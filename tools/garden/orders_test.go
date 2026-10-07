package main

import (
	"testing"
	"time"
)

func ordersGarden() (*Garden, time.Time) {
	now := time.Date(2026, 6, 15, 9, 0, 0, 0, time.UTC)
	g := newTestGarden(now)
	g.Advance(now.Add(time.Second))
	return g, now
}

func TestThreeOrdersAreHandedOutEachDay(t *testing.T) {
	g, now := ordersGarden()
	if len(g.Orders) != ordersPerDay {
		t.Fatalf("%d orders on the first day", len(g.Orders))
	}
	for i, o := range g.Orders {
		if o.Tier != i {
			t.Errorf("order %d has tier %d", i, o.Tier)
		}
		if o.SpeciesRef() == nil || o.SpeciesRef().Kind == KindAquatic || len(o.Wants) == 0 || o.Reward <= 0 {
			t.Errorf("order %d is malformed: %+v", i, o)
		}
	}
	if len(g.Orders[2].Wants) != 2 || g.Orders[2].Bonus == "" {
		t.Error("the hard order asks for two things and throws in seed")
	}
	// Nothing new until tomorrow.
	g.Advance(now.Add(time.Hour))
	if len(g.Orders) != ordersPerDay {
		t.Errorf("%d orders after an hour", len(g.Orders))
	}
	g.Advance(now.Add(25 * time.Hour))
	if len(g.Orders) != 2*ordersPerDay {
		t.Errorf("%d orders the next day, want %d", len(g.Orders), 2*ordersPerDay)
	}
}

func TestOrdersLapseQuietly(t *testing.T) {
	g, now := ordersGarden()
	gold := g.Gold
	g.Advance(now.Add((orderLifeDays*24 + 2) * time.Hour))
	for _, o := range g.Orders {
		if !o.Expires.After(now.Add(orderLifeDays * 24 * time.Hour)) {
			t.Errorf("an old order is still up: %+v", o.Given)
		}
	}
	if g.Gold != gold {
		t.Error("orders lapsing cost gold")
	}
	if len(g.Orders) > maxOrders {
		t.Errorf("%d orders on the board", len(g.Orders))
	}
}

func TestTheSameGardenGetsTheSameOrders(t *testing.T) {
	a, _ := ordersGarden()
	b, _ := ordersGarden()
	for i := range a.Orders {
		if a.Orders[i].Species != b.Orders[i].Species || len(a.Orders[i].Wants) != len(b.Orders[i].Wants) || a.Orders[i].Reward != b.Orders[i].Reward {
			t.Fatalf("order %d differs between identical gardens", i)
		}
		for j := range a.Orders[i].Wants {
			if a.Orders[i].Wants[j] != b.Orders[i].Wants[j] {
				t.Fatalf("order %d condition %d differs", i, j)
			}
		}
	}
}

// Easy orders can be filled by something the shop sells; the others can be
// reached by breeding, because they stay inside the range of genes.
func TestOrdersCanBeFilled(t *testing.T) {
	now := time.Date(2026, 6, 15, 9, 0, 0, 0, time.UTC)
	for seed := int64(1); seed <= 40; seed++ {
		g := newTestGarden(now)
		g.Seed = seed
		for day := 0; day < 5; day++ {
			g.OrdersDay = 0
			g.Orders = nil
			g.refreshOrders(now.Add(time.Duration(day) * 24 * time.Hour))
			for _, o := range g.Orders {
				sp := o.SpeciesRef()
				if o.Tier == 0 {
					ok := false
					for i := range sp.Varieties() {
						gn := sp.VarietyGenome(i)
						all := true
						for _, c := range o.Wants {
							all = all && c.Met(gn)
						}
						ok = ok || all
					}
					if !ok {
						t.Fatalf("seed %d: an easy %s order cannot be met by any shop form: %+v", seed, sp.ID, o.Wants)
					}
				}
				for _, c := range o.Wants {
					if c.Trait != TraitColour && ((c.AtMost && c.Value < 5) || (!c.AtMost && c.Value > 95)) {
						t.Errorf("seed %d: %s asks for the impossible: %+v", seed, sp.ID, c)
					}
				}
			}
		}
	}
}

