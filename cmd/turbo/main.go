// turbo is the BPGo IDE. It opens an interactive text-mode IDE
// inspired by Turbo Pascal 7. The IDE can be driven headlessly via
// the test harness or interactively when connected to a terminal.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/arturoeanton/go-turbo-pascal/internal/compile"
	"github.com/arturoeanton/go-turbo-pascal/internal/ide"
)

type stubCompiler struct{}

func (s *stubCompiler) Compile(src, output string) (string, error) {
	cfg := &compile.CompileConfig{Source: src, SourceFile: "main.pas", Output: output}
	_, err := compile.CompileToVM(cfg)
	if err != nil {
		return "", err
	}
	return "Compiled " + output, nil
}

type stubRunner struct{}

func (s *stubRunner) Run(exe string, args []string) (string, int, error) {
	cfg := &compile.CompileConfig{Source: "program T; begin end.", SourceFile: exe, Output: exe + ".bpi"}
	prog, err := compile.CompileToVM(cfg)
	if err != nil {
		return "", 1, err
	}
	out, code, err := compile.RunVM(prog, args)
	return out, code, err
}

type interactiveRunner struct {
	ide *ide.IDE
}

func (r *interactiveRunner) Run(exe string, args []string) (string, int, error) {
	b := r.ide.Buffers[r.ide.Active]
	cfg := &compile.CompileConfig{Source: b.Text(), SourceFile: b.Filename, Output: exe + ".bpi"}
	prog, err := compile.CompileToVM(cfg)
	if err != nil {
		return "", 1, err
	}
	return compile.RunVM(prog, args)
}

type stubDebugger struct{}

func (s *stubDebugger) SetBreakpoint(file string, line int) {}
func (s *stubDebugger) Step() (string, error)               { return "step", nil }
func (s *stubDebugger) Continue() (string, error)           { return "cont", nil }
func (s *stubDebugger) Watch(expr string) (string, error)   { return "0", nil }

func main() {
	headless := flag.Bool("headless", false, "run in headless mode for tests")
	screen := flag.Bool("screen", false, "render the visual IDE screen and exit")
	project := flag.String("project", "default", "project name")
	showVersion := flag.Bool("version", false, "show version")
	flag.BoolVar(showVersion, "V", false, "show version (alias)")
	flag.Parse()
	if *showVersion {
		fmt.Println("turbo 0.2.0 (BPGo IDE)")
		return
	}
	source := "main.pas"
	if flag.NArg() > 0 {
		source = flag.Arg(0)
	}
	proj := &ide.Project{Name: *project, Source: source, Output: defaultOutput(source)}
	ideInst := ide.New(proj, &stubCompiler{}, &stubRunner{}, &stubDebugger{})
	ideInst.Runner = &interactiveRunner{ide: ideInst}
	ideInst.Buffers[0].SetFilename(source)
	if data, err := os.ReadFile(source); err == nil {
		ideInst.Buffers[0].SetText(string(data))
	} else {
		ideInst.Buffers[0].SetText("program " + sanitizeProgramName(*project) + ";\nbegin\nend.\n")
	}
	if *screen {
		renderVisualDesktop(ideInst, "Ready")
		fmt.Print(ansiReset + "\n")
		return
	}
	if *headless {
		args := flag.Args()
		if len(args) > 0 {
			args = args[1:]
		}
		for _, arg := range args {
			if _, err := ideInst.RunCommand(arg); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}
		return
	}
	if isTerminal(os.Stdin) {
		runVisualIDE(ideInst)
		return
	}
	runScriptedInteractive(ideInst)
}

func runScriptedInteractive(i *ide.IDE) {
	renderDesktop(i)
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("turbo> ")
		if !scanner.Scan() {
			fmt.Println()
			return
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			_, _ = i.RunCommand("Exit")
			return
		}
		out, err := runLine(i, line)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		if out != "" {
			fmt.Println(out)
		}
	}
}

