package main

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// countingSink stands in for the audio player: it counts what the mixer sends
// and remembers whether any of it was actually audible.
type countingSink struct {
	mu     sync.Mutex
	blocks int
	sound  bool
	closed bool
}

func (c *countingSink) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.blocks++
	for i := 0; i+1 < len(p); i += 2 {
		if int16(p[i])|int16(p[i+1])<<8 != 0 {
			c.sound = true
			break
		}
	}
	return len(p), nil
}

func (c *countingSink) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}

func (c *countingSink) stats() (int, bool, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.blocks, c.sound, c.closed
}

func TestAudioStaysQuietUntilAsked(t *testing.T) {
	a := NewAudio(sampleRate)
	if a.Playing() {
		t.Error("audio reports playing before anything started")
	}
	a.Close() // closing an unopened engine must be harmless
	a.Close()
}

func TestAudioWritesSamples(t *testing.T) {
	a := NewAudio(sampleRate)
	sink := &countingSink{}
	a.useSink(sink)
	defer a.Close()

	a.mix.setSequencer(calmMusic(sampleRate, 7))
	if !a.Playing() {
		t.Error("audio does not report playing while music is set")
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if blocks, sound, _ := sink.stats(); blocks > 0 && sound {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	blocks, sound, _ := sink.stats()
	if blocks == 0 {
		t.Fatal("nothing was ever written to the player")
	}
	if !sound {
		t.Error("everything written to the player was silence")
	}
}

func TestCloseStopsFeedingThePlayer(t *testing.T) {
	a := NewAudio(sampleRate)
	sink := &countingSink{}
	a.useSink(sink)
	a.mix.setSequencer(calmMusic(sampleRate, 8))

	time.Sleep(50 * time.Millisecond)
	a.Close()

	blocks, _, closed := sink.stats()
	if !closed {
		t.Error("closing the engine left the player's pipe open")
	}
	time.Sleep(80 * time.Millisecond)
	after, _, _ := sink.stats()
	if after != blocks {
		t.Errorf("%d more blocks were written after Close", after-blocks)
	}
	if a.Playing() {
		t.Error("audio still reports playing after Close")
	}
}

func TestStopMusicReleasesThePlayer(t *testing.T) {
	a := NewAudio(sampleRate)
	sink := &countingSink{}
	a.useSink(sink)
	a.mix.setSequencer(calmMusic(sampleRate, 9))

	a.StopMusic()
	if a.Playing() {
		t.Error("music still playing after StopMusic")
	}
	if _, _, closed := sink.stats(); !closed {
		t.Error("StopMusic left the player process running")
	}
}

func TestAudioCanBeTurnedOffEntirely(t *testing.T) {
	t.Setenv("GARDEN_AUDIO", "off")
	a := NewAudio(sampleRate)
	if a.Available() {
		t.Error("GARDEN_AUDIO=off should disable audio")
	}
	if _, err := findBackend(); err != ErrNoPlayer {
		t.Errorf("findBackend with audio off returned %v", err)
	}
	if err := a.StartMusic(calmMusic(sampleRate, 1)); err == nil {
		t.Error("StartMusic should fail when there is no player")
	}
	if a.Playing() {
		t.Error("audio is playing with no player")
	}
}

func TestPlayerHintNamesThePlayers(t *testing.T) {
	hint := playerHint()
	for _, name := range []string{"pw-play", "paplay", "aplay", "ffplay", "play"} {
		if !strings.Contains(hint, name) {
			t.Errorf("the hint does not mention %s: %q", name, hint)
		}
	}
}

func TestBackendArgumentsCarryTheFormat(t *testing.T) {
	for _, b := range backends {
		args := strings.Join(b.args(sampleRate), " ")
		if !strings.Contains(args, "44100") {
			t.Errorf("%s is not told the sample rate: %q", b.name, args)
		}
		if !strings.Contains(strings.ToLower(args), "s16") && !strings.Contains(args, "signed") {
			t.Errorf("%s is not told the sample format: %q", b.name, args)
		}
	}
}

// The music must stay inside the range a 16-bit sample can hold, whatever the
// mixer is asked to play at once.
func TestMusicNeverClips(t *testing.T) {
	m := newMixer(sampleRate)
	m.setSequencer(calmMusic(sampleRate, 11))
	buf := make([]float64, 4096)
	for i := 0; i < 200; i++ { // about twenty seconds of music
		m.fill(buf)
		for _, v := range buf {
			if v < -1 || v > 1 {
				t.Fatalf("sample %v is outside the -1..1 range", v)
			}
		}
	}
}

func TestCalmMusicIsRepeatable(t *testing.T) {
	render := func(seed int64) []float64 {
		m := newMixer(sampleRate)
		m.setSequencer(calmMusic(sampleRate, seed))
		out := make([]float64, 0, 8192)
		buf := make([]float64, 2048)
		for i := 0; i < 4; i++ {
			m.fill(buf)
			out = append(out, buf...)
		}
		return out
	}
	a, b, c := render(3), render(3), render(4)
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("the same seed drifted apart at sample %d", i)
		}
	}
	same := true
	for i := range a {
		if a[i] != c[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("different seeds produced identical music")
	}
}

func TestCalmMusicKeepsPlaying(t *testing.T) {
	m := newMixer(sampleRate)
	m.setSequencer(calmMusic(sampleRate, 12))

	buf := make([]float64, sampleRate) // a second at a time
	quiet := 0
	for i := 0; i < 60; i++ { // a minute of music
		m.fill(buf)
		loud := false
		for _, v := range buf {
			if v > 0.005 || v < -0.005 {
				loud = true
				break
			}
		}
		if !loud {
			quiet++
		}
	}
	if quiet > 5 {
		t.Errorf("%d of 60 seconds were silent; the piece should keep going", quiet)
	}
}
