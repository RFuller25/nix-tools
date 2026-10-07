package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
	"unicode"
)

// Sharing. A cultivar you have bred can be given to a friend as a short text
// code: paste it into a message, and they paste it into their shed. A code
// carries the species, the seven genes, the name you gave the line and who
// made it. Redeeming one gives three seeds of the line and files it in the
// friend's almanac under their cultivars.
//
// There is no server, so a code can be redeemed once in any one garden, but
// not once in the world: the garden remembers which codes it has used, and
// refuses its own codes. Every export makes a new code, with its own number.
// The seed it grants sells at the ordinary price, so codes are no way to farm
// gold.

const (
	shareVersion = 1
	sharePrefix  = "GD"
	giftSeeds    = 3 // seeds in a gift, whatever the line
	shareNameMax = 40
)

var shareEnc = base32.StdEncoding.WithPadding(base32.NoPadding)

// ShareCode is what a code says.
type ShareCode struct {
	Species string
	Genome  Genome
	Gen     int
	Stable  bool
	Name    string
	From    string
	Origin  uint32 // which garden made it
	Nonce   uint32 // which export it was
}

// cleanText keeps printable text of a sane length.
func cleanText(s string, max int) string {
	var b strings.Builder
	n := 0
	for _, r := range strings.TrimSpace(s) {
		if !unicode.IsPrint(r) || r == '\n' || r == '\t' {
			continue
		}
		if n >= max {
			break
		}
		b.WriteRune(r)
		n++
	}
	return strings.TrimSpace(b.String())
}

// originOf names a garden in a code without giving its seed away.
func (g *Garden) originOf() uint32 { return uint32(hashSerial(g.Seed, 0x0F11) >> 8) }

func (c ShareCode) payload() []byte {
	var b bytes.Buffer
	b.WriteByte(shareVersion)
	b.WriteByte(byte(len(c.Species)))
	b.WriteString(c.Species)
	var h [2]byte
	binary.BigEndian.PutUint16(h[:], c.Genome.Hue)
	b.Write(h[:])
	b.Write([]byte{c.Genome.Sat, c.Genome.Light, c.Genome.Height, c.Genome.Shape, c.Genome.Speed, c.Genome.Yield})
	gen := c.Gen
	if gen > 255 {
		gen = 255
	}
	flags := byte(0)
	if c.Stable {
		flags |= 1
	}
	b.WriteByte(byte(gen))
	b.WriteByte(flags)
	for _, s := range []string{c.Name, c.From} {
		b.WriteByte(byte(len(s)))
		b.WriteString(s)
	}
	var w [4]byte
	binary.BigEndian.PutUint32(w[:], c.Origin)
	b.Write(w[:])
	binary.BigEndian.PutUint32(w[:], c.Nonce)
	b.Write(w[:])
	return b.Bytes()
}

// Encode writes the code as text in groups of five.
func (c ShareCode) Encode() string {
	body := c.payload()
	sum := sha256.Sum256(body)
	raw := shareEnc.EncodeToString(append(body, sum[:4]...))
	var parts []string
	parts = append(parts, sharePrefix+fmt.Sprint(shareVersion))
	for len(raw) > 0 {
		n := min(5, len(raw))
		parts = append(parts, raw[:n])
		raw = raw[n:]
	}
	return strings.Join(parts, "-")
}

// ID names this code, so it can be remembered as used.
func (c ShareCode) ID() string {
	sum := sha256.Sum256(c.payload())
	return hex.EncodeToString(sum[:8])
}

