package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// fake is a stand-in server that records what it was sent.
type fake struct {
	t      *testing.T
	calls  []string
	bodies []map[string]any
	reply  map[string]string
	status map[string]int
}

func newFake(t *testing.T) (*fake, *APIClient) {
	f := &fake{t: t, reply: map[string]string{}, status: map[string]int{}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		if r.Header.Get("X-Rq-Sig") != "key" {
			t.Errorf("missing signature header")
		}
		if !strings.HasPrefix(r.Header.Get("User-Agent"), "Mozilla/") {
			t.Errorf("user agent = %q", r.Header.Get("User-Agent"))
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.calls = append(f.calls, r.URL.Path)
		f.bodies = append(f.bodies, body)
		if s, ok := f.status[r.URL.Path]; ok {
			w.WriteHeader(s)
		}
		_, _ = w.Write([]byte(f.reply[r.URL.Path]))
	}))
	t.Cleanup(srv.Close)
	c := newAPIClient("key", "alice")
	c.base, c.maxJitter = srv.URL, 0
	return f, c
}

func isolate(t *testing.T) {
	d := t.TempDir()
	t.Setenv("LEDGER_CONFIG", filepath.Join(d, "cfg", "config.json"))
	t.Setenv("LEDGER_CACHE", filepath.Join(d, "cache", "board.json"))
}

const boardJSON = `{"w":120,"b":[
 {"i":7,"t":"Rain tomorrow?","c":"alice","s":0,"p":30,"o":[["yes",10],["no",20]],"m":[10,0],"r":-1,"d":1700000000},
 {"i":8,"t":"Old one","c":"bob","s":1,"p":50,"o":[["a",25],["b",25]],"m":[0,0],"r":1,"d":1690000000}]}`

func TestBoardDecodesScrambledKeys(t *testing.T) {
	f, c := newFake(t)
	f.reply[pathBoard] = boardJSON
	b, err := c.Board()
	if err != nil {
		t.Fatal(err)
	}
	if b.Balance != 120 || len(b.Bets) != 2 {
		t.Fatalf("board = %+v", b)
	}
	if got := b.Bets[0]; got.Title != "Rain tomorrow?" || got.Options[1] != (Option{"no", 20}) || got.Mine[0] != 10 {
		t.Fatalf("bet = %+v", got)
	}
	if b.Bets[1].Winner != 1 || b.Bets[1].Status != statusResolved {
		t.Fatalf("resolved bet = %+v", b.Bets[1])
	}
	if f.bodies[0]["u"] != "alice" {
		t.Fatalf("username not sent: %v", f.bodies[0])
	}
}

func TestDetailHistory(t *testing.T) {
	f, c := newFake(t)
	f.reply[pathDetail] = `{"w":5,"i":7,"t":"x","c":"a","s":0,"p":30,"o":[["y",10],["n",20]],"m":[0,0],"r":-1,"d":1,"h":[[1,[0,0]],[2,[10,0]],[3,[10,20]]]}`
	d, err := c.Detail(7)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.History) != 3 || d.History[2].TS != 3 || d.History[2].Totals[1] != 20 || d.Balance != 5 {
		t.Fatalf("detail = %+v", d)
	}
	if f.bodies[0]["i"] != float64(7) {
		t.Fatalf("body = %v", f.bodies[0])
	}
}

func TestErrorCodes(t *testing.T) {
	f, c := newFake(t)
	f.status[pathStake], f.reply[pathStake] = 400, `{"e":2}`
	_, err := c.Stake(1, 0, 999)
	if err == nil || !strings.Contains(err.Error(), "not enough") {
		t.Fatalf("err = %v", err)
	}
	f.status[pathBoard], f.reply[pathBoard] = 404, `{"e":0}`
	if _, err := c.Board(); !isAuthErr(err) {
		t.Fatalf("want auth error, got %v", err)
	}
}

func TestRequestShapes(t *testing.T) {
	f, c := newFake(t)
	f.reply[pathCreate] = `{"w":90,"i":3}`
	f.reply[pathResolve] = `{"w":1,"s":2}`
	if _, err := c.Create("q", []string{"a", "b"}, 1, 10); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Resolve(3, -1); err != nil {
		t.Fatal(err)
	}
	b := f.bodies[0]
	if b["t"] != "q" || b["k"] != float64(1) || b["a"] != float64(10) || len(b["o"].([]any)) != 2 {
		t.Fatalf("create body = %v", b)
	}
	if f.bodies[1]["k"] != float64(-1) {
		t.Fatalf("resolve body = %v", f.bodies[1])
	}
}

func TestConfigPermissions(t *testing.T) {
	isolate(t)
	if err := saveConfig(&Config{APIKey: "k", Username: "u"}); err != nil {
		t.Fatal(err)
	}
	p, _ := configPath()
	st, err := os.Stat(p)
	if err != nil || st.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v err = %v", st.Mode(), err)
	}
	cfg, _ := loadConfig()
	if cfg.APIKey != "k" || cfg.Username != "u" {
		t.Fatalf("cfg = %+v", cfg)
	}
}

