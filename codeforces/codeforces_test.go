package codeforces

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestProblemsParses(t *testing.T) {
	payload := `{"status":"OK","result":{"problems":[
		{"contestId":1234,"index":"A","name":"Two Sum","rating":800,"tags":["math","implementation"]},
		{"contestId":1234,"index":"B","name":"Array Sort","rating":1200,"tags":["sortings"]}
	],"problemStatistics":[
		{"contestId":1234,"index":"A","solvedCount":50000},
		{"contestId":1234,"index":"B","solvedCount":30000}
	]}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	c := NewClient(cfg)

	probs, err := c.Problems(context.Background(), "", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(probs) != 2 {
		t.Fatalf("got %d problems, want 2", len(probs))
	}
	// sorted by solved: 1234A (50000) first, 1234B (30000) second
	p := probs[0]
	if p.ID != "1234A" {
		t.Errorf("id = %q, want 1234A", p.ID)
	}
	if p.Name != "Two Sum" {
		t.Errorf("name = %q, want Two Sum", p.Name)
	}
	if p.Rating != 800 {
		t.Errorf("rating = %d, want 800", p.Rating)
	}
	if p.Tags != "math;implementation" {
		t.Errorf("tags = %q, want math;implementation", p.Tags)
	}
	if p.Solved != 50000 {
		t.Errorf("solved = %d, want 50000", p.Solved)
	}
	if p.URL != "https://codeforces.com/problemset/problem/1234/A" {
		t.Errorf("url = %q", p.URL)
	}
}

func TestProblemsSolvedCountJoined(t *testing.T) {
	// Statistics in reverse order from problems -- must join by key, not index.
	payload := `{"status":"OK","result":{"problems":[
		{"contestId":100,"index":"A","name":"Easy","rating":800,"tags":[]},
		{"contestId":200,"index":"B","name":"Hard","rating":2000,"tags":[]}
	],"problemStatistics":[
		{"contestId":200,"index":"B","solvedCount":5000},
		{"contestId":100,"index":"A","solvedCount":80000}
	]}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	c := NewClient(cfg)

	probs, err := c.Problems(context.Background(), "", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(probs) != 2 {
		t.Fatalf("got %d problems, want 2", len(probs))
	}
	// sorted by solved: 100A (80000) first
	if probs[0].ID != "100A" || probs[0].Solved != 80000 {
		t.Errorf("first: id=%s solved=%d, want 100A/80000", probs[0].ID, probs[0].Solved)
	}
	if probs[1].ID != "200B" || probs[1].Solved != 5000 {
		t.Errorf("second: id=%s solved=%d, want 200B/5000", probs[1].ID, probs[1].Solved)
	}
}

func TestProblemsRatingFilter(t *testing.T) {
	// Also tests that rating==0 is excluded when bounds are set.
	payload := `{"status":"OK","result":{"problems":[
		{"contestId":1,"index":"A","name":"A","rating":800,"tags":[]},
		{"contestId":2,"index":"A","name":"B","rating":1200,"tags":[]},
		{"contestId":3,"index":"A","name":"C","rating":2000,"tags":[]},
		{"contestId":4,"index":"A","name":"D","rating":0,"tags":[]}
	],"problemStatistics":[
		{"contestId":1,"index":"A","solvedCount":100000},
		{"contestId":2,"index":"A","solvedCount":50000},
		{"contestId":3,"index":"A","solvedCount":10000},
		{"contestId":4,"index":"A","solvedCount":5000}
	]}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	c := NewClient(cfg)

	probs, err := c.Problems(context.Background(), "", 1000, 1500)
	if err != nil {
		t.Fatal(err)
	}
	// only "B" with rating 1200 should match; D (rating=0) excluded when bounds set
	if len(probs) != 1 {
		t.Fatalf("got %d problems, want 1", len(probs))
	}
	if probs[0].ID != "2A" {
		t.Errorf("id = %q, want 2A", probs[0].ID)
	}
}

func TestProblemsSortedBySolved(t *testing.T) {
	payload := `{"status":"OK","result":{"problems":[
		{"contestId":1,"index":"A","name":"X","rating":0,"tags":[]},
		{"contestId":2,"index":"A","name":"Y","rating":0,"tags":[]},
		{"contestId":3,"index":"A","name":"Z","rating":0,"tags":[]}
	],"problemStatistics":[
		{"contestId":1,"index":"A","solvedCount":500},
		{"contestId":2,"index":"A","solvedCount":20000},
		{"contestId":3,"index":"A","solvedCount":8000}
	]}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	c := NewClient(cfg)

	probs, err := c.Problems(context.Background(), "", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(probs) != 3 {
		t.Fatalf("got %d problems, want 3", len(probs))
	}
	if probs[0].Solved != 20000 || probs[1].Solved != 8000 || probs[2].Solved != 500 {
		t.Errorf("wrong sort order: %d %d %d", probs[0].Solved, probs[1].Solved, probs[2].Solved)
	}
	if probs[0].Rank != 1 || probs[1].Rank != 2 || probs[2].Rank != 3 {
		t.Errorf("wrong ranks: %d %d %d", probs[0].Rank, probs[1].Rank, probs[2].Rank)
	}
}

func TestGetUser(t *testing.T) {
	payload := `{"status":"OK","result":[{
		"handle":"tourist",
		"firstName":"Gennady",
		"lastName":"Korotkevich",
		"country":"Belarus",
		"rank":"legendary grandmaster",
		"rating":3979,
		"maxRating":3979,
		"friendOfCount":50000
	}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	c := NewClient(cfg)

	users, err := c.UserInfo(context.Background(), []string{"tourist"})
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 {
		t.Fatalf("got %d users, want 1", len(users))
	}
	u := users[0]
	if u.Handle != "tourist" {
		t.Errorf("handle = %q, want tourist", u.Handle)
	}
	if u.Name != "Gennady Korotkevich" {
		t.Errorf("name = %q, want Gennady Korotkevich", u.Name)
	}
	if u.Country != "Belarus" {
		t.Errorf("country = %q, want Belarus", u.Country)
	}
	if u.Rank != "legendary grandmaster" {
		t.Errorf("rank = %q, want legendary grandmaster", u.Rank)
	}
	if u.Rating != 3979 {
		t.Errorf("rating = %d, want 3979", u.Rating)
	}
	if u.MaxRat != 3979 {
		t.Errorf("max_rating = %d, want 3979", u.MaxRat)
	}
	if u.Friends != 50000 {
		t.Errorf("friends = %d, want 50000", u.Friends)
	}
	if u.URL != "https://codeforces.com/profile/tourist" {
		t.Errorf("url = %q", u.URL)
	}
}

func TestGetUserNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"FAILED","comment":"handles: User with handle xyz123abc not found"}`))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	c := NewClient(cfg)

	_, err := c.UserInfo(context.Background(), []string{"xyz123abc"})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

func TestGetUserUnranked(t *testing.T) {
	payload := `{"status":"OK","result":[{
		"handle":"newbie",
		"firstName":"",
		"lastName":"",
		"country":"",
		"rank":"",
		"rating":0,
		"maxRating":0,
		"friendOfCount":0
	}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	c := NewClient(cfg)

	users, err := c.UserInfo(context.Background(), []string{"newbie"})
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 {
		t.Fatalf("got %d users, want 1", len(users))
	}
	u := users[0]
	if u.Rank != "" {
		t.Errorf("rank = %q, want empty", u.Rank)
	}
	if u.Rating != 0 {
		t.Errorf("rating = %d, want 0", u.Rating)
	}
	if u.Name != "" {
		t.Errorf("name = %q, want empty", u.Name)
	}
}

func TestContestsAll(t *testing.T) {
	payload := `{"status":"OK","result":[
		{"id":1901,"name":"Edu Round 160","type":"ICPC","phase":"FINISHED","durationSeconds":9000,"startTimeSeconds":1704153600},
		{"id":1900,"name":"CF Round 900","type":"CF","phase":"FINISHED","durationSeconds":7200,"startTimeSeconds":1704067200},
		{"id":1999,"name":"CF Round 1000","type":"CF","phase":"BEFORE","durationSeconds":9000,"startTimeSeconds":1750000000}
	]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	c := NewClient(cfg)

	contests, err := c.Contests(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(contests) != 3 {
		t.Fatalf("got %d contests, want 3", len(contests))
	}
	// check fields on first returned item (order not guaranteed by library; CLI sorts)
	found := false
	for _, ct := range contests {
		if ct.ID == 1900 {
			found = true
			if ct.Name != "CF Round 900" {
				t.Errorf("name = %q", ct.Name)
			}
			if ct.Phase != "FINISHED" {
				t.Errorf("phase = %q", ct.Phase)
			}
			if ct.Duration != "2h00m" {
				t.Errorf("duration = %q, want 2h00m", ct.Duration)
			}
			if ct.Start != "2024-01-01 00:00" {
				t.Errorf("start = %q, want 2024-01-01 00:00", ct.Start)
			}
			if ct.URL != "https://codeforces.com/contest/1900" {
				t.Errorf("url = %q", ct.URL)
			}
		}
	}
	if !found {
		t.Error("contest 1900 not found in results")
	}
}

func TestContestsDuration(t *testing.T) {
	cases := []struct {
		secs int
		want string
	}{
		{7200, "2h00m"},
		{9000, "2h30m"},
		{3661, "1h01m"},
	}
	for _, tc := range cases {
		got := formatDuration(tc.secs)
		if got != tc.want {
			t.Errorf("formatDuration(%d) = %q, want %q", tc.secs, got, tc.want)
		}
	}
}

func TestAPIFailedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"FAILED","comment":"Call limit exceeded"}`))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	c := NewClient(cfg)

	_, err := c.UserInfo(context.Background(), []string{"tourist"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, ErrNotFound) {
		t.Error("should not be ErrNotFound for 'Call limit exceeded'")
	}
	if !strings.Contains(err.Error(), "Call limit exceeded") {
		t.Errorf("error message %q should contain 'Call limit exceeded'", err.Error())
	}
}

func TestTagsStatic(t *testing.T) {
	tags := Tags()
	if len(tags) < 35 {
		t.Errorf("Tags() returned %d tags, want at least 35", len(tags))
	}
	want := []string{"dp", "greedy", "math", "implementation"}
	found := make(map[string]bool)
	for _, tg := range tags {
		found[tg.Name] = true
	}
	for _, w := range want {
		if !found[w] {
			t.Errorf("tag %q not found in Tags()", w)
		}
	}
	// verify sorted
	for i := 1; i < len(tags); i++ {
		if tags[i].Name < tags[i-1].Name {
			t.Errorf("tags not sorted: %q before %q", tags[i-1].Name, tags[i].Name)
			break
		}
	}
}
