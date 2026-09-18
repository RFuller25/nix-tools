package main

// Auditioning the music without a sound card: RENDER_DIR=/tmp go test -run
// TestRenderCalmMusic writes a minute of the garden's piece to a WAV file.

import (
	"encoding/binary"
	"os"
	"testing"
)

func TestRenderCalmMusic(t *testing.T) {
	dir := os.Getenv("RENDER_DIR")
	if dir == "" {
		t.Skip("set RENDER_DIR to render demo audio")
	}
	m := newMixer(sampleRate)
	m.setSequencer(calmMusic(sampleRate, 20260918))
	samples := make([]float64, sampleRate*60)
	m.fill(samples)

	f, err := os.Create(dir + "/garden-calm.wav")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	data := make([]byte, 2*len(samples))
	for i, s := range samples {
		binary.LittleEndian.PutUint16(data[2*i:], uint16(int16(s*32767)))
	}
	for _, v := range []any{
		[]byte("RIFF"), uint32(36 + len(data)), []byte("WAVEfmt "), uint32(16),
		uint16(1), uint16(1), uint32(sampleRate), uint32(sampleRate * 2),
		uint16(2), uint16(16), []byte("data"), uint32(len(data)),
	} {
		binary.Write(f, binary.LittleEndian, v)
	}
	f.Write(data)
}
