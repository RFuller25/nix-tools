package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
)

// Audio plays the garden's music by generating samples and piping raw PCM to
// whatever audio player the system already has. Nothing is linked in, so the
// binary stays pure Go and a machine with no sound simply stays quiet.

// blockFrames is how many samples are rendered per write. At 44.1kHz this is
// about 23ms, small enough to start and stop music promptly.
const blockFrames = 1024

// sink is the external player's stdin.
type sink interface {
	io.WriteCloser
}

// backend describes one way of getting PCM out of the machine. Each takes
// signed 16-bit little-endian mono samples on standard input.
type backend struct {
	name string
	args func(rate int) []string
}

var backends = []backend{
	{"pw-play", func(r int) []string {
		return []string{"--format", "s16", "--rate", fmt.Sprint(r), "--channels", "1", "--raw", "-"}
	}},
	{"paplay", func(r int) []string {
		return []string{"--raw", "--format=s16le", fmt.Sprintf("--rate=%d", r), "--channels=1", "--latency-msec=60"}
	}},
	{"aplay", func(r int) []string {
		return []string{"-q", "-t", "raw", "-f", "S16_LE", "-r", fmt.Sprint(r), "-c", "1", "-"}
	}},
	{"ffplay", func(r int) []string {
		return []string{"-hide_banner", "-loglevel", "quiet", "-nodisp", "-autoexit",
			"-f", "s16le", "-ar", fmt.Sprint(r), "-ac", "1", "-i", "-"}
	}},
	{"play", func(r int) []string { // sox
		return []string{"-q", "-t", "raw", "-r", fmt.Sprint(r), "-e", "signed", "-b", "16", "-c", "1", "-"}
	}},
}

// Audio owns the mixer, the player process and the goroutine between them.
type Audio struct {
	mix     *mixer
	rate    int
	backend string

	mu     sync.Mutex
	sink   sink
	cmd    *exec.Cmd
	stop   chan struct{}
	done   chan struct{}
	failed atomic.Bool
	err    error
}

// ErrNoPlayer means the machine has no audio player we know how to drive.
var ErrNoPlayer = errors.New("no audio player found (tried pw-play, paplay, aplay, ffplay and sox's play)")

// NewAudio prepares audio without starting anything. Playback begins on the
// first call to StartMusic, so a gardener who never presses the key never
// spawns a process.
func NewAudio(rate int) *Audio {
	return &Audio{mix: newMixer(rate), rate: rate}
}

// Available reports whether a player exists and has not failed on us.
func (a *Audio) Available() bool {
	if os.Getenv("GARDEN_AUDIO") == "off" {
		return false
	}
	if a.failed.Load() {
		return false
	}
	_, err := findBackend()
	return err == nil
}

// Backend names the player in use, for the help screen.
func (a *Audio) Backend() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.backend != "" {
		return a.backend
	}
	if b, err := findBackend(); err == nil {
		return b.name
	}
	return "none"
}

func findBackend() (backend, error) {
	if os.Getenv("GARDEN_AUDIO") == "off" {
		return backend{}, ErrNoPlayer
	}
	for _, b := range backends {
		if _, err := exec.LookPath(b.name); err == nil {
			return b, nil
		}
	}
	return backend{}, ErrNoPlayer
}

// Playing reports whether music is currently sounding.
func (a *Audio) Playing() bool {
	a.mu.Lock()
	running := a.sink != nil
	a.mu.Unlock()
	return running && a.mix.playing()
}

// StartMusic opens the player if needed and sets a piece playing.
func (a *Audio) StartMusic(s *sequencer) error {
	a.mu.Lock()
	if a.sink == nil {
		if err := a.openLocked(); err != nil {
			a.mu.Unlock()
			return err
		}
	}
	a.mu.Unlock()

	a.mix.setSequencer(s)
	return nil
}

// StopMusic silences the music and lets the player process go, so nothing is
// left holding the sound card open.
func (a *Audio) StopMusic() {
	a.mix.setSequencer(nil)
	a.Close()
}

// openLocked spawns the player and the goroutine that feeds it. The caller
// holds a.mu.
func (a *Audio) openLocked() error {
	b, err := findBackend()
	if err != nil {
		return err
	}
	cmd := exec.Command(b.name, b.args(a.rate)...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("opening %s: %w", b.name, err)
	}
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting %s: %w", b.name, err)
	}

	a.cmd, a.sink, a.backend = cmd, stdin, b.name
	a.stop, a.done = make(chan struct{}), make(chan struct{})
	go a.pump(stdin, a.stop, a.done)
	return nil
}

// useSink swaps in a writer instead of a real player. Tests use it; nothing
// else does.
func (a *Audio) useSink(w sink) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sink, a.backend = w, "test"
	a.stop, a.done = make(chan struct{}), make(chan struct{})
	go a.pump(w, a.stop, a.done)
}

// pump renders sample blocks and writes them out. The write blocks until the
// player is ready for more, which is what keeps playback in real time.
func (a *Audio) pump(w io.Writer, stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)

	buf := make([]float64, blockFrames)
	pcm := make([]byte, 2*blockFrames)
	for {
		select {
		case <-stop:
			return
		default:
		}

		a.mix.fill(buf)
		for i, s := range buf {
			n := int16(s * 32767)
			pcm[2*i] = byte(n)
			pcm[2*i+1] = byte(n >> 8)
		}
		if _, err := w.Write(pcm); err != nil {
			a.mu.Lock()
			a.err = err
			a.mu.Unlock()
			a.failed.Store(true)
			return
		}
	}
}

// Err reports why audio stopped, if it did.
func (a *Audio) Err() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.err
}

// Close stops the feeding goroutine and shuts the player down.
func (a *Audio) Close() {
	a.mu.Lock()
	stop, done, s, cmd := a.stop, a.done, a.sink, a.cmd
	a.stop, a.done, a.sink, a.cmd = nil, nil, nil, nil
	a.mu.Unlock()

	if stop == nil {
		return
	}
	close(stop)
	<-done
	if s != nil {
		s.Close()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}
}

// playerHint is the message shown when there is nothing to play through.
func playerHint() string {
	names := make([]string, 0, len(backends))
	for _, b := range backends {
		names = append(names, b.name)
	}
	return "no audio player found — install one of: " + strings.Join(names, ", ")
}
