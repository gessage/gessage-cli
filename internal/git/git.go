package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// GetStagedDiff returns the staged diff (what would be committed).
func GetStagedDiff(ctx context.Context) (string, error) {
	// --staged ensures only staged changes
	cmd := exec.CommandContext(ctx, "git", "diff", "--staged", "--no-color")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git diff failed: %v\n%s", err, out.String())
	}
	return out.String(), nil
}

// CommitWithMessage pipes the message to `git commit -F -`
func CommitWithMessage(ctx context.Context, msg string) error {
	cmd := exec.CommandContext(ctx, "git", "commit", "-F", "-")
	cmd.Stdin = strings.NewReader(msg)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git commit failed: %v\n%s", err, out.String())
	}
	return nil
}
