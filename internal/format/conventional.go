package format

// (replace the entire tail of the file with this)

import (
	"bufio"
	"regexp"
	"strconv"
	"strings"
)

var AllowedTypes = []string{"feat", "fix", "refactor", "docs", "chore", "style", "test", "perf"}

type PromptInput struct {
	Diff         string
	Types        []string
	MaxTitle     int
	MaxBody      int
	UserTypeHint string
}

func BuildPrompt(in PromptInput) string {
	typeList := strings.Join(in.Types, ", ")
	hint := ""
	if in.UserTypeHint != "" {
		hint = "\nUser-specified type hint: " + in.UserTypeHint
	}
	return `Generate a Conventional Commit message from the following staged git diff.
Constraints:
- title <= ` + strconv.Itoa(in.MaxTitle) + ` characters
- optional body lines <= ` + strconv.Itoa(in.MaxBody) + ` columns
- types allowed: ` + typeList + `
Output format:
- First line: "<type>(optional scope): <title>"
- Optional body: wrapped to ` + strconv.Itoa(in.MaxBody) + ` columns; include bullet points if helpful.
- Do not include code fences, backticks, or explanations.

` + hint + `

Diff:
` + in.Diff + `
`
}

type NormalizeOptions struct {
	MaxTitle    int
	MaxBody     int
	Types       []string
	DefaultType string
}

func NormalizeMessage(msg string, opt NormalizeOptions) string {
	msg = strings.TrimSpace(msg)
	msg = stripNonCommitNoise(msg)

	lines := strings.Split(msg, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		return opt.DefaultType + ": update"
	}

	first := strings.TrimSpace(lines[0])
	ty := leadingType(first)
	if !containsCaseInsensitive(opt.Types, ty) {
		first = opt.DefaultType + ": " + first
	}
	if len(first) > opt.MaxTitle {
		first = first[:opt.MaxTitle]
	}

	body := strings.Join(lines[1:], "\n")
	body = wrapLines(body, opt.MaxBody)
	body = strings.TrimRight(body, "\n")

	if strings.TrimSpace(body) == "" {
		return first
	}
	return first + "\n\n" + body
}

func FallbackFromDiff(diff string) string {
	added, removed, files := countDiffStats(diff)
	if len(files) == 0 {
		files = []string{"files"}
	}
	title := "chore: update " + strings.Join(files, ", ")
	if len(title) > 72 {
		title = title[:72]
	}
	var body []string
	if added > 0 {
		body = append(body, "- Additions: "+strconv.Itoa(added))
	}
	if removed > 0 {
		body = append(body, "- Deletions: "+strconv.Itoa(removed))
	}
	if len(body) == 0 {
		return title
	}
	return title + "\n\n" + strings.Join(body, "\n")
}

func wrapLines(s string, width int) string {
	if width <= 0 {
		return s
	}
	var out []string
	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		line := sc.Text()
		for len(line) > width {
			// find last space within width
			breakAt := width
			for i := width; i > 0; i-- {
				if line[i-1] == ' ' {
					breakAt = i
					break
				}
			}
			out = append(out, strings.TrimRight(line[:breakAt], " "))
			line = strings.TrimLeft(line[breakAt:], " ")
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func leadingType(first string) string {
	re := regexp.MustCompile(`^([a-z]+)($begin:math:text$[\\w\\-\\./]+$end:math:text$)?:`)
	m := re.FindStringSubmatch(strings.ToLower(first))
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}

func containsCaseInsensitive(arr []string, v string) bool {
	v = strings.ToLower(strings.TrimSpace(v))
	for _, a := range arr {
		if strings.ToLower(a) == v {
			return true
		}
	}
	return false
}

func stripNonCommitNoise(s string) string {
	// Remove triple backticks blocks if model mistakenly added them
	s = regexp.MustCompile("(?s)```.*?```").ReplaceAllString(s, "")
	// Keep only first non-empty block
	parts := strings.Split(strings.TrimSpace(s), "\n\n")
	if len(parts) > 0 {
		return strings.TrimSpace(parts[0] + "\n\n" + strings.Join(parts[1:], "\n"))
	}
	return s
}

func countDiffStats(diff string) (added, removed int, files []string) {
	seen := map[string]bool{}
	sc := bufio.NewScanner(strings.NewReader(diff))
	for sc.Scan() {
		ln := sc.Text()
		if strings.HasPrefix(ln, "+++ b/") || strings.HasPrefix(ln, "--- a/") {
			// collect file names
			name := strings.TrimPrefix(strings.TrimPrefix(ln, "+++ b/"), "--- a/")
			if name != "/dev/null" && name != "" && !seen[name] {
				seen[name] = true
				files = append(files, name)
			}
		}
		if strings.HasPrefix(ln, "+") && !strings.HasPrefix(ln, "+++") {
			added++
		}
		if strings.HasPrefix(ln, "-") && !strings.HasPrefix(ln, "---") {
			removed++
		}
	}
	return
}
