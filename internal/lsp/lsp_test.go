package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestAnalyzeClean(t *testing.T) {
	diags := Analyze(`program P; begin WriteLn('hola'); end.`)
	if len(diags) != 0 {
		t.Fatalf("expected no diagnostics, got %v", diags)
	}
}

func TestAnalyzeSyntaxError(t *testing.T) {
	diags := Analyze(`program P begin end.`) // missing ';'
	if len(diags) == 0 {
		t.Fatal("expected a syntax diagnostic")
	}
	if diags[0].Line < 1 {
		t.Errorf("diagnostic should carry a line: %+v", diags[0])
	}
}

func TestAnalyzeUnknownIdentifier(t *testing.T) {
	diags := Analyze(`program P; begin nosuchvar := 1; end.`)
	if len(diags) == 0 {
		t.Fatal("expected a semantic diagnostic for an unknown identifier")
	}
}

func frame(msg string) string {
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(msg), msg)
}

func TestServerPublishesDiagnostics(t *testing.T) {
	var in bytes.Buffer
	in.WriteString(frame(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`))
	in.WriteString(frame(`{"jsonrpc":"2.0","method":"initialized","params":{}}`))
	in.WriteString(frame(`{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///t.pas","text":"program P begin end."}}}`))
	in.WriteString(frame(`{"jsonrpc":"2.0","method":"exit"}`))

	var out bytes.Buffer
	srv := NewServer(&in, &out)
	if err := srv.Run(); err != nil {
		t.Fatalf("server: %v", err)
	}

	msgs := readFrames(t, out.Bytes())
	sawInit := false
	sawDiag := false
	for _, m := range msgs {
		if strings.Contains(m, `"capabilities"`) {
			sawInit = true
		}
		if strings.Contains(m, "publishDiagnostics") {
			var note struct {
				Params struct {
					Diagnostics []json.RawMessage `json:"diagnostics"`
				} `json:"params"`
			}
			if err := json.Unmarshal([]byte(m), &note); err == nil && len(note.Params.Diagnostics) > 0 {
				sawDiag = true
			}
		}
	}
	if !sawInit {
		t.Error("expected an initialize response with capabilities")
	}
	if !sawDiag {
		t.Error("expected a publishDiagnostics notification with diagnostics")
	}
}

func readFrames(t *testing.T, data []byte) []string {
	t.Helper()
	r := bufio.NewReader(bytes.NewReader(data))
	var out []string
	for {
		b, err := readMessage(r)
		if err != nil {
			break
		}
		out = append(out, string(b))
	}
	return out
}
