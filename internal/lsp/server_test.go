package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
)

func writeRPC(t *testing.T, w io.Writer, payload string) {
	t.Helper()
	_, err := fmt.Fprintf(w, "Content-Length: %d\r\n\r\n%s", len(payload), payload)
	if err != nil {
		t.Fatal(err)
	}
}

func readRPC(t *testing.T, r *bufio.Reader) map[string]any {
	t.Helper()
	body, err := readFrame(r)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]any{}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestServeInitializeDiagnosticsShutdown(t *testing.T) {
	clientR, serverW := io.Pipe()
	serverR, clientW := io.Pipe()
	done := make(chan error, 1)
	go func() {
		done <- Serve(serverR, serverW)
	}()
	br := bufio.NewReader(clientR)

	writeRPC(t, clientW, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"rootUri":""}}`)
	initMsg := readRPC(t, br)
	if initMsg["id"] == nil {
		t.Fatalf("initialize: %+v", initMsg)
	}
	caps, _ := initMsg["result"].(map[string]any)
	if caps == nil {
		t.Fatalf("no result: %+v", initMsg)
	}
	sc, _ := caps["capabilities"].(map[string]any)
	if sc["hoverProvider"] != true {
		t.Fatalf("capabilities: %+v", sc)
	}

	writeRPC(t, clientW, `{"jsonrpc":"2.0","method":"initialized","params":{}}`)
	writeRPC(t, clientW, `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///t.bb","languageId":"bitshinbasic","version":1,"text":"Function\n"}}}`)

	diag := map[string]any{}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		msg := readRPC(t, br)
		if msg["method"] == "textDocument/publishDiagnostics" {
			diag = msg
			break
		}
	}
	if diag["method"] != "textDocument/publishDiagnostics" {
		t.Fatal("expected diagnostics")
	}
	params, _ := diag["params"].(map[string]any)
	list, _ := params["diagnostics"].([]any)
	if len(list) < 1 {
		t.Fatalf("expected parse error diagnostics, got %+v", params)
	}

	writeRPC(t, clientW, `{"jsonrpc":"2.0","id":2,"method":"shutdown"}`)
	sh := readRPC(t, br)
	if _, ok := sh["result"]; !ok {
		t.Fatalf("shutdown: %+v", sh)
	}
	writeRPC(t, clientW, `{"jsonrpc":"2.0","method":"exit"}`)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not exit")
	}
	_ = clientW.Close()
	_ = serverW.Close()
}

func TestParseDiagnosticsAndSymbols(t *testing.T) {
	bad := parseDiagnostics("Function\n")
	if len(bad) != 1 {
		t.Fatalf("want 1 diagnostic, got %#v", bad)
	}
	if bad[0].Message == "" {
		t.Fatalf("empty message: %#v", bad[0])
	}

	ok := parseDiagnostics("Print 1\n")
	if len(ok) != 0 {
		t.Fatalf("clean program: %#v", ok)
	}

	src := "Function Add(a, b)\n    Return a + b\nEndFunction\nPrint Add(1, 2)\n"
	doc := &document{uri: "file:///x.bb", text: src}
	syms := documentSymbols(doc)
	if len(syms) != 1 || strings.ToLower(syms[0].Name) != "add" {
		t.Fatalf("symbols: %#v", syms)
	}
	locs := definitionAt(doc, position{Line: 3, Character: 6})
	if len(locs) != 1 {
		t.Fatalf("definition: %#v", locs)
	}
	c := newCatalog()
	h := hoverFor(c, doc, position{Line: 0, Character: 0})
	if h == nil || !strings.Contains(h.Contents.Value, "Function") {
		t.Fatalf("hover function: %#v", h)
	}
	list := completions(c, doc, position{Line: 3, Character: 0})
	if len(list.Items) < 10 {
		t.Fatalf("completions too few: %d", len(list.Items))
	}
}

func TestSplitParseErr(t *testing.T) {
	line, col, msg := splitParseErr("line 4:2: expected statement, got ident")
	if line != 4 || col != 2 || msg != "expected statement, got ident" {
		t.Fatalf("%d:%d %q", line, col, msg)
	}
}
