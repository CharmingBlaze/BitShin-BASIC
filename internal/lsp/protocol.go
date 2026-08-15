package lsp

import "encoding/json"

// JSON-RPC / LSP types for BitShin BASIC (stdio, Content-Length framing).

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id"`
	Result  any       `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcNullResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Result  any    `json:"result"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type initializeParams struct {
	RootURI          string            `json:"rootUri"`
	WorkspaceFolders []workspaceFolder `json:"workspaceFolders"`
}

type workspaceFolder struct {
	URI  string `json:"uri"`
	Name string `json:"name"`
}

type initializeResult struct {
	Capabilities serverCaps `json:"capabilities"`
	ServerInfo   serverInfo `json:"serverInfo"`
}

type serverCaps struct {
	TextDocumentSync       int            `json:"textDocumentSync"`
	HoverProvider          bool           `json:"hoverProvider"`
	CompletionProvider     completionOpts `json:"completionProvider"`
	DefinitionProvider     bool           `json:"definitionProvider"`
	DocumentSymbolProvider bool           `json:"documentSymbolProvider"`
}

type completionOpts struct {
	TriggerCharacters []string `json:"triggerCharacters"`
}

type serverInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type textDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

type versionedID struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
}

type didOpenParams struct {
	TextDocument textDocumentItem `json:"textDocument"`
}

type didChangeParams struct {
	TextDocument   versionedID     `json:"textDocument"`
	ContentChanges []contentChange `json:"contentChanges"`
}

type contentChange struct {
	Text string `json:"text"`
}

type didCloseParams struct {
	TextDocument struct {
		URI string `json:"uri"`
	} `json:"textDocument"`
}

type textDocPos struct {
	TextDocument struct {
		URI string `json:"uri"`
	} `json:"textDocument"`
	Position position `json:"position"`
}

type position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type lspRange struct {
	Start position `json:"start"`
	End   position `json:"end"`
}

type diagnostic struct {
	Range    lspRange `json:"range"`
	Severity int      `json:"severity"`
	Source   string   `json:"source"`
	Message  string   `json:"message"`
}

type publishDiagnostics struct {
	URI         string       `json:"uri"`
	Diagnostics []diagnostic `json:"diagnostics"`
}

type hoverResult struct {
	Contents markup `json:"contents"`
}

type markup struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

type completionList struct {
	IsIncomplete bool             `json:"isIncomplete"`
	Items        []completionItem `json:"items"`
}

type completionItem struct {
	Label         string `json:"label"`
	Kind          int    `json:"kind,omitempty"`
	Detail        string `json:"detail,omitempty"`
	Documentation string `json:"documentation,omitempty"`
}

type location struct {
	URI   string   `json:"uri"`
	Range lspRange `json:"range"`
}

type docSymbol struct {
	Name           string      `json:"name"`
	Kind           int         `json:"kind"`
	Range          lspRange    `json:"range"`
	SelectionRange lspRange    `json:"selectionRange"`
	Children       []docSymbol `json:"children,omitempty"`
}

const (
	severityError = 1
	syncFull      = 1
	kindFunction  = 3
	kindConstant  = 21
	kindKeyword   = 14
	kindMethod    = 2
	symFunction   = 12
	symClass      = 5
	symConstant   = 14
	symNamespace  = 3
	symMethod     = 6
	symEnum       = 10
)