func TestCacheIsPerUser(t *testing.T) {
	isolate(t)
	saveCache(boardCache{Username: "alice", Balance: 9, Bets: []Bet{{ID: 1}}})
	if c := loadCache("alice"); c == nil || c.Balance != 9 {
		t.Fatalf("cache = %+v", c)
	}
	if loadCache("bob") != nil {
		t.Fatal("cache leaked across users")
	}
}

// ── model flow ────────────────────────────────────────────────────────────

func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func send(m model, msgs ...tea.Msg) model {
	for _, msg := range msgs {
		next, _ := m.Update(msg)
		m = next.(model)
	}
	return m
}

func typed(m model, s string) model {
	for _, r := range s {
		m = send(m, key(string(r)))
	}
	return m
}

func loaded(t *testing.T) (model, *fake) {
	isolate(t)
	f, c := newFake(t)
	m := newModel(&Config{APIKey: "key", Username: "alice"}, false)
	m.client = c
	var b boardResp
	if err := json.Unmarshal([]byte(boardJSON), &b); err != nil {
		t.Fatal(err)
	}
	m = send(m, tea.WindowSizeMsg{Width: 100, Height: 40}, boardMsg{resp: &b})
	return m, f
}

func TestBoardViewAndCache(t *testing.T) {
	m, _ := loaded(t)
	v := m.View()
	for _, want := range []string{"Rain tomorrow?", "120 BBs", "won: b", "★"} {
		if !strings.Contains(v, want) {
			t.Errorf("board missing %q:\n%s", want, v)
		}
	}
	if loadCache("alice") == nil {
		t.Error("board was not cached")
	}
}

func TestOnlyCreatorSeesResolve(t *testing.T) {
	m, _ := loaded(t)
	m.cursor = 0
	m = send(m, key("enter"), detailMsg{id: 7, d: &Detail{Bet: m.bets[0], Balance: 120}})
	if m.screen != scDetail || !strings.Contains(m.View(), "r resolve") {
		t.Fatalf("creator should see resolve:\n%s", m.View())
	}
	m = send(m, key("esc"))
	m.cursor = 1
	m = send(m, key("enter"), detailMsg{id: 8, d: &Detail{Bet: m.bets[1]}})
	if strings.Contains(m.View(), "resolve") || strings.Contains(m.View(), "b bet") {
		t.Fatalf("closed bet by someone else offers actions:\n%s", m.View())
	}
	m.cursor = 0
	m.bets[0].Creator = "bob"
	m = send(m, key("esc"), key("enter"), detailMsg{id: 7, d: &Detail{Bet: m.bets[0]}})
	if strings.Contains(m.View(), "r resolve") {
		t.Fatal("non-creator offered resolve")
	}
}

func TestStakeFlowUpdatesLocally(t *testing.T) {
	m, _ := loaded(t)
	m = send(m, key("enter"), detailMsg{id: 7, d: &Detail{Bet: m.bets[0], Balance: 120,
		History: []Point{{1, []int{0, 0}}, {2, []int{10, 20}}}}})
	m = send(m, key("b"), key("down"), key("enter"))
	if m.screen != scStake || m.sStep != ssAmount || m.sOption != 1 {
		t.Fatalf("state = %v %v %v", m.screen, m.sStep, m.sOption)
	}
	// over balance is caught before any request
	m = typed(m, "500")
	m = send(m, key("enter"))
	if !m.failed || m.busy {
		t.Fatalf("over-balance stake should fail locally: %q", m.flash)
	}
	m.input.SetValue("25")
	next, cmd := m.Update(key("enter"))
	m = next.(model)
	if !m.busy || cmd == nil {
		t.Fatal("valid stake should start a request")
	}
	m = send(m, stakeMsg{7, 1, 25, &actionResp{Balance: 95}, nil})
	if m.balance != 95 || m.screen != scDetail {
		t.Fatalf("balance %d screen %v", m.balance, m.screen)
	}
	if m.detail.Options[1].Total != 45 || m.detail.Pool != 55 || m.detail.Mine[1] != 25 {
		t.Fatalf("detail = %+v", m.detail.Bet)
	}
	if got := m.bets[0].Pool; got != 55 {
		t.Fatalf("board pool = %d", got)
	}
	if n := len(m.detail.History); n != 3 {
		t.Fatalf("history len = %d", n)
	}
}

