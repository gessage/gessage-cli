package ui

import (
	"fmt"
	"runtime"
	"time"
)

// Spinner is a simple terminal spinner for long-running operations.
type Spinner struct {
	label    string
	frames   []string
	interval time.Duration
	stopCh   chan struct{}
	doneCh   chan struct{}
}

// NewSpinner creates a spinner with a label.
func NewSpinner(label string) *Spinner {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	if runtime.GOOS == "windows" {
		frames = []string{"-", "\\", "|", "/"}
	}
	return &Spinner{
		label:    label,
		frames:   frames,
		interval: 120 * time.Millisecond,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

// Start begins rendering the spinner until Stop is called.
func (s *Spinner) Start() {
	go func() {
		i := 0
		for {
			select {
			case <-s.stopCh:
				// Clear the current line
				fmt.Printf("\r\033[K")
				close(s.doneCh)
				return
			default:
				frame := s.frames[i%len(s.frames)]
				fmt.Printf("\r%s %s", frame, s.label)
				time.Sleep(s.interval)
				i++
			}
		}
	}()
}

// Stop stops the spinner and clears the line.
func (s *Spinner) Stop() {
	select {
	case <-s.doneCh:
		return
	default:
		close(s.stopCh)
		<-s.doneCh
	}
}
