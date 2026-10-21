package client

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

// PollInterval is slower than the CLI's 3s: a single apply can wait on several
// resources at once, all inside the api's 100 requests a minute.
var PollInterval = 5 * time.Second

// Poll calls check every PollInterval until it says done or ctx ends. ctx
// carries the resource's timeout.
func Poll(ctx context.Context, check func(context.Context) (status string, done bool, err error)) (string, error) {
	last := ""
	for {
		status, done, err := check(ctx)
		if err != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return last, gaveUp(last)
			}
			return status, err
		}
		last = status
		if done {
			return status, nil
		}

		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return last, gaveUp(last)
			}
			return last, ctx.Err()
		case <-time.After(PollInterval):
		}
	}
}

func gaveUp(status string) error {
	if status == "" {
		return errors.New("gave up waiting, raise the timeout with a timeouts block")
	}
	return fmt.Errorf("still %s, gave up waiting, raise the timeout with a timeouts block", status)
}

// WaitStatus polls a plain text /status endpoint until the status is one of
// want. A 404 reads as Terminated, the resource is gone. Failed ends the wait
// as an error unless it is wanted.
func (c *Client) WaitStatus(ctx context.Context, statusPath string, want ...string) (string, error) {
	status, err := Poll(ctx, func(ctx context.Context) (string, bool, error) {
		status, err := c.GetText(ctx, statusPath, nil)
		if IsNotFound(err) {
			return "Terminated", true, nil
		}
		if err != nil {
			return "", false, err
		}
		status = strings.TrimSpace(status)
		return status, slices.Contains(want, status) || status == "Failed", nil
	})
	if err != nil {
		return status, err
	}
	if !slices.Contains(want, status) {
		return status, fmt.Errorf("it is %s, expected %s", status, strings.Join(want, " or "))
	}
	return status, nil
}