func TestResolveNeedsConfirmAndVoidRow(t *testing.T) {
	m, _ := loaded(t)
	m = send(m, key("enter"), detailMsg{id: 7, d: &Detail{Bet: m.bets[0]}})
	m = send(m, key("r"), key("down"), key("down"))
	if m.rOption != 2 {
		t.Fatalf("void row not reachable: %d", m.rOption)
	}
	if !strings.Contains(m.View(), "VOID") {
		t.Fatal("no void row")
	}
	next, cmd := m.Update(key("enter"))
	m = next.(model)
	if !m.rConfirm || cmd != nil {
		t.Fatal("enter must ask for confirmation, not fire")
	}
	m = send(m, key("n"))
	if m.rConfirm {
		t.Fatal("n should cancel")
	}
	m = send(m, key("enter"))
	next, cmd = m.Update(key("y"))
	m = next.(model)
	if !m.busy || cmd == nil {
		t.Fatal("y should fire the request")
	}
	m = send(m, resolveMsg{7, -1, &actionResp{Balance: 130, Status: statusVoid}, nil})
	if m.bets[0].Status != statusVoid || m.balance != 130 {
		t.Fatalf("bet = %+v balance %d", m.bets[0], m.balance)
	}
}

func TestCreateFlow(t *testing.T) {
	m, _ := loaded(t)
	m = send(m, key("n"))
	m = typed(m, "Who wins?")
	m = send(m, key("enter"))
	m = send(m, key("enter")) // empty with no outcomes: refused
	if m.cStep != csOptions || !m.failed {
		t.Fatal("should need two outcomes")
	}
	for _, o := range []string{"Ann", "Ben"} {
		m = typed(m, o)
		m = send(m, key("enter"))
	}
	m = send(m, key("enter")) // finish outcomes
	if m.cStep != csSide {
		t.Fatalf("step = %v", m.cStep)
	}
	m = send(m, key("down"), key("enter"))
	if m.cStep != csAmount || m.cSide != 1 {
		t.Fatalf("step %v side %d", m.cStep, m.cSide)
	}
	m.input.SetValue("4")
	m = send(m, key("enter"))
	if !m.failed || m.busy {
		t.Fatalf("under-minimum accepted: %q", m.flash)
	}
	m.input.SetValue("10")
	next, cmd := m.Update(key("enter"))
	m = next.(model)
	if !m.busy || cmd == nil {
		t.Fatal("valid create should fire")
	}
	m = send(m, createMsg{"Who wins?", []string{"Ann", "Ben"}, 1, 10, &actionResp{Balance: 110, ID: 9}, nil})
	if m.bets[0].ID != 9 || m.bets[0].Options[1].Total != 10 || m.bets[0].Mine[1] != 10 || m.screen != scBoard {
		t.Fatalf("bets[0] = %+v", m.bets[0])
	}
}

func TestNoBackgroundPolling(t *testing.T) {
	m, _ := loaded(t)
	// Init is the only thing that may fetch on its own; updates from window
	// resizes and idle keys must not start requests.
	for _, msg := range []tea.Msg{tea.WindowSizeMsg{Width: 80, Height: 30}, key("j"), key("k")} {
		next, cmd := m.Update(msg)
		m = next.(model)
		if cmd != nil {
			t.Fatalf("%T started a command", msg)
		}
	}
}

func TestGraph(t *testing.T) {
	pts := []Point{{100, []int{0, 0}}, {200, []int{10, 0}}, {300, []int{10, 30}}, {400, []int{40, 30}}}
	g := renderGraph(pts, 2, 60, 8)
	if lines := strings.Count(g, "\n"); lines != 8 {
		t.Fatalf("rows = %d\n%s", lines+1, g)
	}
	for _, want := range []string{"100%", " 50%", "  0%", "━", "┃"} {
		if !strings.Contains(g, want) {
			t.Errorf("graph missing %q", want)
		}
	}
	if !strings.Contains(renderGraph([]Point{{1, []int{0, 0}}}, 2, 60, 8), "no stakes") {
		t.Error("empty pool should say so")
	}
	if renderGraph(pts, 2, 5, 8) != "" {
		t.Error("tiny width should draw nothing")
	}
}

func TestOdds(t *testing.T) {
	if got := oddsLine(25, 100); !strings.Contains(got, "25%") || !strings.Contains(got, "x4.00") {
		t.Fatalf("odds = %q", got)
	}
	if !strings.Contains(oddsLine(0, 100), "no money") {
		t.Fatal("zero stake")
	}
}

func TestSetupSavesOnlyAfterValidation(t *testing.T) {
	isolate(t)
	f, c := newFake(t)
	f.status[pathBoard], f.reply[pathBoard] = 404, `{"e":0}`
	t.Setenv("LEDGER_URL", c.base)
	m := newModel(nil, true)
	m = typed(m, "key")
	m = send(m, key("enter"))
	m = typed(m, "alice")
	next, cmd := m.Update(key("enter"))
	m = next.(model)
	if cmd == nil {
		t.Fatal("no validation command")
	}
	if p, _ := configPath(); fileExists(p) {
		t.Fatal("config written before validation")
	}
}

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }
