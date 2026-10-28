package main

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// Orders. Once a week the village posts three requests: a foxglove at least
// so tall, a zinnia near a certain red. Deliver a flowering plant that fits and
// it is taken away (not composted: it is simply consumed by the order) and you
// are paid. An order nobody fills lapses when the next week's board goes up;
// there is no penalty and no timer to beat.
//
// The three are of rising difficulty. The first is easy (a named form from the
// shop already fits), the second wants something past the shop shelf, and the
// third wants two things at once, one of them well past the shelf. They are
// always for three different species. Pay is a flat sum for the tier, a
// multiple of the seed's price, and a bonus for how far past the shelf the
// order reaches, so a harder request is worth far more than an easy one.

const (
	ordersPerWeek = 3
	maxOrders     = 9
	colourMatch   = 0.22 // how near a colour must be to the one asked for

	orderEasyBase = 15
	orderEasyMult = 3
	orderMidBase  = 40
	orderMidMult  = 8
	orderHardBase = 90
	orderHardMult = 16
	orderPointPay = 2 // gold per point of difficulty (a gene point past the shelf, or a point of colour distance)

	// How far past the best the shop sells a stretch order asks, as a range
	// the generator picks from; a hard order asks for more.
	stretchMin, stretchMax = 10, 20
	hardMin, hardMax       = 18, 30
)

// Condition is one thing an order asks of a plant.
type Condition struct {
	Trait  Trait  `json:"trait"`
	AtMost bool   `json:"at_most,omitempty"` // numeric traits: at most, rather than at least
	Value  uint8  `json:"value,omitempty"`
	Colour string `json:"colour,omitempty"` // for colour: the shade asked for, as #rrggbb
}

// Met reports whether a genome satisfies the condition.
func (c Condition) Met(gn Genome) bool {
	if c.Trait == TraitColour {
		target, ok := parseColor(c.Colour)
		return ok && colourDistance(gn.RGB(), target) <= colourMatch
	}
	v := gn.value(c.Trait)
	if c.AtMost {
		return v <= c.Value
	}
	return v >= c.Value
}

// Text says what the condition asks, in words.
func (c Condition) Text(sp *Species) string {
	if c.Trait == TraitColour {
		target, _ := parseColor(c.Colour)
		return "a colour near " + colourName(target)
	}
	probe := Genome{Height: c.Value}
	switch c.Trait {
	case TraitHeight:
		if c.AtMost {
			return fmt.Sprintf("no taller than %s", sp.HeightText(probe))
		}
		return fmt.Sprintf("at least %s tall", sp.HeightText(probe))
	case TraitShape:
		if c.AtMost {
			return fmt.Sprintf("slim and upright (shape %d or less)", c.Value)
		}
		return fmt.Sprintf("full and bushy (shape %d or more)", c.Value)
	case TraitSpeed:
		if c.AtMost {
			return fmt.Sprintf("slow-growing (speed %d or less)", c.Value)
		}
		return fmt.Sprintf("quick-growing (speed %d or more)", c.Value)
	}
	if c.AtMost {
		return fmt.Sprintf("sparing with seed (yield %d or less)", c.Value)
	}
	return fmt.Sprintf("a generous cropper (yield %d or more)", c.Value)
}

// Order is one request on the board.
type Order struct {
	ID      int         `json:"id"`
	Species string      `json:"species"`
	Tier    int         `json:"tier"` // 0 easy, 1 middling, 2 hard
	Wants   []Condition `json:"wants"`
	Reward  int         `json:"reward"`
	Client  string      `json:"client,omitempty"` // who is asking
	Bonus   string      `json:"bonus,omitempty"`  // a species of seed thrown in, for the hard ones
	Given   time.Time   `json:"given"`
	Expires time.Time   `json:"expires"`
}

func (o Order) SpeciesRef() *Species { return SpeciesByID(o.Species) }

// TierName is how hard the order is, in a word.
func (o Order) TierName() string { return [...]string{"easy", "a stretch", "hard"}[o.Tier] }

// Fits reports whether a plant would fill the order.
func (o Order) Fits(p *Plot) bool {
	if p.Empty() || p.SpeciesID != o.Species || p.Growth < 1 {
		return false
	}
	gn := p.Genes()
	for _, c := range o.Wants {
		if !c.Met(gn) {
			return false
		}
	}
	return true
}

func dayKey(t time.Time) int {
	y, m, d := t.Date()
	return y*10000 + int(m)*100 + d
}

// orderPool is the species an order may ask for: anything on the shop shelf
// that is not a water plant.
func (g *Garden) orderPool() []*Species {
	var out []*Species
	for _, sp := range AllSpecies() {
		if sp.Kind != KindAquatic && g.Unlocked(sp) {
			out = append(out, sp)
		}
	}
	return out
}

