package main

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// Lineage. Every plant, packet and cultivar carries the record of how its line
// came to be: where the first generation came from, and then one Step for each
// generation after it, saying what the parents were, what the seedling came out
// as, and which genes mutated or drifted on the way. The genetics screen reads
// the record back, so the history is the real one and not a guess: each step is
// made at the moment the seed is sown, from the same roll that decided the
// seedling's genes.
//
// A save from before lineages were kept has plants without a record. They are
// shown with an honest "record begins here" marker and carry on recording.

// geneCount is how many genes a genome has: hue, saturation, lightness, height,
// shape, speed and yield, in the order geneNames lists them.
const geneCount = len(geneNames)

// lineageMax bounds how many steps one record keeps. When a line runs longer,
// the first generation stays (it is where the line came from) and the oldest of
// the rest are dropped.
const lineageMax = 60

// StepKind says how a generation came about.
type StepKind string

const (
	stepShop      StepKind = "shop"      // bought from the shop as a named form
	stepCultivar  StepKind = "cultivar"  // bought from the shop as a cultivar
	stepWild      StepKind = "wild"      // the gnomes' wild seed
	stepGift      StepKind = "gift"      // a friend's code
	stepSelf      StepKind = "self"      // seed from a plant pollinated by itself
	stepCross     StepKind = "cross"     // seed from a plant pollinated by another
	stepVolunteer StepKind = "volunteer" // sowed itself in a neighbouring bed
	stepGnome     StepKind = "gnome"     // changed in place by the gnomes
	stepMatch     StepKind = "match"     // a gnome-matched cross of two species
	stepLab       StepKind = "lab"       // copied exactly at the gene lab
)

// Origin reports whether the kind starts a line, rather than continuing one.
func (k StepKind) Origin() bool {
	switch k {
	case stepShop, stepCultivar, stepWild, stepGift, stepLab:
		return true
	}
	return false
}

// Shift is one gene that landed away from where its parents pointed: a
// mutation, or drift big enough to notice. From is where the gene would have
// been without it (the middle of the parents, or the gene before a gnome's
// spell) and To is where it ended up.
type Shift struct {
	Gene     int  `json:"g"`
	From     int  `json:"f"`
	To       int  `json:"t"`
	Mutation bool `json:"m,omitempty"` // a rare jump, rather than ordinary drift
}

// Delta is how far the gene moved, round the colour wheel for hue.
func (s Shift) Delta() int {
	if s.Gene == 0 {
		return hueDelta(s.From, s.To)
	}
	return s.To - s.From
}

// hueDelta is the signed shortest way round the colour wheel from a to b.
func hueDelta(a, b int) int {
	d := ((b-a)%360 + 540) % 360
	return d - 180
}

// Step is one generation of a line.
type Step struct {
	Gen     int       `json:"gen"`
	Kind    StepKind  `json:"kind"`
	Note    string    `json:"note,omitempty"`
	Genome  Genome    `json:"genome"` // what the generation came out as
	Mother  *Genome   `json:"mother,omitempty"`
	Father  *Genome   `json:"father,omitempty"`
	Shifts  []Shift   `json:"shifts,omitempty"`
	Species string    `json:"species,omitempty"` // set when a step changed the species
	At      time.Time `json:"at,omitempty"`
}

// Mutations counts the mutations in a step.
func (s Step) Mutations() int {
	n := 0
	for _, sh := range s.Shifts {
		if sh.Mutation {
			n++
		}
	}
	return n
}

// Centre is where the parents pointed: the noise-free blend of mother and
// father. For a generation with no parents it is the genome itself.
func (s Step) Centre() Genome {
	if s.Mother == nil || s.Father == nil {
		return s.Genome
	}
	return meanGenome(*s.Mother, *s.Father)
}