func runVisualIDE(i *ide.IDE) {
	oldState, rawOK := enterRawMode()
	if !rawOK {
		fmt.Fprintln(os.Stderr, "turbo: cannot enter terminal raw mode; mouse disabled. Try running from a real terminal or use --screen/--headless.")
		return
	}
	defer restoreTerminal(oldState)
	fmt.Print("\x1b[?25l")
	fmt.Print("\x1b[?1000h\x1b[?1006h")
	defer fmt.Print("\x1b[?1006l\x1b[?1000l\x1b[0m\x1b[?25h\x1b[2J\x1b[H")
	status := "F2 Save  F3 Open  F9 Compile  Ctrl-R Run  Ctrl-O Open  Ctrl-W Watch  Ctrl-X Exit"
	r := bufio.NewReader(os.Stdin)
	for {
		renderVisualDesktop(i, status)
		key, payload, err := readIDEKey(r)
		if err != nil {
			return
		}
		switch key {
		case "quit", "alt-x", "ctrl-x":
			_, _ = i.RunCommand("Exit")
			return
		case "f2", "ctrl-s", "alt-s":
			out, err := saveActiveFile(i)
			status = statusText(out, err)
		case "f3", "ctrl-o", "alt-o":
			name := visualPrompt("Open file: ")
			if name != "" {
				out, err := runLine(i, "open "+name)
				status = statusText(out, err)
			}
		case "f9", "alt-c":
			out, err := i.RunCommand("Compile")
			status = statusText(out, err)
		case "ctrl-f9", "ctrl-r", "alt-r":
			out, err := i.RunCommand("Run")
			status = statusText(out, err)
		case "f7", "f8":
			out, err := i.RunCommand("DebugStep")
			status = statusText(out, err)
		case "ctrl-f8", "ctrl-w", "alt-w":
			expr := visualPrompt("Watch expression: ")
			if expr != "" {
				out, err := i.RunCommand("Watch", expr)
				status = statusText(out, err)
			}
		case "up":
			i.Buffers[i.Active].MoveCursor(0, -1)
		case "down":
			i.Buffers[i.Active].MoveCursor(0, 1)
		case "left":
			i.Buffers[i.Active].MoveCursor(-1, 0)
		case "right":
			i.Buffers[i.Active].MoveCursor(1, 0)
		case "enter":
			i.Buffers[i.Active].InsertNewline()
		case "backspace":
			i.Buffers[i.Active].Backspace()
		case "delete":
			i.Buffers[i.Active].Delete()
		case "mouse":
			parts := strings.Fields(payload)
			if len(parts) == 4 {
				_, _ = i.RunCommand("Mouse", parts...)
				status = "Mouse event at " + parts[0] + "," + parts[1]
			}
		case "text":
			for i2 := 0; i2 < len(payload); i2++ {
				i.Buffers[i.Active].InsertChar(payload[i2])
			}
		}
	}
}

func renderDesktop(i *ide.IDE) {
	file := i.Project.Source
	if i.Buffers[i.Active].Filename != "" {
		file = i.Buffers[i.Active].Filename
	}
	fmt.Println("+------------------------------------------------------------------------------+")
	fmt.Println("| File  Edit  Search  Run  Compile  Debug  Tools  Options  Window  Help        |")
	fmt.Println("+------------------------------------------------------------------------------+")
	fmt.Printf("| BPGo Turbo IDE 0.2.0        Project: %-18s File: %-18s |\n", trimCell(i.Project.Name, 18), trimCell(file, 18))
	fmt.Println("|                                                                              |")
	fmt.Println("|  F3 Open  F2 Save  F9 Compile  Ctrl-F9 Run  F7 Trace  F8 Step  Ctrl-F8 Watch |")
	fmt.Println("|  Mouse: use 'mouse X Y BUTTON DOWN' for scripted mouse events.               |")
	fmt.Println("+------------------------------------------------------------------------------+")
	fmt.Println("Type help for TP7-compatible commands, exit to quit.")
}

const (
	ansiReset      = "\x1b[0m"
	ansiBlue       = "\x1b[44;37m"
	ansiGrayMenu   = "\x1b[47;30m"
	ansiCyanTitle  = "\x1b[46;30m"
	ansiWhitePanel = "\x1b[47;34m"
	ansiYellow     = "\x1b[44;33m"
)