// traitSpan is the lowest and highest value of a trait among a species' named forms.
func traitSpan(sp *Species, t Trait) (lo, hi uint8) {
	lo, hi = 255, 0
	for i := range sp.Varieties() {
		v := sp.VarietyGenome(i).value(t)
		if v < lo {
			lo = v
		}
		if v > hi {
			hi = v
		}
	}
	return lo, hi
}

var numericTraits = []Trait{TraitHeight, TraitShape, TraitSpeed, TraitYield}

// orderClients are the villagers who post orders.
var orderClients = []string{
	"the vicar", "the baker", "old Mrs Pennywhistle", "the schoolmaster", "the ferryman",
	"the innkeeper", "the miller's daughter", "the parish council", "a travelling tinker",
	"the blacksmith", "the squire's cook", "the midwife", "the lamplighter", "the beekeeper",
	"a wedding party", "the harvest committee", "the curate", "the postmistress",
}

// difficulty is how many points past the shop shelf a condition reaches: gene
// points beyond the best or worst named form, or, for a colour, a hundred times
// its distance from the nearest named form's.
func (c Condition) difficulty(sp *Species) int {
	if c.Trait == TraitColour {
		target, ok := parseColor(c.Colour)
		if !ok {
			return 0
		}
		best := math.MaxFloat64
		for i := range sp.Varieties() {
			if d := colourDistance(sp.VarietyGenome(i).RGB(), target); d < best {
				best = d
			}
		}
		return int(math.Round(math.Min(best, 0.6) * 100))
	}
	lo, hi := traitSpan(sp, c.Trait)
	if c.AtMost {
		return max(0, int(lo)-int(c.Value))
	}
	return max(0, int(c.Value)-int(hi))
}

// makeOrder builds order number slot (0 to 2) for a week, for a species not in
// taken. It is a pure function of the garden's seed, the week and what the
// shop stocks.
func (g *Garden) makeOrder(week, slot int, pool []*Species, taken map[string]bool, now time.Time) Order {
	pick := func(salt int64) float64 { return hashUnit(g.Seed, int64(week), int64(slot)*64+salt+0x0DE5) }
	start := int(pick(1) * float64(len(pool)))
	sp := pool[start%len(pool)]
	for i := 0; i < len(pool) && taken[sp.ID]; i++ {
		sp = pool[(start+i)%len(pool)]
	}
	taken[sp.ID] = true
	vi := int(pick(2)*float64(len(sp.Varieties()))) % len(sp.Varieties())
	form := sp.VarietyGenome(vi)
	t := numericTraits[int(pick(3)*float64(len(numericTraits)))%len(numericTraits)]
	// A second, different gene for orders that ask for more than one.
	t2 := numericTraits[(int(pick(8)*float64(len(numericTraits)-1))+1+indexOfTrait(t))%len(numericTraits)]

	beyond := func(trait Trait, by uint8, salt int64) Condition {
		lo, hi := traitSpan(sp, trait)
		if pick(salt) < 0.5 {
			return Condition{Trait: trait, Value: uint8(math.Min(92, float64(hi)+float64(by)))}
		}
		return Condition{Trait: trait, AtMost: true, Value: uint8(math.Max(8, float64(lo)-float64(by)))}
	}
	span := func(lo, hi int, salt int64) uint8 { return uint8(lo + int(pick(salt)*float64(hi-lo+1))%(hi-lo+1)) }
	colourOf := func(gn Genome) Condition {
		return Condition{Trait: TraitColour, Colour: gn.Hex()}
	}

	o := Order{Species: sp.ID, Tier: slot, Given: now, Expires: weekEnd(now),
		Client: orderClients[int(pick(9)*float64(len(orderClients)))%len(orderClients)]}
	base := sp.seedPrice()
	var tierBase, tierMult int
	switch slot {
	case 0:
		// Something a named form already is.
		if pick(4) < 0.4 {
			o.Wants = []Condition{colourOf(form)}
		} else if form.value(t) >= 50 {
			o.Wants = []Condition{{Trait: t, Value: form.value(t) - 4}}
		} else {
			o.Wants = []Condition{{Trait: t, AtMost: true, Value: form.value(t) + 4}}
		}
		tierBase, tierMult = orderEasyBase, orderEasyMult
	case 1:
		// Past the shelf in one gene, and often a second the shelf can manage.
		o.Wants = []Condition{beyond(t, span(stretchMin, stretchMax, 5), 10)}
		if pick(11) < 0.4 {
			if form.value(t2) >= 50 {
				o.Wants = append(o.Wants, Condition{Trait: t2, Value: form.value(t2) - 4})
			} else {
				o.Wants = append(o.Wants, Condition{Trait: t2, AtMost: true, Value: form.value(t2) + 4})
			}
		}
		tierBase, tierMult = orderMidBase, orderMidMult
	default:
		// Well past the shelf in one gene, plus either a colour the shelf does
		// not have or a second gene a little past it.
		o.Wants = []Condition{beyond(t, span(hardMin, hardMax, 5), 10)}
		if pick(11) < 0.6 {
			odd := form
			odd.Hue = uint16(math.Mod(float64(form.Hue)+90+120*pick(6), 360))
			odd.Sat = uint8(math.Max(float64(form.Sat), 55))
			o.Wants = append(o.Wants, colourOf(odd))
		} else {
			o.Wants = append(o.Wants, beyond(t2, span(6, 12, 12), 13))
		}
		tierBase, tierMult = orderHardBase, orderHardMult
		others := g.orderPool()
		o.Bonus = others[int(pick(7)*float64(len(others)))%len(others)].ID
	}
	points := 0
	for _, c := range o.Wants {
		points += c.difficulty(sp)
	}
	o.Reward = tierMult*base + tierBase + orderPointPay*points
	return o
}

