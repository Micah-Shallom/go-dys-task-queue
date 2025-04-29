package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dispatcher := NewDispatcher(10, 5)
	dispatcher.Start(ctx)

	//simulate task submission
	go func() {
		for i := 1; i < 1_000; i++ {
			fmt.Println("Running")
			priority := i%3 + 1 //cycle through priorities 1,2,3
			task := Task{ID: i, Priority: priority, Name: fmt.Sprintf("Task-%d", i)}
			dispatcher.Submit(task)
			time.Sleep(500 * time.Millisecond)
		}
	}()

	//metrics reporting
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				fmt.Println(dispatcher.metrics.Report())
				time.Sleep(2 * time.Second)
			}
		}
	}()

	time.Sleep(5 * time.Second)
	cancel()
	dispatcher.wg.Wait()
	fmt.Println("Shutdown complete")
}
