package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// Audio turns the mixer's samples into sound by piping raw PCM to whatever
// player the machine already has. Nothing is linked in, so the binary stays
// pure Go, and a machine with no sound simply plays in silence.

// blockFrames is how many samples are rendered per write: about 23ms at
// 44.1kHz, short enough that a bleep lands when you press the key.
const blockFrames = 1024

type sink interface {
	io.WriteCloser
}

// backend is one way of getting PCM out of the machine. Each takes signed
// 16-bit little-endian mono samples on standard input.
type backend struct {
	name string
	args func(rate int) []string
}

var backends = []backend{
	{"pw-play", func(r int) []string {
		return []string{"--format", "s16", "--rate", fmt.Sprint(r), "--channels", "1", "--raw", "-"}
	}},
	{"paplay", func(r int) []string {
		return []string{"--raw", "--format=s16le", fmt.Sprintf("--rate=%d", r), "--channels=1", "--latency-msec=40"}
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

// ErrNoPlayer means the machine has no audio player we know how to drive.
var ErrNoPlayer = errors.New("no audio player found (tried pw-play, paplay, aplay, ffplay and sox's play)")

// Audio owns the mixer, the player process and the goroutine between them.
//
// Muting is not a volume of zero: it shuts the player down entirely, so a
// muted game holds no sound device and burns no CPU. Unmuting brings the
// music back where it left off.
type Audio struct {
	mix  *mixer
	rate int

	mu      sync.Mutex
	muted   bool
	music   *sequencer
	sink    sink
	cmd     *exec.Cmd
	backend string
	stop    chan struct{}
	done    chan struct{}
	err     error
	broken  bool
}

func NewAudio(rate int) *Audio {
	return &Audio{mix: newMixer(rate), rate: rate}
}

// Available reports whether there is a player to use.
func (a *Audio) Available() bool {
	a.mu.Lock()
	broken := a.broken
	a.mu.Unlock()
	if broken {
		return false
	}
	_, err := findBackend()
	return err == nil
}

// Backend names the player in use, for the menu.
func (a *Audio) Backend() string {
	a.mu.Lock()
	name := a.backend
	a.mu.Unlock()
	if name != "" {
		return name
	}
	if b, err := findBackend(); err == nil {
		return b.name
	}
	return "none"
}

func findBackend() (backend, error) {
	if os.Getenv("GAMER_AUDIO") == "off" {
		return backend{}, ErrNoPlayer
	}
	for _, b := range backends {
		if _, err := exec.LookPath(b.name); err == nil {
			return b, nil
		}
	}
	return backend{}, ErrNoPlayer
}

// Muted reports whether sound is currently switched off.
func (a *Audio) Muted() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.muted
}

// SetMuted switches sound off or back on, stopping and restarting the player.
func (a *Audio) SetMuted(muted bool) {
	a.mu.Lock()
	if a.muted == muted {
		a.mu.Unlock()
		return
	}
	a.muted = muted
	music := a.music
	a.mu.Unlock()

	if muted {
		a.mix.setSequencer(nil)
		a.closePlayer()
		return
	}
	if music != nil {
		a.SetMusic(music)
	}
}

// Play sounds an effect. It is a no-op when muted or when there is nothing to
// play through, so callers never have to check.
func (a *Audio) Play(vs ...voice) {
	if len(vs) == 0 || !a.open() {
		return
	}
	a.mix.play(vs...)
}

// SetMusic starts a piece looping. Passing nil stops the music but leaves
// sound effects working.
func (a *Audio) SetMusic(s *sequencer) {
	a.mu.Lock()
	a.music = s
	a.mu.Unlock()

	if s == nil {
		a.mix.setSequencer(nil)
		return
	}
	if !a.open() {
		return
	}
	a.mix.setSequencer(s)
}

// open makes sure a player is running, unless muted or unavailable.
func (a *Audio) open() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.muted || a.broken {
		return false
	}
	if a.sink != nil {
		return true
	}

	b, err := findBackend()
	if err != nil {
		a.err = err
		return false
	}
	cmd := exec.Command(b.name, b.args(a.rate)...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		a.err = fmt.Errorf("opening %s: %w", b.name, err)
		return false
	}
	if err := cmd.Start(); err != nil {
		a.err = fmt.Errorf("starting %s: %w", b.name, err)
		return false
	}

	a.cmd, a.sink, a.backend = cmd, stdin, b.name
	a.stop, a.done = make(chan struct{}), make(chan struct{})
	go a.pump(stdin, a.stop, a.done)
	return true
}

// useSink swaps in a writer instead of a real player. Tests use it.
func (a *Audio) useSink(w sink) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sink, a.backend = w, "test"
	a.stop, a.done = make(chan struct{}), make(chan struct{})
	go a.pump(w, a.stop, a.done)
}

// pump renders blocks of samples and writes them out. The write blocks until
// the player wants more, which is what keeps playback in real time.
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
			a.err, a.broken = err, true
			a.mu.Unlock()
			return
		}
	}
}

// playing reports whether a player is running right now.
func (a *Audio) playing() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.sink != nil
}

// Err reports why sound stopped, if it did.
func (a *Audio) Err() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.err
}

// closePlayer stops the feeding goroutine and shuts the player down, keeping
// the remembered music so unmuting can pick it up again.
func (a *Audio) closePlayer() {
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

// Close shuts everything down for good.
func (a *Audio) Close() {
	a.mix.setSequencer(nil)
	a.closePlayer()
}

// playerHint is the message shown when there is nothing to play through.
func playerHint() string {
	names := make([]string, 0, len(backends))
	for _, b := range backends {
		names = append(names, b.name)
	}
	return "no audio player found — install one of: " + strings.Join(names, ", ")
}
