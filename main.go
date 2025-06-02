package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	taskStream := make(chan Task, 1000)

	//create fixed size worker pool with 5 workers
	dispatcher := NewDispatcher(2)
	go TaskFeeder(ctx, taskStream)

	dispatcher.Start(ctx)
	go Producer(ctx, taskStream, dispatcher)

	//metrics reporting
	setupMetricsReporting(ctx, dispatcher)

	
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Wait for termination signal
	sig := <-sigCh
	fmt.Printf("Received signal: %s. Shutting down...\n", sig)

	cancel()
	dispatcher.StopDispatch()
	dispatcher.wg.Wait()
	fmt.Println("Shutdown complete")
}
