package main

// vault.go provides the mechanisms and configurations to fetch secrets from vault.

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/funnylookinhat/easyjson"
)

// VaultConfig is a set of values for reading secrets from a Vault server over HTTP.
type VaultConfig struct {
	Address string // e.g. https://path.to.vault:8200
	Token   string
	Path    string // The path to the secrets to dump.
}

// GenerateVaultConfig using arguments and environment variables: VAULT_ADDR,
// VAULT_TOKEN, and VAULT_PATH
func GenerateVaultConfig(address *string, token *string, path *string) VaultConfig {
	config := VaultConfig{}

	// Then if any options are still blank we read the environment variables.
	if len(*address) > 0 {
		config.Address = *address
	} else {
		config.Address = os.Getenv("VAULT_ADDR")
	}

	// Ensure that the address doesn't end in a trailing slash.
	if strings.HasSuffix(config.Address, "/") {
		config.Address = config.Address[:len(config.Address)-1]
	}

	if len(*token) > 0 {
		config.Token = *token
	} else {
		config.Token = os.Getenv("VAULT_TOKEN")
	}

	if len(*path) > 0 {
		config.Path = *path
	} else {
		config.Path = os.Getenv("VAULT_PATH")
	}

	return config
}

// ValidateVaultConfig ensure that all required values are present
func ValidateVaultConfig(config VaultConfig) error {
	if len(config.Address) == 0 {
		return errors.New("Missing Vault address")
	}

	if len(config.Path) == 0 {
		return errors.New("Missing Vault secret path")
	}

	if len(config.Token) == 0 {
		return errors.New("Missing Vault token")
	}

	return nil
}

// GetVaultSecrets fetches secrets from vault and returns a map[string]string
func GetVaultSecrets(config VaultConfig) (map[string]string, error) {
	client := &http.Client{}

	requestURL := fmt.Sprintf("%s/v1/%s", config.Address, config.Path)

	req, err := http.NewRequest("GET", requestURL, nil)

	if err != nil {
		return nil, err
	}

	req.Header.Add("X-Vault-Token", config.Token)

	resp, err := client.Do(req)

	defer resp.Body.Close()

	if err != nil {
		return nil, err
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	bodyJSON, err := easyjson.DecodeJson(bodyBytes)

	if err != nil {
		return nil, err
	}

	vaultData, err := easyjson.GetMap(bodyJSON, "data")

	if err != nil {
		return nil, err
	}

	vaultSecrets := make(map[string]string)

	for k, v := range vaultData {
		vaultSecrets[fmt.Sprintf("%s", k)] = fmt.Sprintf("%s", v)
	}

	return vaultSecrets, nil
}
