package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"time"
)

// The server keeps its paths and JSON keys deliberately opaque; this file is
// the only place that knows the legend (see docs/bb-api.md in the website).

const defaultBaseURL = "https://rhysfuller.com"

const (
	pathBoard   = "/api/v/kx3q/"
	pathDetail  = "/api/v/m8tz/"
	pathCreate  = "/api/v/p2wr/"
	pathStake   = "/api/v/h6nc/"
	pathResolve = "/api/v/j9ud/"
	pathWins    = "/api/v/r5vk/"
)

const userAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:128.0) Gecko/20100101 Firefox/128.0"

const (
	minCreate = 5
	maxOpts   = 8
)

const (
	statusOpen = iota
	statusResolved
	statusVoid
)

// Option is [label, total] on the wire.
type Option struct {
	Label string
	Total int
}

func (o *Option) UnmarshalJSON(b []byte) error {
	var raw [2]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if err := json.Unmarshal(raw[0], &o.Label); err != nil {
		return err
	}
	return json.Unmarshal(raw[1], &o.Total)
}

func (o Option) MarshalJSON() ([]byte, error) {
	return json.Marshal([]any{o.Label, o.Total})
}

type Bet struct {
	ID      int      `json:"i"`
	Title   string   `json:"t"`
	Creator string   `json:"c"`
	Status  int      `json:"s"`
	Pool    int      `json:"p"`
	Options []Option `json:"o"`
	Mine    []int    `json:"m"`
	Winner  int      `json:"r"`
	Created int64    `json:"d"`
	// Paid is what the bet returned to this user (payout or refund), zero
	// while it is open or if they lost. Resolved is when it closed.
	Paid     int   `json:"y"`
	Resolved int64 `json:"z"`
}

// clone copies the slices, so the board's copy of a bet and the open detail
// never share storage.
func (b Bet) clone() Bet {
	b.Options = append([]Option(nil), b.Options...)
	b.Mine = append([]int(nil), b.Mine...)
	return b
}

// Point is the running total per option just after one stake: [ts, [totals]].
type Point struct {
	TS     int64
	Totals []int
}

func (p *Point) UnmarshalJSON(b []byte) error {
	var raw [2]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if err := json.Unmarshal(raw[0], &p.TS); err != nil {
		return err
	}
	return json.Unmarshal(raw[1], &p.Totals)
}

func (p Point) MarshalJSON() ([]byte, error) {
	return json.Marshal([]any{p.TS, p.Totals})
}

type Detail struct {
	Bet
	Balance int     `json:"w"`
	History []Point `json:"h"`
}

type boardResp struct {
	Balance int   `json:"w"`
	Bets    []Bet `json:"b"`
}

type actionResp struct {
	Balance int `json:"w"`
	ID      int `json:"i"`
	Status  int `json:"s"`
	Paid    int `json:"y"`
}

// APIError carries the server's numeric code.
type APIError struct {
	Code   int
	Status int
}

var errMessages = map[int]string{
	0: "rejected: check your key and username",
	1: "that was not accepted",
	2: "not enough BBs, or under the minimum",
	3: "that bet is already closed",
	4: "only the creator can do that",
	5: "no such bet",
}

func (e *APIError) Error() string {
	if m, ok := errMessages[e.Code]; ok {
		return m
	}
	return fmt.Sprintf("error %d", e.Status)
}

type APIClient struct {
	secret   string
	username string
	base     string
	http     *http.Client
	// maxJitter is the longest random pause before each request, so a burst
	// of key presses does not make a regular beat on the wire.
	maxJitter time.Duration
}

func newAPIClient(secret, username string) *APIClient {
	base := defaultBaseURL
	if v := os.Getenv("LEDGER_URL"); v != "" {
		base = v
	}
	return &APIClient{
		secret:    secret,
		username:  username,
		base:      base,
		http:      &http.Client{Timeout: 10 * time.Second},
		maxJitter: 1200 * time.Millisecond,
	}
}

func (c *APIClient) post(path string, req map[string]any, out any) error {
	if c.maxJitter > 0 {
		time.Sleep(time.Duration(rand.Int63n(int64(c.maxJitter))))
	}
	req["u"] = c.username
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	r, err := http.NewRequest(http.MethodPost, c.base+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Rq-Sig", c.secret)
	r.Header.Set("User-Agent", userAgent)
	r.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(r)
	if err != nil {
		return fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return fmt.Errorf("read error: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		var e struct {
			E int `json:"e"`
		}
		if json.Unmarshal(data, &e) != nil {
			return &APIError{Code: -1, Status: resp.StatusCode}
		}
		return &APIError{Code: e.E, Status: resp.StatusCode}
	}
	return json.Unmarshal(data, out)
}

func (c *APIClient) Board() (*boardResp, error) {
	var out boardResp
	if err := c.post(pathBoard, map[string]any{}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Wins lists every bet this user has won, newest first.
func (c *APIClient) Wins() (*boardResp, error) {
	var out boardResp
	if err := c.post(pathWins, map[string]any{}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *APIClient) Detail(id int) (*Detail, error) {
	var out Detail
	if err := c.post(pathDetail, map[string]any{"i": id}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *APIClient) Create(title string, labels []string, side, amount int) (*actionResp, error) {
	var out actionResp
	err := c.post(pathCreate, map[string]any{"t": title, "o": labels, "k": side, "a": amount}, &out)
	return &out, err
}

func (c *APIClient) Stake(id, option, amount int) (*actionResp, error) {
	var out actionResp
	err := c.post(pathStake, map[string]any{"i": id, "k": option, "a": amount}, &out)
	return &out, err
}

// Resolve settles the bet; option -1 voids it.
func (c *APIClient) Resolve(id, option int) (*actionResp, error) {
	var out actionResp
	err := c.post(pathResolve, map[string]any{"i": id, "k": option}, &out)
	return &out, err
}

func isAuthErr(err error) bool {
	var e *APIError
	return errors.As(err, &e) && e.Code == 0
}
