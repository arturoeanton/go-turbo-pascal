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
				"textDocumentSync": 1, // full document sync
			},
			"serverInfo": map[string]interface{}{
				"name":    "bpgo-pls",
				"version": "0.1.0",
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
	}
	return false
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

func parseClose(raw json.RawMessage) string {
	var p struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	json.Unmarshal(raw, &p)
	return p.TextDocument.URI
}
