package main

import (
	"os"
	"testing"
)

// Tests must never spawn a real audio player, however many keys they press.
func TestMain(m *testing.M) {
	os.Setenv("GARDEN_AUDIO", "off")
	os.Exit(m.Run())
}
