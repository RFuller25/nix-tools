package main

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// The village. Where gold goes once the beds are full: hired help who do the
// chores for a weekly wage, scenery that makes the garden a little kinder, a
// gene lab for copying and forecasting plants, craftspeople who turn a plant
// into a keepsake, and the colours and borders the garden is drawn in.
//
// Nothing here is needed to play, and nothing here can hurt you: a helper you
// cannot pay simply stays at home that week.

const (
	labForecastCost = 15 // a likely spread of seedlings from two flowers
	labForecastKids = 6
	labCloneBase    = 40 // a copy of a plant: this, plus the multiple below of a seed's worth
	labCloneMult    = 6
	labCloneSeeds   = 2
	labPurifyBase   = 20 // make a packet breed true: this, plus the multiple below of its worth
	labPurifyMult   = 3
)

// ---- hired help --------------------------------------------------------------

// Helper is someone on the payroll.
type Helper struct {
	ID       string `json:"id"`
	PaidWeek int    `json:"paid_week"` // the week their wage was last paid for
}

type helperDef struct {
	ID    string
	Name  string
	Blurb string
	Hire  int // paid once, which also covers the first week's wage
	Wage  int // paid each Monday
}

var helperDefs = []helperDef{
	{"waterer", "Dilly the water-carrier", "Walks the rows with two buckets and waters any bed that has got dry. You will not need to water by hand again.", 60, 10},
	{"weeder", "Bracken the hoe-man", "Clears any bed that has gone properly weedy, before it hurts the plant. He takes the weeding gold himself.", 50, 8},
	{"gatherer", "Tilly Seedcoat", "Gathers seed from any plant carrying a good crop and files it in the shed, crossed or selfed just as if you had done it.", 120, 18},
}

func helperDefByID(id string) *helperDef {
	for i := range helperDefs {
		if helperDefs[i].ID == id {
			return &helperDefs[i]
		}
	}
	return nil
}

func (g *Garden) helper(id string) *Helper {
	for i := range g.Helpers {
		if g.Helpers[i].ID == id {
			return &g.Helpers[i]
		}
	}
	return nil
}

// Hired reports whether someone is on the payroll.
func (g *Garden) Hired(id string) bool { return g.helper(id) != nil }

// HelperWorking reports whether someone hired has been paid for the week.
func (g *Garden) HelperWorking(id string, at time.Time) bool {
	h := g.helper(id)
	return h != nil && h.PaidWeek == weekKey(at)
}

// HireHelper takes someone on, paying the fee, which covers this week's wage.
func (g *Garden) HireHelper(id string, now time.Time) error {
	d := helperDefByID(id)
	if d == nil {
		return fmt.Errorf("nobody of that name")
	}
	if g.Hired(id) {
		return fmt.Errorf("%s already works for you", d.Name)
	}
	g.settleWages(now)
	if !g.spend(d.Hire) {
		return fmt.Errorf("%s asks %s to start", d.Name, goldLabel(d.Hire))
	}
	g.Helpers = append(g.Helpers, Helper{ID: id, PaidWeek: weekKey(now)})
	g.Log(now, "Took on %s for %s, and %s a week after.", d.Name, goldLabel(d.Hire), goldLabel(d.Wage))
	return nil
}

// DismissHelper lets someone go. There is no refund.
func (g *Garden) DismissHelper(id string, now time.Time) error {
	for i := range g.Helpers {
		if g.Helpers[i].ID == id {
			g.Helpers = append(g.Helpers[:i], g.Helpers[i+1:]...)
			g.Log(now, "Thanked %s and let them go.", helperDefByID(id).Name)
			return nil
		}
	}
	return fmt.Errorf("they do not work for you")
}

// settleWages pays everyone on the payroll once for each new week. Anyone
// who cannot be paid stays at home until the next Monday.
func (g *Garden) settleWages(at time.Time) {
	week := weekKey(at)
	if g.WagesWeek == week {
		return
	}
	g.WagesWeek = week
	for i := range g.Helpers {
		h := &g.Helpers[i]
		d := helperDefByID(h.ID)
		if d == nil {
			continue
		}
		if g.spend(d.Wage) {
			h.PaidWeek = week
			g.WagesPaid += d.Wage
		} else {
			g.Log(at, "Could not pay %s this week, so they stayed at home.", d.Name)
		}
	}
}