// Spread is how widely the seedlings of these parents scatter on a gene: the
// standard deviation of the roll, the same figure the breeding itself uses.
func (s Step) Spread(gene int) float64 {
	if s.Mother == nil || s.Father == nil {
		return 0
	}
	return spreadFor(*s.Mother, *s.Father, gene)
}

// spreadFor is the standard deviation breeding gives a gene, from the parents.
func spreadFor(a, b Genome, gene int) float64 {
	if gene == 0 {
		gap := hueDistance(float64(a.Hue), float64(b.Hue))
		wa, wb := float64(a.Sat)+1, float64(b.Sat)+1
		weight := math.Max(0.25, (wa+wb)/200)
		return (noiseBase*2 + noiseFromGap*gap) * weight
	}
	return noiseBase + noiseFromGap*math.Abs(float64(a.geneAt(gene))-float64(b.geneAt(gene)))
}

// driftNote is how far, in points, a gene must land from its parents' middle
// before the record notes it as drift (colour wheel degrees for hue).
func driftNote(gene int) int {
	if gene == 0 {
		return 15
	}
	return 8
}

// shiftsFor lists the genes of a seedling worth noting: those that mutated,
// and those that drifted a noticeable way from the parents' middle.
func shiftsFor(mother, father, child Genome, mutated [geneCount]bool) []Shift {
	centre := meanGenome(mother, father)
	var out []Shift
	for i := 0; i < geneCount; i++ {
		from, to := centre.geneAt(i), child.geneAt(i)
		d := to - from
		if i == 0 {
			d = hueDelta(from, to)
			// A colourless parent pair says nothing about hue.
			if centre.Sat < 12 && child.Sat < 12 && !mutated[i] {
				continue
			}
		}
		if mutated[i] || iabs(d) >= driftNote(i) {
			out = append(out, Shift{Gene: i, From: from, To: to, Mutation: mutated[i]})
		}
	}
	return out
}

// diffShifts lists every gene that changed between two genomes, for a change
// made in place such as a gnome's spell.
func diffShifts(before, after Genome, mutation bool) []Shift {
	var out []Shift
	for i := 0; i < geneCount; i++ {
		if before.geneAt(i) != after.geneAt(i) {
			out = append(out, Shift{Gene: i, From: before.geneAt(i), To: after.geneAt(i), Mutation: mutation})
		}
	}
	return out
}

func iabs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// extendLineage returns a record with one more step. It never shares memory
// with the record it was given, so a plant, its packets and the cultivar they
// found each own their copy. A record that has grown past lineageMax drops the
// oldest of its later steps but keeps its first.
func extendLineage(h []Step, s Step) []Step {
	out := make([]Step, 0, len(h)+1)
	out = append(out, h...)
	out = append(out, s)
	if len(out) > lineageMax {
		trimmed := append([]Step{out[0]}, out[len(out)-(lineageMax-1):]...)
		out = trimmed
	}
	return out
}

// copyLineage is an independent copy of a record.
func copyLineage(h []Step) []Step {
	if len(h) == 0 {
		return nil
	}
	return append([]Step(nil), h...)
}

// ---- reading a record back ------------------------------------------------

// geneChange is one noted event along a line, for the summary.
type geneChange struct {
	Gen      int
	Gene     int
	From, To int
	Mutation bool
}

// LineageReport is a record read as a whole: how deep it is, where it began,
// how far it has wandered from the first generation, and every mutation and
// drift on the way.
type LineageReport struct {
	Steps     []Step
	Depth     int  // generations after the first in the record
	Known     bool // false when the record is empty (a plant from before lineages)
	Origin    Step
	Mutations []geneChange
	Drifts    []geneChange
	Wander    float64 // distance from the first generation's genes to the last's
	Peak      float64 // the furthest any generation got from the first
	Sources   []string
}

