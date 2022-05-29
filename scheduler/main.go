package main

import (
	"fmt"
	"log"
	"time"

	"github.com/madflojo/tasks"
)

// https://github.com/madflojo/tasks/blob/main/tasks_test.go
func main() {
	// Start the Scheduler
	scheduler := tasks.New()
	defer scheduler.Stop()

	// Add a one time only task for 60 seconds from now
	id, err := scheduler.Add(&tasks.Task{
		Interval: 15 * time.Second,
		RunOnce:  true,
		TaskFunc: func() error {
			fmt.Println(">>>>")
			return nil
		},
		ErrFunc: func(e error) {
			log.Printf("An error occurred when executing task %v", e)
		},
	})
	if err != nil {
		// Do Stuff
	}
	fmt.Println(id)

	// select {}
	time.Sleep(1 * time.Hour)

	//// Add a task
	//id, err := scheduler.Add(&tasks.Task{
	//	Interval: time.Duration(30 * time.Second),
	//	TaskFunc: func() error {
	//		// Put your logic here
	//	}(),
	//})
	//if err != nil {
	//	// Do Stuff
	//}
}
