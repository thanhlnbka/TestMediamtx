package main

import (
	"bytes"
	"fmt"
	"net/http"
	"time"
)

func main() {
	url := "http://localhost:5000/verify_token"
	jsonStr := []byte(`{"token": "secret_token", "id":"dasdasd"}`)
	maxRetries := 3 // maximum number of retries
	retries := 0    // current number of retries

	for retries < maxRetries {
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
		if err != nil {
			panic(err)
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			fmt.Printf("Retrying in 2 seconds...\n")
			time.Sleep(2 * time.Second) // sleep for 2 seconds before retrying
			retries++
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			fmt.Println("Status: OK")
			// Process response here
			break
		} else {
			fmt.Println("Status: Unauthorized")
			// Process error response here
			break
		}
	}

	if retries == maxRetries {
		fmt.Println("Maximum retries reached")
		// Handle maximum retries reached error here
	}
}