func indexOfTrait(t Trait) int {
	for i, x := range numericTraits {
		if x == t {
			return i
		}
	}
	return 0
}

// weekEnd is the start of next Monday, when this week's board comes down.
func weekEnd(t time.Time) time.Time {
	d := int(t.Weekday()+6) % 7 // Monday is 0
	y, m, day := t.Date()
	return time.Date(y, m, day+7-d, 0, 0, 0, 0, t.Location())
}

// refreshOrders posts this week's orders if they have not been posted, and
// takes down the ones that have lapsed.
func (g *Garden) refreshOrders(now time.Time) {
	live := g.Orders[:0]
	for _, o := range g.Orders {
		if now.Before(o.Expires) {
			live = append(live, o)
		}
	}
	g.Orders = live

	week := weekKey(now)
	if g.OrdersWeek == week {
		return
	}
	g.OrdersWeek = week
	pool := g.orderPool()
	if len(pool) == 0 {
		return
	}
	taken := map[string]bool{}
	for slot := 0; slot < ordersPerWeek && len(g.Orders) < maxOrders; slot++ {
		o := g.makeOrder(week, slot, pool, taken, now)
		g.OrderSeq++
		o.ID = g.OrderSeq
		g.Orders = append(g.Orders, o)
	}
}

// OrderByID finds an order on the board.
func (g *Garden) OrderByID(id int) (int, *Order) {
	for i := range g.Orders {
		if g.Orders[i].ID == id {
			return i, &g.Orders[i]
		}
	}
	return -1, nil
}

// OrdersFor lists the orders a plant would fill, best-paying first.
func (g *Garden) OrdersFor(p *Plot) []Order {
	var out []Order
	for _, o := range g.Orders {
		if o.Fits(p) {
			out = append(out, o)
		}
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].Reward > out[b].Reward })
	return out
}

// QualifyingBeds lists the beds whose plants would fill an order.
func (g *Garden) QualifyingBeds(o Order) []int {
	var out []int
	for i := range g.Plots {
		if o.Fits(&g.Plots[i]) {
			out = append(out, i)
		}
	}
	return out
}

// Deliver fills an order with the plant in a bed. The plant is removed and
// used up by the order; the bed is left bare, with its soil as it was.
func (g *Garden) Deliver(orderID, bed int, now time.Time) (int, error) {
	oi, o := g.OrderByID(orderID)
	if o == nil {
		return 0, fmt.Errorf("that order has lapsed")
	}
	if bed < 0 || bed >= len(g.Plots) {
		return 0, fmt.Errorf("no such bed")
	}
	p := &g.Plots[bed]
	sp := o.SpeciesRef()
	switch {
	case p.Empty():
		return 0, fmt.Errorf("bed %d is empty", bed+1)
	case p.SpeciesID != o.Species:
		return 0, fmt.Errorf("the order wants a %s", sp.Common)
	case p.Growth < 1:
		return 0, fmt.Errorf("%s is not in flower yet", p.DisplayName())
	case !o.Fits(p):
		return 0, fmt.Errorf("%s is not what was asked for", p.DisplayName())
	}
	name := p.DisplayName()
	reward := o.Reward
	bonus := o.Bonus
	*p = Plot{Moisture: p.Moisture, Weeds: p.Weeds, Pond: p.Pond, PH: p.PH, Richness: p.Richness}
	g.earn(reward)
	g.OrdersDone++
	g.Orders = append(g.Orders[:oi], g.Orders[oi+1:]...)
	msg := fmt.Sprintf("Delivered %s to an order for %s: %s.", name, sp.Common, goldLabel(reward))
	if bsp := SpeciesByID(bonus); bsp != nil {
		g.AddPacket(Packet{SpeciesID: bsp.ID, A: bsp.VarietyGenome(0), B: bsp.VarietyGenome(0), Count: 2, Pure: true, From: "an order"})
		msg += fmt.Sprintf(" Two %s seeds came with it.", bsp.Common)
	}
	g.Log(now, "%s", msg)
	return reward, nil
}