// helpersAct lets the hired help do their rounds for one step of time.
func (g *Garden) helpersAct(at time.Time) {
	if len(g.Helpers) == 0 {
		return
	}
	water, weed, gather := g.HelperWorking("waterer", at), g.HelperWorking("weeder", at), g.HelperWorking("gatherer", at)
	for i := range g.Plots {
		p := &g.Plots[i]
		if water && !p.Empty() && !p.Pond && p.Moisture < 0.3 {
			p.Moisture = 0.85
		}
		if weed && p.Weeds > 0.55 {
			p.Weeds = 0
		}
		if gather && !p.Empty() && p.Growth >= 1 && p.Pods >= 2 {
			g.Gather(i, at)
		}
	}
}

// ---- scenery -----------------------------------------------------------------

type boostKind string

const (
	boostNone     boostKind = ""
	boostPollen   boostKind = "pollen"   // bees and moths find more mates
	boostVisitors boostKind = "visitors" // more calls from wildlife
	boostWater    boostKind = "water"    // soil dries more slowly
	boostWeeds    boostKind = "weeds"    // weeds come more slowly
	boostGrowth   boostKind = "growth"   // plants grow a little faster
	boostFair     boostKind = "fair"     // the judges look kindly on your entry
)

type sceneryDef struct {
	ID     string
	Name   string
	Glyph  string
	Blurb  string
	Price  int
	Effect boostKind
	Amount float64 // the change, as a fraction: +0.4 is forty per cent more
}

var sceneryDefs = []sceneryDef{
	{"gnome", "Garden gnome", "☺", "A small, red-hatted fellow who looks pleased with the beds. He does nothing at all, and you will be glad of him.", 40, boostNone, 0},
	{"bench", "Bench", "▭", "Somewhere to sit and watch the bees. A lovely thing to have, and that is the whole of it.", 60, boostNone, 0},
	{"scarecrow", "Scarecrow", "♅", "Weeds seem to think twice about a bed with a scarecrow near it.", 80, boostWeeds, -0.20},
	{"birdbath", "Bird bath", "♁", "Finches, butterflies and the rest call more often when there is water and a place to preen.", 90, boostVisitors, 0.40},
	{"waterbutt", "Water butt", "◍", "Rain caught off the shed roof keeps every bed's soil damp a good deal longer.", 110, boostWater, -0.20},
	{"pergola", "Pergola", "▥", "Roses on a frame, in plain view of the judges. Your fair entries score four per cent higher.", 140, boostFair, 0.04},
	{"beehive", "Beehive", "⬡", "A straw skep at the end of the row. Bees and moths find a mate for a flower forty per cent more often.", 160, boostPollen, 0.40},
	{"sundial", "Sundial", "◔", "Stone, brass and a motto about time. It tells the hour correctly, which is more than can be said for most things in a garden.", 200, boostNone, 0},
	{"fountain", "Fountain", "♨", "Water falling in the middle of the garden does something to the plants. They grow four per cent faster.", 320, boostGrowth, 0.04},
}

func sceneryDefByID(id string) *sceneryDef {
	for i := range sceneryDefs {
		if sceneryDefs[i].ID == id {
			return &sceneryDefs[i]
		}
	}
	return nil
}

// HasScenery reports whether a piece is standing in the garden.
func (g *Garden) HasScenery(id string) bool { return g.Scenery[id] }

// BuyScenery buys a piece. There is only one of each.
func (g *Garden) BuyScenery(id string, now time.Time) error {
	d := sceneryDefByID(id)
	if d == nil {
		return fmt.Errorf("the village has none of those")
	}
	if g.Scenery[id] {
		return fmt.Errorf("you already have a %s", d.Name)
	}
	if !g.spend(d.Price) {
		return fmt.Errorf("a %s costs %s", d.Name, goldLabel(d.Price))
	}
	if g.Scenery == nil {
		g.Scenery = map[string]bool{}
	}
	g.Scenery[id] = true
	g.Log(now, "Set up a %s in the garden for %s.", d.Name, goldLabel(d.Price))
	return nil
}

// boost is the product of what the scenery does to one thing, 1 if nothing.
func (g *Garden) boost(e boostKind) float64 {
	if len(g.Scenery) == 0 {
		return 1
	}
	f := 1.0
	for _, d := range sceneryDefs {
		if d.Effect == e && g.Scenery[d.ID] {
			f *= 1 + d.Amount
		}
	}
	return f
}

