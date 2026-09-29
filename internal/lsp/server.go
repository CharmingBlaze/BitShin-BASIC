package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

const version = "1.0.0"

type server struct {
	out  io.Writer
	mu   sync.Mutex
	docs map[string]*document
	cat  *catalog
	exit bool
}

// Serve runs a stdio JSON-RPC language server until an `exit` notification.
func Serve(in io.Reader, out io.Writer) error {
	s := &server{
		out:  out,
		docs: map[string]*document{},
		cat:  newCatalog(),
	}
	r := bufio.NewReader(in)
	for !s.exit {
		body, err := readFrame(r)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := s.handle(body); err != nil {
			return err
		}
	}
	return nil
}

func readFrame(r *bufio.Reader) ([]byte, error) {
	n := 0
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		low := strings.ToLower(line)
		if strings.HasPrefix(low, "content-length:") {
			v := strings.TrimSpace(line[len("Content-Length:"):])
			n, err = strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("lsp: bad Content-Length")
			}
		}
	}
	if n <= 0 {
		return nil, fmt.Errorf("lsp: missing Content-Length")
	}
	body := make([]byte, n)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, err
	}
	return body, nil
}

func (s *server) write(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err = fmt.Fprintf(s.out, "Content-Length: %d\r\n\r\n%s", len(b), b)
	return err
}

func (s *server) reply(id any, result any) error {
	if result == nil {
		return s.write(rpcNullResponse{JSONRPC: "2.0", ID: id, Result: nil})
	}
	return s.write(rpcResponse{JSONRPC: "2.0", ID: id, Result: result})
}

func (s *server) replyErr(id any, code int, msg string) error {
	return s.write(rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: msg}})
}

func (s *server) notify(method string, params any) error {
	return s.write(map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	})
}

func (s *server) handle(body []byte) error {
	msg := rpcRequest{}
	if err := json.Unmarshal(body, &msg); err != nil {
		return err
	}
	switch msg.Method {
	case "initialize":
		p := initializeParams{}
		_ = json.Unmarshal(msg.Params, &p)
		root := uriToPath(p.RootURI)
		if root == "" && len(p.WorkspaceFolders) > 0 {
			root = uriToPath(p.WorkspaceFolders[0].URI)
		}
		if root == "" {
			wd, err := os.Getwd()
			if err == nil {
				root = wd
			}
		}
		s.cat.loadRoot(root)
		return s.reply(msg.ID, initializeResult{
			Capabilities: serverCaps{
				TextDocumentSync:       syncFull,
				HoverProvider:          true,
				CompletionProvider:     completionOpts{TriggerCharacters: []string{"."}},
				DefinitionProvider:     true,
				DocumentSymbolProvider: true,
				RenameProvider:         true,
			},
			ServerInfo: serverInfo{Name: "bsls", Version: version},
		})
	case "initialized", "textDocument/didSave":
		return nil
	case "shutdown":
		return s.reply(msg.ID, nil)
	case "exit":
		s.exit = true
		return nil
	case "textDocument/didOpen":
		p := didOpenParams{}
		if err := json.Unmarshal(msg.Params, &p); err != nil {
			return nil
		}
		s.put(p.TextDocument.URI, p.TextDocument.Text, p.TextDocument.Version)
		return s.publish(p.TextDocument.URI)
	case "textDocument/didChange":
		p := didChangeParams{}
		if err := json.Unmarshal(msg.Params, &p); err != nil {
			return nil
		}
		if len(p.ContentChanges) == 0 {
			return nil
		}
		s.put(p.TextDocument.URI, p.ContentChanges[len(p.ContentChanges)-1].Text, p.TextDocument.Version)
		return s.publish(p.TextDocument.URI)
	case "textDocument/didClose":
		p := didCloseParams{}
		_ = json.Unmarshal(msg.Params, &p)
		s.mu.Lock()
		delete(s.docs, p.TextDocument.URI)
		s.mu.Unlock()
		return s.notify("textDocument/publishDiagnostics", publishDiagnostics{URI: p.TextDocument.URI, Diagnostics: []diagnostic{}})
	case "textDocument/hover":
		p := textDocPos{}
		_ = json.Unmarshal(msg.Params, &p)
		doc := s.get(p.TextDocument.URI)
		if doc == nil {
			return s.reply(msg.ID, nil)
		}
		return s.reply(msg.ID, hoverFor(s.cat, doc, p.Position))
	case "textDocument/completion":
		p := textDocPos{}
		_ = json.Unmarshal(msg.Params, &p)
		doc := s.get(p.TextDocument.URI)
		if doc == nil {
			return s.reply(msg.ID, completionList{})
		}
		return s.reply(msg.ID, completions(s.cat, doc, p.Position))
	case "textDocument/definition":
		p := textDocPos{}
		_ = json.Unmarshal(msg.Params, &p)
		doc := s.get(p.TextDocument.URI)
		if doc == nil {
			return s.reply(msg.ID, []location{})
		}
		return s.reply(msg.ID, definitionAcross(s.allDocs(), doc, p.Position))
	case "textDocument/rename":
		p := renameParams{}
		_ = json.Unmarshal(msg.Params, &p)
		doc := s.get(p.TextDocument.URI)
		if doc == nil {
			return s.reply(msg.ID, nil)
		}
		return s.reply(msg.ID, renameFunc(s.allDocs(), doc, p.Position, p.NewName))
	case "textDocument/documentSymbol":
		p := didCloseParams{}
		_ = json.Unmarshal(msg.Params, &p)
		doc := s.get(p.TextDocument.URI)
		if doc == nil {
			return s.reply(msg.ID, []docSymbol{})
		}
		return s.reply(msg.ID, documentSymbols(doc))
	default:
		if msg.ID != nil {
			return s.replyErr(msg.ID, -32601, "method not found: "+msg.Method)
		}
		return nil
	}
}

func (s *server) put(uri, text string, version int) {
	s.mu.Lock()
	s.docs[uri] = &document{uri: uri, text: text, version: version}
	s.mu.Unlock()
}

func (s *server) get(uri string) *document {
	s.mu.Lock()
	d := s.docs[uri]
	s.mu.Unlock()
	return d
}

func (s *server) publish(uri string) error {
	doc := s.get(uri)
	if doc == nil {
		return nil
	}
	diags := parseDiagnostics(doc.text)
	if len(diags) == 0 {
		diags = semanticDiags(s.cat, doc)
	}
	return s.notify("textDocument/publishDiagnostics", publishDiagnostics{
		URI:         uri,
		Diagnostics: diags,
	})
}

func (s *server) allDocs() []*document {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*document, 0, len(s.docs))
	for _, d := range s.docs {
		out = append(out, d)
	}
	return out
}

func uriToPath(uri string) string {
	if uri == "" {
		return ""
	}
	u, err := url.Parse(uri)
	if err != nil {
		return uri
	}
	p := u.Path
	if runtime.GOOS == "windows" && strings.HasPrefix(p, "/") && len(p) >= 3 && p[2] == ':' {
		p = strings.TrimPrefix(p, "/")
	}
	if u.Host != "" && runtime.GOOS == "windows" {
		p = u.Host + "/" + strings.TrimPrefix(p, "/")
	}
	return filepath.FromSlash(p)
}
