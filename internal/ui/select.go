package ui

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
)

// Select renders an interactive up/down arrow list and returns the chosen index.
// If the terminal is not interactive, it falls back to a simple numeric prompt.
func Select(label string, options []string, initialIndex int) (int, error) {
	if len(options) == 0 {
		return 0, errors.New("no options to select from")
	}

	// On Windows, use a simple numbered prompt for compatibility
	if runtime.GOOS == "windows" {
		fmt.Println(label)
		for i, opt := range options {
			fmt.Printf("  %d) %s\n", i+1, opt)
		}
		fmt.Print("Enter number > ")
		in := bufio.NewReader(os.Stdin)
		s, _ := in.ReadString('\n')
		s = strings.TrimSpace(s)
		for i := range options {
			if fmt.Sprintf("%d", i+1) == s {
				return i, nil
			}
		}
		return 0, errors.New("invalid selection")
	}

	// Best-effort TTY check: if stdin is not a char device, fallback to numbered prompt
	st, _ := os.Stdin.Stat()
	if st == nil || (st.Mode()&os.ModeCharDevice) == 0 {
		// Fallback: numbered prompt
		fmt.Println(label)
		for i, opt := range options {
			fmt.Printf("  %d) %s\n", i+1, opt)
		}
		fmt.Print("Enter number > ")
		in := bufio.NewReader(os.Stdin)
		s, _ := in.ReadString('\n')
		s = strings.TrimSpace(s)
		for i := range options {
			if fmt.Sprintf("%d", i+1) == s {
				return i, nil
			}
		}
		return 0, errors.New("invalid selection")
	}

	// Save current tty state
	saveCmd := exec.Command("sh", "-c", "stty -g < /dev/tty")
	savedStateBytes, err := saveCmd.Output()
	if err != nil {
		return 0, err
	}
	savedState := strings.TrimSpace(string(savedStateBytes))
	// Ensure we restore at the end
	defer exec.Command("sh", "-c", fmt.Sprintf("stty %s < /dev/tty", savedState)).Run()
	// Enter raw -echo mode
	if err := exec.Command("sh", "-c", "stty raw -echo < /dev/tty").Run(); err != nil {
		return 0, err
	}

	// Handle Ctrl-C gracefully
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	defer signal.Stop(c)

	selected := initialIndex
	if selected < 0 || selected >= len(options) {
		selected = 0
	}

	clear := func() { fmt.Printf("\r\033[K") }
	redraw := func() {
		fmt.Print("\x1b[?25l") // hide cursor
		defer fmt.Print("\x1b[?25h")

		fmt.Printf("\r\033[K%s\n", label)
		for i, opt := range options {
			clear()
			if i == selected {
				fmt.Printf("\r> %s\n", opt)
			} else {
				fmt.Printf("\r  %s\n", opt)
			}
		}
		fmt.Print("\r\033[KUse ↑/↓, Enter to select, q to cancel\n")
	}

	redraw()
	// Read directly from the TTY to avoid buffered stdin issues
	tty, err := os.OpenFile("/dev/tty", os.O_RDONLY, 0)
	if err != nil {
		return 0, err
	}
	defer tty.Close()
	reader := bufio.NewReader(tty)
	for {
		select {
		case <-c:
			return 0, errors.New("cancelled")
		default:
		}

		b, err := reader.ReadByte()
		if err != nil {
			return 0, err
		}

		if b == 'q' || b == 'Q' { // quit
			return 0, errors.New("cancelled")
		}
		if b == '\r' || b == '\n' { // enter
			// Move cursor below the list cleanly
			fmt.Println()
			return selected, nil
		}

		if b == 0x1b { // ESC sequence
			next1, _ := reader.ReadByte()
			next2, _ := reader.ReadByte()
			if next1 == '[' {
				switch next2 {
				case 'A': // up
					if selected > 0 {
						selected--
						recount := len(options) + 2
						fmt.Printf("\x1b[%dA", recount) // move cursor up to top
						redraw()
					}
				case 'B': // down
					if selected < len(options)-1 {
						selected++
						recount := len(options) + 2
						fmt.Printf("\x1b[%dA", recount)
						redraw()
					}
				}
			}
		}
	}
}