func renderVisualDesktop(i *ide.IDE, status string) {
	width, height := terminalSize()
	if width < 60 {
		width = 60
	}
	if height < 18 {
		height = 18
	}
	b := i.Buffers[i.Active]
	file := b.Filename
	if file == "" {
		file = i.Project.Source
	}
	fmt.Print("\x1b[H")
	fmt.Print(ansiBlue)
	for row := 0; row < height; row++ {
		fmt.Print(strings.Repeat(" ", width))
		if row < height-1 {
			fmt.Print("\n")
		}
	}
	fmt.Print("\x1b[H")
	fmt.Print(ansiGrayMenu + padRight(" File  Edit  Search  Run  Compile  Debug  Tools  Options  Window  Help", width))
	fmt.Print(ansiBlue)
	writeAt(2, 2, ansiCyanTitle+centerText(" BPGo Turbo Pascal 7 IDE ", width-4)+ansiBlue)
	writeAt(4, 3, ansiYellow+"Project: "+trimCell(i.Project.Name, 16)+"   File: "+trimCell(file, 42)+ansiBlue)
	editHeight := height - 8
	if editHeight < 8 {
		editHeight = 8
	}
	drawBox(2, 4, width-4, editHeight, " Edit ")
	visibleLines := editHeight - 2
	for row := 0; row < visibleLines; row++ {
		lineIdx := b.TopLine + row
		text := ""
		if lineIdx < len(b.Lines) {
			text = b.Lines[lineIdx]
		}
		writeAt(4, 5+row, ansiBlue+padRight(trimCell(text, width-8), width-8)+ansiReset)
	}
	statusY := height - 3
	drawBox(2, statusY, width-4, 2, " Output/Status ")
	writeAt(4, statusY+1, ansiYellow+padRight(trimCell(status, width-8), width-8)+ansiBlue)
	writeAt(1, height, ansiGrayMenu+padRight(" F1 Help  F2 Save  F3 Open  F9 Compile  Ctrl-R Run  Ctrl-W Watch  Ctrl-X Exit ", width)+ansiReset)
	cursorY := 5 + b.CursorY - b.TopLine
	cursorX := 4 + b.CursorX
	if cursorY >= 5 && cursorY < statusY && cursorX < width-4 {
		fmt.Printf("\x1b[%d;%dH\x1b[?25h", cursorY, cursorX)
	}
}

func drawBox(x, y, w, h int, title string) {
	writeAt(x, y, ansiWhitePanel+"+"+strings.Repeat("-", w-2)+"+"+ansiBlue)
	if title != "" && len(title) < w-4 {
		writeAt(x+2, y, ansiWhitePanel+title+ansiBlue)
	}
	for row := 1; row < h-1; row++ {
		writeAt(x, y+row, ansiWhitePanel+"|"+ansiBlue+strings.Repeat(" ", w-2)+ansiWhitePanel+"|"+ansiBlue)
	}
	writeAt(x, y+h-1, ansiWhitePanel+"+"+strings.Repeat("-", w-2)+"+"+ansiBlue)
}

func writeAt(x, y int, s string) {
	fmt.Printf("\x1b[%d;%dH%s", y, x, s)
}

