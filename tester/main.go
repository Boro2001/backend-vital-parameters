package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"sync"
	"time"
)

// Function to make a POST request. You need to implement this.
func makePostRequest(wg *sync.WaitGroup, results chan<- float64) {
	// Simulate a POST request with some operation here...
	// For the sake of example, let's assume it returns the time taken for the request
	startTime := time.Now()

	// ... Your HTTP request logic here ...

	duration := time.Since(startTime).Seconds()
	results <- duration // Send the time taken for the request to the results channel

	// Note: No need to call wg.Done() here since it's deferred in the caller goroutine
}

// Function to perform the test with the specified number of users and requests
func performTest(users, requests int) {
	var wg sync.WaitGroup
	results := make(chan float64, users*requests)

	// Start timing the test
	testStartTime := time.Now()

	for i := 0; i < users; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < requests; j++ {
				makePostRequest(&wg, results)
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// Record results and write to CSV file concurrently
	go recordResults(users, requests, results, testStartTime)
}

// Function to record results into a CSV file
func recordResults(users, requests int, results <-chan float64, testStartTime time.Time) {
	// Create a CSV file to store the results
	fileName := fmt.Sprintf("load_test_%d_users.csv", users)
	file, err := os.Create(fileName)
	if err != nil {
		fmt.Printf("Error creating CSV file: %v\n", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header to CSV file
	if err := writer.Write([]string{"Request", "Time (s)"}); err != nil {
		fmt.Printf("Error writing header to CSV file: %v\n", err)
		return
	}

	var totalRequestTime float64
	count := 0

	for result := range results {
		count++
		totalRequestTime += result
		if err := writer.Write([]string{fmt.Sprintf("%d", count), fmt.Sprintf("%f", result)}); err != nil {
			fmt.Printf("Error writing to CSV file: %v\n", err)
			return
		}
	}

	// Calculate and print average request time
	avgRequestTime := totalRequestTime / float64(users*requests)
	totalDuration := time.Since(testStartTime).Seconds()

	fmt.Printf("Test completed for %d users making %d requests each.\n", users, requests)
	fmt.Printf("Total time for test: %f seconds\n", totalDuration)
	fmt.Printf("Average request time: %f seconds\n", avgRequestTime)
}

func main() {
	// Example usage of performTest function:
	// Start test with 1 user making 20 requests
	performTest(1, 20)

	// Wait for input to exit, to see the output in console-based environments
	fmt.Println("Press 'Enter' to exit...")
	fmt.Scanln()
}