// reportLineage reads a record. now is the genome of the plant or line the
// record belongs to, which ends the trail even when the last step is older.
func reportLineage(steps []Step) LineageReport {
	r := LineageReport{Steps: steps, Known: len(steps) > 0}
	if len(steps) == 0 {
		return r
	}
	r.Origin = steps[0]
	r.Depth = max(0, steps[len(steps)-1].Gen-steps[0].Gen)
	for _, s := range steps {
		for _, sh := range s.Shifts {
			ev := geneChange{Gen: s.Gen, Gene: sh.Gene, From: sh.From, To: sh.To, Mutation: sh.Mutation}
			if sh.Mutation {
				r.Mutations = append(r.Mutations, ev)
			} else {
				r.Drifts = append(r.Drifts, ev)
			}
		}
		if d := s.Genome.Distance(steps[0].Genome); d > r.Peak {
			r.Peak = d
		}
		if s.Kind.Origin() && s.Note != "" {
			r.Sources = append(r.Sources, s.Note)
		}
	}
	r.Wander = steps[len(steps)-1].Genome.Distance(steps[0].Genome)
	return r
}

// geneTrace is one gene's value at each step.
func geneTrace(steps []Step, gene int) []int {
	out := make([]int, len(steps))
	for i, s := range steps {
		out[i] = s.Genome.geneAt(gene)
	}
	return out
}

// sparkline draws values from 0 to 100 as a row of blocks.
func sparkline(vals []int) string {
	const blocks = "▁▂▃▄▅▆▇█"
	rs := []rune(blocks)
	var b strings.Builder
	for _, v := range vals {
		i := v * len(rs) / 101
		i = max(0, min(len(rs)-1, i))
		b.WriteRune(rs[i])
	}
	return b.String()
}

// shiftWords says what a shift did: "height 48 → 74".
func shiftWords(s Shift) string {
	name := geneNames[s.Gene]
	if s.Gene == 0 {
		return fmt.Sprintf("%s %d° → %d°", name, s.From, s.To)
	}
	return fmt.Sprintf("%s %d → %d", name, s.From, s.To)
}

// kindWords is how a step reads on the timeline.
func kindWords(k StepKind) string {
	switch k {
	case stepShop:
		return "bought from the shop"
	case stepCultivar:
		return "bought as a cultivar"
	case stepWild:
		return "brewed wild by the gnomes"
	case stepGift:
		return "a gift"
	case stepSelf:
		return "selfed"
	case stepCross:
		return "crossed"
	case stepVolunteer:
		return "sowed itself"
	case stepGnome:
		return "changed by the gnomes"
	case stepMatch:
		return "matched by the gnomes"
	case stepLab:
		return "copied at the gene lab"
	}
	return "unrecorded"
}

// howItWorks explains a step in terms of the rules, with the actual numbers:
// the breeding rule is "the average of the parents, plus noise that widens
// with how unlike they are, plus a rare mutation".
func howItWorks(s Step) string {
	switch s.Kind {
	case stepLab:
		return "The gene lab copies a plant gene for gene, so seed of this kind comes exactly true to the plant it was taken from."
	case stepShop, stepCultivar, stepWild:
		return "Seed of this kind comes exactly true, so the plant is the form itself."
	case stepGift:
		return "A gift comes exactly true to the line it was made from."
	case stepGnome:
		return "A gnome's spell moves the genes of a plant that is already growing; it leaves its line."
	}
	if s.Mother == nil || s.Father == nil {
		return ""
	}
	gap := s.Mother.Distance(*s.Father)
	var how string
	switch {
	case gap <= stableGap:
		how = fmt.Sprintf("The parents are alike (%.0f%% apart), so the seedlings scatter only a little round their middle.", gap*100)
	case gap <= 0.20:
		how = fmt.Sprintf("The parents differ by %.0f%%: the seedlings scatter moderately round the blend of the two.", gap*100)
	default:
		how = fmt.Sprintf("The parents are unlike (%.0f%% apart): the seedlings scatter widely round the blend of the two.", gap*100)
	}
	return how
}