// SceneryStrip is the glyphs of what stands in the garden, in the shop's order.
func (g *Garden) SceneryStrip() string {
	out := ""
	for _, d := range sceneryDefs {
		if g.Scenery[d.ID] {
			out += d.Glyph
		}
	}
	return out
}

// ---- the gene lab ------------------------------------------------------------

// labReady reports a bed whose plant is in flower, so the lab can read it.
func (g *Garden) labReady(idx int) bool {
	if idx < 0 || idx >= len(g.Plots) {
		return false
	}
	p := &g.Plots[idx]
	return !p.Empty() && p.Species() != nil && p.Growth >= 1
}

// Forecast shows a likely spread of seedlings from crossing two flowers of one
// species. It is a sample, not a promise: the real seedlings are rolled
// afterwards, from the same rule.
func (g *Garden) Forecast(a, b int, now time.Time) ([]Genome, error) {
	if !g.labReady(a) || !g.labReady(b) || a == b {
		return nil, fmt.Errorf("the lab needs two different flowers in bloom")
	}
	pa, pb := &g.Plots[a], &g.Plots[b]
	if pa.SpeciesID != pb.SpeciesID {
		return nil, fmt.Errorf("the lab forecasts crosses of one species; the gnomes match strangers")
	}
	if !g.spend(labForecastCost) {
		return nil, fmt.Errorf("a forecast costs %s", goldLabel(labForecastCost))
	}
	g.LabUses++
	out := make([]Genome, labForecastKids)
	for i := range out {
		out[i] = breed(g.Seed, int64(0x1AB0+g.LabUses*16+i), pa.Genes(), pb.Genes())
	}
	return out, nil
}

// CloneCost is what the lab charges to copy a plant: a base, plus more the
// further the plant is from anything on the shop shelf.
func (g *Garden) CloneCost(idx int) int {
	if !g.labReady(idx) {
		return 0
	}
	p := &g.Plots[idx]
	gn := p.Genes()
	return labCloneBase + labCloneMult*seedValue(Packet{SpeciesID: p.SpeciesID, A: gn, B: gn, Count: 1, Pure: true})
}

// Clone copies a plant gene for gene into a packet of seed that comes exactly
// true. The plant itself is not touched.
func (g *Garden) Clone(idx int, now time.Time) error {
	if !g.labReady(idx) {
		return fmt.Errorf("the lab can only copy a plant in bloom")
	}
	p := &g.Plots[idx]
	cost := g.CloneCost(idx)
	if !g.spend(cost) {
		return fmt.Errorf("copying %s costs %s", p.DisplayName(), goldLabel(cost))
	}
	gn := p.Genes()
	variety, _ := p.Species().NearestVariety(gn)
	g.AddPacket(Packet{
		SpeciesID: p.SpeciesID, A: gn, B: gn, Count: labCloneSeeds, Pure: true, Gen: p.Gen, Variety: variety,
		From: "the gene lab", Descent: "copied from " + p.DisplayName(), Kind: stepLab, Line: p.Line, Label: p.LineName,
	})
	g.LabUses++
	g.Log(now, "The gene lab copied %s into %d seeds for %s.", p.DisplayName(), labCloneSeeds, goldLabel(cost))
	return nil
}

// PurifyCost is what the lab charges to make a packet breed true.
func (g *Garden) PurifyCost(packet int) int {
	if packet < 0 || packet >= len(g.Shed) || g.Shed[packet].Pure {
		return 0
	}
	pk := g.Shed[packet]
	return labPurifyBase + labPurifyMult*seedValue(pk)*pk.Count
}

// Purify settles a packet of mixed seed on its middle gene, so every seed
// sown from it comes exactly true.
func (g *Garden) Purify(packet int, now time.Time) error {
	if packet < 0 || packet >= len(g.Shed) || g.Shed[packet].Species() == nil {
		return fmt.Errorf("no such packet")
	}
	pk := &g.Shed[packet]
	if pk.Pure {
		return fmt.Errorf("that seed already comes true")
	}
	cost := g.PurifyCost(packet)
	if !g.spend(cost) {
		return fmt.Errorf("purifying that packet costs %s", goldLabel(cost))
	}
	mean := pk.Mean()
	pk.A, pk.B, pk.Pure = mean, mean, true
	pk.Variety, _ = pk.Species().NearestVariety(mean)
	g.LabUses++
	g.Log(now, "The gene lab purified a packet of %s for %s: it now comes true.", pk.Species().Common, goldLabel(cost))
	return nil
}

