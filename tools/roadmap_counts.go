//go:build ignore

// Recounts docs/ROADMAP.md: for each numbered section of the full
// checklist, how many items are done, partial, and missing, and rewrites
// the "Where things stand" table. Run from the repo root after ticking
// items:
//
//	go run tools/roadmap_counts.go
package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

const path = "docs/ROADMAP.md"

func main() {
	b, err := os.ReadFile(path)
	if err != nil {
		fail(err)
	}
	crlf := strings.Contains(string(b), "\r\n")
	text := strings.ReplaceAll(string(b), "\r\n", "\n")

	full := strings.Index(text, "\n## Full checklist")
	if full < 0 {
		fail(fmt.Errorf("no Full checklist heading"))
	}
	section := regexp.MustCompile(`^### (\d+\. .+)$`)
	type row struct {
		title         string
		done, part, n int
	}
	var rows []row
	for _, line := range strings.Split(text[full:], "\n") {
		if m := section.FindStringSubmatch(line); m != nil {
			rows = append(rows, row{title: m[1]})
			continue
		}
		if strings.HasPrefix(line, "## ") || len(rows) == 0 {
			if strings.HasPrefix(line, "## ") && !strings.HasPrefix(line, "## Full checklist") {
				break
			}
			continue
		}
		r := &rows[len(rows)-1]
		switch {
		case strings.HasPrefix(line, "- [x]"):
			r.done++
			r.n++
		case strings.HasPrefix(line, "- [ ]"):
			if strings.Contains(line, "**Partial:**") {
				r.part++
			}
			r.n++
		}
	}

	var out strings.Builder
	out.WriteString("| Area | Done | Partial | Missing | Coverage |\n| --- | --- | --- | --- | --- |\n")
	var d, p, n int
	anchor := regexp.MustCompile(`[^a-z0-9 -]`)
	for _, r := range rows {
		a := strings.ReplaceAll(anchor.ReplaceAllString(strings.ToLower(r.title), ""), " ", "-")
		fmt.Fprintf(&out, "| [%s](#%s) | %d | %d | %d | %d%% |\n", r.title, a, r.done, r.part, r.n-r.done-r.part, pct(r.done, r.part, r.n))
		d, p, n = d+r.done, p+r.part, n+r.n
	}
	fmt.Fprintf(&out, "| **Total** | **%d** | **%d** | **%d** | **%d%%** |\n", d, p, n-d-p, pct(d, p, n))

	start := strings.Index(text, "| Area | Done | Partial | Missing | Coverage |")
	if start < 0 {
		fail(fmt.Errorf("no summary table"))
	}
	end := strings.Index(text[start:], "\n\n")
	text = text[:start] + strings.TrimSuffix(out.String(), "\n") + text[start+end:]
	if crlf {
		text = strings.ReplaceAll(text, "\n", "\r\n")
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		fail(err)
	}
	fmt.Printf("%s: %d done, %d partial, %d missing (%d%%)\n", path, d, p, n-d-p, pct(d, p, n))
}

// pct counts a partial item as half.
func pct(done, part, n int) int {
	if n == 0 {
		return 0
	}
	return int(float64(done)*100/float64(n) + float64(part)*50/float64(n) + 0.5)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
