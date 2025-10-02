package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {
	callsPerLoopStr := os.Getenv("CALLS_PER_LOOP")
	if callsPerLoopStr == "" {
		callsPerLoopStr = "200"
	}
	callsPerLoop, err := strconv.Atoi(callsPerLoopStr)
	if err != nil {
		log.Fatalf("Invalid number for CALLS_PER_LOOP: %v", err)
	}

	apiURL := "https://jsonplaceholder.typicode.com/posts"
	callCount := 0

	log.Printf("🚀 Starting Go API caller. Making %d calls per loop.", callsPerLoop)

	for {
		for i := 0; i < callsPerLoop; i++ {
			resp, err := http.Get(fmt.Sprintf("%s/%d", apiURL, i+1))
			if err != nil {
				log.Printf("❌ Request failed: %v", err)
				continue
			}

			if resp.StatusCode == http.StatusOK {
				callCount++
				log.Printf("✅ Call #%d successful (Status: %s)", callCount, resp.Status)
			} else {
				log.Printf("❌ Call failed with status: %s", resp.Status)
			}
			resp.Body.Close()

			time.Sleep(100 * time.Millisecond)
		}

		log.Printf("Loop finished. Made %d calls. Waiting 10 seconds before next loop.", callsPerLoop)
		time.Sleep(10 * time.Second)
	}
}
