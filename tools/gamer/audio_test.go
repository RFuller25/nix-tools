package main

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// countingSink stands in for the audio player: it counts what the mixer sends
// and remembers whether any of it was audible.
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

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestAudioStaysQuietUntilSomethingPlays(t *testing.T) {
	a := NewAudio(sampleRate)
	if a.playing() {
		t.Error("a player was opened before any sound was asked for")
	}
	a.Close()
	a.Close() // closing twice must be harmless
}

func TestSoundEffectsReachThePlayer(t *testing.T) {
	a := NewAudio(sampleRate)
	sink := &countingSink{}
	a.useSink(sink)
	defer a.Close()

	a.Play(sfxLines(4)...)
	waitFor(t, "a sound effect to be written", func() bool {
		_, sound, _ := sink.stats()
		return sound
	})
}

func TestMuteStopsThePlayerAndUnmuteBringsItBack(t *testing.T) {
	a := NewAudio(sampleRate)
	sink := &countingSink{}
	a.useSink(sink)

	a.SetMusic(chiptune(sampleRate, 1, gameBPM))
	waitFor(t, "music to start", func() bool {
		n, _, _ := sink.stats()
		return n > 0
	})

	a.SetMuted(true)
	if !a.Muted() {
		t.Fatal("audio does not report being muted")
	}
	if a.playing() {
		t.Error("muting left the player process running")
	}
	if _, _, closed := sink.stats(); !closed {
		t.Error("muting did not close the player's pipe")
	}

	// Muted, effects must be swallowed rather than queued up.
	blocks, _, _ := sink.stats()
	a.Play(sfxBoom()...)
	time.Sleep(60 * time.Millisecond)
	if after, _, _ := sink.stats(); after != blocks {
		t.Errorf("%d blocks were written while muted", after-blocks)
	}

	// Unmuting with no player available must not panic or wedge.
	a.SetMuted(false)
	if a.Muted() {
		t.Error("audio is still muted after unmuting")
	}
	a.Close()
}

func TestMutedAudioOpensNoPlayer(t *testing.T) {
	a := NewAudio(sampleRate)
	a.SetMuted(true)
	a.Play(sfxEat()...)
	a.SetMusic(chiptune(sampleRate, 2, menuBPM))
	if a.playing() {
		t.Error("a muted engine opened a player anyway")
	}
}

func TestAudioCanBeTurnedOffEntirely(t *testing.T) {
	t.Setenv("GAMER_AUDIO", "off")
	a := NewAudio(sampleRate)
	if a.Available() {
		t.Error("GAMER_AUDIO=off should disable audio")
	}
	if _, err := findBackend(); err != ErrNoPlayer {
		t.Errorf("findBackend with audio off returned %v", err)
	}
	a.Play(sfxMove()...)
	a.SetMusic(chiptune(sampleRate, 3, menuBPM))
	if a.playing() {
		t.Error("a player was started with audio switched off")
	}
}

func TestCloseStopsFeedingThePlayer(t *testing.T) {
	a := NewAudio(sampleRate)
	sink := &countingSink{}
	a.useSink(sink)
	a.SetMusic(chiptune(sampleRate, 4, gameBPM))

	waitFor(t, "music to start", func() bool {
		n, _, _ := sink.stats()
		return n > 0
	})
	a.Close()

	blocks, _, closed := sink.stats()
	if !closed {
		t.Error("Close left the player's pipe open")
	}
	time.Sleep(80 * time.Millisecond)
	if after, _, _ := sink.stats(); after != blocks {
		t.Errorf("%d more blocks were written after Close", after-blocks)
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
