package ui

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ReadChoice reads a short token from stdin.
func ReadChoice() (string, error) {
	in := bufio.NewReader(os.Stdin)
	s, err := in.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.ToLower(strings.TrimSpace(s)), nil
}

// EditInEditor opens $EDITOR if set; otherwise allows a simple inline edit.
func EditInEditor(initial string) (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		fmt.Println("\nEnter new message, end with a single line containing only '.' :")
		return readMultilineWithTerminator(initial)
	}
	tmp, err := os.CreateTemp("", "gessage-*.txt")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(initial); err != nil {
		return "", err
	}
	tmp.Close()

	cmd := exec.Command(editor, tmp.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}

	b, err := os.ReadFile(tmp.Name())
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func readMultilineWithTerminator(initial string) (string, error) {
	fmt.Println("--- current ---")
	fmt.Println(initial)
	fmt.Println("----------------")
	fmt.Println("(Edit lines below; type '.' on its own line to finish)")
	var lines []string
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		t := sc.Text()
		if strings.TrimSpace(t) == "." {
			break
		}
		lines = append(lines, t)
	}
	if err := sc.Err(); err != nil {
		return "", err
	}
	txt := strings.Join(lines, "\n")
	if strings.TrimSpace(txt) == "" {
		return initial, nil
	}
	return txt, nil
}
