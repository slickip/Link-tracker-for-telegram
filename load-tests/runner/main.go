package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type Config struct {
	BaseURL       string
	Users         int
	ChatCount     int64
	Duration      time.Duration
	RampUp        time.Duration
	GetRatio      int
	MutationRatio int
}

type Result struct {
	Request string
	Status  int
	Latency time.Duration
	Err     error
}

func main() {
	cfg := Config{
		BaseURL:       getEnv("SCRAPPER_URL", "http://localhost:8081"),
		Users:         getEnvInt("LOAD_USERS", 16),
		ChatCount:     int64(getEnvInt("LOAD_CHAT_COUNT", 1000)),
		Duration:      getEnvDuration("LOAD_DURATION", 5*time.Minute),
		RampUp:        getEnvDuration("LOAD_RAMP_UP", time.Minute),
		GetRatio:      getEnvInt("LOAD_GET_RATIO", 100),
		MutationRatio: getEnvInt("LOAD_MUTATION_RATIO", 1),
	}

	fmt.Printf("Load test config: %+v\n", cfg)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.RampUp+cfg.Duration)
	defer cancel()

	results := make(chan Result, 10000)

	var wg sync.WaitGroup
	var started int64

	for i := 0; i < cfg.Users; i++ {
		wg.Add(1)

		go func(workerID int) {
			defer wg.Done()

			delay := time.Duration(workerID) * cfg.RampUp / time.Duration(cfg.Users)
			time.Sleep(delay)

			atomic.AddInt64(&started, 1)

			runWorker(ctx, cfg, workerID, results)
		}(i)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	all := make([]Result, 0)
	for result := range results {
		all = append(all, result)
	}

	printReport(all, cfg.Duration)
}

func runWorker(ctx context.Context, cfg Config, workerID int, results chan<- Result) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	rnd := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)))

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		chatID := rnd.Int63n(cfg.ChatCount) + 1
		operation := rnd.Intn(cfg.GetRatio + cfg.MutationRatio)

		if operation < cfg.GetRatio {
			results <- doGetList(ctx, client, cfg.BaseURL, chatID)
			continue
		}

		results <- doPostList(ctx, client, cfg.BaseURL, chatID, workerID)
	}
}

func doGetList(ctx context.Context, client *http.Client, baseURL string, chatID int64) Result {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/list", nil)
	if err != nil {
		return Result{Request: "GET /list", Latency: time.Since(start), Err: err}
	}

	req.Header.Set("Tg-Chat-Id", strconv.FormatInt(chatID, 10))

	resp, err := client.Do(req)
	if err != nil {
		return Result{Request: "GET /list", Latency: time.Since(start), Err: err}
	}
	defer resp.Body.Close()

	return Result{
		Request: "GET /list",
		Status:  resp.StatusCode,
		Latency: time.Since(start),
	}
}

func doPostList(ctx context.Context, client *http.Client, baseURL string, chatID int64, workerID int) Result {
	start := time.Now()

	body := map[string]any{
		"url": fmt.Sprintf(
			"https://stackoverflow.com/questions/%d/load-test-question-%d",
			time.Now().UnixNano()%1000000000,
			workerID,
		),
		"tags": []string{"load"},
	}

	data, err := json.Marshal(body)
	if err != nil {
		return Result{Request: "POST /links", Latency: time.Since(start), Err: err}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/links", bytes.NewReader(data))
	if err != nil {
		return Result{Request: "POST /links", Latency: time.Since(start), Err: err}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Tg-Chat-Id", strconv.FormatInt(chatID, 10))

	resp, err := client.Do(req)
	if err != nil {
		return Result{Request: "POST /links", Latency: time.Since(start), Err: err}
	}
	defer resp.Body.Close()

	return Result{
		Request: "POST /links",
		Status:  resp.StatusCode,
		Latency: time.Since(start),
	}
}

func printReport(results []Result, duration time.Duration) {
	grouped := map[string][]Result{}

	for _, result := range results {
		grouped[result.Request] = append(grouped[result.Request], result)
	}

	fmt.Println()
	fmt.Println("| Request | RPS | Avg, ms | p50, ms | p99, ms | 200 | 502/504 | 500 | Other statuses | Errors |")
	fmt.Println("|---|---:|---:|---:|---:|---:|---:|---:|---|---:|")

	for request, items := range grouped {
		latencies := make([]time.Duration, 0, len(items))

		status200 := 0
		status502504 := 0
		status500 := 0
		errorsCount := 0
		otherStatuses := make(map[int]int)

		for _, item := range items {
			latencies = append(latencies, item.Latency)

			if item.Err != nil {
				errorsCount++
				continue
			}

			switch item.Status {
			case http.StatusOK:
				status200++
			case http.StatusBadGateway, http.StatusGatewayTimeout:
				status502504++
			case http.StatusInternalServerError:
				status500++
			default:
				if item.Status != 0 {
					otherStatuses[item.Status]++
				}
			}
		}

		sort.Slice(latencies, func(i, j int) bool {
			return latencies[i] < latencies[j]
		})

		avg := average(latencies)
		p50 := percentile(latencies, 0.50)
		p99 := percentile(latencies, 0.99)
		rps := float64(len(items)) / duration.Seconds()

		fmt.Printf(
			"| %s | %.2f | %.2f | %.2f | %.2f | %d | %d | %d | %v | %d |\n",
			request,
			rps,
			float64(avg.Microseconds())/1000,
			float64(p50.Microseconds())/1000,
			float64(p99.Microseconds())/1000,
			status200,
			status502504,
			status500,
			otherStatuses,
			errorsCount,
		)
	}
}

func average(values []time.Duration) time.Duration {
	if len(values) == 0 {
		return 0
	}

	var total time.Duration
	for _, value := range values {
		total += value
	}

	return total / time.Duration(len(values))
}

func percentile(values []time.Duration, p float64) time.Duration {
	if len(values) == 0 {
		return 0
	}

	index := int(float64(len(values)-1) * p)
	return values[index]
}

func getEnv(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	return value
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return parsed
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}

	return parsed
}
