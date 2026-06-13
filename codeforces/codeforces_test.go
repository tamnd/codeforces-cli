package codeforces

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetSendsUserAgent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request carried no User-Agent")
		}
		_, _ = w.Write([]byte(`{"status":"OK","result":[]}`))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	c := NewClient(cfg)

	body, err := c.get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if len(body) == 0 {
		t.Error("expected non-empty body")
	}
}

func TestGetRetriesOn503(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"status":"OK","result":[]}`))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	cfg.Retries = 5
	c := NewClient(cfg)

	body, err := c.get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if len(body) == 0 {
		t.Errorf("expected body after retries, hits=%d", hits)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
}

func TestGetAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"FAILED","comment":"handles: User with handle abc123xyz not found"}`))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	c := NewClient(cfg)

	var v []wireUser
	err := c.getJSON(context.Background(), srv.URL, &v)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != ErrNotFound {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

func TestProblemsFilter(t *testing.T) {
	payload := `{
		"status": "OK",
		"result": {
			"problems": [
				{"contestId": 1, "index": "A", "name": "Easy", "rating": 800, "tags": ["math"]},
				{"contestId": 2, "index": "B", "name": "Medium", "rating": 1200, "tags": ["dp"]},
				{"contestId": 3, "index": "C", "name": "Hard", "rating": 1400, "tags": ["graphs"]}
			],
			"problemStatistics": [
				{"contestId": 1, "index": "A", "solvedCount": 50000},
				{"contestId": 2, "index": "B", "solvedCount": 20000},
				{"contestId": 3, "index": "C", "solvedCount": 5000}
			]
		}
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	c := NewClient(cfg)

	probs, err := c.Problems(context.Background(), "", 1200, 1400)
	if err != nil {
		t.Fatal(err)
	}
	if len(probs) != 2 {
		t.Fatalf("got %d problems, want 2", len(probs))
	}
	// sorted by solved count descending: Medium (20000) first, Hard (5000) second
	if probs[0].ID != "2B" {
		t.Errorf("first problem id = %q, want %q", probs[0].ID, "2B")
	}
	if probs[1].ID != "3C" {
		t.Errorf("second problem id = %q, want %q", probs[1].ID, "3C")
	}
	if probs[0].Rank != 1 {
		t.Errorf("first rank = %d, want 1", probs[0].Rank)
	}
}