func centerText(s string, width int) string {
	if len(s) >= width {
		return trimCell(s, width)
	}
	left := (width - len(s)) / 2
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", width-left-len(s))
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

func runLine(i *ide.IDE, line string) (string, error) {
	cmd, rest, _ := strings.Cut(line, " ")
	switch strings.ToLower(cmd) {
	case "help", "?":
		return "commands: new, open FILE, project FILE, save [FILE], show, insert TEXT, line N, find TEXT, replace OLD NEW, compile/f9, run/ctrl-f9, break N, trace/f7, step/f8, watch EXPR, mouse X Y BUTTON DOWN, desktop, exit", nil
	case "desktop":
		renderDesktop(i)
		return "", nil
	case "new":
		return i.RunCommand("New")
	case "project":
		name := strings.TrimSpace(rest)
		if name == "" {
			return i.RunCommand("ProjectInfo")
		}
		source, output, err := loadProjectFile(name)
		if err != nil {
			return "", err
		}
		out, err := i.RunCommand("OpenProject", strings.TrimSuffix(filepath.Base(name), filepath.Ext(name)), source, output)
		if err != nil {
			return "", err
		}
		if source != "" {
			data, err := os.ReadFile(source)
			if err != nil {
				return "", err
			}
			_, err = i.RunCommand("Open", source, string(data))
			if err != nil {
				return "", err
			}
		}
		return "opened project " + out, nil
	case "open":
		name := strings.TrimSpace(rest)
		if name == "" {
			return "", fmt.Errorf("open requires a file")
		}
		data, err := os.ReadFile(name)
		if err != nil {
			return "", err
		}
		i.Project.Source = name
		i.Project.Output = defaultOutput(name)
		return i.RunCommand("Open", name, string(data))
	case "save":
		name := strings.TrimSpace(rest)
		if name == "" {
			name = i.Buffers[i.Active].Filename
		}
		if name == "" {
			name = i.Project.Source
		}
		text, err := i.RunCommand("SaveAs", name)
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(name, []byte(text), 0o644); err != nil {
			return "", err
		}
		i.Project.Source = name
		i.Project.Output = defaultOutput(name)
		return "saved " + name, nil
	case "show":
		return i.Buffers[i.Active].Text(), nil
	case "insert":
		i.Buffers[i.Active].InsertString(rest)
		return "", nil
	case "line":
		return i.RunCommand("GotoLine", strings.TrimSpace(rest))
	case "find":
		return i.RunCommand("Find", rest)
	case "replace":
		parts := strings.SplitN(rest, " ", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("replace requires OLD NEW")
		}
		return i.RunCommand("Replace", parts[0], parts[1])
	case "compile", "f9":
		return i.RunCommand("Compile")
	case "build":
		return i.RunCommand("Build")
	case "run", "ctrl-f9":
		args := strings.Fields(rest)
		return i.RunCommand("Run", args...)
	case "break":
		return i.RunCommand("SetBreakpoint", strings.TrimSpace(rest))
	case "trace", "f7", "step", "f8":
		return i.RunCommand("DebugStep")
	case "cont", "continue":
		return i.RunCommand("DebugContinue")
	case "watch", "ctrl-f8":
		return i.RunCommand("Watch", strings.TrimSpace(rest))
	case "mouse":
		parts := strings.Fields(rest)
		if len(parts) != 4 {
			return "", fmt.Errorf("mouse requires X Y BUTTON DOWN")
		}
		return i.RunCommand("Mouse", parts...)
	}
	return "", fmt.Errorf("unknown command %q", cmd)
}

func loadProjectFile(name string) (string, string, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return "", "", err
	}
	source := ""
	output := ""
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			if source == "" && strings.HasSuffix(strings.ToLower(line), ".pas") {
				source = line
			}
			continue
		}
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "source", "main", "file":
			source = strings.TrimSpace(value)
		case "output", "exe":
			output = strings.TrimSpace(value)
		}
	}
	if source != "" && !filepath.IsAbs(source) {
		source = filepath.Join(filepath.Dir(name), source)
	}
	if output == "" {
		output = defaultOutput(source)
	}
	return source, output, nil
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func enterRawMode() (string, bool) {
	out, err := exec.Command("sh", "-c", "stty -g < /dev/tty").Output()
	if err != nil {
		return "", false
	}
	old := strings.TrimSpace(string(out))
	if err := exec.Command("sh", "-c", "stty raw -echo -echoctl -ixon < /dev/tty").Run(); err != nil {
		if err := exec.Command("sh", "-c", "stty raw -echo -ixon < /dev/tty").Run(); err != nil {
			return "", false
		}
	}
	return old, true
}

func restoreTerminal(state string) {
	if state == "" {
		return
	}
	_ = exec.Command("sh", "-c", "stty "+state+" < /dev/tty").Run()
}

func terminalSize() (int, int) {
	cmd := exec.Command("sh", "-c", "stty size < /dev/tty")
	out, err := cmd.Output()
	if err != nil {
		return 80, 25
	}
	parts := strings.Fields(string(out))
	if len(parts) != 2 {
		return 80, 25
	}
	rows, err1 := strconv.Atoi(parts[0])
	cols, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || rows <= 0 || cols <= 0 {
		return 80, 25
	}
	return cols, rows
}

func readIDEKey(r *bufio.Reader) (string, string, error) {
	b, err := r.ReadByte()
	if err != nil {
		return "", "", err
	}
	switch b {
	case 3: // Ctrl-C
		return "quit", "", nil
	case 15: // Ctrl-O
		return "ctrl-o", "", nil
	case 18: // Ctrl-R
		return "ctrl-r", "", nil
	case 19: // Ctrl-S
		return "ctrl-s", "", nil
	case 23: // Ctrl-W
		return "ctrl-w", "", nil
	case 24: // Ctrl-X
		return "ctrl-x", "", nil
	case 13, 10:
		return "enter", "", nil
	case 8, 127:
		return "backspace", "", nil
	case 27:
		return readEscapeKey(r)
	}
	if b >= 32 && b <= 126 {
		return "text", string([]byte{b}), nil
	}
	return "", "", nil
}

