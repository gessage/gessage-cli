package sanitize

import (
	"regexp"
	"strings"
)

type Stats struct {
	RedactedCount int
}

// Common secret patterns; extend as needed (AWS, GitHub tokens, JWTs, etc.)
var patterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(api[-_ ]?key|secret|token|password|passwd|pwd)\s*[:=]\s*['"]?([A-Za-z0-9_\-=\./+]{6,})['"]?`),
	regexp.MustCompile(`(?i)authorization:\s*Bearer\s+[A-Za-z0-9_\-=\./+]{10,}`),
	regexp.MustCompile(`(?i)(x-amz-security-token|aws_secret_access_key|aws_access_key_id)\s*[:=]\s*[A-Za-z0-9/\+=]{8,}`),
	regexp.MustCompile(`(?i)(PRIVATE KEY-----[\s\S]+?-----END [A-Z ]+-----)`),
}

func Redact(diff string) (string, Stats) {
	s := diff
	stat := Stats{}
	for _, re := range patterns {
		s = re.ReplaceAllStringFunc(s, func(m string) string {
			stat.RedactedCount++
			return re.ReplaceAllString(m, "[REDACTED]")
		})
	}
	// Also nuke obvious .env style lines entirely
	lines := strings.Split(s, "\n")
	for i, ln := range lines {
		if looksSensitive(ln) {
			lines[i] = "[REDACTED LINE]"
			stat.RedactedCount++
		}
	}
	return strings.Join(lines, "\n"), stat
}

func looksSensitive(line string) bool {
	l := strings.ToLower(line)
	return strings.Contains(l, "secret=") ||
		strings.Contains(l, "password=") ||
		strings.Contains(l, "token=") ||
		strings.Contains(l, "api_key=") ||
		strings.Contains(l, "apikey=")
}
