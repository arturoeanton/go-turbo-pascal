package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
)

// Server is a minimal LSP server speaking JSON-RPC 2.0 over a stream.
type Server struct {
	in   *bufio.Reader
	out  io.Writer
	mu   sync.Mutex
	docs map[string]string
}

// NewServer creates a server reading requests from in and writing to out.
func NewServer(in io.Reader, out io.Writer) *Server {
	return &Server{in: bufio.NewReader(in), out: out, docs: map[string]string{}}
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
}

type rpcNotification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

type position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type rng struct {
	Start position `json:"start"`
	End   position `json:"end"`
}

type lspDiagnostic struct {
	Range    rng    `json:"range"`
	Severity int    `json:"severity"`
	Source   string `json:"source"`
	Message  string `json:"message"`
}

// Run processes messages until the client exits or the stream closes.
func (s *Server) Run() error {
	for {
		body, err := readMessage(s.in)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		var req rpcRequest
		if err := json.Unmarshal(body, &req); err != nil {
			continue
		}
		if stop := s.handle(req); stop {
			return nil
		}
	}
}

// handle dispatches a single message, returning true to stop the server.
func (s *Server) handle(req rpcRequest) bool {
	switch req.Method {
	case "initialize":
		s.respond(req.ID, map[string]interface{}{
			"capabilities": map[string]interface{}{
				"textDocumentSync":       1, // full document sync
				"hoverProvider":          true,
				"definitionProvider":     true,
				"documentSymbolProvider": true,
				"completionProvider":     map[string]interface{}{"triggerCharacters": []string{"."}},
			},
			"serverInfo": map[string]interface{}{
				"name":    "bpgo-pls",
				"version": "0.1.3",
			},
		})
	case "initialized":
		// no-op
	case "shutdown":
		s.respond(req.ID, nil)
	case "exit":
		return true
	case "textDocument/didOpen":
		uri, text := parseOpen(req.Params)
		if uri != "" {
			s.setDoc(uri, text)
			s.publish(uri, text)
		}
	case "textDocument/didChange":
		uri, text := parseChange(req.Params)
		if uri != "" {
			s.setDoc(uri, text)
			s.publish(uri, text)
		}
	case "textDocument/didClose":
		uri := parseClose(req.Params)
		if uri != "" {
			s.delDoc(uri)
			s.publishDiags(uri, nil)
		}
	case "textDocument/documentSymbol":
		uri := parseDocPos(req.Params).uri
		s.respond(req.ID, s.documentSymbols(uri))
	case "textDocument/hover":
		p := parseDocPos(req.Params)
		s.respond(req.ID, s.hover(p.uri, p.line, p.character))
	case "textDocument/definition":
		p := parseDocPos(req.Params)
		s.respond(req.ID, s.definition(p.uri, p.line, p.character))
	case "textDocument/completion":
		uri := parseDocPos(req.Params).uri
		s.respond(req.ID, s.completion(uri))
	}
	return false
}

// docOf returns the current text of a document.
func (s *Server) docOf(uri string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.docs[uri]
}

// documentSymbols answers textDocument/documentSymbol.
func (s *Server) documentSymbols(uri string) []map[string]interface{} {
	out := []map[string]interface{}{}
	for _, sym := range Symbols(s.docOf(uri)) {
		out = append(out, map[string]interface{}{
			"name":           sym.Name,
			"detail":         sym.Detail,
			"kind":           sym.Kind,
			"range":          symRange(sym),
			"selectionRange": symRange(sym),
		})
	}
	return out
}

// hover answers textDocument/hover with the declaration of the symbol under
// the cursor.
func (s *Server) hover(uri string, line, character int) interface{} {
	src := s.docOf(uri)
	word := wordAt(src, line, character)
	if word == "" {
		return nil
	}
	if sym, ok := findSymbol(src, word); ok {
		return map[string]interface{}{
			"contents": map[string]interface{}{
				"kind":  "markdown",
				"value": "```pascal\n" + sym.Detail + "\n```",
			},
		}
	}
	return nil
}

// definition answers textDocument/definition.
func (s *Server) definition(uri string, line, character int) interface{} {
	src := s.docOf(uri)
	word := wordAt(src, line, character)
	if word == "" {
		return nil
	}
	if sym, ok := findSymbol(src, word); ok {
		return map[string]interface{}{
			"uri":   uri,
			"range": symRange(sym),
		}
	}
	return nil
}

// completion answers textDocument/completion with document symbols plus the
// Pascal keywords.
func (s *Server) completion(uri string) map[string]interface{} {
	items := []map[string]interface{}{}
	seen := map[string]bool{}
	for _, sym := range Symbols(s.docOf(uri)) {
		if seen[sym.Name] {
			continue
		}
		seen[sym.Name] = true
		items = append(items, map[string]interface{}{
			"label":  sym.Name,
			"kind":   completionKind(sym.Kind),
			"detail": sym.Detail,
		})
	}
	for _, kw := range pascalKeywords {
		items = append(items, map[string]interface{}{"label": kw, "kind": symKeyword})
	}
	return map[string]interface{}{"isIncomplete": false, "items": items}
}

