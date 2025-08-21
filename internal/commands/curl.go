package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// NewCurlCmd executes a single HTTP request using curl-like arguments.
// Example:
//
//	stress-test curl -X POST https://httpbin.org/post -H 'Content-Type: application/json' -d '{"a":1}'
func NewCurlCmd() *cobra.Command {
	var showStats bool

	cmd := &cobra.Command{
		Use:   "curl [curl-args...]",
		Short: "Execute a curl-style request and print the response",
		Long: `Execute a single HTTP request using a small subset of curl flags.

Supported flags (subset):
  -X, --request METHOD         Set HTTP method
  -H, --header 'K: V'          Add header (repeatable)
  -d, --data [--data-raw...]   Request body (switches to POST if method not set)
  -i                           Include response headers in output
  -I, --head                   Use HEAD method
  --url URL                    Explicit URL (or pass URL as the last arg)

By default, only the response body is written to stdout. Use --stats to
print timing, status code and body size to stderr (useful for piping).`,
		Example: `# GET and include headers
stress-test curl -i https://httpbin.org/get

# POST JSON with headers and show stats to stderr
stress-test curl -X POST https://httpbin.org/post -H 'Content-Type: application/json' \
  -d '{"hello":"world"}' --stats`,
		DisableFlagParsing: true, // we'll parse args ourselves
		Args:               cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Since DisableFlagParsing is true, show help manually on -h/--help or no args
			if len(args) == 0 {
				return cmd.Help()
			}
			for _, a := range args {
				if a == "-h" || a == "--help" {
					return cmd.Help()
				}
			}
			// Drop leading literal "curl" if provided
			if args[0] == "curl" {
				args = args[1:]
			}
			method, target, hdr, body, include, err := parseCurlArgs(args)
			if err != nil {
				return err
			}
			if target == "" {
				return errors.New("missing URL in curl arguments")
			}
			if _, err := url.ParseRequestURI(target); err != nil {
				return fmt.Errorf("invalid URL: %w", err)
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, method, target, strings.NewReader(body))
			if err != nil {
				return err
			}
			for k, vals := range hdr {
				for _, v := range vals {
					req.Header.Add(k, v)
				}
			}

			client := &http.Client{}
			start := time.Now()
			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			out := cmd.OutOrStdout()
			if include {
				// Status line
				fmt.Fprintf(out, "HTTP/1.1 %d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode))
				// Headers
				for k, vals := range resp.Header {
					for _, v := range vals {
						fmt.Fprintf(out, "%s: %s\n", k, v)
					}
				}
				fmt.Fprintln(out)
			}

			// Copy body to stdout and count bytes written
			n, copyErr := io.Copy(out, resp.Body)

			// Optionally print stats to stderr to avoid contaminating stdout/pipes
			if showStats {
				elapsed := time.Since(start)
				fmt.Fprintf(cmd.ErrOrStderr(), "\nTime: %s\nStatus: %d\nBody bytes: %d\n", elapsed, resp.StatusCode, n)
			}

			return copyErr
		},
	}
	cmd.Flags().BoolVar(&showStats, "stats", false, "Print request time, status and body size to stderr")
	return cmd
}

// parseCurlArgs parses a subset of curl flags: -X/--request, -H/--header, -d/--data*, -i, and URL.
func parseCurlArgs(args []string) (method string, target string, headers http.Header, body string, include bool, err error) {
	headers = make(http.Header)
	method = http.MethodGet

	var bodies []string

	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-X", "--request":
			i++
			if i >= len(args) {
				return "", "", nil, "", false, errors.New("-X/--request requires a value")
			}
			method = strings.ToUpper(args[i])
		case "-H", "--header":
			i++
			if i >= len(args) {
				return "", "", nil, "", false, errors.New("-H/--header requires a value")
			}
			kv := args[i]
			parts := strings.SplitN(kv, ":", 2)
			if len(parts) != 2 {
				return "", "", nil, "", false, fmt.Errorf("invalid header format: %q", kv)
			}
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			if k == "" {
				return "", "", nil, "", false, fmt.Errorf("invalid header key in: %q", kv)
			}
			headers.Add(k, v)
		case "-d", "--data", "--data-raw", "--data-binary", "--data-ascii":
			i++
			if i >= len(args) {
				return "", "", nil, "", false, errors.New("-d/--data* requires a value")
			}
			bodies = append(bodies, args[i])
			if method == http.MethodGet {
				method = http.MethodPost // curl commonly defaults to POST when -d is used
			}
		case "-A", "--user-agent":
			i++
			if i >= len(args) {
				return "", "", nil, "", false, errors.New("-A/--user-agent requires a value")
			}
			headers.Set("User-Agent", args[i])
		case "-I", "--head":
			method = http.MethodHead
		case "-i":
			include = true
		case "--url":
			i++
			if i >= len(args) {
				return "", "", nil, "", false, errors.New("--url requires a value")
			}
			target = args[i]
		default:
			// If it looks like a URL and target not yet set, treat as URL.
			if strings.HasPrefix(a, "http://") || strings.HasPrefix(a, "https://") {
				if target == "" {
					target = a
					continue
				}
			}
			// ignore unrecognized flags for now
		}
	}

	if len(bodies) > 0 {
		body = strings.Join(bodies, "&")
	}
	return
}
