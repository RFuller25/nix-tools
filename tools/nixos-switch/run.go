package main

import (
	"bufio"
	"errors"
	"io"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type cmdStartedMsg struct {
	reader *bufio.Reader
	pr     *io.PipeReader
	exitCh chan int
}

type sudoOKMsg struct{}
type sudoNeededMsg struct{}
type sudoFailedMsg struct{ err error }

// checkSudo returns sudoOKMsg if sudo credentials are cached, sudoNeededMsg if not.
func checkSudo() tea.Cmd {
	return func() tea.Msg {
		if exec.Command("sudo", "-n", "true").Run() == nil {
			return sudoOKMsg{}
		}
		return sudoNeededMsg{}
	}
}

// authenticateSudo pipes password to sudo -S -v to cache credentials.
func authenticateSudo(password string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("sudo", "-S", "-v")
		cmd.Stdin = strings.NewReader(password + "\n")
		if out, err := cmd.CombinedOutput(); err != nil {
			_ = out
			return sudoFailedMsg{err: err}
		}
		return sudoOKMsg{}
	}
}

func startBuild(flakePath, hostname string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("nh",
			"-e", "sudo",
			"os", "switch",
			"--update",
			"--diff", "always",
			"--no-nom",
			"--hostname", hostname,
			flakePath)

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
