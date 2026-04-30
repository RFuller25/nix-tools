package main

import (
	"bufio"
	"io"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

func startRebuild(flakePath, hostname string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("sudo", "nixos-rebuild", "switch",
			"--flake", flakePath+"#"+hostname)

		pr, pw := io.Pipe()
		cmd.Stdout = pw
		cmd.Stderr = pw

		if err := cmd.Start(); err != nil {
			pw.Close()
			return errMsg{err}
		}

		go func() {
			defer pw.Close()
			cmd.Wait()
		}()

		return cmdStartedMsg{reader: bufio.NewReader(pr)}
	}
}
