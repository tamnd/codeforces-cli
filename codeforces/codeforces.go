// Package codeforces is the library behind the cf command: the HTTP client,
// request shaping, and the typed data models for Codeforces.
//
// The Codeforces public API lives at https://codeforces.com/api and requires
// no authentication for read-only access. All responses share the envelope
// {"status":"OK","result":<payload>} or {"status":"FAILED","comment":"reason"}.
package codeforces

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

// DefaultUserAgent identifies the client to the Codeforces API.
const DefaultUserAgent = "cf/dev (+https://github.com/tamnd/codeforces-cli)"

// ErrNotFound is returned when the API reports the requested resource does not exist.
var ErrNotFound = errors.New("not found")

// Config holds constructor parameters.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Retries   int
	Timeout   time.Duration
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL:   "https://codeforces.com/api",
		UserAgent: DefaultUserAgent,
		Rate:      200 * time.Millisecond,
		Retries:   3,
		Timeout:   30 * time.Second,
	}
}

// Client talks to the Codeforces public API.
type Client struct {
	httpClient *http.Client
	userAgent  string
	baseURL    string
	rate       time.Duration
	retries    int
	mu         sync.Mutex
	last       time.Time
}

// NewClient returns a Client with the given config.
func NewClient(cfg Config) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: cfg.Timeout},
		userAgent:  cfg.UserAgent,
		baseURL:    cfg.BaseURL,
		rate:       cfg.Rate,
		retries:    cfg.Retries,
	}
}

// get fetches a URL with pacing and retries.
func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, rawURL)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.rate <= 0 {
		return
	}
	if wait := c.rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

// getJSON fetches rawURL, decodes the Codeforces envelope, and extracts result into v.
func (c *Client) getJSON(ctx context.Context, rawURL string, v any) error {
	body, err := c.get(ctx, rawURL)
	if err != nil {
		return err
	}
	// decode envelope first to check status
	var env struct {
		Status  string          `json:"status"`
		Comment string          `json:"comment"`
		Result  json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("decode %s: %w", rawURL, err)
	}
	if env.Status != "OK" {
		if strings.Contains(strings.ToLower(env.Comment), "not found") {
			return ErrNotFound
		}
		return fmt.Errorf("api error: %s", env.Comment)
	}
	if err := json.Unmarshal(env.Result, v); err != nil {
		return fmt.Errorf("decode result %s: %w", rawURL, err)
	}
	return nil
}

// ─── wire types ──────────────────────────────────────────────────────────────

type wireProblem struct {
	ContestID int      `json:"contestId"`
	Index     string   `json:"index"`
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	Points    float64  `json:"points"`
	Rating    int      `json:"rating"`
	Tags      []string `json:"tags"`
}

type wireProblemStat struct {
	ContestID   int    `json:"contestId"`
	Index       string `json:"index"`
	SolvedCount int    `json:"solvedCount"`
}

type wireProblemSet struct {
	Problems   []wireProblem     `json:"problems"`
	Statistics []wireProblemStat `json:"problemStatistics"`
}

type wireContest struct {
	ID               int    `json:"id"`
	Name             string `json:"name"`
	Type             string `json:"type"`
	Phase            string `json:"phase"`
	Frozen           bool   `json:"frozen"`
	DurationSeconds  int    `json:"durationSeconds"`
	StartTimeSeconds int64  `json:"startTimeSeconds"`
}

type wireUser struct {
	Handle        string `json:"handle"`
	FirstName     string `json:"firstName"`
	LastName      string `json:"lastName"`
	Country       string `json:"country"`
	Rank          string `json:"rank"`
	Rating        int    `json:"rating"`
	MaxRating     int    `json:"maxRating"`
	FriendOfCount int    `json:"friendOfCount"`
}

// ─── public methods ───────────────────────────────────────────────────────────

// Problems fetches the problemset and returns merged Problem records.
// tag may be empty (no filter). minRating and maxRating may be 0 (no bound).
// Results are sorted by solved count descending.
func (c *Client) Problems(ctx context.Context, tag string, minRating, maxRating int) ([]Problem, error) {
	params := url.Values{}
	if tag != "" {
		params.Set("tags", tag)
	}
	rawURL := c.baseURL + "/problemset.problems"
	if len(params) > 0 {
		rawURL += "?" + params.Encode()
	}

	var ps wireProblemSet
	if err := c.getJSON(ctx, rawURL, &ps); err != nil {
		return nil, err
	}

	// build solved count map
	statMap := make(map[string]int, len(ps.Statistics))
	for _, s := range ps.Statistics {
		key := fmt.Sprintf("%d%s", s.ContestID, s.Index)
		statMap[key] = s.SolvedCount
	}

	var out []Problem
	for _, p := range ps.Problems {
		if p.Rating == 0 && (minRating > 0 || maxRating > 0) {
			continue
		}
		if minRating > 0 && p.Rating < minRating {
			continue
		}
		if maxRating > 0 && p.Rating > maxRating {
			continue
		}
		id := fmt.Sprintf("%d%s", p.ContestID, p.Index)
		rec := Problem{
			ID:     id,
			Name:   p.Name,
			Rating: p.Rating,
			Tags:   strings.Join(p.Tags, ";"),
			Solved: statMap[id],
			URL:    fmt.Sprintf("https://codeforces.com/problemset/problem/%d/%s", p.ContestID, p.Index),
		}
		out = append(out, rec)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Solved > out[j].Solved
	})

	for i := range out {
		out[i].Rank = i + 1
	}
	return out, nil
}

// UserInfo fetches one or more user profiles by handle.
func (c *Client) UserInfo(ctx context.Context, handles []string) ([]User, error) {
	rawURL := c.baseURL + "/user.info?handles=" + url.QueryEscape(strings.Join(handles, ";"))
	var users []wireUser
	if err := c.getJSON(ctx, rawURL, &users); err != nil {
		return nil, err
	}
	out := make([]User, len(users))
	for i, u := range users {
		name := strings.TrimSpace(u.FirstName + " " + u.LastName)
		out[i] = User{
			Handle:  u.Handle,
			Name:    name,
			Country: u.Country,
			Rank:    u.Rank,
			Rating:  u.Rating,
			MaxRat:  u.MaxRating,
			Friends: u.FriendOfCount,
			URL:     "https://codeforces.com/profile/" + u.Handle,
		}
	}
	return out, nil
}

// Contests fetches the contest list. gym=false returns regular contests only.
func (c *Client) Contests(ctx context.Context, gym bool) ([]Contest, error) {
	gymStr := "false"
	if gym {
		gymStr = "true"
	}
	rawURL := c.baseURL + "/contest.list?gym=" + gymStr
	var contests []wireContest
	if err := c.getJSON(ctx, rawURL, &contests); err != nil {
		return nil, err
	}
	out := make([]Contest, len(contests))
	for i, wc := range contests {
		out[i] = Contest{
			ID:       wc.ID,
			Name:     wc.Name,
			Phase:    wc.Phase,
			Duration: formatDuration(wc.DurationSeconds),
			Start:    formatStart(wc.StartTimeSeconds),
			URL:      fmt.Sprintf("https://codeforces.com/contest/%d", wc.ID),
		}
	}
	return out, nil
}
