package main

// vault.go provides the mechanisms and configurations to fetch secrets from vault.

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// VaultConfig is a set of values for reading secrets from a Vault server over HTTP.
type VaultConfig struct {
	Address string // e.g. https://path.to.vault:8200
	Token   string
	Path    string // The path to the secrets to dump.
}

// VaultSecretResponse is a partial representation of the reponse that comes
// back when fetching secrets.
type VaultSecretResponse struct {
	Errors []string `json:"errors"`
	// The data that comes back for secrets can be of any type, but the keys will
	// always be strings.  So rather than have map[string]string, which fails to
	// unmarshal, we just use map[string]interface{}
	Data map[string]interface{} `json:"data"`
}

// VaultRenewResponse handles fields we care about from renewing the token.
type VaultRenewResponse struct {
	Errors []string `json:"errors"`
	Auth   struct {
		LeaseDuration int64 `json:"lease_duration"`
	}
}

// GenerateVaultConfig using arguments and environment variables: VAULT_ADDR,
// VAULT_TOKEN, and VAULT_PATH
func GenerateVaultConfig(address *string, token *string, path *string) (VaultConfig, error) {
	config := VaultConfig{
		Address: *address,
		Token:   *token,
		Path:    *path,
	}

	// Then if any options are still blank we read the environment variables.
	if len(config.Address) == 0 {
		config.Address = os.Getenv("VAULT_ADDR")
	}
	if len(config.Token) == 0 {
		config.Token = os.Getenv("VAULT_TOKEN")
	}
	if len(config.Path) == 0 {
		config.Path = os.Getenv("VAULT_PATH")
	}

	// Ensure that the address doesn't end in a trailing slash.
	if strings.HasSuffix(config.Address, "/") {
		config.Address = config.Address[:len(config.Address)-1]
	}

	if len(config.Address) == 0 {
		return config, errors.New("missing vault address")
	}

	_, err := url.ParseRequestURI(config.Address)

	if err != nil {
		return config, fmt.Errorf("invalid vault address: %s", err)
	}

	if len(config.Path) == 0 {
		return config, errors.New("missing vault secret path")
	}

	if len(config.Token) == 0 {
		return config, errors.New("missing vault token")
	}

	return config, nil
}

// Make a request to the vault service with a given method.
func makeVaultRequest(method string, path string, config VaultConfig) ([]byte, error) {
	client := &http.Client{}

	requestURL := fmt.Sprintf("%s/%s", config.Address, path)

	req, err := http.NewRequest(method, requestURL, nil)

	if err != nil {
		return nil, err
	}

	req.Header.Add("X-Vault-Token", config.Token)

	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if err != nil {
		return nil, err
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	if len(bodyBytes) == 0 {
		return nil, fmt.Errorf(
			"vault server error (HTTP status %d): empty response",
			resp.StatusCode)
	}

	return bodyBytes, nil
}

// GetVaultSecrets fetches secrets from vault and returns a map[string]interface{}
func GetVaultSecrets(config VaultConfig) (map[string]interface{}, error) {
	bodyBytes, err := makeVaultRequest("GET", "v1/"+config.Path, config)

	if err != nil {
		return nil, err
	}

	var vaultSecretResponse VaultSecretResponse

	err = json.Unmarshal(bodyBytes, &vaultSecretResponse)

	if err != nil {
		return nil, err
	}

	if len(vaultSecretResponse.Errors) > 0 {
		return nil, fmt.Errorf(
			"vault server error: %s",
			strings.Join(vaultSecretResponse.Errors, ","))
	}

	return vaultSecretResponse.Data, nil
}

// RenewVaultToken attempts to renew the token provided in the config, returns
// the lease expiration and an error.
func RenewVaultToken(config VaultConfig) (int64, error) {
	bodyBytes, err := makeVaultRequest("POST", "v1/auth/token/renew-self", config)

	if err != nil {
		return 0, err
	}

	var vaultRenewResponse VaultRenewResponse

	err = json.Unmarshal(bodyBytes, &vaultRenewResponse)

	if err != nil {
		return 0, err
	}

	if len(vaultRenewResponse.Errors) > 0 {
		return 0, fmt.Errorf(
			"vault server error: %s",
			strings.Join(vaultRenewResponse.Errors, ","))
	}

	return vaultRenewResponse.Auth.LeaseDuration, nil
}
