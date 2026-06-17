package dap

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func frame(msg string) string {
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(msg), msg)
}

func TestDebugSession(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "d.pas")
	src := `program D;
var x: Integer;
begin
  x := 1;
  x := 2;
  WriteLn(x);
end.`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	var in bytes.Buffer
	in.WriteString(frame(`{"seq":1,"type":"request","command":"initialize","arguments":{}}`))
	in.WriteString(frame(fmt.Sprintf(`{"seq":2,"type":"request","command":"launch","arguments":{"program":%q}}`, path)))
	in.WriteString(frame(`{"seq":3,"type":"request","command":"setBreakpoints","arguments":{"breakpoints":[{"line":5}]}}`))
	in.WriteString(frame(`{"seq":4,"type":"request","command":"configurationDone","arguments":{}}`))
	in.WriteString(frame(`{"seq":5,"type":"request","command":"continue","arguments":{}}`))
	in.WriteString(frame(`{"seq":6,"type":"request","command":"disconnect","arguments":{}}`))

	var out bytes.Buffer
	if err := NewServer(&in, &out).Run(); err != nil {
		t.Fatalf("run: %v", err)
	}

	msgs := readFrames(out.Bytes())
	var sawInitialized, sawStopped, sawTerminated, sawOutput bool
	for _, m := range msgs {
		var ev struct {
			Type  string `json:"type"`
			Event string `json:"event"`
			Body  struct {
				Reason string `json:"reason"`
				Output string `json:"output"`
			} `json:"body"`
		}
		if json.Unmarshal([]byte(m), &ev); ev.Type != "event" {
			continue
		}
		switch ev.Event {
		case "initialized":
			sawInitialized = true
		case "stopped":
			if ev.Body.Reason == "breakpoint" {
				sawStopped = true
			}
		case "terminated":
			sawTerminated = true
		case "output":
			if strings.Contains(ev.Body.Output, "2") {
				sawOutput = true
			}
		}
	}
	if !sawInitialized {
		t.Error("missing initialized event")
	}
	if !sawStopped {
		t.Error("missing stopped(breakpoint) event")
	}
	if !sawOutput {
		t.Error("missing output event with program output")
	}
	if !sawTerminated {
		t.Error("missing terminated event")
	}
}

func readFrames(data []byte) []string {
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
