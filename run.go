package main

// run.go includes functions for running processes with provided environment
// variables.

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

// RunWithEnvVars runs command with the provided environment variables and returns
// a channel for when the error processes.
func RunWithEnvVars(command []string, envVars map[string]string) (chan error, error) {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Add the environment variables to the command.
	env := os.Environ()
	for k, v := range envVars {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = env

	// Start - and then wait for an exit.
	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	// Trap all signals and pass them on to the process.

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
		sig := <-sigs
		err := cmd.Process.Signal(sig)
		if err != nil {
			done <- err
		}
	}()

	return done, nil
}
