package interp

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"bitshinbasic/internal/ast"
)

// SetDebug pauses on the given source lines. in and out carry continue/step/print commands.
func (in *Interp) SetDebug(lines []int, inR io.Reader, out io.Writer) {
	in.debug = true
	in.breaks = map[int]bool{}
	for _, ln := range lines {
		if ln > 0 {
			in.breaks[ln] = true
		}
	}
	if inR != nil {
		in.debugIn = bufio.NewReader(inR)
	}
	if out == nil {
		out = io.Discard
	}
	in.debugOut = out
}

func (in *Interp) maybeBreak(s ast.Stmt) {
	if s == nil || in.debugOut == nil {
		return
	}
	line, _ := s.Pos()
	if line < 1 {
		return
	}
	if in.stepNext {
		in.stepNext = false
		in.debugPause(line)
		return
	}
	if in.breaks[line] {
		in.debugPause(line)
	}
}

func (in *Interp) debugPause(line int) {
	fmt.Fprintf(in.debugOut, "break %d\n", line)
	env := in.env()
	if env != nil {
		for name, v := range env.vars {
			fmt.Fprintf(in.debugOut, "  %s = %s\n", name, v.String())
		}
	}
	if in.debugIn == nil {
		return
	}
	for {
		fmt.Fprint(in.debugOut, "debug> ")
		text, err := in.debugIn.ReadString('\n')
		if err != nil {
			return
		}
		fields := strings.Fields(text)
		if len(fields) == 0 {
			continue
		}
		switch strings.ToLower(fields[0]) {
		case "c", "continue", "g", "go":
			return
		case "s", "step", "n":
			in.stepNext = true
			return
		case "p", "print", "w", "watch":
			if len(fields) < 2 {
				continue
			}
			if v, ok := in.env().get(fields[1]); ok {
				fmt.Fprintf(in.debugOut, "%s = %s\n", fields[1], v.String())
			} else {
				fmt.Fprintf(in.debugOut, "%s = 0\n", fields[1])
			}
		case "q", "quit":
			in.err = fmt.Errorf("stopped at line %d", line)
			in.status = StatusEnd
			in.stack = nil
			return
		default:
			fmt.Fprintln(in.debugOut, "commands: c continue, s step, p name, q quit")
		}
	}
}
