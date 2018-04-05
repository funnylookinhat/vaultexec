package main

// vault.go provides the mechanisms and configurations to fetch secrets from vault.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

// VaultConfig is a set of values for reading secrets from a Vault server over HTTP.
type VaultConfig struct {
	Address   string `json:"address"` // e.g. https://path.to.vault:8200
	Token     string `json:"token"`
	Path      string `json:"path"`       // The path to the secrets to dump.
	PathDelim string `json:"path-delim"` // Delimeter for multiple paths
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

// VaultLookupTokenResponse is used just for determining renewability
type VaultLookupTokenResponse struct {
	Errors []string `json:"errors"`
	Data   struct {
		Renewable bool `json:"renewable"`
	}
}

// GenerateVaultConfig creates a new vault config by running a given command on
// the system.  Will merge the passed in config with the environment variables
// passed to vaultexec to run the command.
func GenerateVaultConfig(generateConfig *string, config VaultConfig) (VaultConfig, error) {
	cmd := exec.Command(*generateConfig)

	var stdoutBytes bytes.Buffer
	cmd.Stdout = &stdoutBytes

	// We'll just pipe stderr back to stderr
	cmd.Stderr = os.Stderr

	// Merge vault config environment variables
	env := os.Environ()
	if len(config.Address) > 0 {
		env = append(env, fmt.Sprintf("VAULT_ADDR=%s", config.Address))
	}
	if len(config.Token) > 0 {
		env = append(env, fmt.Sprintf("VAULT_TOKEN=%s", config.Token))
	}
	if len(config.Path) > 0 {
		env = append(env, fmt.Sprintf("VAULT_PATH=%s", config.Path))
	}
	if len(config.PathDelim) > 0 {
		env = append(env, fmt.Sprintf("VAULT_PATH_DELIM=%s", config.PathDelim))
	}
	cmd.Env = env

	err := cmd.Run()
	if err != nil {
		return config, err
	}

	var stdoutVaultConfig VaultConfig

	err = json.Unmarshal(stdoutBytes.Bytes(), &stdoutVaultConfig)

	if err != nil {
		return config, err
	}

	if len(stdoutVaultConfig.Address) > 0 {
		config.Address = stdoutVaultConfig.Address
	}
	if len(stdoutVaultConfig.Token) > 0 {
		config.Token = stdoutVaultConfig.Token
	}
	if len(stdoutVaultConfig.Path) > 0 {
		config.Path = stdoutVaultConfig.Path
	}
	if len(stdoutVaultConfig.PathDelim) > 0 {
		config.PathDelim = stdoutVaultConfig.PathDelim
	}

	return config, nil
}

// NewVaultConfig creates a new VaultConfig by handling the parameters and
// substituting env when appropriate
func NewVaultConfig(address *string, token *string, path *string, pathDelim *string) (VaultConfig, error) {
	config := VaultConfig{
		Address:   *address,
		Token:     *token,
		Path:      *path,
		PathDelim: *pathDelim,
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

	// Because we default path delimeter to a comma, we check if it's blank or
	// if it's the default value - and then only swap in the environment value if
	// it's not blank.
	if len(config.PathDelim) == 0 || config.PathDelim == "," {
		if len(os.Getenv("VAULT_PATH_DELIM")) != 0 {
			config.PathDelim = os.Getenv("VAULT_PATH_DELIM")
		}
	}

	// Ensure that the address doesn't end in a trailing slash.
	if strings.HasSuffix(config.Address, "/") {
		config.Address = config.Address[:len(config.Address)-1]
	}

	return config, nil
}

// ValidateVaultConfig validates a given vaultconfig and returns an error if invalid.
func ValidateVaultConfig(config VaultConfig) error {

	if len(config.Address) == 0 {
		return errors.New("missing vault address")
	}

	_, err := url.ParseRequestURI(config.Address)

	if err != nil {
		return fmt.Errorf("invalid vault address: %s", err)
	}

	if len(config.Path) == 0 {
		return errors.New("missing vault secret path")
	}

	if len(config.Token) == 0 {
		return errors.New("missing vault token")
	}

	if len(config.PathDelim) == 0 {
		return errors.New("missing vault secret path delimeter")
	}

	return nil
}

// Make a request to the vault service with a given method.
func makeVaultRequest(method string, path string, config VaultConfig) ([]byte, error) {
	client := &http.Client{}

	requestURL := config.Address + "/" + path

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

// GetVaultSecrets loops through all of the secret paths that are provided and
// returns a single map representing the merged results of every lookup from vault.
func GetVaultSecrets(config VaultConfig) (map[string]interface{}, error) {
	var err error
	var secrets map[string]interface{}

	// These are the secrets we will return by merging the results of each fetch.
	mergedSecrets := make(map[string]interface{})

	paths := strings.Split(config.Path, config.PathDelim)

	for _, path := range paths {
		secrets, err = GetVaultSecretsAtPath(path, config)
		if err != nil {
			return nil, err
		}

		for k, v := range secrets {
			mergedSecrets[k] = v
		}
	}

	return mergedSecrets, nil
}

// GetVaultSecretsAtPath does a lookup for a specific secret path from vault
// and returns a map with the result.
func GetVaultSecretsAtPath(path string, config VaultConfig) (map[string]interface{}, error) {
	bodyBytes, err := makeVaultRequest("GET", "v1/"+path, config)

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

// GetVaultTokenRenewable returns whether or not a VaultConfig has a renewable token
func GetVaultTokenRenewable(config VaultConfig) (bool, error) {
	bodyBytes, err := makeVaultRequest("GET", "v1/auth/token/lookup-self", config)

	if err != nil {
		return false, err
	}

	var vaultLookupTokenResponse VaultLookupTokenResponse

	err = json.Unmarshal(bodyBytes, &vaultLookupTokenResponse)

	if err != nil {
		return false, err
	}

	if len(vaultLookupTokenResponse.Errors) > 0 {
		return false, fmt.Errorf(
			"vault server error: %s",
			strings.Join(vaultLookupTokenResponse.Errors, ","))
	}

	return vaultLookupTokenResponse.Data.Renewable, nil
}