// ---- commissions -------------------------------------------------------------

type commissionDef struct {
	ID    string
	Name  string
	Maker string
	Blurb string
	Base  int
	Mult  int // times the seed's shop price
}

var commissionDefs = []commissionDef{
	{"potpourri", "Potpourri jar", "Mrs Dunnock", "The petals dried and jarred with a little lavender. Quick work, and the jar goes on your shelf.", 20, 1},
	{"frame", "Pressed-flower frame", "Mr Ashby", "The whole flower pressed flat under glass in an oak frame, labelled in a careful hand.", 30, 2},
	{"wreath", "Dried wreath", "The Misses Thorne", "The plant dried and woven on a willow ring, with a ribbon. It hangs on the door for years.", 45, 3},
}

func commissionDefByID(id string) *commissionDef {
	for i := range commissionDefs {
		if commissionDefs[i].ID == id {
			return &commissionDefs[i]
		}
	}
	return nil
}

// Keepsake is a plant made into something to keep.
type Keepsake struct {
	ID      int       `json:"id"`
	Kind    string    `json:"kind"`
	Species string    `json:"species"`
	Name    string    `json:"name"`
	Hex     string    `json:"hex"`
	Made    time.Time `json:"made"`
}

// CommissionCost is the fee for making a plant into a keepsake.
func (g *Garden) CommissionCost(kind string, idx int) int {
	d := commissionDefByID(kind)
	if d == nil || idx < 0 || idx >= len(g.Plots) || g.Plots[idx].Species() == nil {
		return 0
	}
	return d.Base + d.Mult*g.Plots[idx].Species().seedPrice()
}

// Commission turns the plant in a bed into a keepsake. The plant is used up
// as it is for an order, and the bed keeps its soil.
func (g *Garden) Commission(kind string, idx int, now time.Time) error {
	d := commissionDefByID(kind)
	if d == nil {
		return fmt.Errorf("no one makes that")
	}
	if !g.labReady(idx) {
		return fmt.Errorf("only a plant in bloom can be made into a keepsake")
	}
	p := &g.Plots[idx]
	cost := g.CommissionCost(kind, idx)
	if !g.spend(cost) {
		return fmt.Errorf("%s asks %s", d.Maker, goldLabel(cost))
	}
	g.KeepsakeSeq++
	k := Keepsake{ID: g.KeepsakeSeq, Kind: kind, Species: p.SpeciesID, Name: p.FullName(), Hex: p.Genes().Hex(), Made: now}
	g.Keepsakes = append(g.Keepsakes, k)
	*p = Plot{Moisture: p.Moisture, Weeds: p.Weeds, Pond: p.Pond, PH: p.PH, Richness: p.Richness}
	g.Log(now, "%s made %s into a %s for %s.", d.Maker, k.Name, d.Name, goldLabel(cost))
	return nil
}

// ---- themes and borders -------------------------------------------------------

type themeDef struct {
	ID     string
	Name   string
	Price  int
	Accent string // titles, the selected bed, good news
	Label  string // headings and the card border
	Latin  string // species names
}

var themeDefs = []themeDef{
	{"meadow", "Meadow", 0, "114", "108", "152"},
	{"dusk", "Dusk", 80, "#b48ead", "#81a1c1", "#88c0d0"},
	{"rosewater", "Rosewater", 80, "#e58fa8", "#c9a0b4", "#f0c4d0"},
	{"seaglass", "Sea glass", 80, "#6fd3c4", "#5aa9b8", "#a8e0e0"},
	{"ink", "Ink", 60, "#e0e0e0", "#a0a0a0", "#c8c8c8"},
	{"ember", "Ember", 120, "#f0a050", "#c08060", "#f5cf9f"},
}

func themeDefByID(id string) *themeDef {
	for i := range themeDefs {
		if themeDefs[i].ID == id {
			return &themeDefs[i]
		}
	}
	return &themeDefs[0]
}

