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
- Optional body: wrapped to ` + strconv.Itoa(in.MaxBody) + ` columns.
- Output ONLY the commit message. No steps, no tables, no quotes, no extra text.
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

	// Pick the first line that looks like a proper Conventional Commit title
	lines := strings.Split(msg, "\n")
	titleIdx := -1
	var title string
	titleRe := regexp.MustCompile(`(?i)^(feat|fix|refactor|docs|chore|style|test|perf)(\([^)]+\))?:\s+.+$`)
	for i, ln := range lines {
		l := strings.TrimSpace(ln)
		if l == "" {
			continue
		}
		if titleRe.MatchString(l) {
			titleIdx = i
			title = l
			break
		}
	}
	if titleIdx == -1 {
		// fallback: use first non-empty line as title
		for _, ln := range lines {
			l := strings.TrimSpace(ln)
			if l != "" {
				title = l
				break
			}
		}
	}
	if title == "" {
		return opt.DefaultType + ": update"
	}

	ty := leadingType(title)
	if !containsCaseInsensitive(opt.Types, ty) {
		title = opt.DefaultType + ": " + title
	}
	if len(title) > opt.MaxTitle {
		title = title[:opt.MaxTitle]
	}

	// Body is the text after the title, filtered to remove instructions/tables
	var bodyLines []string
	if titleIdx >= 0 && titleIdx+1 < len(lines) {
		for _, ln := range lines[titleIdx+1:] {
			t := strings.TrimSpace(ln)
			if t == "" {
				bodyLines = append(bodyLines, "")
				continue
			}
			// Drop tables and numbered instructions
			if strings.HasPrefix(t, "|") || regexp.MustCompile(`^\d+\.`).MatchString(t) {
				continue
			}
			if strings.Contains(strings.ToLower(t), "conventional commit") && (strings.Contains(strings.ToLower(t), "generate") || strings.Contains(strings.ToLower(t), "steps")) {
				continue
			}
			bodyLines = append(bodyLines, t)
		}
	}
	body := strings.Join(bodyLines, "\n")
	body = wrapLines(body, opt.MaxBody)
	body = strings.TrimSpace(body)

	if body == "" {
		return title
	}
	return title + "\n\n" + body
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
	re := regexp.MustCompile(`^([a-z]+)(\([\w\-\./]+\))?:`)
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
	// Remove markdown tables
	lines := strings.Split(strings.TrimSpace(s), "\n")
	var out []string
	for _, ln := range lines {
		t := strings.TrimSpace(ln)
		if strings.HasPrefix(t, "|") {
			continue
		}
		out = append(out, ln)
	}
	return strings.Join(out, "\n")
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
