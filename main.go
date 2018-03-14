package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

// Simple function to clean up golang error checking for main()
func errCheck(err error) {
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "vaultexec - Run commands with secrets from Vault.\n")
		fmt.Fprintf(os.Stderr, "Usage: vaultexec [options] command arg1 arg2 arg3\n")
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "Providing any command line option will override the equivalent environment variable.\n")
	}

	// First read command line options.
	address := flag.String("address", "", "https://path.to.vault:8200 - Can also be set with the ENV VAULT_ADDR")
	token := flag.String("token", "", "xxxxxxxx-yyyy-yyyy-yyyy-xxxxxxxxxxxx - Can also be set with the ENV VAULT_TOKEN")
	path := flag.String("path", "", "path/to/secrets/location - Can also be set with the ENV VAULT_PATH")
	generateConfig := flag.String(
		"generate-config",
		"",
		`A command to run to generate the vault config.
		Will be passed all environment variables that were passed to VaultExec, along with any of the
		flags that were passed to vaultexec (as environment variables).
		Must output a JSON formatted object with an address, token, and path key to stdout.`)

	flag.Parse()

	cmd := flag.Args()

	if len(cmd) == 0 {
		errCheck(errors.New("Must provide a command"))
	}

	config, err := NewVaultConfig(address, token, path)
	errCheck(err)

	if len(*generateConfig) > 0 {
		config, err = GenerateVaultConfig(generateConfig, config)
		errCheck(err)
	}

	errCheck(ValidateVaultConfig(config))

	vaultSecrets, err := GetVaultSecrets(config)
	errCheck(err)

	// Renew the token periodically (half of every lease duration), starting
	// right now.
	go func() {
		leaseTimeout := 0 * time.Second
		for {
			time.Sleep(leaseTimeout * time.Second)
			leaseDuration, err := RenewVaultToken(config)
			if err != nil {
				log.Printf("error renewing vault token: %s", err)
				// If there was an error renewing the token, it should stop trying to
				// renew (otherwise it will repeatedly try to renew with no delay)
				return
			}
			leaseTimeout = time.Duration(leaseDuration) / 2
		}
	}()

	// This is a blocking call that runs several go-funcs to manage sending
	// signals to the process.
	errCheck(RunWithEnvVars(cmd, vaultSecrets))

	os.Exit(0)
}
