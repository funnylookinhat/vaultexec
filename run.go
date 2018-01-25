package main

// run.go includes functions for running processes with provided environment
// variables.

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

// RunWithEnvVars runs command with the provided environment variables and returns
// a channel for when the error processes.
func RunWithEnvVars(command []string, envVars map[string]string) error {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Add the environment variables to the command.
	env := os.Environ()
	for k, v := range envVars {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = env

	// Start command, trap and send all signals.
	err := cmd.Start()
	if err != nil {
		return err
	}

	sigs := make(chan os.Signal)

	signal.Notify(
		sigs,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGKILL,
		syscall.SIGQUIT,
	)

	// Send any trapped signals to the process, if we fail to pass it on, then
	// return the error to the channel so that the process can quit.
	go func() {
		log.Println("VaultExec - Waiting for Signals")
		// TODO range over rather than read from a channel that you know will close
		// Reading on a closed channel just gives back the zero value[0]
		//
		// [0] - https://dave.cheney.net/2014/03/19/channel-axioms
		for sig := range sigs {
			log.Println("VaultExec - Received Signal: ", sig)
			err := cmd.Process.Signal(sig)
			if err != nil {
				log.Println("VaultExec - Error sending signal to process: ", err)
			}
		}
	}()

	/*
		TODO think about possibility for race condition. What happens if the
		receiver channel closes and signal package tries to send before we shut
		down? Sending on a closed channel panics [0]

		You'll also want to find some way to tell the signal package to stop
		forwarding signals on the channel and then synchronize on that to close down
		resources [1]

		[0] - https://dave.cheney.net/2014/03/19/channel-axioms
		[1] - https://golang.org/pkg/os/signal/#Stop
	*/
	defer close(sigs)

	return cmd.Wait()
}
