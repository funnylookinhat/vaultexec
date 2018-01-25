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
//
// TODO: Verify & document that command-line args *always* take precedence over
// environment variables and document it as such. If you intend to run this in a
// container or something, you lose the ability to change values from the host
// environment without changing the Docker image since the command-line
// variables take precedent.
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

	// TODO validate Address has a protocol (https, etc.). Could cause an error in
	// GetVaultSecrets you could catch now.
	if len(config.Address) == 0 {
		return config, errors.New("Missing Vault address")
	}

	if len(config.Path) == 0 {
		return config, errors.New("Missing Vault secret path")
	}

	if len(config.Token) == 0 {
		return config, errors.New("Missing Vault token")
	}

	return config, nil
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
	// TODO Handle vault return status code
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("vault server returned status: %d", resp.StatusCode)
	}

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if err != nil {
		return nil, err
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	// TODO error on empty bodyBytes

	if err != nil {
		return nil, err
	}

	bodyJSON, err := easyjson.DecodeJson(bodyBytes)

	if err != nil {
		return nil, err
	}

	/*
		// Here is an alternative to simplify your JSON extraction since the desired
		// data is just a string->string object anyway. You don't get to use your
		// easyjson package, but since the shape of the data is simple, you can just
		// use an anonymous struct to describe what you want, and then just return
		// the data directly.

		var data struct {
			Data map[string]string `json:"data"`
		}
		if err := json.Unmarshal(bodyBytes, &data); err != nil {
			return nil, err
		}
		return data
	*/

	vaultData, err := easyjson.GetMap(bodyJSON, "data")

	if err != nil {
		return nil, err
	}

	vaultSecrets := make(map[string]string)

	// TODO Don't allocate a new string for k when the key type from GetMap is
	// already a string. All of this is just because the value type of GetMap is
	// interface{}?
	for k, v := range vaultData {
		vaultSecrets[fmt.Sprintf("%s", k)] = fmt.Sprintf("%s", v)
	}

	return vaultSecrets, nil
}
