//go:build ignore

// Regenerates ide/frontend/src/lib/data/commands.json from docs/COMMANDS.md,
// so the IDE's completion, hover, and command search match the docs. Table
// rows give a command per code span in the first cell; elsewhere a code span
// counts only if it names a registered runtime command or a language
// builtin from internal/interp. Run from the repo root:
//
//	go run tools/gen_commands_json.go
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode"

	"moonbasic/internal/runtime"
)

type command struct {
	Syntax      string `json:"syntax"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

type category struct {
	Name     string    `json:"name"`
	Commands []command `json:"commands"`
}

type catalog struct {
	Categories []category `json:"categories"`
	Commands   []command  `json:"commands"`
}

func main() {
	f, err := os.Open("docs/COMMANDS.md")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()

	known := map[string]bool{}
	for _, n := range runtime.CommandNames() {
		known[strings.ToLower(n)] = true
	}
	// Builtins are registered as n("name", ...) in the interpreter.
	if src, err := os.ReadFile("internal/interp/interp.go"); err == nil {
		for _, m := range regexp.MustCompile(`\bn\("([a-z0-9_.]+)"`).FindAllStringSubmatch(string(src), -1) {
			known[m[1]] = true
		}
	}
	isKnown := func(name string) bool {
		return known[strings.ToLower(strings.TrimRight(name, "$#%"))]
	}

	var out catalog
	seen := map[string]bool{}
	cur := -1
	fence := false
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "```") {
			fence = !fence
			continue
		}
		if fence {
			continue
		}
		if strings.HasPrefix(line, "## ") {
			out.Categories = append(out.Categories, category{Name: strings.TrimSpace(line[3:])})
			cur = len(out.Categories) - 1
			continue
		}
		if cur < 0 {
			continue
		}
		var spans []string
		desc := ""
		table := false
		switch {
		case strings.HasPrefix(line, "| `"):
			// Table row: every command in the first cell, described by the second.
			cells := splitRow(line)
			if len(cells) < 2 {
				continue
			}
			spans, desc, table = codeSpans(cells[0]), cells[1], true
		case strings.HasPrefix(line, "`"):
			// Prose list: `A x` / `B y` — what they do.
			list, rest, found := strings.Cut(line, " — ")
			spans, desc = codeSpans(list), strings.TrimSpace(rest)
			if found {
				spans = append(spans, codeSpans(rest)...)
			}
		case strings.HasPrefix(line, "|"):
			continue
		default:
			spans = codeSpans(line)
		}
		for _, syntax := range spans {
			name := commandName(syntax)
			if name == "" || (!table && !isKnown(name)) {
				continue
			}
			c := command{Syntax: syntax, Name: name, Description: desc, Category: out.Categories[cur].Name}
			out.Categories[cur].Commands = append(out.Categories[cur].Commands, c)
			if !seen[strings.ToLower(name)] {
				seen[strings.ToLower(name)] = true
				out.Commands = append(out.Commands, c)
			}
		}
	}
	if err := sc.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	// Drop headings with no command table, such as the intro.
	kept := out.Categories[:0]
	for _, c := range out.Categories {
		if len(c.Commands) > 0 {
			kept = append(kept, c)
		}
	}
	out.Categories = kept

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	path := "ide/frontend/src/lib/data/commands.json"
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("%s: %d commands in %d categories\n", path, len(out.Commands), len(out.Categories))
}

// splitRow splits a markdown table row on pipes outside backticks, so
// `A \| B` and `a | b` inside code spans stay in one cell.
func splitRow(line string) []string {
	line = strings.TrimSuffix(strings.TrimPrefix(line, "|"), "|")
	var cells []string
	var cell strings.Builder
	inCode := false
	for i := 0; i < len(line); i++ {
		ch := line[i]
		switch {
		case ch == '\\' && i+1 < len(line) && line[i+1] == '|':
			cell.WriteByte('|')
			i++
		case ch == '`':
			inCode = !inCode
			cell.WriteByte(ch)
		case ch == '|' && !inCode:
			cells = append(cells, strings.TrimSpace(cell.String()))
			cell.Reset()
		default:
			cell.WriteByte(ch)
		}
	}
	cells = append(cells, strings.TrimSpace(cell.String()))
	return cells
}

// codeSpans returns the text inside each `code span` of s.
func codeSpans(s string) []string {
	var spans []string
	for {
		i := strings.IndexByte(s, '`')
		if i < 0 {
			return spans
		}
		j := strings.IndexByte(s[i+1:], '`')
		if j < 0 {
			return spans
		}
		spans = append(spans, strings.TrimSpace(s[i+1:i+1+j]))
		s = s[i+1+j+1:]
	}
}

// commandName is the leading identifier of the syntax: letters, digits,
// dots, and the BASIC type suffixes. Rows that start with anything else
// (a literal like [x,y,z] or $RRGGBB) are examples, not commands.
func commandName(syntax string) string {
	syntax = strings.TrimSpace(syntax)
	if syntax == "" || !unicode.IsLetter(rune(syntax[0])) {
		return ""
	}
	i := 0
	for i < len(syntax) {
		r := rune(syntax[i])
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '.' || r == '$' || r == '#' || r == '%' {
			i++
			continue
		}
		break
	}
	name := strings.TrimRight(syntax[:i], ".")
	if len(name) < 2 {
		return ""
	}
	return name
}
