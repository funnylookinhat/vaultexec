package main

// This is a utility binary for testing the signal control for vaultexec.

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
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
		fmt.Println("SignalEcho - Waiting for signals...")
		for {
			sig := <-sigs
			fmt.Println("SignalEcho - Received Signal: ", sig)
		}
	}()

	defer close(sigs)

	// Wait forever
	select {}
}
