package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/JeanGrijp/stress-test/internal/runner"
	"github.com/spf13/cobra"
)

// NewRampCmd runs multiple phases with increasing concurrency (a "ramp").
// Each phase executes a fixed number of requests with its own concurrency.
// Example:
//
//	stress-test ramp --url=https://example.com --steps=3 --start-concurrency=5 --step-concurrency=10 --requests-per-step=200
func NewRampCmd() *cobra.Command {
	var (
		targetURL        string
		steps            int
		startConcurrency int
		stepConcurrency  int
		requestsPerStep  int
		perStepDuration  time.Duration
		sleepBetween     time.Duration
		timeout          time.Duration
		method           string
		headers          []string
		body             string
		rps              float64
		stepRps          float64
		output           string
		outFile          string
	)

	cmd := &cobra.Command{
		Use:   "ramp",
		Short: "Run multiple phases with increasing concurrency",
		Long: `Run a multi-phase test ramping concurrency between phases.

You can choose one of three modes per phase:
	A) Requests mode:      --requests-per-step > 0
	B) Duration mode:      --per-step-duration > 0 (max throughput per concurrency)
	C) Duration + Rate:    --per-step-duration > 0 and --rps > 0 (paced RPS target)

Per-phase concurrency is computed as: start + i*step for i in [0..steps-1].
Between phases you may sleep with --sleep-between.

Printed metrics include per-phase summaries and an overall final summary.
You can export the final summary as JSON.

Important combinations:
	- Requests mode: do not set --per-step-duration or --rps
	- Duration mode: set --per-step-duration, leave --requests-per-step=0, --rps=0
	- Rate mode:     set --per-step-duration and --rps (optionally --step-rps)`,
		Example: `# 3 phases, +5 concurrency per phase, 200 requests per phase
stress-test ramp --url https://example.com --steps 3 --start-concurrency 5 \
	--step-concurrency 5 --requests-per-step 200

# Duration mode: 2 phases of 15s each, starting concurrency 10 then +10
stress-test ramp --url https://example.com --steps 2 --start-concurrency 10 \
	--step-concurrency 10 --per-step-duration 15s

# Rate mode: 3 phases of 20s, start 50 rps and +25 rps per phase
stress-test ramp --url https://example.com --steps 3 --start-concurrency 20 \
	--step-concurrency 0 --per-step-duration 20s --rps 50 --step-rps 25 \
	--requests-per-step 0 --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// validations
			if targetURL == "" {
				return errors.New("--url is required")
			}
			if _, err := url.ParseRequestURI(targetURL); err != nil {
				return fmt.Errorf("invalid --url: %w", err)
			}
			if steps <= 0 {
				return errors.New("--steps must be > 0")
			}
			if startConcurrency <= 0 {
				return errors.New("--start-concurrency must be > 0")
			}
			// Valid modes:
			// A) requests mode: requests-per-step>0, per-step-duration==0, rps==0
			// B) time mode (max throughput): per-step-duration>0, requests-per-step==0, rps==0
			// C) time+rate mode: per-step-duration>0, rps>0, requests-per-step==0
			if requestsPerStep > 0 {
				if perStepDuration > 0 || rps > 0 {
					return errors.New("requests mode: do not set --per-step-duration or --rps when using --requests-per-step")
				}
			} else if perStepDuration > 0 {
				// ok, either time mode or time+rate
			} else {
				return errors.New("must set either --requests-per-step (>0) or --per-step-duration (>0)")
			}

			// normalize method
			method = strings.ToUpper(strings.TrimSpace(method))
			if method == "" {
				method = http.MethodGet
			}
			switch method {
			case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead, http.MethodOptions:
			default:
				return fmt.Errorf("unsupported --method: %s", method)
			}

			// parse headers "Key: Value"
			hdr := make(http.Header)
			for _, h := range headers {
				parts := strings.SplitN(h, ":", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid --header format (use 'Key: Value'): %q", h)
				}
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				if key == "" {
					return fmt.Errorf("invalid --header key in: %q", h)
				}
				hdr.Add(key, val)
			}

			opts := runner.Options{Method: method, Headers: hdr, Body: []byte(body)}

			overallStart := time.Now()
			overall := runner.Report{StatusCounts: map[int]int{}}

			for i := 0; i < steps; i++ {
				concurrency := startConcurrency + i*stepConcurrency
				ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
				var rep runner.Report
				var err error
				if requestsPerStep > 0 {
					fmt.Fprintf(cmd.ErrOrStderr(), "Phase %d/%d: concurrency=%d, requests=%d\n", i+1, steps, concurrency, requestsPerStep)
					rep, err = runner.RunWithOptions(ctx, targetURL, requestsPerStep, concurrency, opts)
				} else {
					rpsPhase := rps + float64(i)*stepRps
					if rpsPhase > 0 {
						fmt.Fprintf(cmd.ErrOrStderr(), "Phase %d/%d: concurrency=%d, duration=%s, rate=%.2frps\n", i+1, steps, concurrency, perStepDuration, rpsPhase)
						rep, err = runner.RunForDurationWithRate(ctx, targetURL, perStepDuration, concurrency, opts, rpsPhase)
					} else {
						fmt.Fprintf(cmd.ErrOrStderr(), "Phase %d/%d: concurrency=%d, duration=%s\n", i+1, steps, concurrency, perStepDuration)
						rep, err = runner.RunForDuration(ctx, targetURL, perStepDuration, concurrency, opts)
					}
				}
				cancel()
				if err != nil {
					return fmt.Errorf("phase %d failed: %w", i+1, err)
				}

				// print per-phase summary
				fmt.Fprintf(cmd.OutOrStdout(), "Phase %d: time=%s, rps=%.2f, http200=%d, errors=%d\n", i+1, rep.Duration, rep.RPS(), rep.Succeeded200, rep.Errors)

				// aggregate results
				overall.TotalRequests += rep.TotalRequests
				overall.Succeeded200 += rep.Succeeded200
				overall.Errors += rep.Errors
				for code, count := range rep.StatusCounts {
					overall.StatusCounts[code] += count
				}

				if sleepBetween > 0 && i < steps-1 {
					time.Sleep(sleepBetween)
				}
			}

			overall.Duration = time.Since(overallStart)

			switch strings.ToLower(strings.TrimSpace(output)) {
			case "", "text":
				// final summary (text)
				fmt.Fprintln(cmd.OutOrStdout(), "---")
				fmt.Fprintf(cmd.OutOrStdout(), "Overall time: %s\n", overall.Duration)
				fmt.Fprintf(cmd.OutOrStdout(), "Total requests: %d\n", overall.TotalRequests)
				fmt.Fprintf(cmd.OutOrStdout(), "Overall RPS: %.2f\n", overall.RPS())
				fmt.Fprintf(cmd.OutOrStdout(), "HTTP 200: %d\n", overall.Succeeded200)
				if len(overall.StatusCounts) > 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "Other status codes:")
					for code, count := range overall.StatusCounts {
						if code == 200 {
							continue
						}
						fmt.Fprintf(cmd.OutOrStdout(), "- %d: %d\n", code, count)
					}
				}
				if overall.Errors > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "Errors: %d\n", overall.Errors)
				}
				return nil
			case "json":
				type jsonOut struct {
					URL           string         `json:"url"`
					Steps         int            `json:"steps"`
					StartConc     int            `json:"start_concurrency"`
					StepConc      int            `json:"step_concurrency"`
					Mode          string         `json:"mode"`
					PerStep       string         `json:"per_step"`
					Method        string         `json:"method"`
					DurationMS    int64          `json:"duration_ms"`
					TotalRequests int            `json:"total_requests"`
					RPS           float64        `json:"rps"`
					HTTP200       int            `json:"http_200"`
					Errors        int            `json:"errors"`
					StatusCounts  map[string]int `json:"status_counts"`
					Timestamp     string         `json:"timestamp"`
				}
				sc := make(map[string]int, len(overall.StatusCounts))
				for k, v := range overall.StatusCounts {
					sc[fmt.Sprintf("%d", k)] = v
				}
				mode := ""
				per := ""
				if requestsPerStep > 0 {
					mode = "requests"
					per = fmt.Sprintf("requests_per_step=%d", requestsPerStep)
				} else if rps > 0 || stepRps > 0 {
					mode = "duration+rate"
					per = fmt.Sprintf("per_step_duration=%s,rps_start=%.2f,step_rps=%.2f", perStepDuration, rps, stepRps)
				} else {
					mode = "duration"
					per = fmt.Sprintf("per_step_duration=%s", perStepDuration)
				}
				payload := jsonOut{
					URL:           targetURL,
					Steps:         steps,
					StartConc:     startConcurrency,
					StepConc:      stepConcurrency,
					Mode:          mode,
					PerStep:       per,
					Method:        method,
					DurationMS:    overall.Duration.Milliseconds(),
					TotalRequests: overall.TotalRequests,
					RPS:           overall.RPS(),
					HTTP200:       overall.Succeeded200,
					Errors:        overall.Errors,
					StatusCounts:  sc,
					Timestamp:     time.Now().UTC().Format(time.RFC3339),
				}
				data, err := json.MarshalIndent(payload, "", "  ")
				if err != nil {
					return err
				}
				if outFile != "" {
					return os.WriteFile(outFile, data, 0644)
				}
				_, _ = cmd.OutOrStdout().Write(append(data, '\n'))
				return nil
			default:
				return fmt.Errorf("unsupported --output: %s", output)
			}
		},
	}

	cmd.Flags().StringVar(&targetURL, "url", "", "Target URL to test")
	cmd.Flags().IntVar(&steps, "steps", 3, "Number of ramp phases")
	cmd.Flags().IntVar(&startConcurrency, "start-concurrency", 5, "Concurrency at the first phase")
	cmd.Flags().IntVar(&stepConcurrency, "step-concurrency", 5, "Concurrency increment per phase")
	cmd.Flags().IntVar(&requestsPerStep, "requests-per-step", 100, "Total requests per phase")
	cmd.Flags().DurationVar(&perStepDuration, "per-step-duration", 0, "Per-phase duration (alternative to requests-per-step)")
	cmd.Flags().DurationVar(&sleepBetween, "sleep-between", 0, "Sleep duration between phases")
	cmd.Flags().DurationVar(&timeout, "timeout", 60*time.Second, "Per-phase timeout")
	cmd.Flags().StringVar(&method, "method", http.MethodGet, "HTTP method (GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS)")
	cmd.Flags().StringArrayVar(&headers, "header", nil, "HTTP header in 'Key: Value' format (repeatable)")
	cmd.Flags().StringVar(&body, "body", "", "HTTP request body (string)")
	cmd.Flags().Float64Var(&rps, "rps", 0, "Target requests per second per phase (requires --per-step-duration)")
	cmd.Flags().Float64Var(&stepRps, "step-rps", 0, "RPS increment per phase")
	cmd.Flags().StringVar(&output, "output", "text", "Output format: text|json")
	cmd.Flags().StringVar(&outFile, "out-file", "", "Write final summary to file (only for --output=json by default)")
	_ = cmd.MarkFlagRequired("url")

	return cmd
}
