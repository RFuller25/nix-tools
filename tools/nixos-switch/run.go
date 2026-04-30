package main

import (
	"bufio"
	"errors"
	"io"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

type cmdStartedMsg struct {
	reader *bufio.Reader
	pr     *io.PipeReader
	exitCh chan int
}

func startRebuild(flakePath, hostname string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("sudo", "nixos-rebuild", "switch",
			"--flake", flakePath+"#"+hostname)

		pr, pw := io.Pipe()
		cmd.Stdout = pw
		cmd.Stderr = pw

		if err := cmd.Start(); err != nil {
			pw.Close()
			pr.Close()
			return errMsg{err}
		}

		exitCh := make(chan int, 1)
		go func() {
			defer pw.Close()
			if err := cmd.Wait(); err != nil {
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) {
					exitCh <- exitErr.ExitCode()
					return
				}
				exitCh <- 1
				return
			}
			exitCh <- 0
		}()

		return cmdStartedMsg{reader: bufio.NewReader(pr), pr: pr, exitCh: exitCh}
	}
}