// findSymbol returns the declared symbol matching name (case-insensitive).
func findSymbol(src, name string) (Symbol, bool) {
	low := strings.ToLower(name)
	for _, sym := range Symbols(src) {
		if strings.ToLower(sym.Name) == low {
			return sym, true
		}
	}
	return Symbol{}, false
}

// symRange builds a one-token LSP range from a symbol's 1-based position.
func symRange(sym Symbol) map[string]interface{} {
	line := sym.Line - 1
	if line < 0 {
		line = 0
	}
	col := sym.Col - 1
	if col < 0 {
		col = 0
	}
	return map[string]interface{}{
		"start": map[string]int{"line": line, "character": col},
		"end":   map[string]int{"line": line, "character": col + len(sym.Name)},
	}
}

// completionKind maps an LSP SymbolKind to a CompletionItemKind.
func completionKind(symbolKind int) int {
	switch symbolKind {
	case symFunction:
		return 3 // Function
	case symVariable:
		return 6 // Variable
	case symConstant:
		return 21 // Constant
	case symClass:
		return 7 // Class
	case symEnum:
		return 13 // Enum
	case symModule:
		return 9 // Module
	}
	return 1 // Text
}

// pascalKeywords are offered as completion items.
var pascalKeywords = []string{
	"begin", "end", "program", "unit", "uses", "interface", "implementation",
	"procedure", "function", "const", "type", "var", "record", "array", "set",
	"of", "string", "integer", "real", "boolean", "char", "if", "then", "else",
	"case", "for", "to", "downto", "do", "while", "repeat", "until", "with",
	"class", "object", "constructor", "destructor", "virtual", "property",
	"read", "write", "try", "except", "finally", "raise", "nil", "true", "false",
	"and", "or", "not", "div", "mod", "in",
}

func (s *Server) setDoc(uri, text string) {
	s.mu.Lock()
	s.docs[uri] = text
	s.mu.Unlock()
}

func (s *Server) delDoc(uri string) {
	s.mu.Lock()
	delete(s.docs, uri)
	s.mu.Unlock()
}

// publish computes and sends diagnostics for a document.
func (s *Server) publish(uri, text string) {
	var diags []lspDiagnostic
	for _, d := range Analyze(text) {
		line := d.Line - 1
		if line < 0 {
			line = 0
		}
		col := d.Col - 1
		if col < 0 {
			col = 0
		}
		diags = append(diags, lspDiagnostic{
			Range:    rng{Start: position{line, col}, End: position{line, col + 1}},
			Severity: 1, // Error
			Source:   "bpgo",
			Message:  d.Message,
		})
	}
	s.publishDiags(uri, diags)
}

func (s *Server) publishDiags(uri string, diags []lspDiagnostic) {
	if diags == nil {
		diags = []lspDiagnostic{}
	}
	s.notify("textDocument/publishDiagnostics", map[string]interface{}{
		"uri":         uri,
		"diagnostics": diags,
	})
}

func (s *Server) respond(id json.RawMessage, result interface{}) {
	s.write(rpcResponse{JSONRPC: "2.0", ID: id, Result: result})
}

func (s *Server) notify(method string, params interface{}) {
	s.write(rpcNotification{JSONRPC: "2.0", Method: method, Params: params})
}

func (s *Server) write(v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	fmt.Fprintf(s.out, "Content-Length: %d\r\n\r\n%s", len(b), b)
}

// readMessage reads one LSP-framed JSON-RPC message.
func readMessage(r *bufio.Reader) ([]byte, error) {
	length := 0
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if strings.HasPrefix(strings.ToLower(line), "content-length:") {
			fmt.Sscanf(strings.TrimSpace(line[len("content-length:"):]), "%d", &length)
		}
	}
	if length <= 0 {
		return nil, fmt.Errorf("lsp: missing content-length")
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

// --- param parsing ---

func parseOpen(raw json.RawMessage) (string, string) {
	var p struct {
		TextDocument struct {
			URI  string `json:"uri"`
			Text string `json:"text"`
		} `json:"textDocument"`
	}
	json.Unmarshal(raw, &p)
	return p.TextDocument.URI, p.TextDocument.Text
}

func parseChange(raw json.RawMessage) (string, string) {
	var p struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		ContentChanges []struct {
			Text string `json:"text"`
		} `json:"contentChanges"`
	}
	json.Unmarshal(raw, &p)
	if len(p.ContentChanges) == 0 {
		return p.TextDocument.URI, ""
	}
	// Full sync: the last change carries the whole document.
	return p.TextDocument.URI, p.ContentChanges[len(p.ContentChanges)-1].Text
}

// docPos is a document URI plus an optional 0-based cursor position.
type docPos struct {
	uri       string
	line      int
	character int
}

func parseDocPos(raw json.RawMessage) docPos {
	var p struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Position struct {
			Line      int `json:"line"`
			Character int `json:"character"`
		} `json:"position"`
	}
	json.Unmarshal(raw, &p)
	return docPos{uri: p.TextDocument.URI, line: p.Position.Line, character: p.Position.Character}
}

func parseClose(raw json.RawMessage) string {
	var p struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	json.Unmarshal(raw, &p)
	return p.TextDocument.URI
}
