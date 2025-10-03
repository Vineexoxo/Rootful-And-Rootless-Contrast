package main

import (
	"log"
	"math/rand"
	"net/http"
	"time"
)

func main() {
	// A list of APIs to choose from randomly.
	apiURLs := []string{
		"https://api.publicapis.org/entries",
		"https://catfact.ninja/fact",
		"https://dogapi.dog/api/v2/facts",
		"https://www.boredapi.com/api/activity",
		"https://api.coindesk.com/v1/bpi/currentprice.json",
		"https://jsonplaceholder.typicode.com/photos", // A "heavy" API call for bigger spikes
	}

	// Create a new random number generator seeded with the current time.
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	callCount := 0
	log.Printf("🚀 Starting Go API caller with bursty random calls.")

	// --- MODIFICATION START ---
	// The outer loop runs forever, choosing a new task each time.
	for {
		// 1. Choose a random API for this entire batch of calls.
		randomIndex := r.Intn(len(apiURLs))
		chosenURL := apiURLs[randomIndex]

		// 2. Choose a random number of times to call this API (e.g., between 10 and 50 times).
		repetitions := r.Intn(41) + 10 // Generates a random number from 10 to 50

		log.Printf("🔥 Starting new burst: Calling %s for %d times.", chosenURL, repetitions)

		// 3. The inner loop runs for the random number of repetitions.
		for i := 0; i < repetitions; i++ {
			resp, err := http.Get(chosenURL)
			if err != nil {
				log.Printf("❌ Request to %s failed: %v", chosenURL, err)
				continue // Skip to the next call in the burst
			}

			if resp.StatusCode == http.StatusOK {
				callCount++
				log.Printf("✅ Call #%d to %s successful (Status: %s)", callCount, chosenURL, resp.Status)
			} else {
				log.Printf("❌ Call to %s failed with status: %s", chosenURL, resp.Status)
			}
			resp.Body.Close()

			// A short delay between each call within the burst.
			time.Sleep(200 * time.Millisecond)
		}

		log.Printf("✅ Burst finished. Waiting before starting a new one.")
		// A longer pause between bursts to make the changes more noticeable.
		time.Sleep(5 * time.Second)
	}
	// --- MODIFICATION END ---
}

// package main

// import (
// 	"fmt"
// 	"log"
// 	"net/http"
// 	"os"
// 	"strconv"
// 	"time"
// )

// func main() {
// 	callsPerLoopStr := os.Getenv("CALLS_PER_LOOP")
// 	if callsPerLoopStr == "" {
// 		callsPerLoopStr = "200"
// 	}
// 	callsPerLoop, err := strconv.Atoi(callsPerLoopStr)
// 	if err != nil {
// 		log.Fatalf("Invalid number for CALLS_PER_LOOP: %v", err)
// 	}

// 	// apiURL := "https://jsonplaceholder.typicode.com/posts"
// 	apiURL := "https://dogapi.dog/api/v2/groups"
// 	callCount := 0

// 	log.Printf("🚀 Starting Go API caller. Making %d calls per loop.", callsPerLoop)

// 	for {
// 		for i := 0; i < callsPerLoop; i++ {
// 			resp, err := http.Get(fmt.Sprintf("%s/%d", apiURL, i+1))
// 			if err != nil {
// 				log.Printf("❌ Request failed: %v", err)
// 				continue
// 			}

// 			if resp.StatusCode == http.StatusOK {
// 				callCount++
// 				log.Printf("✅ Call #%d successful (Status: %s)", callCount, resp.Status)
// 			} else {
// 				log.Printf("❌ Call failed with status: %s", resp.Status)
// 			}
// 			resp.Body.Close()

// 			time.Sleep(100 * time.Millisecond)
// 		}

// 		log.Printf("Loop finished. Made %d calls. Waiting 10 seconds before next loop.", callsPerLoop)
// 		time.Sleep(10 * time.Second)
// 	}
// }
