package main

import (
	"fmt"

	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/orchestrator"
)

func main() {
	const numWorkers = 10
	jobs := make(chan models.Submission, 100)
	results := make(chan []models.Verdict, 100)

	// Start worker pool
	for w := 1; w <= numWorkers; w++ {
		go orchestrator.InitializeWorker(w, jobs, results)
	}

	// Simulate influx of submissions
	for i := 1; i <= 50; i++ {
		jobs <- models.Submission{
			Id:            fmt.Sprint(i),
			Language:      "c",
			BuildFiles:    []string{"array.c", "ganjilgenap.c"},
			TCInputFiles:  []string{"1.in", "2.in", "3.in", "4.in", "5.in", "6.in", "7.in", "8.in", "9.in", "10.in"},
			TCOutputFiles: []string{"1.out", "2.out", "3.out", "4.out", "5.out", "6.out", "7.out", "8.out", "9.out", "10.out"},
		}
	}
	close(jobs)

	// Collect results
	for i := 1; i <= 50; i++ {
		fmt.Println(<-results)
	}
}
