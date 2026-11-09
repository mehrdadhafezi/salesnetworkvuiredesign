// Command crm-healthcheck probes a local health endpoint for container
// health checks (R3). Distroless images have no shell or curl.
//
//	crm-healthcheck http://127.0.0.1:8080/health/ready
//
// Exit 0 when the endpoint answers 200 within 3 s, 1 otherwise, 2 on usage.
// Only loopback http URLs are accepted: it can never probe another host.
package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

func main() { os.Exit(run(os.Args[1:], os.Stderr)) }

func run(args []string, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: crm-healthcheck http://127.0.0.1:<port>/<path>")
		return 2
	}
	u, err := url.Parse(args[0])
	if err != nil || u.Scheme != "http" || u.User != nil {
		fmt.Fprintln(stderr, "crm-healthcheck: a loopback http URL is required")
		return 2
	}
	ip := net.ParseIP(u.Hostname())
	if u.Hostname() != "localhost" && (ip == nil || !ip.IsLoopback()) {
		fmt.Fprintln(stderr, "crm-healthcheck: only loopback hosts are allowed")
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return 2
	}
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintln(stderr, "crm-healthcheck: unavailable")
		return 1
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(stderr, "crm-healthcheck: status %d\n", resp.StatusCode)
		return 1
	}
	return 0
}
