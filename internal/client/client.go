// Package client talks to the Cloider api, https://cloider.com/api, as an API
// token. It is the cloider CLI's api client cut down to what the provider manages.
package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/time/rate"
)

type Client struct {
	BaseURL string

	token     string
	userAgent string
	http      *http.Client
	limiter   *rate.Limiter
}

func New(baseURL, token, userAgent string, insecure bool) *Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if insecure {
		// Only for a local api with a self signed certificate, behind CLOIDER_INSECURE
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	return &Client{
		BaseURL:   strings.TrimRight(baseURL, "/"),
		token:     token,
		userAgent: userAgent,
		http:      &http.Client{Transport: transport, Timeout: 2 * time.Minute},
		// The api allows 10 requests a second and 100 a minute per client. A plan
		// refreshes every resource at once and waits poll, so this stays under it
		// instead of leaning on the 429 retries.
		limiter: rate.NewLimiter(rate.Limit(1.5), 10),
	}
}

// The api answers a rate limited request with a 429, or with a 503.
const maxAttempts = 4

func (c *Client) newRequest(ctx context.Context, method, path string, query url.Values, body any) (*http.Request, error) {
	u := c.BaseURL + "/" + strings.TrimLeft(path, "/")
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("User-Agent", c.userAgent)
	return req, nil
}

// send makes the request, retrying what the rate limit turned away. Only reads
// are retried on a 502/503/504: a write may have gone through.
func (c *Client) send(ctx context.Context, method, path string, query url.Values, body any) (*http.Response, error) {
	idempotent := method == http.MethodGet || method == http.MethodHead

	for attempt := 1; ; attempt++ {
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, err
		}
		req, err := c.newRequest(ctx, method, path, query, body)
		if err != nil {
			return nil, err
		}

		start := time.Now()
		resp, err := c.http.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			if idempotent && attempt < maxAttempts {
				tflog.Debug(ctx, "cloider api request failed, retrying", map[string]any{"method": method, "url": req.URL.Redacted(), "error": err.Error()})
				if !sleep(ctx, backoff(attempt, "")) {
					return nil, ctx.Err()
				}
				continue
			}
			return nil, fmt.Errorf("can't reach %s: %w", c.BaseURL, err)
		}
		tflog.Debug(ctx, "cloider api request", map[string]any{
			"method": method, "url": req.URL.Redacted(), "status": resp.StatusCode, "duration": time.Since(start).Round(time.Millisecond).String(),
		})

		retryable := resp.StatusCode == http.StatusTooManyRequests ||
			(idempotent && (resp.StatusCode == http.StatusBadGateway || resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusGatewayTimeout))
		if retryable && attempt < maxAttempts {
			wait := backoff(attempt, resp.Header.Get("Retry-After"))
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if !sleep(ctx, wait) {
				return nil, ctx.Err()
			}
			continue
		}

		return resp, nil
	}
}

func backoff(attempt int, retryAfter string) time.Duration {
	if s, err := strconv.Atoi(retryAfter); err == nil && s > 0 && s <= 60 {
		return time.Duration(s) * time.Second
	}
	return time.Duration(1<<(attempt-1)) * time.Second
}

func sleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

// Do sends a request and decodes the answer into out: json into a pointer to
// anything, as is into a *string, nowhere when out is nil.
func (c *Client) Do(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	resp, err := c.send(ctx, method, path, query, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 300 {
		return parseError(resp.StatusCode, data)
	}

	switch out := out.(type) {
	case nil:
		return nil
	case *string:
		*out = string(data)
		return nil
	default:
		// Some lists answer null rather than [] when empty, which leaves out as is
		if len(bytes.TrimSpace(data)) == 0 {
			return nil
		}
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("unexpected answer from %s %s: %w", method, path, err)
		}
		return nil
	}
}

func (c *Client) Get(ctx context.Context, path string, query url.Values, out any) error {
	return c.Do(ctx, http.MethodGet, path, query, nil, out)
}

func (c *Client) Post(ctx context.Context, path string, body any, out any) error {
	return c.Do(ctx, http.MethodPost, path, nil, body, out)
}

func (c *Client) Put(ctx context.Context, path string, body any, out any) error {
	return c.Do(ctx, http.MethodPut, path, nil, body, out)
}

func (c *Client) Delete(ctx context.Context, path string, body any, out any) error {
	return c.Do(ctx, http.MethodDelete, path, nil, body, out)
}

// GetText is for the endpoints that answer plain text: statuses, kubeconfigs,
// passwords, certificates.
func (c *Client) GetText(ctx context.Context, path string, query url.Values) (string, error) {
	var text string
	err := c.Get(ctx, path, query, &text)
	return text, err
}
