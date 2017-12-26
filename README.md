# VaultExec

The least intrusive way to use Vault with your application.  VaultExec will
fetch secrets, add them to the environment, and then launch your application
(passing along any signals it should receive).  Ideally, most environments
should be able to simply prefix the run command with `vaultexec`.

## Usage

VaultExec can be configured both by command line options and environment variables:

- Address of vault server:
    - `-address http://vault.host:8200`
    - `VAULT_ADDR`
- Vault access token:
    - `-token xxxxxxxx-yyyy-yyyy-yyyy-xxxxxxxxxxxx`
    - `VAULT_TOKEN`
- Vault secret path:
    - `-path secrets/for/my/app`
    - `VAULT_PATH`

## Getting VaultExec

Check out the releases and grab the appropriately built binary: https://github.com/funnylookinhat/vaultexec/releases

## Building VaultExec Locally

Requirements:

- govendor
- gox

Run the following to generate release binaries:

`gox -output="bin/{{.Dir}}_{{.OS}}_{{.Arch}}"`
