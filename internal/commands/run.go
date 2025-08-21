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

// NewRunCmd returns the `run` subcommand to execute a simple HTTP load test.
func NewRunCmd() *cobra.Command {
	var (
		targetURL   string
		total       int
		concurrency int
		timeout     time.Duration
		method      string
		headers     []string
		body        string
		output      string
		outFile     string
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a load test against a target URL",
		Long: `Run a fixed number of HTTP requests with a given concurrency.

By default, requests use GET and no body. You can change the method, add
headers, or send a body. Results can be printed in human-readable text or
exported as JSON for automation.

Key metrics: total time, total requests, requests/sec (RPS), 200 OK count,
per-status counts and error count.

Flags overview:
	--url            Target URL (required)
	--requests       Total number of requests (required)
	--concurrency    Number of worker goroutines (default 10)
	--timeout        Overall test timeout
	--method         HTTP method (GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS)
	--header         Repeatable HTTP header in 'Key: Value' format
	--body           Request body (string)
	--output         text|json (default text)
	--out-file       If set with --output=json, write JSON to file`,
		Example: `# 100 requests with concurrency 10
stress-test run --url https://example.com --requests 100 --concurrency 10

# POST JSON with custom header
stress-test run --url https://httpbin.org/post --requests 50 --concurrency 5 \
	--method POST --header 'Content-Type: application/json' --body '{"a":1}'

# Save machine-readable output
stress-test run --url https://example.com --requests 200 --concurrency 20 \
	--output json --out-file result.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if targetURL == "" {
				return errors.New("--url is required")
			}
			if _, err := url.ParseRequestURI(targetURL); err != nil {
				return fmt.Errorf("invalid --url: %w", err)
			}
			if total <= 0 {
				return errors.New("--requests must be > 0")
			}
			if concurrency <= 0 {
				return errors.New("--concurrency must be > 0")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()

			// normalize and validate method
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

			opts := runner.Options{
				Method:  method,
				Headers: hdr,
				Body:    []byte(body),
			}

			rep, err := runner.RunWithOptions(ctx, targetURL, total, concurrency, opts)
			if err != nil {
				return err
			}

			// Output
			switch strings.ToLower(strings.TrimSpace(output)) {
			case "", "text":
				// human-readable
				fmt.Fprintf(cmd.OutOrStdout(), "Total time: %s\n", rep.Duration)
				fmt.Fprintf(cmd.OutOrStdout(), "Total requests: %d\n", rep.TotalRequests)
				fmt.Fprintf(cmd.OutOrStdout(), "Requests/sec: %.2f\n", rep.RPS())
				fmt.Fprintf(cmd.OutOrStdout(), "HTTP 200: %d\n", rep.Succeeded200)
				if len(rep.StatusCounts) > 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "Other status codes:")
					for code, count := range rep.StatusCounts {
						if code == 200 { // already printed separately
							continue
						}
						fmt.Fprintf(cmd.OutOrStdout(), "- %d: %d\n", code, count)
					}
				}
				if rep.Errors > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "Errors: %d\n", rep.Errors)
				}
				return nil
			case "json":
				// machine-readable
				type jsonOut struct {
					URL           string         `json:"url"`
					Method        string         `json:"method"`
					DurationMS    int64          `json:"duration_ms"`
					TotalRequests int            `json:"total_requests"`
					RPS           float64        `json:"rps"`
					HTTP200       int            `json:"http_200"`
					Errors        int            `json:"errors"`
					StatusCounts  map[string]int `json:"status_counts"`
					Timestamp     string         `json:"timestamp"`
				}
				sc := make(map[string]int, len(rep.StatusCounts))
				for k, v := range rep.StatusCounts {
					sc[fmt.Sprintf("%d", k)] = v
				}
				payload := jsonOut{
					URL:           targetURL,
					Method:        method,
					DurationMS:    rep.Duration.Milliseconds(),
					TotalRequests: rep.TotalRequests,
					RPS:           rep.RPS(),
					HTTP200:       rep.Succeeded200,
					Errors:        rep.Errors,
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
	cmd.Flags().IntVar(&total, "requests", 0, "Total number of requests")
	cmd.Flags().IntVar(&concurrency, "concurrency", 10, "Number of concurrent workers")
	cmd.Flags().DurationVar(&timeout, "timeout", 60*time.Second, "Overall test timeout")
	cmd.Flags().StringVar(&method, "method", http.MethodGet, "HTTP method (GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS)")
	cmd.Flags().StringArrayVar(&headers, "header", nil, "HTTP header in 'Key: Value' format (repeatable)")
	cmd.Flags().StringVar(&body, "body", "", "HTTP request body (string)")
	cmd.Flags().StringVar(&output, "output", "text", "Output format: text|json")
	cmd.Flags().StringVar(&outFile, "out-file", "", "Write output to file (only for --output=json by default)")
	err := cmd.MarkFlagRequired("url")
	if err != nil {
		return nil
	}
	err = cmd.MarkFlagRequired("requests")
	if err != nil {
		return nil
	}

	return cmd
}
