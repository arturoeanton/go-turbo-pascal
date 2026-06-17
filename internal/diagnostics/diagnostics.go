// Package diagnostics defines the central diagnostic catalog used by
// every component of BPGo. Diagnostics are stable: codes and messages
// must not change because conformance tests reference them.
package diagnostics

import (
	"fmt"
	"sort"
	"strings"
)

type Severity int

const (
	SevError Severity = iota
	SevWarning
	SevInfo
)

func (s Severity) String() string {
	switch s {
	case SevError:
		return "Error"
	case SevWarning:
		return "Warning"
	default:
		return "Info"
	}
}

type Category int

const (
	CatCompile Category = iota
	CatRuntime
	CatIO
	CatGraph
	CatOverlay
	CatDebug
	CatIDE
)

func (c Category) String() string {
	return [...]string{"Compile", "Runtime", "I/O", "Graph", "Overlay", "Debug", "IDE"}[c]
}

type Diagnostic struct {
	Code     int
	Category Category
	Severity Severity
	Name     string
	Message  string
}

type Entry struct {
	Diagnostic
	Help string
}

var catalog = map[string]Entry{}

func Register(d Diagnostic, help string) {
	k := fmt.Sprintf("%d:%d", d.Category, d.Code)
	catalog[k] = Entry{Diagnostic: d, Help: help}
}

func Get(category Category, code int) (Entry, bool) {
	k := fmt.Sprintf("%d:%d", category, code)
	e, ok := catalog[k]
	return e, ok
}

func Format(category Category, code int, file string, line, col int) string {
	e, ok := Get(category, code)
	if !ok {
		return fmt.Sprintf("%s %d: unknown diagnostic (file %s line %d col %d)",
			category, code, file, line, col)
	}
	loc := ""
	if file != "" {
		loc = fmt.Sprintf(" (%s, line %d, col %d)", file, line, col)
	}
	return fmt.Sprintf("%s %d: %s%s", e.Category, e.Code, e.Message, loc)
}

