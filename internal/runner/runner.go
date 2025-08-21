package runner

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"sync"
	"time"
)

// Report summarizes a test execution.
type Report struct {
	Duration      time.Duration
	TotalRequests int
	Succeeded200  int
	StatusCounts  map[int]int
	Errors        int
}

// RPS returns requests per second.
func (r Report) RPS() float64 {
	if r.Duration <= 0 {
		return 0
	}
	return float64(r.TotalRequests) / r.Duration.Seconds()
}

// Options configures request details for the load test.
type Options struct {
	Method  string
	Headers http.Header
	Body    []byte
}

// Run executes a simple HTTP load test using defaults (GET, no headers, no body).
func Run(ctx context.Context, targetURL string, total, concurrency int) (Report, error) {
	return RunWithOptions(ctx, targetURL, total, concurrency, Options{Method: http.MethodGet})
}

// RunWithOptions executes a HTTP load test against targetURL with custom options.
func RunWithOptions(ctx context.Context, targetURL string, total, concurrency int, opts Options) (Report, error) {
	start := time.Now()
	rep := Report{TotalRequests: total, StatusCounts: make(map[int]int)}

	client := &http.Client{}
	defer client.CloseIdleConnections()

	// Work distribution
	jobs := make(chan struct{})
	var wg sync.WaitGroup
	var mu sync.Mutex

	worker := func() {
		defer wg.Done()
		for range jobs {
			// respect context cancellation
			if ctx.Err() != nil {
				return
			}
			var bodyReader *bytes.Reader
			if len(opts.Body) > 0 {
				bodyReader = bytes.NewReader(opts.Body)
			}
			method := opts.Method
			if method == "" {
				method = http.MethodGet
			}
			var req *http.Request
			var err error
			if bodyReader != nil {
				req, err = http.NewRequestWithContext(ctx, method, targetURL, bodyReader)
			} else {
				req, err = http.NewRequestWithContext(ctx, method, targetURL, nil)
			}
			if err != nil {
				mu.Lock()
				rep.Errors++
				mu.Unlock()
				continue
			}
			// set headers
			if opts.Headers != nil {
				for k, vals := range opts.Headers {
					for _, v := range vals {
						req.Header.Add(k, v)
					}
				}
			}
			resp, err := client.Do(req)
			if err != nil {
				mu.Lock()
				rep.Errors++
				mu.Unlock()
				continue
			}
			// drain and close body to allow connection reuse
			_ = resp.Body.Close()
			mu.Lock()
			rep.StatusCounts[resp.StatusCode]++
			if resp.StatusCode == http.StatusOK {
				rep.Succeeded200++
			}
			mu.Unlock()
		}
	}

	// Start workers
	if concurrency < 1 {
		concurrency = 1
	}
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go worker()
	}

	// Enqueue jobs
	go func() {
		defer close(jobs)
		for i := 0; i < total; i++ {
			select {
			case <-ctx.Done():
				return
			case jobs <- struct{}{}:
			}
		}
	}()

	wg.Wait()
	rep.Duration = time.Since(start)
	return rep, nil
}

// RunForDuration executes requests for a given duration at fixed concurrency.
func RunForDuration(ctx context.Context, targetURL string, d time.Duration, concurrency int, opts Options) (Report, error) {
	start := time.Now()
	rep := Report{StatusCounts: make(map[int]int)}

	client := &http.Client{}
	defer client.CloseIdleConnections()

	var wg sync.WaitGroup
	var mu sync.Mutex

	// Deadline goroutine to cancel via context if needed
	end := time.After(d)

	worker := func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case <-end:
				return
			default:
			}

			var body io.Reader
			if len(opts.Body) > 0 {
				body = bytes.NewReader(opts.Body)
			} else {
				body = nil
			}
			method := opts.Method
			if method == "" {
				method = http.MethodGet
			}
			req, err := http.NewRequestWithContext(ctx, method, targetURL, body)
			if err != nil {
				mu.Lock()
				rep.Errors++
				mu.Unlock()
				continue
			}
			if opts.Headers != nil {
				for k, vals := range opts.Headers {
					for _, v := range vals {
						req.Header.Add(k, v)
					}
				}
			}
			resp, err := client.Do(req)
			if err != nil {
				mu.Lock()
				rep.Errors++
				mu.Unlock()
				continue
			}
			_ = resp.Body.Close()
			mu.Lock()
			rep.TotalRequests++
			rep.StatusCounts[resp.StatusCode]++
			if resp.StatusCode == http.StatusOK {
				rep.Succeeded200++
			}
			mu.Unlock()
		}
	}

	if concurrency < 1 {
		concurrency = 1
	}
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go worker()
	}
	wg.Wait()
	rep.Duration = time.Since(start)
	return rep, nil
}

// RunForDurationWithRate executes requests for a given duration at a target RPS,
// using a simple paced job generator and a fixed number of workers.
func RunForDurationWithRate(ctx context.Context, targetURL string, d time.Duration, concurrency int, opts Options, rps float64) (Report, error) {
	start := time.Now()
	rep := Report{StatusCounts: make(map[int]int)}

	if rps <= 0 {
		return rep, nil
	}

	client := &http.Client{}
	defer client.CloseIdleConnections()

	jobs := make(chan struct{}, 1024)
	var wg sync.WaitGroup
	var mu sync.Mutex

	// paced generator
	tickerInterval := time.Second
	if rps > 0 {
		tickerInterval = time.Duration(float64(time.Second) / rps)
		if tickerInterval <= 0 {
			tickerInterval = time.Nanosecond
		}
	}
	ticker := time.NewTicker(tickerInterval)
	end := time.After(d)

	genDone := make(chan struct{})
	go func() {
		defer close(genDone)
		defer close(jobs)
		for {
			select {
			case <-ctx.Done():
				return
			case <-end:
				return
			case <-ticker.C:
				select {
				case jobs <- struct{}{}:
				case <-ctx.Done():
					return
				case <-end:
					return
				}
			}
		}
	}()

	worker := func() {
		defer wg.Done()
		for range jobs {
			if ctx.Err() != nil {
				return
			}
			var body io.Reader
			if len(opts.Body) > 0 {
				body = bytes.NewReader(opts.Body)
			}
			method := opts.Method
			if method == "" {
				method = http.MethodGet
			}
			req, err := http.NewRequestWithContext(ctx, method, targetURL, body)
			if err != nil {
				mu.Lock()
				rep.Errors++
				mu.Unlock()
				continue
			}
			if opts.Headers != nil {
				for k, vals := range opts.Headers {
					for _, v := range vals {
						req.Header.Add(k, v)
					}
				}
			}
			resp, err := client.Do(req)
			if err != nil {
				mu.Lock()
				rep.Errors++
				mu.Unlock()
				continue
			}
			_ = resp.Body.Close()
			mu.Lock()
			rep.TotalRequests++
			rep.StatusCounts[resp.StatusCode]++
			if resp.StatusCode == http.StatusOK {
				rep.Succeeded200++
			}
			mu.Unlock()
		}
	}

	if concurrency < 1 {
		concurrency = 1
	}
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go worker()
	}
	<-genDone
	wg.Wait()
	ticker.Stop()
	rep.Duration = time.Since(start)
	return rep, nil
}
