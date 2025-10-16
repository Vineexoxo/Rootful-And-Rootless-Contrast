package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"
)

// LargeResponseSize is increased to 50 MB to heavily stress network I/O throughput.
const LargeResponseSize = 50 * 1024 * 1024 // 50 MB of data

// LargePayload will hold a pre-allocated large byte slice of data.
var LargePayload []byte

func init() {
	// Initialize the large payload once at startup.
	// We use simple bytes instead of strings for slightly better performance.
	LargePayload = make([]byte, LargeResponseSize)
	for i := 0; i < LargeResponseSize; i++ {
		LargePayload[i] = byte(i % 256)
	}

	log.Printf("Payload initialized to %d bytes (%.2f MB).", LargeResponseSize, float64(LargeResponseSize)/(1024*1024))
}

// stressHandler simulates a workload that triggers high Network I/O and stresses the system's GC.
func stressHandler(w http.ResponseWriter, r *http.Request) {
	// --- I/O Stress ---
	// Set headers for a large binary transfer
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", LargeResponseSize))

	// Write the large payload. This forces high network throughput,
	// which is the weakest area for rootless user-space networking stacks.
	_, err := w.Write(LargePayload)
	if err != nil {
		// Log error, but don't stop the server
		log.Printf("Error writing response: %v", err)
	}

	// --- GC Stress (Simulating memory pressure) ---
	// Force the Go runtime to trigger garbage collection frequently for comparison.
	// This increases the frequency of syscalls related to memory management (freeing memory to the OS),
	// potentially magnifying the overhead of User Namespace ID mapping.
	// This is critical for showing CPU overhead difference.
	debug.FreeOSMemory()

	// No sleep to maximize throughput.
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	http.HandleFunc("/", stressHandler)

	log.Printf("🔥 Starting EXTREME I/O Stress Server on port %s", port)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// root full
// vihanvashishth@Vihans-MacBook-Air api_caller % wrk -t4 -c10 -d30s http://localhost:8082/
// Running 30s test @ http://localhost:8082/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency     1.06s   118.33ms   1.61s    86.10%
//     Req/Sec     2.57      2.75    10.00     89.15%
//   223 requests in 30.02s, 11.03GB read
// Requests/sec:      7.43
// Transfer/sec:    376.20MB

// vihanvashishth@Vihans-MacBook-Air api_caller % wrk -t4 -c10 -d30s http://localhost:8082/
// Running 30s test @ http://localhost:8082/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency     1.13s   132.96ms   1.80s    72.55%
//     Req/Sec     3.31      3.43    10.00     82.47%
//   205 requests in 30.05s, 10.21GB read
//   Socket errors: connect 0, read 0, write 0, timeout 1
// Requests/sec:      6.82
// Transfer/sec:    347.86MB

// vihanvashishth@Vihans-MacBook-Air api_caller % wrk -t4 -c10 -d30s http://localhost:8082/
// Running 30s test @ http://localhost:8082/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency     1.14s   159.44ms   1.86s    82.21%
//     Req/Sec     2.84      3.14    10.00     86.07%
//   208 requests in 30.03s, 10.30GB read
// Requests/sec:      6.93
// Transfer/sec:    351.08MB

// vihanvashishth@Vihans-MacBook-Air api_caller % wrk -t4 -c10 -d30s http://localhost:8082/
// Running 30s test @ http://localhost:8082/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency     1.18s   138.84ms   1.62s    79.10%
//     Req/Sec     2.56      2.89    10.00     89.29%
//   201 requests in 30.02s, 9.90GB read
// Requests/sec:      6.70
// Transfer/sec:    337.72MB

// vihanvashishth@Vihans-MacBook-Air api_caller % wrk -t4 -c10 -d30s http://localhost:8082/
// Running 30s test @ http://localhost:8082/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency     1.14s   105.49ms   1.72s    72.41%
//     Req/Sec     2.69      3.07    10.00     86.70%
//   204 requests in 30.01s, 10.15GB read
//   Socket errors: connect 0, read 0, write 0, timeout 1
// Requests/sec:      6.80
// Transfer/sec:    346.47MB

// vihanvashishth@Vihans-MacBook-Air api_caller % wrk -t4 -c10 -d30s http://localhost:8082/
// Running 30s test @ http://localhost:8082/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency     1.16s   137.22ms   1.83s    82.76%
//     Req/Sec     2.08      2.47    10.00     90.72%
//   203 requests in 30.01s, 10.07GB read
// Requests/sec:      6.77
// Transfer/sec:    343.54MB

// vihanvashishth@Vihans-MacBook-Air api_caller % wrk -t4 -c10 -d30s http://localhost:8082/
// Running 30s test @ http://localhost:8082/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency     1.16s    59.58ms   1.35s    84.58%
//     Req/Sec     3.41      3.67    10.00     79.29%
//   201 requests in 30.01s, 10.07GB read
// Requests/sec:      6.70
// Transfer/sec:    343.69MB

// vihanvashishth@Vihans-MacBook-Air api_caller % wrk -t4 -c10 -d30s http://localhost:8082/
// Running 30s test @ http://localhost:8082/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency     1.23s    75.45ms   1.54s    84.38%
//     Req/Sec     3.36      3.53    10.00     83.15%
//   192 requests in 30.01s, 9.49GB read
// Requests/sec:      6.40
// Transfer/sec:    323.91MB

// vihanvashishth@Vihans-MacBook-Air api_caller % wrk -t4 -c10 -d30s http://localhost:8082/
// Running 30s test @ http://localhost:8082/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency     1.15s   107.18ms   1.48s    73.66%
//     Req/Sec     3.02      3.21    10.00     85.85%
//   205 requests in 30.01s, 10.16GB read
// Requests/sec:      6.83
// Transfer/sec:    346.64MB

// vihanvashishth@Vihans-MacBook-Air api_caller % wrk -t4 -c10 -d30s http://localhost:8082/
// Running 30s test @ http://localhost:8082/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency     1.27s    95.37ms   1.47s    58.70%
//     Req/Sec     3.23      3.92    10.00     78.08%
//   184 requests in 30.01s, 9.25GB read
// Requests/sec:      6.13
// Transfer/sec:    315.48MB

// rootless
// vihanvashishth@Vihans-MacBook-Air api_caller % wrk -t4 -c10 -d30s http://localhost:8081/

// Running 30s test @ http://localhost:8081/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency     1.13s   152.42ms   1.80s    80.38%
//     Req/Sec     2.79      3.00    10.00     88.20%
//   209 requests in 30.08s, 10.36GB read
// Requests/sec:      6.95
// Transfer/sec:    352.79MB

// vihanvashishth@Vihans-MacBook-Air api_caller % wrk -t4 -c10 -d30s http://localhost:8082/
// Running 30s test @ http://localhost:8082/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency     1.12s   118.09ms   1.50s    73.81%
//     Req/Sec     2.92      2.75    10.00     85.24%
//   210 requests in 30.01s, 10.50GB read
// Requests/sec:      7.00
// Transfer/sec:    358.15MB

// vihanvashishth@Vihans-MacBook-Air api_caller % wrk -t4 -c10 -d30s http://localhost:8082/
// Running 30s test @ http://localhost:8082/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency     1.20s   100.66ms   1.55s    77.32%
//     Req/Sec     2.73      2.93    10.00     89.89%
//   194 requests in 30.01s, 9.69GB read
// Requests/sec:      6.46
// Transfer/sec:    330.75MB

// vihanvashishth@Vihans-MacBook-Air api_caller % wrk -t4 -c10 -d30s http://localhost:8082/
// Running 30s test @ http://localhost:8082/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency     1.33s    99.57ms   1.55s    61.93%
//     Req/Sec     3.16      3.60    10.00     83.73%
//   176 requests in 30.05s, 8.79GB read
// Requests/sec:      5.86
// Transfer/sec:    299.37MB

// vihanvashishth@Vihans-MacBook-Air api_caller % wrk -t4 -c10 -d30s http://localhost:8082/
// Running 30s test @ http://localhost:8082/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency     1.30s   171.56ms   1.84s    81.01%
//     Req/Sec     2.28      2.41    10.00     75.84%
//   179 requests in 30.03s, 8.99GB read
// Requests/sec:      5.96
// Transfer/sec:    306.44MB

// vihanvashishth@Vihans-MacBook-Air api_caller % wrk -t4 -c10 -d30s http://localhost:8082/
// Running 30s test @ http://localhost:8082/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency     1.21s   128.79ms   1.70s    88.08%
//     Req/Sec     3.22      3.68    10.00     79.77%
//   193 requests in 30.01s, 9.71GB read
// Requests/sec:      6.43
// Transfer/sec:    331.20MB

// vihanvashishth@Vihans-MacBook-Air api_caller % wrk -t4 -c10 -d30s http://localhost:8082/
// Running 30s test @ http://localhost:8082/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency     1.13s    88.18ms   1.49s    79.81%
//     Req/Sec     2.54      2.46    10.00     90.29%
//   208 requests in 30.01s, 10.40GB read
// Requests/sec:      6.93
// Transfer/sec:    354.85MB

// vihanvashishth@Vihans-MacBook-Air api_caller % wrk -t4 -c10 -d30s http://localhost:8082/
// Running 30s test @ http://localhost:8082/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency     1.19s    56.89ms   1.37s    79.19%
//     Req/Sec     3.47      4.16    10.00     72.46%
//   197 requests in 30.01s, 9.86GB read
// Requests/sec:      6.56
// Transfer/sec:    336.41MB

// vihanvashishth@Vihans-MacBook-Air api_caller % wrk -t4 -c10 -d30s http://localhost:8082/
// Running 30s test @ http://localhost:8082/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency     1.07s   126.81ms   1.92s    79.91%
//     Req/Sec     2.71      2.93    10.00     87.62%
//   215 requests in 30.01s, 10.77GB read
//   Socket errors: connect 0, read 0, write 0, timeout 1
// Requests/sec:      7.16
// Transfer/sec:    367.60MB

// vihanvashishth@Vihans-MacBook-Air api_caller % wrk -t4 -c10 -d30s http://localhost:8082/
// Running 30s test @ http://localhost:8082/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency     1.14s    67.92ms   1.38s    79.23%
//     Req/Sec     2.73      2.46    10.00     92.20%
//   207 requests in 30.01s, 10.30GB read
// Requests/sec:      6.90
// Transfer/sec:    351.33MB

// In VM rootful
// Running 30s test @ http://localhost:8082/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency    34.84ms    6.44ms  96.39ms   74.20%
//     Req/Sec    57.27      7.88   120.00     81.20%
//   6877 requests in 30.09s, 335.92GB read
// Requests/sec:    228.51
// Transfer/sec:     11.16GB

// Running 30s test @ http://localhost:8082/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency    36.58ms    7.03ms  94.00ms   72.25%
//     Req/Sec    54.54      8.08    80.00     83.15%
//   6559 requests in 30.09s, 320.46GB read
// Requests/sec:    217.95
// Transfer/sec:     10.65GB

// Running 30s test @ http://localhost:8082/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency    37.81ms    7.86ms 107.26ms   73.43%
//     Req/Sec    52.73      8.84    80.00     76.42%
//   6331 requests in 30.03s, 309.39GB read
// Requests/sec:    210.84
// Transfer/sec:     10.30GB

// Running 30s test @ http://localhost:8082/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency    35.48ms    6.43ms  78.12ms   72.58%
//     Req/Sec    56.21      7.68    79.00     83.81%
//   6755 requests in 30.10s, 329.99GB read
// Requests/sec:    224.45
// Transfer/sec:     10.96GB

// Running 30s test @ http://localhost:8082/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency    37.50ms    8.35ms 162.67ms   79.41%
//     Req/Sec    53.30      8.66    70.00     79.83%
//   6399 requests in 30.04s, 312.55GB read
// Requests/sec:    213.03
// Transfer/sec:     10.41GB

// Running 30s test @ http://localhost:8082/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency    38.56ms    9.66ms 147.69ms   80.47%
//     Req/Sec    51.84      9.65    80.00     75.59%
//   6225 requests in 30.03s, 304.13GB read
// Requests/sec:    207.30
// Transfer/sec:     10.13GB

// Running 30s test @ http://localhost:8082/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency    39.29ms    7.97ms 141.31ms   75.56%
//     Req/Sec    50.82      8.51    70.00     46.57%
//   6102 requests in 30.03s, 298.06GB read
// Requests/sec:    203.18
// Transfer/sec:      9.92GB

//Running 30s test @ http://localhost:8082/
// 4 threads and 10 connections
// Thread Stats   Avg      Stdev     Max   +/- Stdev
//   Latency    39.43ms    7.04ms  94.77ms   70.29%
//   Req/Sec    50.57      7.93    70.00     46.74%
// 6073 requests in 30.04s, 296.72GB read
// Requests/sec:    202.14
// Transfer/sec:      9.88GB

// Running 30s test @ http://localhost:8082/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency    40.57ms    7.10ms  84.83ms   69.78%
//     Req/Sec    49.15      8.12    80.00     43.73%
//   5903 requests in 30.04s, 288.40GB read
// Requests/sec:    196.53
// Transfer/sec:      9.60GB

// Running 30s test @ http://localhost:8082/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency    43.12ms    8.80ms 113.51ms   74.03%
//     Req/Sec    46.23      8.40    78.00     78.85%
//   5557 requests in 30.06s, 271.48GB read
// Requests/sec:    184.87
// Transfer/sec:      9.03GB

// In VM rootless
// Running 30s test @ http://localhost:8083/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency    42.14ms    9.21ms 112.86ms   74.19%
//     Req/Sec    47.33      8.63    70.00     75.17%
//   5687 requests in 30.05s, 277.84GB read
// Requests/sec:    189.23
// Transfer/sec:      9.25GB

// Running 30s test @ http://localhost:8083/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency    42.27ms    8.73ms 113.83ms   73.63%
//     Req/Sec    47.17      8.79    70.00     74.00%
//   5667 requests in 30.05s, 276.83GB read
// Requests/sec:    188.58
// Transfer/sec:      9.21GB

// Running 30s test @ http://localhost:8083/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency    43.42ms    8.07ms  92.18ms   69.79%
//     Req/Sec    45.90      7.58    70.00     83.78%
//   5515 requests in 30.04s, 269.42GB read
// Requests/sec:    183.59
// Transfer/sec:      8.97GB

// Running 30s test @ http://localhost:8083/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency    46.31ms   10.50ms 157.96ms   75.93%
//     Req/Sec    43.09      8.41    70.00     82.44%
//   5182 requests in 30.08s, 253.22GB read
// Requests/sec:    172.29
// Transfer/sec:      8.42GB

// Running 30s test @ http://localhost:8083/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency    43.66ms    8.19ms  96.38ms   72.28%
//     Req/Sec    45.64      8.04    70.00     82.11%
//   5487 requests in 30.06s, 268.09GB read
// Requests/sec:    182.55
// Transfer/sec:      8.92GB

// Running 30s test @ http://localhost:8083/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency    61.30ms   17.13ms 168.12ms   75.81%
//     Req/Sec    32.48      9.62    70.00     69.97%
//   3912 requests in 30.10s, 191.22GB read
// Requests/sec:    129.98
// Transfer/sec:      6.35GB

// Running 30s test @ http://localhost:8083/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency    56.09ms   16.91ms 212.32ms   71.55%
//     Req/Sec    35.61     10.92    70.00     68.32%
//   4283 requests in 30.09s, 209.24GB read
// Requests/sec:    142.33
// Transfer/sec:      6.95GB

// Running 30s test @ http://localhost:8083/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency    52.11ms   13.67ms 132.07ms   66.83%
//     Req/Sec    38.19     10.92    70.00     74.39%
//   4600 requests in 30.10s, 224.78GB read
// Requests/sec:    152.81
// Transfer/sec:      7.47GB

// Running 30s test @ http://localhost:8083/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency    42.38ms   12.66ms 156.26ms   79.38%
//     Req/Sec    47.20     11.73    80.00     58.68%
//   5665 requests in 30.08s, 276.82GB read
// Requests/sec:    188.34
// Transfer/sec:      9.20GB

// Running 30s test @ http://localhost:8083/
//   4 threads and 10 connections
//   Thread Stats   Avg      Stdev     Max   +/- Stdev
//     Latency    36.60ms    6.51ms  69.40ms   69.64%
//     Req/Sec    54.49      7.50    80.00     84.90%
//   6554 requests in 30.10s, 320.20GB read
// Requests/sec:    217.75
// Transfer/sec:     10.64GB
