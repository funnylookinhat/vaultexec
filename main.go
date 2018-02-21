package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
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

	flag.Parse()

	cmd := flag.Args()

	if len(cmd) == 0 {
		errCheck(errors.New("Must provide a command"))
	}

	config, err := GenerateVaultConfig(address, token, path)
	errCheck(err)

	vaultSecrets, err := GetVaultSecrets(config)
	errCheck(err)

	// TODO - Renew vault secret.

	// This is a blocking call that runs several go-funcs to manage sending
	// signals to the process.

	errCheck(RunWithEnvVars(cmd, vaultSecrets))

	os.Exit(0)
}