// DecodeShare reads a code, checking it as it goes. Whatever is wrong with it,
// the answer is an error in plain words, never a half-made plant.
func DecodeShare(code string) (ShareCode, error) {
	var c ShareCode
	clean := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || r == '-' {
			return -1
		}
		return unicode.ToUpper(r)
	}, code)
	prefix := sharePrefix + fmt.Sprint(shareVersion)
	if !strings.HasPrefix(clean, sharePrefix) {
		return c, fmt.Errorf("that is not a garden code (they begin %s-)", prefix)
	}
	if !strings.HasPrefix(clean, prefix) {
		return c, fmt.Errorf("that code is from a different version of the garden")
	}
	raw, err := shareEnc.DecodeString(strings.TrimPrefix(clean, prefix))
	if err != nil || len(raw) < 24 {
		return c, fmt.Errorf("that code is damaged: check you copied all of it")
	}
	body, sum := raw[:len(raw)-4], raw[len(raw)-4:]
	want := sha256.Sum256(body)
	if !bytes.Equal(sum, want[:4]) {
		return c, fmt.Errorf("that code is damaged: check you copied all of it")
	}

	r := bytes.NewReader(body)
	short := fmt.Errorf("that code is damaged: it ends too soon")
	version, _ := r.ReadByte()
	if version != shareVersion {
		return c, fmt.Errorf("that code is from a different version of the garden")
	}
	readStr := func() (string, bool) {
		n, err := r.ReadByte()
		if err != nil || int(n) > r.Len() {
			return "", false
		}
		buf := make([]byte, n)
		_, _ = r.Read(buf)
		return string(buf), true
	}
	species, ok := readStr()
	if !ok {
		return c, short
	}
	var g [8]byte
	if n, _ := r.Read(g[:]); n < 8 {
		return c, short
	}
	genGen, _ := r.ReadByte()
	flags, _ := r.ReadByte()
	name, ok1 := readStr()
	from, ok2 := readStr()
	var tail [8]byte
	if n, _ := r.Read(tail[:]); !ok1 || !ok2 || n < 8 || r.Len() != 0 {
		return c, short
	}

	c.Species = species
	c.Genome = Genome{
		Hue: binary.BigEndian.Uint16(g[0:2]), Sat: g[2], Light: g[3],
		Height: g[4], Shape: g[5], Speed: g[6], Yield: g[7],
	}
	c.Gen, c.Stable = int(genGen), flags&1 != 0
	c.Name, c.From = cleanText(name, shareNameMax), cleanText(from, shareNameMax)
	c.Origin = binary.BigEndian.Uint32(tail[0:4])
	c.Nonce = binary.BigEndian.Uint32(tail[4:8])

	if SpeciesByID(c.Species) == nil {
		return c, fmt.Errorf("that code is for a plant this garden does not know (%q)", cleanText(c.Species, 20))
	}
	gn := c.Genome
	if gn.Hue >= 360 || gn.Sat > 100 || gn.Light < 3 || gn.Light > 97 || gn.Height > 100 || gn.Shape > 100 || gn.Speed > 100 || gn.Yield > 100 {
		return c, fmt.Errorf("that code carries genes outside the possible range")
	}
	if c.Name == "" {
		return c, fmt.Errorf("that code has no name for the plant")
	}
	return c, nil
}

// ExportCultivar writes the code for one of your cultivars. Each call makes a
// code of its own.
func (g *Garden) ExportCultivar(id int) (string, ShareCode, error) {
	c := g.CultivarByID(id)
	if c == nil {
		return "", ShareCode{}, fmt.Errorf("no such cultivar")
	}
	if c.SpeciesRef() == nil {
		return "", ShareCode{}, fmt.Errorf("%s is a plant this garden does not know", c.Name)
	}
	g.Exports++
	code := ShareCode{
		Species: c.Species, Genome: c.Genome, Gen: c.Gen, Stable: c.Stable,
		Name: cleanText(c.Name, shareNameMax), From: cleanText(g.Gardener, shareNameMax),
		Origin: g.originOf(), Nonce: uint32(hashSerial(g.Seed, int64(c.ID), int64(g.Exports)) >> 8),
	}
	return code.Encode(), code, nil
}

// Redemption is what receiving a code gave.
type Redemption struct {
	Cultivar Cultivar
	Packet   Packet
	NewLine  bool // false when the line was already in the almanac
}

// Redeem takes a friend's code: three seeds in the shed, and the line in the
// almanac. A code works once in a garden, and not in the garden that made it.
func (g *Garden) Redeem(text string, now time.Time) (Redemption, error) {
	c, err := DecodeShare(text)
	if err != nil {
		return Redemption{}, err
	}
	if c.Origin == g.originOf() {
		return Redemption{}, fmt.Errorf("that is a code from your own garden; it is for sharing")
	}
	id := c.ID()
	for _, used := range g.Redeemed {
		if used == id {
			return Redemption{}, fmt.Errorf("you have already used that code")
		}
	}
	sp := SpeciesByID(c.Species)

	// The line goes in the almanac, unless it is already there.
	var line *Cultivar
	newLine := false
	if existing := g.cultivarFor(sp, c.Genome); existing != nil {
		line = existing
	} else {
		g.CultivarSeq++
		variety, _ := sp.NearestVariety(c.Genome)
		g.Cultivars = append(g.Cultivars, Cultivar{
			ID: g.CultivarSeq, Name: c.Name, Species: sp.ID, Genome: c.Genome, Gen: c.Gen,
			Descent: "a gift" + fromText(c.From), Found: now, Stable: c.Stable, Named: true,
			Variety: variety, From: c.From, Gifted: true,
		})
		line = &g.Cultivars[len(g.Cultivars)-1]
		newLine = true
	}
	if c.Stable {
		line.Stable = true
	}

	streak := 0
	if c.Stable {
		streak = stableRuns
	}
	pk := Packet{
		SpeciesID: sp.ID, A: c.Genome, B: c.Genome, Count: giftSeeds, Gen: c.Gen,
		Variety: line.Variety, Label: line.Name, From: "a gift" + fromText(c.From), Descent: "a gift" + fromText(c.From),
		Streak: streak, Line: line.ID,
	}
	g.AddPacket(pk)
	g.Redeemed = append(g.Redeemed, id)
	g.Received++
	g.Log(now, "Received %d seeds of ‘%s’ (%s) as a gift%s.", giftSeeds, line.Name, sp.Common, fromText(c.From))
	return Redemption{Cultivar: *line, Packet: pk, NewLine: newLine}, nil
}

func fromText(from string) string {
	if from == "" {
		return ""
	}
	return " from " + from
}