func TestDeliveringConsumesThePlant(t *testing.T) {
	g, now := ordersGarden()
	g.Gold = 100
	o := g.Orders[0]
	sp := o.SpeciesRef()
	// Grow the first form that satisfies it.
	var gn Genome
	for i := range sp.Varieties() {
		cand := sp.VarietyGenome(i)
		ok := true
		for _, c := range o.Wants {
			ok = ok && c.Met(cand)
		}
		if ok {
			gn = cand
			break
		}
	}
	if gn.Blank() {
		t.Fatal("no form fits the easy order")
	}
	g.sow(3, sp, gn, 0, now)
	g.Plots[3].Growth = 1
	g.Plots[3].Richness = 0.42

	if _, err := g.Deliver(o.ID, 4, now); err == nil {
		t.Error("delivered an empty bed")
	}
	g.Plots[3].Growth = 0.5
	if _, err := g.Deliver(o.ID, 3, now); err == nil {
		t.Error("delivered a plant not yet in flower")
	}
	g.Plots[3].Growth = 1

	before := g.Gold
	got, err := g.Deliver(o.ID, 3, now)
	if err != nil || got != o.Reward || g.Gold != before+o.Reward {
		t.Fatalf("delivery gave %d, %v; gold %d→%d", got, err, before, g.Gold)
	}
	if !g.Plots[3].Empty() {
		t.Error("the plant is still in the bed")
	}
	if g.Plots[3].Richness != 0.42 {
		t.Errorf("richness is %.2f: the plant was composted, but an order consumes it", g.Plots[3].Richness)
	}
	if _, o2 := g.OrderByID(o.ID); o2 != nil || g.OrdersDone != 1 {
		t.Error("the order was not taken off the board")
	}
	if _, err := g.Deliver(o.ID, 3, now); err == nil {
		t.Error("filled the same order twice")
	}
}

func TestWrongPlantsAreRefused(t *testing.T) {
	g, now := ordersGarden()
	g.Gold = 100
	o := g.Orders[0]
	other := "tomato"
	if o.Species == other {
		other = "basil"
	}
	if err := g.Plant(0, SpeciesByID(other), 0, now); err != nil {
		t.Fatal(err)
	}
	g.Plots[0].Growth = 1
	if _, err := g.Deliver(o.ID, 0, now); err == nil {
		t.Error("the wrong species was accepted")
	}
	if g.Plots[0].Empty() {
		t.Error("a refused delivery still removed the plant")
	}
}

func TestOrderKeyFlow(t *testing.T) {
	g, now := ordersGarden()
	o := g.Orders[0]
	sp := o.SpeciesRef()
	for i := range sp.Varieties() {
		cand := sp.VarietyGenome(i)
		ok := true
		for _, c := range o.Wants {
			ok = ok && c.Met(cand)
		}
		if ok {
			g.sow(0, sp, cand, 0, now)
			g.Plots[0].Growth = 1
			break
		}
	}
	m := newModel(g, t.TempDir()+"/g.json", now)
	m.width, m.height = 110, 36
	var cur = keyPress(m, "tab") // shed
	cur = keyPress(cur, "tab")   // orders
	if cur.(model).screen != screenOrders {
		t.Fatal("tab does not reach the order board")
	}
	if v := cur.View(); len(v) == 0 {
		t.Fatal("blank order board")
	}
	before := g.Gold
	cur = keyPress(cur, "enter")
	if g.Gold <= before || !g.Plots[0].Empty() {
		t.Errorf("enter did not deliver: gold %d→%d, bed %q", before, g.Gold, g.Plots[0].SpeciesID)
	}
}