func readEscapeKey(r *bufio.Reader) (string, string, error) {
	b, err := r.ReadByte()
	if err != nil {
		return "quit", "", nil
	}
	if b == 'O' {
		c, err := r.ReadByte()
		if err != nil {
			return "", "", err
		}
		switch c {
		case 'P':
			return "f1", "", nil
		case 'Q':
			return "f2", "", nil
		case 'R':
			return "f3", "", nil
		case 'S':
			return "f4", "", nil
		}
	}
	switch b {
	case 'x', 'X':
		return "alt-x", "", nil
	case 's', 'S':
		return "alt-s", "", nil
	case 'o':
		return "alt-o", "", nil
	case 'c', 'C':
		return "alt-c", "", nil
	case 'r', 'R':
		return "alt-r", "", nil
	case 'w', 'W':
		return "alt-w", "", nil
	}
	if b != '[' {
		return "", "", nil
	}
	seq := []byte{}
	for {
		c, err := r.ReadByte()
		if err != nil {
			return "", "", err
		}
		seq = append(seq, c)
		if (c >= 'A' && c <= 'Z') || c == '~' || c == 'M' || c == 'm' {
			break
		}
	}
	s := string(seq)
	if s == "M" {
		buf := make([]byte, 3)
		if _, err := io.ReadFull(r, buf); err != nil {
			return "", "", nil
		}
		button := int(buf[0]) - 32
		x := int(buf[1]) - 32
		y := int(buf[2]) - 32
		return "mouse", strconv.Itoa(x) + " " + strconv.Itoa(y) + " " + strconv.Itoa(button) + " 1", nil
	}
	switch s {
	case "A":
		return "up", "", nil
	case "B":
		return "down", "", nil
	case "C":
		return "right", "", nil
	case "D":
		return "left", "", nil
	case "3~":
		return "delete", "", nil
	case "11~":
		return "f1", "", nil
	case "12~":
		return "f2", "", nil
	case "13~":
		return "f3", "", nil
	case "18~":
		return "f7", "", nil
	case "19~":
		return "f8", "", nil
	case "19;5~":
		return "ctrl-f8", "", nil
	case "20~":
		return "f9", "", nil
	case "20;5~":
		return "ctrl-f9", "", nil
	}
	if strings.HasPrefix(s, "<") && (strings.HasSuffix(s, "M") || strings.HasSuffix(s, "m")) {
		payload := strings.TrimRight(strings.TrimPrefix(s, "<"), "Mm")
		parts := strings.Split(payload, ";")
		if len(parts) == 3 {
			button := parts[0]
			x := parts[1]
			y := parts[2]
			pressed := "1"
			if strings.HasSuffix(s, "m") {
				pressed = "0"
			}
			return "mouse", x + " " + y + " " + button + " " + pressed, nil
		}
	}
	return "", "", nil
}

func visualPrompt(label string) string {
	width, height := terminalSize()
	fmt.Print("\x1b[" + strconv.Itoa(height-1) + ";1H" + ansiGrayMenu + padRight(label, width) + "\x1b[" + strconv.Itoa(height-1) + ";" + strconv.Itoa(len(label)+1) + "H" + ansiReset)
	_ = exec.Command("sh", "-c", "stty sane < /dev/tty").Run()
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	_, _ = enterRawMode()
	return strings.TrimSpace(line)
}

func saveActiveFile(i *ide.IDE) (string, error) {
	name := i.Buffers[i.Active].Filename
	if name == "" {
		name = i.Project.Source
	}
	if name == "" {
		name = visualPrompt("Save as: ")
	}
	if name == "" {
		return "", fmt.Errorf("no filename")
	}
	text, err := i.RunCommand("SaveAs", name)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(name, []byte(text), 0o644); err != nil {
		return "", err
	}
	return "Saved " + name, nil
}

func statusText(out string, err error) string {
	if err != nil {
		return "Error: " + err.Error()
	}
	if out == "" {
		return "Ready"
	}
	return out
}

func defaultOutput(source string) string {
	base := strings.TrimSuffix(source, filepath.Ext(source))
	if base == "" {
		base = "main"
	}
	return base + ".exe"
}

func sanitizeProgramName(name string) string {
	if name == "" {
		return "T"
	}
	var b strings.Builder
	for _, r := range name {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "T"
	}
	return b.String()
}

func trimCell(s string, width int) string {
	if len(s) <= width {
		return s
	}
	if width <= 3 {
		return s[:width]
	}
	return s[:width-3] + "..."
}
