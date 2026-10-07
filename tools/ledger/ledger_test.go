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
 {"i":8,"t":"Old one","c":"bob","s":1,"p":50,"o":[["a",25],["b",25]],"m":[10,0],"r":1,"d":1690000000,"y":0,"z":1690001000},
 {"i":9,"t":"Cheap win","c":"bob","s":1,"p":60,"o":[["a",20],["b",40]],"m":[0,20],"r":1,"d":1690000000,"y":60,"z":1690002000},
 {"i":10,"t":"Sat out","c":"bob","s":1,"p":10,"o":[["a",5],["b",5]],"m":[0,0],"r":0,"d":1690000000,"y":0,"z":1690003000},
 {"i":11,"t":"Voided","c":"bob","s":2,"p":10,"o":[["a",5],["b",5]],"m":[5,0],"r":-1,"d":1690000000,"y":5,"z":1690004000}]}`

func TestBoardDecodesScrambledKeys(t *testing.T) {
	f, c := newFake(t)
	f.reply[pathBoard] = boardJSON
	b, err := c.Board()
	if err != nil {
		t.Fatal(err)
	}
	if b.Balance != 120 || len(b.Bets) != 5 {
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
	for _, want := range []string{"Rain tomorrow?", "120 BBs", "in for 10", "★"} {
		if !strings.Contains(v, want) {
			t.Errorf("board missing %q:\n%s", want, v)
		}
	}
	if loadCache("alice") == nil {
		t.Error("board was not cached")
	}
}

// Completed wagers are not live wagers: the board lists only open bets.
func TestBoardListsOnlyOpenBets(t *testing.T) {
	m, _ := loaded(t)
	v := m.View()
	for _, gone := range []string{"Old one", "Cheap win", "Sat out", "Voided", "you won", "you lost", "refunded", "no stake"} {
		if strings.Contains(v, gone) {
			t.Errorf("the board still shows %q:\n%s", gone, v)
		}
	}
	if got := m.active(); len(got) != 1 || m.bets[got[0]].ID != 7 {
		t.Fatalf("active bets = %v", got)
	}
	if m.selected() == nil || m.selected().ID != 7 {
		t.Fatal("the cursor is not on the open bet")
	}
	// Down does not walk onto hidden bets.
	m = send(m, key("down"), key("down"))
	if m.cursor != 0 {
		t.Errorf("cursor moved to %d on a one-bet board", m.cursor)
	}

	m2, _ := loaded(t)
	m2.bets = m2.bets[1:] // nothing open at all
	m2.clampCursor()
	if v := m2.View(); !strings.Contains(v, "no open bets") {
		t.Errorf("an empty board should say so:\n%s", v)
	}
	if m2.selected() != nil {
		t.Error("enter on an empty board has something selected")
	}
}

// A bet leaves the board the moment it is resolved or voided.
func TestResolvingTakesABetOffTheBoard(t *testing.T) {
	m, _ := loaded(t)
	m = send(m, key("enter"), detailMsg{id: 7, d: &Detail{Bet: m.bets[0]}})
	m = send(m, resolveMsg{7, 1, &actionResp{Balance: 90, Status: statusResolved, Paid: 0}, nil})
	m = send(m, key("esc"))
	if len(m.active()) != 0 || strings.Contains(m.View(), "Rain tomorrow?") {
		t.Fatalf("a resolved bet is still on the board:\n%s", m.View())
	}
	if m.cursor != 0 {
		t.Errorf("cursor is %d", m.cursor)
	}
	// It is still reachable, and says how it went, from its own page.
	m.detail = &Detail{Bet: m.bets[0]}
	m.screen = scDetail
	if !strings.Contains(m.View(), "you lost 10") {
		t.Errorf("the closed bet's page lost its result:\n%s", m.View())
	}
}

func TestBetPagesNeverSayWonUnlessYouBackedTheWinner(t *testing.T) {
	m, _ := loaded(t)
	for _, c := range []struct {
		id   int
		want string
		not  string
	}{
		{8, "you lost 10", "you won"},
		{9, "you won +40", "you lost"},
		{10, "no stake", "you won"},
		{11, "refunded 5", "you won"},
	} {
		m.screen = scBoard
		m.detail = &Detail{Bet: m.bets[m.betIndex(c.id)]}
		m.screen = scDetail
		v := m.View()
		if !strings.Contains(v, c.want) || strings.Contains(v, c.not) {
			t.Errorf("bet %d detail: want %q, not %q:\n%s", c.id, c.want, c.not, v)
		}
		if strings.Contains(v, "won:") {
			t.Errorf("bet %d still labels the outcome as won:\n%s", c.id, v)
		}
	}
}

func TestWonTab(t *testing.T) {
	m, f := loaded(t)
	f.reply[pathWins] = `{"w":120,"b":[{"i":9,"t":"Cheap win","c":"bob","s":1,"p":60,"o":[["a",20],["b",40]],"m":[0,20],"r":1,"d":1,"y":60,"z":1690002000},
	 {"i":3,"t":"Earlier win","c":"bob","s":1,"p":30,"o":[["a",10],["b",20]],"m":[10,0],"r":0,"d":1,"y":30,"z":1680000000}]}`
	next, cmd := m.Update(key("2"))
	m = next.(model)
	if m.tab != tabWon || !m.busy || cmd == nil {
		t.Fatal("opening the Won tab should fetch once")
	}
	var wins boardResp
	if err := json.Unmarshal([]byte(f.reply[pathWins]), &wins); err != nil {
		t.Fatal(err)
	}
	m = send(m, winsMsg{resp: &wins})
	v := m.View()
	for _, want := range []string{"Won", "2 wins", "+60 BBs net", "Cheap win", "staked 20 → paid 60 (+40)", "Earlier win", "staked 10 → paid 30 (+20)"} {
		if !strings.Contains(v, want) {
			t.Errorf("won tab missing %q:\n%s", want, v)
		}
	}
	// going back and forth must not refetch
	m = send(m, key("1"))
	next, cmd = m.Update(key("2"))
	m = next.(model)
	if cmd != nil || m.busy {
		t.Fatal("won tab refetched without being asked")
	}
	// cursor + enter open the highlighted win
	m = send(m, key("down"))
	next, cmd = m.Update(key("enter"))
	m = next.(model)
	if m.screen != scDetail || m.detail.ID != 3 || cmd == nil {
		t.Fatalf("enter opened %+v", m.detail)
	}
	// r on the tab refetches the wins, not the board
	m = send(m, detailMsg{id: 3, d: &Detail{Bet: m.detail.Bet}}, key("esc"))
	_, cmd = m.Update(key("r"))
	if cmd == nil {
		t.Fatal("r should refresh")
	}
}

func TestWinsInvalidatedByResolving(t *testing.T) {
	m, _ := loaded(t)
	m.wonLoaded = true
	m = send(m, key("enter"), detailMsg{id: 7, d: &Detail{Bet: m.bets[0]}})
	m = send(m, resolveMsg{7, 0, &actionResp{Balance: 150, Status: statusResolved, Paid: 30}, nil})
	if m.wonLoaded {
		t.Fatal("won list should be refetched next visit")
	}
	if m.detail.Paid != 30 || !strings.Contains(m.View(), "you won +20") {
		t.Fatalf("detail after resolve:\n%s", m.View())
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

// ── leaderboard ───────────────────────────────────────────────────────────

const leadersJSON = `{"w":120,"l":[["bob",340],["carol",180],["alice",120],["dave",40]]}`

func TestLeadersDecode(t *testing.T) {
	f, c := newFake(t)
	f.reply[pathLeaders] = leadersJSON
	r, err := c.Leaders()
	if err != nil {
		t.Fatal(err)
	}
	if r.Balance != 120 || len(r.Rows) != 4 || r.Rows[0] != (Leader{"bob", 340}) || r.Rows[2].Name != "alice" {
		t.Fatalf("leaders = %+v", r)
	}
	if f.bodies[0]["u"] != "alice" {
		t.Fatalf("username not sent: %v", f.bodies[0])
	}
}

func TestMissingLeaderboardIsSaidPlainly(t *testing.T) {
	f, c := newFake(t)
	f.status[pathLeaders], f.reply[pathLeaders] = 404, `not found`
	_, err := c.Leaders()
	if err == nil || !strings.Contains(err.Error(), "no leaderboard yet") {
		t.Fatalf("err = %v", err)
	}
	// A rejected key is still reported as one.
	f.status[pathLeaders], f.reply[pathLeaders] = 404, `{"e":0}`
	if _, err := c.Leaders(); !isAuthErr(err) {
		t.Fatalf("want auth error, got %v", err)
	}
}

func TestLeaderboardTab(t *testing.T) {
	m, f := loaded(t)
	f.reply[pathLeaders] = leadersJSON
	next, cmd := m.Update(key("3"))
	m = next.(model)
	if m.tab != tabLeaders || !m.busy || cmd == nil {
		t.Fatal("opening the leaderboard should fetch once")
	}
	var lr leadersResp
	if err := json.Unmarshal([]byte(leadersJSON), &lr); err != nil {
		t.Fatal(err)
	}
	m = send(m, leadersMsg{resp: &lr})
	v := m.View()
	for _, want := range []string{"3 Leaderboard", "4 players", "you are 3rd", " 1. bob", "340 BBs", " 3. alice", "← you", " 4. dave"} {
		if !strings.Contains(v, want) {
			t.Errorf("leaderboard missing %q:\n%s", want, v)
		}
	}
	if i1, i2, i3 := strings.Index(v, " 1. bob"), strings.Index(v, " 2. carol"), strings.Index(v, " 3. alice"); !(0 <= i1 && i1 < i2 && i2 < i3) {
		t.Errorf("not in order of balance:\n%s", v)
	}

	// Going back and forth does not refetch; r does.
	m = send(m, key("1"))
	next, cmd = m.Update(key("3"))
	m = next.(model)
	if cmd != nil || m.busy {
		t.Fatal("the leaderboard refetched without being asked")
	}
	_, cmd = m.Update(key("r"))
	if cmd == nil {
		t.Fatal("r should refresh the leaderboard")
	}
	// enter on it opens nothing.
	m2 := send(m, key("enter"))
	if m2.screen != scBoard {
		t.Error("enter on the leaderboard opened a bet")
	}
}

func TestTabCyclesThroughAllThree(t *testing.T) {
	m, _ := loaded(t)
	m.wonLoaded, m.leadersLoaded = true, true
	seen := []int{m.tab}
	for i := 0; i < 3; i++ {
		m = send(m, tea.KeyMsg{Type: tea.KeyTab})
		seen = append(seen, m.tab)
	}
	if seen[0] != tabBoard || seen[1] != tabWon || seen[2] != tabLeaders || seen[3] != tabBoard {
		t.Errorf("tab visited %v", seen)
	}
	m = send(m, tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.tab != tabLeaders {
		t.Errorf("shift+tab from the board went to %d", m.tab)
	}
}

func TestBalancesMoveSoTheLeaderboardIsRefetched(t *testing.T) {
	m, _ := loaded(t)
	m.leadersLoaded = true
	m = send(m, key("enter"), detailMsg{id: 7, d: &Detail{Bet: m.bets[0], Balance: 120}})
	m.sOption = 0
	m = send(m, stakeMsg{id: 7, option: 0, amount: 5, resp: &actionResp{Balance: 115}})
	if m.leadersLoaded {
		t.Error("staking changed your balance, but the leaderboard was kept")
	}
}

func TestLeaderboardScrollsAndFitsSmallWindows(t *testing.T) {
	m, _ := loaded(t)
	var rows []Leader
	for i := 0; i < 40; i++ {
		rows = append(rows, Leader{Name: "player" + strings.Repeat("x", i%7), Balance: 1000 - i*10})
	}
	rows[25].Name = "alice"
	m.tab = tabLeaders
	m = send(m, tea.WindowSizeMsg{Width: 50, Height: 16}, leadersMsg{resp: &leadersResp{Balance: 5, Rows: rows}})
	top := m.View()
	for i := 0; i < 12; i++ {
		m = send(m, key("down"))
	}
	if m.View() == top {
		t.Error("scrolling changed nothing")
	}
	if n := len(strings.Split(m.View(), "\n")); n > 16+2 {
		t.Errorf("view is %d lines in a window of 16", n)
	}
	if !strings.Contains(top, "you are 26th") {
		t.Errorf("rank not shown:\n%s", top)
	}
}
