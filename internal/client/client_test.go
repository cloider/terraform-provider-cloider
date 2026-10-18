package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestErrorShapes(t *testing.T) {
	e := parseError(400, []byte(`{"errors":{"_":["Something failed!"],"cluster_name":["Too short"]}}`))
	if e.Error() != "Something failed!\ncluster_name: Too short" {
		t.Fatalf("message: %q", e.Error())
	}
	if e := parseError(404, []byte("Not Found")); e.Error() != "Not Found" || !IsNotFound(e) {
		t.Fatalf("plain text 404: %q", e.Error())
	}
	if e := parseError(503, []byte("<html>busy</html>")); e.Error() != "503 Service Unavailable" {
		t.Fatalf("html body: %q", e.Error())
	}
	if !MayHaveGoneThrough(parseError(504, nil)) || MayHaveGoneThrough(parseError(400, nil)) {
		t.Fatal("only a 502 or 504 may have gone through")
	}
}

func TestClientSendsTokenAndRetriesRateLimit(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer cldr_test" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("User-Agent") != "terraform-provider-cloider/test" {
			t.Errorf("User-Agent = %q", r.Header.Get("User-Agent"))
		}
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		fmt.Fprint(w, `[{"hashed_id":"x1","cluster_name":"web"}]`)
	}))
	defer server.Close()

	var clusters []Cluster
	if err := New(server.URL, "cldr_test", "terraform-provider-cloider/test", false).Get(context.Background(), "/clusters", nil, &clusters); err != nil {
		t.Fatal(err)
	}
	if len(clusters) != 1 || clusters[0].HashedId != "x1" || calls.Load() != 2 {
		t.Fatalf("got %+v after %d calls", clusters, calls.Load())
	}
}

func TestClientDoesNotRetryWritesOn504(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusGatewayTimeout)
	}))
	defer server.Close()

	err := New(server.URL, "", "test", false).Post(context.Background(), "/buckets", map[string]string{}, nil)
	if !MayHaveGoneThrough(err) || calls.Load() != 1 {
		t.Fatalf("a write may have gone through, it must not be retried: %v after %d calls", err, calls.Load())
	}
}

func TestWaitStatusReadsGoneAsTerminated(t *testing.T) {
	PollInterval = 10 * time.Millisecond
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			fmt.Fprint(w, "Terminating\n")
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	status, err := New(server.URL, "", "test", false).WaitStatus(context.Background(), "/buckets/b1/status", "Terminated")
	if err != nil || status != "Terminated" {
		t.Fatalf("got %q, %v", status, err)
	}
}

func TestWaitStatusFailsOnFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Failed")
	}))
	defer server.Close()

	if _, err := New(server.URL, "", "test", false).WaitStatus(context.Background(), "/buckets/b1/status", "Terminated"); err == nil {
		t.Fatal("Failed must end the wait as an error")
	}
}
