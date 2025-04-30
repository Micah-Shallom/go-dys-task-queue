package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	//simulate task generation
	dataStore := generateData()

	//create fixed size worker pool with 5 workers
	dispatcher := NewDispatcher(10)
	dispatcher.Start(ctx)

	go Producer(dataStore, dispatcher)

	// //metrics reporting
	// go func() {
	// 	ticker := time.NewTicker(3 * time.Second)
	// 	defer ticker.Stop()

	// 	for {
	// 		select {
	// 		case <-ctx.Done():
	// 			return
	// 		default:
	// 			fmt.Println("📊 " + dispatcher.metrics.Report())
	// 		}
	// 	}
	// }()

	time.Sleep(45 * time.Second)
	cancel()
	dispatcher.wg.Wait()
	fmt.Println("Shutdown complete")
}
