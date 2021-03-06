# VaultExec

The least intrusive way to use Vault with your application.  VaultExec will
fetch secrets, add them to the environment, and then launch your application
(passing along any signals it should receive).  Ideally, most environments
should be able to simply prefix the run command with `vaultexec`.

## Usage

VaultExec can be configured both by command line options and environment variables:

- Address of vault server:
    - Option: `-address http://vault.host:8200`
    - Environment: `VAULT_ADDR`
- Vault access token:
    - Option: `-token xxxxxxxx-yyyy-yyyy-yyyy-xxxxxxxxxxxx`
    - Environment: `VAULT_TOKEN`
- Vault secret path:
    - Option: `-path secrets/for/my/app`
    - Environment: `VAULT_PATH`
    - You can provide multiple comma-separated paths within the same argument.
    - Note that secret paths will be read in order, and if a key already exists
      it will be overwritten by a later secret if it has the same key.
    - If commas are required for your path names, you can change teh delimiter.
- Vault secret path delimiter:
    - Option: `-path-delim ,`
    - Environment: `VAULT_PATH_DELIM`
- Additionally, you can provide a binary command to run to generate a vault config:
    - Option: `--generate-config some-binary`
    - This will be run with the environment variables that were passed to VaultExec
      along with appending any address, token, or secret that was passed as
      command line arguments.
    - This command MUST return only JSON in stdout; it may have any of the following attributes: address, token, path
    - The returned values will be merged with the configuration that vaultexec was started with.

## Examples

**With environment variables:**

```
export VAULT_ADDR=http://my.vault.host:8200
export VAULT_TOKEN=a44cb316-4bf9-4c16-bbed-ae37e068683d
export VAULT_PATH=secrets/my-app/test/all
vaultexec myapp
```

**With options:**

```
vaultexec -address http://my.vault.host:8200 \
  -token a44cb316-4bf9-4c16-bbed-ae37e068683d \
  -path secrets/for/my/app \
  myapp
```

**With generate-config:**
```
# some-generator must be in the PATH
vaultexec --generate-config some-generator myapp
```

**With multiple secret paths:**
```
vaultexec -address http://my.vault.host:8200 \
  -token a44cb316-4bf9-4c16-bbed-ae37e068683d \
  -path secrets/for/my/app,secrets/another/secret/path \
  myapp
```

**Or using a custom delimiter:**
```
vaultexec -address http://my.vault.host:8200 \
  -token a44cb316-4bf9-4c16-bbed-ae37e068683d \
  -path secrets/for/my/app,1__secrets/for/my/app,2 \
  -path-delim __ \
  myapp
```

**In a Dockerfile:**

```
FROM node:8.9.2-alpine

# Install VaultExec
ADD https://github.com/funnylookinhat/vaultexec/releases/download/v0.0.2/vaultexec_linux_amd64 /usr/local/bin/vaultexec
RUN chmod +x /usr/local/bin/vaultexec

# Add your files, etc.

CMD ["vaultexec", "node", "/app/server.js"]
```

## Getting VaultExec

Check out the releases and grab the appropriately built binary:
https://github.com/funnylookinhat/vaultexec/releases

## Building VaultExec Locally

Requirements:

- gox

Run the following to generate release binaries for all platforms:

`gox -output="bin/{{.Dir}}_{{.OS}}_{{.Arch}}" -tags='netgo' -ldflags='-w'`

## Testing Locally

Requirements:

- Docker Compose

Start up the docker container:

`docker-compose up`

This should print out the values from `test/secrets.json` along with the rest
of the environment.

If you want to do more thorough testing, the easiest mechanism is to generate
binaries from within docker and test there as well.  For example:

```
docker-compose run app sh
cd test/
go build -o signal_echo signal_echo.go
cd ../
go build -o vaultexec main.go run.go vault.go
./vaultexec test/signal_echo
```

At this point you should have signal_echo running with the provided environment
variables, and printing out any signals that are send.  Hitting `Control+C`
should print out an `Interrupt` message received from each binary.

Hit `Control+Z` to send the process to the background.

```
/go/src/vaultexec # ./vaultexec test/signal_echo
2017/12/28 18:43:19 VaultExec - Waiting for Signals
SignalEcho - Waiting for signals...
^C2017/12/28 18:43:20 VaultExec - Received Signal:  interrupt
SignalEcho - Received Signal:  interrupt
^Z[1]+  Stopped                    ./vaultexec test/signal_echo
```

Find the PID with `ps aux`:

```
/go/src/vaultexec # ps aux
PID   USER     TIME   COMMAND
    1 root       0:00 sh
   48 root       0:00 ./vaultexec test/signal_echo
   54 root       0:00 test/signal_echo
   62 root       0:00 ps aux
```

You can kill the process manually with `kill -9`:

```
/go/src/vaultexec # kill -9 54
```

And get any remaining output by bringing the vaultexec process back to the foreground:

```
/go/src/vaultexec # fg
./vaultexec test/signal_echo
2017/12/28 18:45:39 signal: killed
```
