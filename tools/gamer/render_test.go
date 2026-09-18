package main

// Auditioning the audio without a sound card: RENDER_DIR=/tmp go test -run
// TestRenderDemos writes the theme and every sound effect to WAV files.

import (
	"encoding/binary"
	"os"
	"testing"
)

func writeWAV(t *testing.T, path string, samples []float64) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	data := make([]byte, 2*len(samples))
	for i, s := range samples {
		binary.LittleEndian.PutUint16(data[2*i:], uint16(int16(s*32767)))
	}
	hdr := []any{
		[]byte("RIFF"), uint32(36 + len(data)), []byte("WAVEfmt "), uint32(16),
		uint16(1), uint16(1), uint32(sampleRate), uint32(sampleRate * 2),
		uint16(2), uint16(16), []byte("data"), uint32(len(data)),
	}
	for _, v := range hdr {
		binary.Write(f, binary.LittleEndian, v)
	}
	f.Write(data)
}

func TestRenderDemos(t *testing.T) {
	dir := os.Getenv("RENDER_DIR")
	if dir == "" {
		t.Skip("set RENDER_DIR to render demo audio")
	}

	// The looping theme.
	m := newMixer(sampleRate)
	m.setSequencer(chiptune(sampleRate, 7, gameBPM))
	theme := make([]float64, sampleRate*20)
	m.fill(theme)
	writeWAV(t, dir+"/gamer-theme.wav", theme)

	// Every effect in turn, over the theme, a second apart.
	m2 := newMixer(sampleRate)
	m2.setSequencer(chiptune(sampleRate, 7, gameBPM))
	effects := [][]voice{
		sfxMove(), sfxSelect(), sfxShift(), sfxRotate(), sfxLock(), sfxHardDrop(),
		sfxLines(1), sfxLines(4), sfxLevelUp(), sfxHold(), sfxSlide(), sfxMerge(8),
		sfxMerge(1024), sfxEat(), sfxPick(), sfxSwap(), sfxReveal(), sfxFlag(),
		sfxBoom(), sfxGameOver(), sfxWin(),
	}
	out := make([]float64, 0, sampleRate*len(effects))
	block := make([]float64, sampleRate)
	for _, e := range effects {
		m2.play(e...)
		m2.fill(block)
		out = append(out, block...)
	}
	writeWAV(t, dir+"/gamer-sfx.wav", out)
}