// applyTheme repaints the styles that carry the garden's accent colours.
func applyTheme(id string) {
	t := themeDefByID(id)
	accent, label, latin := lipgloss.Color(t.Accent), lipgloss.Color(t.Label), lipgloss.Color(t.Latin)
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(accent)
	okStyle = lipgloss.NewStyle().Foreground(accent)
	labelStyle = lipgloss.NewStyle().Foreground(label)
	latinStyle = lipgloss.NewStyle().Italic(true).Foreground(latin)
	selBorder = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accent)
	cardBorder = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(label).Padding(0, 1)
}

type borderDef struct {
	ID    string
	Name  string
	Price int
	Shape lipgloss.Border
}

func uniformBorder(top, side, corner string) lipgloss.Border {
	return lipgloss.Border{Top: top, Bottom: top, Left: side, Right: side,
		TopLeft: corner, TopRight: corner, BottomLeft: corner, BottomRight: corner}
}

var borderDefs = []borderDef{
	{"rounded", "Rounded", 0, lipgloss.RoundedBorder()},
	{"classic", "Classic", 30, lipgloss.NormalBorder()},
	{"ascii", "Plain ASCII", 30, uniformBorder("-", "|", "+")},
	{"double", "Double line", 40, lipgloss.DoubleBorder()},
	{"thick", "Thick", 40, lipgloss.ThickBorder()},
	{"stitched", "Stitched", 50, lipgloss.Border{Top: "╌", Bottom: "╌", Left: "┆", Right: "┆", TopLeft: "┌", TopRight: "┐", BottomLeft: "└", BottomRight: "┘"}},
	{"dotted", "Dotted", 50, uniformBorder("·", ":", "·")},
	{"hedge", "Clipped hedge", 90, uniformBorder("▒", "▒", "▒")},
}

func borderDefByID(id string) *borderDef {
	for i := range borderDefs {
		if borderDefs[i].ID == id {
			return &borderDefs[i]
		}
	}
	return &borderDefs[0]
}

// bedShape is the border the garden's beds are drawn with.
func (g *Garden) bedShape() lipgloss.Border { return borderDefByID(g.Border).Shape }

// OwnsTheme reports whether a theme has been bought (the first is free).
func (g *Garden) OwnsTheme(id string) bool {
	d := themeDefByID(id)
	return d.Price == 0 || g.Themes[d.ID]
}

// OwnsBorder reports whether a border has been bought (the first is free).
func (g *Garden) OwnsBorder(id string) bool {
	d := borderDefByID(id)
	return d.Price == 0 || g.Borders[d.ID]
}

// CurrentTheme and CurrentBorder are the ids in use, the default if unset.
func (g *Garden) CurrentTheme() string  { return themeDefByID(g.Theme).ID }
func (g *Garden) CurrentBorder() string { return borderDefByID(g.Border).ID }

// UseTheme buys a theme if it is not yet owned, and puts it on. It reports
// whether gold changed hands.
func (g *Garden) UseTheme(id string, now time.Time) (bought bool, err error) {
	d := themeDefByID(id)
	if d.ID != id {
		return false, fmt.Errorf("the village has no such colours")
	}
	if !g.OwnsTheme(id) {
		if !g.spend(d.Price) {
			return false, fmt.Errorf("the %s colours cost %s", d.Name, goldLabel(d.Price))
		}
		if g.Themes == nil {
			g.Themes = map[string]bool{}
		}
		g.Themes[id], bought = true, true
		g.Log(now, "Bought the %s colours for %s.", d.Name, goldLabel(d.Price))
	}
	g.Theme = id
	return bought, nil
}

// UseBorder buys a border if it is not yet owned, and puts it on every bed.
func (g *Garden) UseBorder(id string, now time.Time) (bought bool, err error) {
	d := borderDefByID(id)
	if d.ID != id {
		return false, fmt.Errorf("the village has no such border")
	}
	if !g.OwnsBorder(id) {
		if !g.spend(d.Price) {
			return false, fmt.Errorf("the %s border costs %s", d.Name, goldLabel(d.Price))
		}
		if g.Borders == nil {
			g.Borders = map[string]bool{}
		}
		g.Borders[id], bought = true, true
		g.Log(now, "Bought the %s bed border for %s.", d.Name, goldLabel(d.Price))
	}
	g.Border = id
	return bought, nil
}

// wagesLabel is the weekly wage bill, in words.
func (g *Garden) wagesLabel() string {
	total := 0
	for _, h := range g.Helpers {
		if d := helperDefByID(h.ID); d != nil {
			total += d.Wage
		}
	}
	return goldLabel(total)
}