func init() {
	compileCodes := []struct {
		code int
		name string
		msg  string
		help string
	}{
		{1, "Unexpected symbol", "Unexpected symbol", "Check syntax; missing operator or separator."},
		{2, "Identifier expected", "Identifier expected", "A reserved word was used where an identifier is required."},
		{3, "Unknown identifier", "Unknown identifier", "The name is not declared in the current scope."},
		{4, "Duplicate identifier", "Duplicate identifier", "The identifier is already declared in this scope."},
		{5, "Syntax error", "Syntax error", "Token does not match the grammar."},
		{6, "Error in real constant", "Error in real constant", "Real literal is malformed."},
		{7, "Error in integer constant", "Error in integer constant", "Integer literal is malformed or out of range."},
		{8, "String constant exceeds line", "String constant exceeds line", "Use string concatenation with '+'."},
		{9, "Unterminated string", "Unterminated string", "Add a closing single quote."},
		{10, "Expected quote", "Expected closing quote", "Add a closing single quote."},
		{11, "Expected =", "Expected '='", "Add '='."},
		{12, "Expected :=", "Expected ':='", "Use ':=' for assignment."},
		{13, "Type identifier expected", "Type identifier expected", "Provide a type name."},
		{14, "Expected of", "Expected OF", "Add OF (e.g. array, case, set, file)."},
		{15, "Expected .", "Expected '.'", "Add '.'."},
		{16, "Too many nested procedures", "Too many nested procedures", "Reduce nesting or use units."},
		{17, "Bad type", "Bad type", "Type is not valid in this context."},
		{18, "Expected END", "Expected END", "Close the open block with END."},
		{26, "Type mismatch", "Type mismatch", "The expression type does not match the destination."},
		{27, "Invalid subrange base type", "Invalid subrange base type", "Subrange base must be an ordinal."},
		{28, "Lower bound > upper bound", "Lower bound > upper bound", "Swap bounds."},
		{29, "Ordinal expected", "Ordinal expected", "Provide an ordinal type."},
		{30, "Integer constant expected", "Integer constant expected", "Provide an integer constant."},
		{31, "Constant expected", "Constant expected", "Provide a constant expression."},
		{32, "Integer or real constant expected", "Integer or real constant expected", "Provide a numeric constant."},
		{33, "Pointer type identifier expected", "Pointer type identifier expected", "Use '^T'."},
		{34, "Invalid function result type", "Invalid function result type", "Use a scalar, pointer or string type."},
		{35, "Label identifier expected", "Label identifier expected", "Provide a numeric label."},
		{36, "BEGIN expected", "BEGIN expected", "Start the block with BEGIN."},
		{37, "Statement part too large", "Statement part too large", "Split into smaller procedures."},
		{38, "Expected DO", "Expected DO", "Add DO."},
		{39, "Expected THEN", "Expected THEN", "Add THEN."},
		{40, "Too many variables", "Too many variables", "Reduce variable count or split unit."},
		{41, "Undefined type", "Undefined type", "Declare the type."},
		{42, "File not allowed here", "File not allowed here", "Files have restrictions in this context."},
		{43, "String length mismatch", "String length mismatch", "Source and destination strings differ in declared length."},
		{44, "String constant expected", "String constant expected", "Use a string literal."},
		{45, "Integer or real variable expected", "Integer or real variable expected", "Provide a numeric variable."},
		{46, "Ordinal variable expected", "Ordinal variable expected", "Provide an ordinal variable."},
		{47, "Character expression expected", "Character expression expected", "Provide a Char-compatible expression."},
		{48, "Structured variable expected", "Structured variable expected", "Provide a record/array/file."},
		{49, "Constant expression expected", "Constant expression expected", "Use a constant."},
		{50, "Integer expression expected", "Integer expression expected", "Use an integer expression."},
		{51, "Boolean expression expected", "Boolean expression expected", "Use a boolean expression."},
		{52, "Operand types do not match", "Operand types do not match", "Operator is not defined for these types."},
		{53, "Field identifier expected", "Field identifier expected", "Use a record field name."},
		{54, "Object file too large", "Object file too large", "Reduce code or split the unit."},
		{55, "Undefined external", "Undefined external", "Provide the external symbol or library."},
		{56, "Invalid object file record", "Invalid object file record", "OMF record is not supported."},
		{57, "Code segment too large", "Code segment too large", "Code cannot exceed 64KB without overlays."},
		{58, "Data segment too large", "Data segment too large", "Data cannot exceed 64KB."},
		{84, "Unit name mismatch", "Unit name mismatch", "Unit identifier does not match filename."},
		{85, "Unit version mismatch", "Unit version mismatch", "Recompile the unit."},
		{86, "Duplicate unit name", "Duplicate unit name", "A unit appears twice in uses."},
		{87, "Unit cycle detected", "Unit cycle error", "Remove the circular uses."},
		{88, "Unit not found", "Unit not found", "Add the unit path or create the unit."},
	}
	for _, c := range compileCodes {
		Register(Diagnostic{Code: c.code, Category: CatCompile, Severity: SevError, Name: c.name, Message: c.msg}, c.help)
	}
	runtimeCodes := []struct {
		code int
		name string
		msg  string
		help string
	}{
		{1, "Invalid function number", "Invalid function number", "Use a supported DOS function."},
		{2, "File not found", "File not found", "Check path and filename."},
		{3, "Path not found", "Path not found", "Check directory path."},
		{4, "Too many open files", "Too many open files", "Close some files or raise CONFIG FILES."},
		{5, "File access denied", "File access denied", "Check permissions or readonly."},
		{6, "Invalid file handle", "Invalid file handle", "File is not open."},
		{12, "Invalid file access code", "Invalid file access code", "Use Reset/Rewrite/Append."},
		{15, "Invalid drive number", "Invalid drive number", "Use 0..26 for drives."},
		{16, "Cannot remove current directory", "Cannot remove current directory", "Change to another directory first."},
		{17, "Not same device", "Not same device", "Rename/Move across drives not allowed."},
		{18, "No more files", "No more files", "End of FindFirst/FindNext search."},
		{100, "Disk read error", "Disk read error", "Check the disk."},
		{101, "Disk write error", "Disk write error", "Disk is full or write protected."},
		{102, "File not assigned", "File not assigned", "Call Assign first."},
		{103, "File not open", "File not open", "Open the file before I/O."},
		{104, "File not open for input", "File not open for input", "Use Reset."},
		{105, "File not open for output", "File not open for output", "Use Rewrite/Append."},
		{106, "Invalid numeric format", "Invalid numeric format", "Input is not a valid number."},
		{150, "Division by zero", "Division by zero", "Add a zero check."},
		{151, "Range check error", "Range check error", "Enable {$R-} or fix the value."},
		{152, "Stack overflow", "Stack overflow", "Reduce recursion depth."},
		{153, "Heap overflow", "Heap overflow", "Free memory or increase heap."},
		{154, "Invalid pointer operation", "Invalid pointer operation", "Pointer is nil or freed."},
		{155, "Floating point overflow", "Floating point overflow", "Value too large for Real."},
		{156, "Floating point division by zero", "Floating point division by zero", "Check the divisor."},
		{157, "Invalid floating point operation", "Invalid floating point operation", "Sqrt/Ln/ArcTan of invalid input."},
		{158, "Floating point underflow", "Floating point underflow", "Result too small for Real."},
		{159, "Integer overflow", "Integer overflow", "Use LongInt or a wider type."},
		{160, "Invalid variant operation", "Invalid variant operation", "Variant holds wrong type."},
		{161, "Invalid variant typecast", "Invalid variant typecast", "Cast is not valid for the stored type."},
		{162, "Dispatch error", "Dispatch error", "Method not found in dispatch table."},
		{200, "Division by zero (delay loop bug)", "Division by zero", "Replaced by runtime stub."},
		{201, "Range check", "Range check error", "{$R+} is on."},
		{202, "Stack overflow", "Stack overflow", "{$S+} is on."},
		{203, "Heap overflow", "Heap overflow", "Heap is exhausted."},
		{204, "Invalid pointer", "Invalid pointer operation", "Pointer is invalid."},
		{205, "Floating point overflow", "Floating point overflow", "{$N+}/{$E+} is on."},
		{206, "Floating point underflow", "Floating point underflow", "{$N+}/{$E+} is on."},
		{207, "Invalid 8087 opcode", "Invalid 8087 opcode", "Disassemble and check."},
	}
	for _, c := range runtimeCodes {
		Register(Diagnostic{Code: c.code, Category: CatRuntime, Severity: SevError, Name: c.name, Message: c.msg}, c.help)
	}
	ioCodes := []struct {
		code int
		name string
		msg  string
		help string
	}{
		{2, "File not found", "File not found", "Check path and filename."},
		{3, "Path not found", "Path not found", "Check directory path."},
		{5, "Access denied", "File access denied", "Check permissions or readonly."},
		{32, "Sharing violation", "Sharing violation", "File is locked by another process."},
		{100, "Disk read error", "Disk read error", "Check the disk."},
		{101, "Disk write error", "Disk write error", "Disk is full or write protected."},
	}
	for _, c := range ioCodes {
		Register(Diagnostic{Code: c.code, Category: CatIO, Severity: SevError, Name: c.name, Message: c.msg}, c.help)
	}
	graphCodes := []struct {
		code int
		name string
		msg  string
		help string
	}{
		{0, "No error", "No error", "All is well."},
		{-1, "Graphics error", "Graphics error", "Generic."},
		{-2, "InitGraph error", "InitGraph error", "Check driver and path."},
		{-3, "No font", "No font", "Install a font."},
		{-4, "No load mem", "Not enough memory to load driver", "Free memory."},
		{-5, "No scan mem", "Not enough memory to scan fill", "Free memory."},
		{-6, "No flood mem", "Not enough memory to flood fill", "Free memory."},
		{-7, "Image too large", "Image too large for buffer", "Reduce viewport."},
		{-8, "Invalid font", "Invalid font", "Use a valid BGI font."},
		{-9, "Invalid driver", "Invalid driver", "Use a valid BGI driver."},
		{-10, "Invalid mode", "Invalid mode", "Use a valid mode for the driver."},
		{-11, "Invalid fill", "Invalid fill", "Use a known fill style."},
		{-12, "Palette index out of range", "Palette index out of range", "Use 0..GetMaxColor."},
		{-13, "Invalid image", "Invalid image buffer", "GetImage then PutImage."},
		{-14, "Invalid linestyle", "Invalid line style", "Use a known line style."},
		{-15, "Out of memory", "Out of memory", "Free memory."},
		{-16, "Out of viewport", "Out of viewport", "Coordinates are outside the viewport."},
		{-17, "Invalid viewport", "Invalid viewport", "Use a non-empty viewport."},
	}
	for _, c := range graphCodes {
		Register(Diagnostic{Code: c.code, Category: CatGraph, Severity: SevError, Name: c.name, Message: c.msg}, c.help)
	}
	ovlCodes := []struct {
		code int
		name string
		msg  string
		help string
	}{
		{0, "OK", "Overlay loaded", ""},
		{-1, "OVR error", "Overlay error", "Generic."},
		{-2, "OVR file not found", "Overlay file not found", "Check OVR filename."},
		{-3, "OVR out of memory", "Out of memory", "Increase overlay buffer."},
		{-4, "OVR read error", "Overlay read error", "Check disk."},
	}
	for _, c := range ovlCodes {
		Register(Diagnostic{Code: c.code, Category: CatOverlay, Severity: SevError, Name: c.name, Message: c.msg}, c.help)
	}
	dbgCodes := []struct {
		code int
		name string
		msg  string
		help string
	}{
		{1, "No source", "No source for address", "Compile with {$D+}."},
		{2, "Invalid breakpoint", "Invalid breakpoint", "Set a valid line breakpoint."},
		{3, "Symbol not found", "Symbol not found", "Check name and scope."},
		{4, "Process not running", "Process not running", "Run the program first."},
	}
	for _, c := range dbgCodes {
		Register(Diagnostic{Code: c.code, Category: CatDebug, Severity: SevError, Name: c.name, Message: c.msg}, c.help)
	}
	ideCodes := []struct {
		code int
		name string
		msg  string
		help string
	}{
		{1, "File not found", "File not found", "Check the file path."},
		{2, "Save failed", "Save failed", "Check disk space and permissions."},
		{3, "Build failed", "Build failed", "See compile output."},
		{4, "Run failed", "Run failed", "Check executable and parameters."},
	}
	for _, c := range ideCodes {
		Register(Diagnostic{Code: c.code, Category: CatIDE, Severity: SevError, Name: c.name, Message: c.msg}, c.help)
	}
}

// Codes returns the list of registered codes for a category. Mainly used by
// the conformance harness to verify all expected codes are present.
func Codes(category Category) []int {
	seen := map[int]bool{}
	for k := range catalog {
		var c Category
		if _, err := fmt.Sscanf(k, "%d:", &c); err == nil {
			if c == category {
				var code int
				fmt.Sscanf(k, "%d:%d", &c, &code)
				seen[code] = true
			}
		}
	}
	out := make([]int, 0, len(seen))
	for c := range seen {
		out = append(out, c)
	}
	sort.Ints(out)
	return out
}

// List returns all registered diagnostics sorted by category and code.
func List() []Entry {
	keys := make([]string, 0, len(catalog))
	for k := range catalog {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]Entry, 0, len(keys))
	for _, k := range keys {
		out = append(out, catalog[k])
	}
	return out
}

// Search is a simple case-insensitive substring search for help.
func Search(query string) []Entry {
	q := strings.ToUpper(query)
	var out []Entry
	for _, e := range catalog {
		if strings.Contains(strings.ToUpper(e.Name), q) || strings.Contains(strings.ToUpper(e.Message), q) {
			out = append(out, e)
		}
	}
	return out
}
