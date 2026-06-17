// Package dap implements a minimal Debug Adapter Protocol server for BPGo
// Pascal programs, driving the ir.Debugger engine. It supports launching a
// program, line breakpoints, continue / step, a single thread/stack frame and
// inspection of global variables — enough for editor debugging of .pas files.
package dap

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/arturoeanton/go-turbo-pascal/internal/codegen"
	"github.com/arturoeanton/go-turbo-pascal/internal/ir"
)

// Server speaks DAP over a stream.
type Server struct {
	in     *bufio.Reader
	out    io.Writer
	mu     sync.Mutex
	seq    int
	dbg    *ir.Debugger
	source string
}

// NewServer creates a DAP server reading from in and writing to out.
func NewServer(in io.Reader, out io.Writer) *Server {
	return &Server{in: bufio.NewReader(in), out: out}
}

type dapRequest struct {
	Seq       int             `json:"seq"`
	Type      string          `json:"type"`
	Command   string          `json:"command"`
	Arguments json.RawMessage `json:"arguments"`
}

// Run processes DAP messages until disconnect or EOF.
func (s *Server) Run() error {
	for {
		body, err := readMessage(s.in)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		var req dapRequest
		if err := json.Unmarshal(body, &req); err != nil {
			continue
		}
		if s.handle(req) {
			return nil
		}
	}
}

func (s *Server) handle(req dapRequest) bool {
	switch req.Command {
	case "initialize":
		s.respond(req, map[string]interface{}{"supportsConfigurationDoneRequest": true})
		s.event("initialized", nil)
	case "launch":
		s.launch(req)
	case "setBreakpoints":
		s.setBreakpoints(req)
	case "configurationDone":
		s.respond(req, nil)
		s.runUntilStop("breakpoint")
	case "threads":
		s.respond(req, map[string]interface{}{
			"threads": []map[string]interface{}{{"id": 1, "name": "main"}},
		})
	case "stackTrace":
		line := 0
		if s.dbg != nil {
			line = s.dbg.Line()
		}
		s.respond(req, map[string]interface{}{
			"stackFrames": []map[string]interface{}{{
				"id": 1, "name": "main", "line": line, "column": 1,
				"source": map[string]interface{}{"path": s.source},
			}},
			"totalFrames": 1,
		})
	case "scopes":
		s.respond(req, map[string]interface{}{
			"scopes": []map[string]interface{}{{
				"name": "Globals", "variablesReference": 1, "expensive": false,
			}},
		})
	case "variables":
		s.respond(req, map[string]interface{}{"variables": s.variables()})
	case "continue":
		s.respond(req, map[string]interface{}{"allThreadsContinued": true})
		s.runUntilStop("breakpoint")
	case "next", "stepIn", "stepOut":
		s.respond(req, nil)
		s.stepUntilStop()
	case "disconnect":
		s.respond(req, nil)
		return true
	default:
		s.respond(req, nil)
	}
	return false
}

func (s *Server) launch(req dapRequest) {
	var args struct {
		Program string `json:"program"`
	}
	json.Unmarshal(req.Arguments, &args)
	s.source = args.Program
	src, err := os.ReadFile(args.Program)
	if err != nil {
		s.respondErr(req, err.Error())
		return
	}
	prog, err := codegen.Compile(string(src), args.Program)
	if err != nil {
		s.respondErr(req, err.Error())
		return
	}
	vm := codegen.NewVM(prog, nil, "")
	s.dbg = ir.NewDebugger(vm)
	s.dbg.Start()
	s.respond(req, nil)
}

func (s *Server) setBreakpoints(req dapRequest) {
	var args struct {
		Breakpoints []struct {
			Line int `json:"line"`
		} `json:"breakpoints"`
	}
	json.Unmarshal(req.Arguments, &args)
	var lines []int
	var verified []map[string]interface{}
	for _, b := range args.Breakpoints {
		lines = append(lines, b.Line)
		verified = append(verified, map[string]interface{}{"verified": true, "line": b.Line})
	}
	if s.dbg != nil {
		s.dbg.SetBreakpoints(lines)
	}
	s.respond(req, map[string]interface{}{"breakpoints": verified})
}

func (s *Server) runUntilStop(reason string) {
	if s.dbg == nil {
		return
	}
	stopped, _ := s.dbg.Continue()
	if stopped {
		s.event("stopped", map[string]interface{}{
			"reason": reason, "threadId": 1, "allThreadsStopped": true,
		})
		return
	}
	s.terminate()
}

func (s *Server) stepUntilStop() {
	if s.dbg == nil {
		return
	}
	stopped, _ := s.dbg.StepLine()
	if stopped {
		s.event("stopped", map[string]interface{}{
			"reason": "step", "threadId": 1, "allThreadsStopped": true,
		})
		return
	}
	s.terminate()
}

func (s *Server) terminate() {
	if s.dbg != nil {
		if out := s.dbg.Output(); out != "" {
			s.event("output", map[string]interface{}{"category": "stdout", "output": out})
		}
	}
	s.event("terminated", nil)
}

func (s *Server) variables() []map[string]interface{} {
	if s.dbg == nil {
		return []map[string]interface{}{}
	}
	var out []map[string]interface{}
	for name, v := range s.dbg.Globals() {
		if strings.HasPrefix(name, "_") {
			continue // internal globals
		}
		out = append(out, map[string]interface{}{
			"name": name, "value": v.String(), "variablesReference": 0,
		})
	}
	return out
}

// --- protocol I/O ---

func (s *Server) respond(req dapRequest, body interface{}) {
	s.write(map[string]interface{}{
		"type": "response", "request_seq": req.Seq, "success": true,
		"command": req.Command, "body": body,
	})
}

func (s *Server) respondErr(req dapRequest, msg string) {
	s.write(map[string]interface{}{
		"type": "response", "request_seq": req.Seq, "success": false,
		"command": req.Command, "message": msg,
	})
}

func (s *Server) event(name string, body interface{}) {
	s.write(map[string]interface{}{"type": "event", "event": name, "body": body})
}

func (s *Server) write(msg map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	msg["seq"] = s.seq
	b, err := json.Marshal(msg)
	if err != nil {
		return
	}
	fmt.Fprintf(s.out, "Content-Length: %d\r\n\r\n%s", len(b), b)
}

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
		return nil, fmt.Errorf("dap: missing content-length")
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}
