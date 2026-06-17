// Package lexer implements the Turbo Pascal 7 / Borland Pascal 7 source
// tokenizer. It is intentionally independent of the parser: tokens carry
// position information, kind, literal text and (where applicable) a
// canonicalized value. The lexer is case-insensitive for keywords and
// identifiers but always preserves the original spelling for diagnostics
// and source maps.
package lexer

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type TokenKind int

const (
	TokEOF TokenKind = iota
	TokIdent
	TokInt
	TokReal
	TokHex
	TokString
	TokKeyword
	TokDirective
	TokOp
	TokComma
	TokSemicolon
	TokColon
	TokPeriod
	TokRange
	TokLParen
	TokRParen
	TokLBracket
	TokRBracket
	TokCaret
	TokAt
	TokAssign
	TokEqual
	TokComment
	TokError
)

func (k TokenKind) String() string {
	switch k {
	case TokEOF:
		return "EOF"
	case TokIdent:
		return "ident"
	case TokInt:
		return "int"
	case TokReal:
		return "real"
	case TokHex:
		return "hex"
	case TokString:
		return "string"
	case TokKeyword:
		return "kw"
	case TokDirective:
		return "directive"
	case TokOp:
		return "op"
	case TokComma:
		return ","
	case TokSemicolon:
		return ";"
	case TokColon:
		return ":"
	case TokPeriod:
		return "."
	case TokRange:
		return ".."
	case TokLParen:
		return "("
	case TokRParen:
		return ")"
	case TokLBracket:
		return "["
	case TokRBracket:
		return "]"
	case TokCaret:
		return "^"
	case TokAt:
		return "@"
	case TokAssign:
		return ":="
	case TokEqual:
		return "="
	case TokComment:
		return "comment"
	case TokError:
		return "error"
	}
	return "?"
}

type Token struct {
	Kind  TokenKind
	Text  string
	Lower string
	Line  int
	Col   int
	Int   int64
	Real  float64
	Hex   uint64
	Str   string
}

// Keywords and reserved words of TP7/BP7 (strict mode). Order matters for
// the lookup table only; comparison is case-insensitive via Lower.
var keywords = map[string]string{
	"and":            "AND",
	"array":          "ARRAY",
	"asm":            "ASM",
	"begin":          "BEGIN",
	"case":           "CASE",
	"class":          "CLASS",
	"property":       "PROPERTY",
	"protected":      "PROTECTED",
	"published":      "PUBLISHED",
	"const":          "CONST",
	"constructor":    "CONSTRUCTOR",
	"destructor":     "DESTRUCTOR",
	"div":            "DIV",
	"do":             "DO",
	"downto":         "DOWNTO",
	"try":            "TRY",
	"except":         "EXCEPT",
	"finally":        "FINALLY",
	"raise":          "RAISE",
	"else":           "ELSE",
	"end":            "END",
	"external":       "EXTERNAL",
	"far":            "FAR",
	"file":           "FILE",
	"for":            "FOR",
	"forward":        "FORWARD",
	"function":       "FUNCTION",
	"goto":           "GOTO",
	"halt":           "HALT",
	"if":             "IF",
	"implementation": "IMPLEMENTATION",
	"in":             "IN",
	"inherited":      "INHERITED",
	"initialization": "INITIALIZATION",
	"finalization":   "FINALIZATION",
	"inline":         "INLINE",
	"interface":      "INTERFACE",
	"interrupt":      "INTERRUPT",
	"label":          "LABEL",
	"mod":            "MOD",
	"near":           "NEAR",
	"nil":            "NIL",
	"not":            "NOT",
	"object":         "OBJECT",
	"of":             "OF",
	"or":             "OR",
	"override":       "OVERRIDE",
	"abstract":       "ABSTRACT",
	"static":         "STATIC",
	"private":        "PRIVATE",
	"public":         "PUBLIC",
	"virtual":        "VIRTUAL",
	"packed":         "PACKED",
	"procedure":      "PROCEDURE",
	"program":        "PROGRAM",
	"record":         "RECORD",
	"repeat":         "REPEAT",
	"set":            "SET",
	"shl":            "SHL",
	"shr":            "SHR",
	"string":         "STRING",
	"then":           "THEN",
	"to":             "TO",
	"type":           "TYPE",
	"unit":           "UNIT",
	"until":          "UNTIL",
	"uses":           "USES",
	"var":            "VAR",
	"while":          "WHILE",
	"with":           "WITH",
	"xor":            "XOR",
}

var directives = map[string]bool{
	"A": true, "B": true, "D": true, "E": true, "F": true, "G": true,
	"I": true, "L": true, "M": true, "N": true, "O": true, "P": true,
	"Q": true, "R": true, "S": true, "T": true, "V": true, "X": true,
	"DEFINE": true, "UNDEF": true, "IFDEF": true, "IFNDEF": true,
	"IFOPT": true, "ELSE": true, "ENDIF": true,
}

type Lexer struct {
	src      []byte
	pos      int
	line     int
	col      int
	tokens   []Token
	errors   []string
	curLine  int
	curCol   int
	inString bool
}

func New(src string) *Lexer {
	l := &Lexer{src: []byte(src), line: 1, col: 1}
	l.run()
	return l
}

func NewBytes(src []byte) *Lexer {
	l := &Lexer{src: src, line: 1, col: 1}
	l.run()
	return l
}

func (l *Lexer) Tokens() []Token  { return l.tokens }
func (l *Lexer) Errors() []string { return l.errors }

func (l *Lexer) run() {
	for l.pos < len(l.src) {
		if l.skipWhitespace() {
			continue
		}
		if l.peek() == '{' && l.pos+1 < len(l.src) && l.src[l.pos+1] == '$' {
			l.scanDirective()
			continue
		}
		if l.peek() == '(' && l.pos+1 < len(l.src) && l.src[l.pos+1] == '*' {
			l.scanParenComment()
			continue
		}
		if l.peek() == '{' {
			l.scanBraceComment()
			continue
		}
		if l.peek() == '\'' {
			l.scanString()
			continue
		}
		// '#' control char reference (only legal adjacent to a string literal).
		if l.peek() == '#' && l.lastTokenWasString() {
			l.scanHashRef()
			continue
		}
		c := l.peek()
		if isIdentStart(c) {
			l.scanIdentOrKeyword()
			continue
		}
		if isDigit(c) || (c == '$' && l.pos+1 < len(l.src) && isHexDigit(l.src[l.pos+1])) {
			l.scanNumber()
			continue
		}
		l.scanPunct()
	}
	l.tokens = append(l.tokens, Token{Kind: TokEOF, Line: l.line, Col: l.col})
}

func (l *Lexer) lastTokenWasString() bool {
	if len(l.tokens) == 0 {
		return false
	}
	return l.tokens[len(l.tokens)-1].Kind == TokString
}

func (l *Lexer) scanHashRef() {
	line, col := l.mark()
	l.advance() // '#'
	num := 0
	had := false
	for l.pos < len(l.src) && isDigit(l.src[l.pos]) {
		num = num*10 + int(l.src[l.pos]-'0')
		l.advance()
		had = true
	}
	if !had {
		l.errf("expected digits after '#'")
		return
	}
	// Emit a TokInt representing the control char so the parser/IR layer
	// can fold it into the previous string.
	l.emit(Token{Kind: TokInt, Text: "#" + itoa(num), Int: int64(num), Line: line, Col: col})
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

func (l *Lexer) peek() byte {
	if l.pos >= len(l.src) {
		return 0
	}
	return l.src[l.pos]
}

func (l *Lexer) advance() byte {
	if l.pos >= len(l.src) {
		return 0
	}
	c := l.src[l.pos]
	l.pos++
	if c == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	return c
}

func (l *Lexer) skipWhitespace() bool {
	any := false
	for l.pos < len(l.src) {
		c := l.src[l.pos]
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			l.advance()
			any = true
			continue
		}
		break
	}
	return any
}

func (l *Lexer) mark() (int, int) {
	return l.line, l.col
}

func (l *Lexer) emit(t Token) {
	l.tokens = append(l.tokens, t)
}

func (l *Lexer) errf(format string, args ...any) {
	line, col := l.line, l.col
	l.errors = append(l.errors, fmt.Sprintf("line %d col %d: "+format, append([]any{line, col}, args...)...))
}

func (l *Lexer) scanBraceComment() {
	start := l.pos
	l.advance() // '{'
	for l.pos < len(l.src) {
		if l.src[l.pos] == '}' {
			l.advance()
			_ = string(l.src[start:l.pos])
			return
		}
		l.advance()
	}
	l.errf("unterminated {{ ... }} comment")
}

func (l *Lexer) scanParenComment() {
	start := l.pos
	l.advance() // '('
	l.advance() // '*'
	for l.pos < len(l.src) {
		if l.src[l.pos] == '*' && l.pos+1 < len(l.src) && l.src[l.pos+1] == ')' {
			l.advance()
			l.advance()
			_ = string(l.src[start:l.pos])
			return
		}
		l.advance()
	}
	l.errf("unterminated (* ... *) comment")
}

func (l *Lexer) scanDirective() {
	start := l.pos
	l.advance() // '{'
	l.advance() // '$'
	for l.pos < len(l.src) {
		if l.src[l.pos] == '}' {
			l.advance()
			_ = string(l.src[start:l.pos])
			return
		}
		l.advance()
	}
	l.errf("unterminated {{$...}} directive")
}

func extractDirectiveBody(s string) string {
	// Trim outermost markers
	if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
		s = s[1 : len(s)-1]
	} else if strings.HasPrefix(s, "(*") && strings.HasSuffix(s, "*)") {
		s = s[2 : len(s)-2]
	}
	s = strings.TrimPrefix(s, "$")
	return strings.TrimSpace(s)
}

func (l *Lexer) scanString() {
	line, col := l.mark()
	start := l.pos
	l.advance() // opening quote
	var sb strings.Builder
	for l.pos < len(l.src) {
		c := l.src[l.pos]
		if c == '\'' {
			if l.pos+1 < len(l.src) && l.src[l.pos+1] == '\'' {
				sb.WriteByte('\'')
				l.advance()
				l.advance()
				continue
			}
			l.advance()
			l.emit(Token{Kind: TokString, Text: string(l.src[start:l.pos]), Str: sb.String(), Line: line, Col: col})
			return
		}
		if c == '#' {
			// control char reference: read decimal digits.
			l.advance()
			num := 0
			had := false
			for l.pos < len(l.src) && l.src[l.pos] >= '0' && l.src[l.pos] <= '9' {
				num = num*10 + int(l.src[l.pos]-'0')
				l.advance()
				had = true
			}
			if !had {
				l.errf("expected digits after '#' in string")
				continue
			}
			if num > 255 {
				l.errf("character code %d out of range in string", num)
			}
			sb.WriteByte(byte(num))
			continue
		}
		sb.WriteByte(c)
		l.advance()
	}
	l.errf("unterminated string literal")
	l.emit(Token{Kind: TokError, Text: string(l.src[start:l.pos]), Line: line, Col: col})
}

func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func isIdentCont(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isHexDigit(c byte) bool {
	return isDigit(c) || (c >= 'A' && c <= 'F') || (c >= 'a' && c <= 'f')
}

func (l *Lexer) scanIdentOrKeyword() {
	line, col := l.mark()
	start := l.pos
	for l.pos < len(l.src) && isIdentCont(l.src[l.pos]) {
		l.advance()
	}
	text := string(l.src[start:l.pos])
	lower := strings.ToLower(text)
	// "do" and "to" can be used as identifiers in TP7 (e.g. method "Do").
	// Treat them as identifiers if they are not lowercase to keep method
	// names valid while still allowing "do"/"to" as keywords when lowercase.
	if (lower == "do" || lower == "to") && text != lower {
		l.emit(Token{Kind: TokIdent, Text: text, Lower: lower, Line: line, Col: col})
		return
	}
	if _, ok := keywords[lower]; ok {
		l.emit(Token{Kind: TokKeyword, Text: text, Lower: lower, Line: line, Col: col})
		return
	}
	l.emit(Token{Kind: TokIdent, Text: text, Lower: lower, Line: line, Col: col})
}

func (l *Lexer) scanNumber() {
	line, col := l.mark()
	start := l.pos
	if l.peek() == '$' {
		l.advance()
		var v uint64
		for l.pos < len(l.src) && isHexDigit(l.src[l.pos]) {
			d := hexVal(l.src[l.pos])
			v = v*16 + d
			l.advance()
		}
		l.emit(Token{Kind: TokHex, Text: string(l.src[start:l.pos]), Hex: v, Line: line, Col: col})
		return
	}
	for l.pos < len(l.src) && isDigit(l.src[l.pos]) {
		l.advance()
	}
	isReal := false
	if l.pos < len(l.src) && l.src[l.pos] == '.' {
		// Look ahead to disambiguate range operator: only consume '.' if next
		// char is a digit (real literal).
		if l.pos+1 < len(l.src) && isDigit(l.src[l.pos+1]) {
			isReal = true
			l.advance()
			for l.pos < len(l.src) && isDigit(l.src[l.pos]) {
				l.advance()
			}
		}
	}
	if l.pos < len(l.src) && (l.src[l.pos] == 'e' || l.src[l.pos] == 'E') {
		isReal = true
		l.advance()
		if l.pos < len(l.src) && (l.src[l.pos] == '+' || l.src[l.pos] == '-') {
			l.advance()
		}
		for l.pos < len(l.src) && isDigit(l.src[l.pos]) {
			l.advance()
		}
	}
	text := string(l.src[start:l.pos])
	if isReal {
		f, err := parseReal(text)
		if err != nil {
			l.errf("invalid real literal %q", text)
		}
		l.emit(Token{Kind: TokReal, Text: text, Real: f, Line: line, Col: col})
		return
	}
	v := int64(0)
	for _, c := range text {
		v = v*10 + int64(c-'0')
	}
	l.emit(Token{Kind: TokInt, Text: text, Int: v, Line: line, Col: col})
}

func hexVal(c byte) uint64 {
	switch {
	case c >= '0' && c <= '9':
		return uint64(c - '0')
	case c >= 'A' && c <= 'F':
		return uint64(c - 'A' + 10)
	case c >= 'a' && c <= 'f':
		return uint64(c - 'a' + 10)
	}
	return 0
}

func parseReal(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%g", &f)
	return f, err
}

func (l *Lexer) scanPunct() {
	line, col := l.mark()
	c := l.advance()
	switch c {
	case ',':
		l.emit(Token{Kind: TokComma, Text: ",", Line: line, Col: col})
	case ';':
		l.emit(Token{Kind: TokSemicolon, Text: ";", Line: line, Col: col})
	case ':':
		if l.peek() == '=' {
			l.advance()
			l.emit(Token{Kind: TokAssign, Text: ":=", Line: line, Col: col})
		} else {
			l.emit(Token{Kind: TokColon, Text: ":", Line: line, Col: col})
		}
	case '.':
		if l.peek() == '.' {
			l.advance()
			l.emit(Token{Kind: TokRange, Text: "..", Line: line, Col: col})
		} else {
			l.emit(Token{Kind: TokPeriod, Text: ".", Line: line, Col: col})
		}
	case '(':
		l.emit(Token{Kind: TokLParen, Text: "(", Line: line, Col: col})
	case ')':
		l.emit(Token{Kind: TokRParen, Text: ")", Line: line, Col: col})
	case '[':
		l.emit(Token{Kind: TokLBracket, Text: "[", Line: line, Col: col})
	case ']':
		l.emit(Token{Kind: TokRBracket, Text: "]", Line: line, Col: col})
	case '^':
		l.emit(Token{Kind: TokCaret, Text: "^", Line: line, Col: col})
	case '@':
		l.emit(Token{Kind: TokAt, Text: "@", Line: line, Col: col})
	case '+', '-', '*', '/', '=', '<', '>', '!':
		if c == '<' && l.peek() == '=' {
			l.advance()
			l.emit(Token{Kind: TokOp, Text: "<=", Line: line, Col: col})
			return
		}
		if c == '<' && l.peek() == '>' {
			l.advance()
			l.emit(Token{Kind: TokOp, Text: "<>", Line: line, Col: col})
			return
		}
		if c == '>' && l.peek() == '=' {
			l.advance()
			l.emit(Token{Kind: TokOp, Text: ">=", Line: line, Col: col})
			return
		}
		if c == '=' && l.peek() == '=' {
			l.advance()
			l.emit(Token{Kind: TokOp, Text: "==", Line: line, Col: col})
			return
		}
		if c == '=' {
			l.emit(Token{Kind: TokEqual, Text: "=", Line: line, Col: col})
			return
		}
		l.emit(Token{Kind: TokOp, Text: string(c), Line: line, Col: col})
	default:
		// Unknown character: emit as op so the parser can complain with a
		// proper diagnostic instead of panicking.
		l.errf("unexpected character %q", rune(c))
		l.emit(Token{Kind: TokError, Text: string(c), Line: line, Col: col})
	}
}

// Helpers used by tests and other packages.

func IsKeyword(s string) bool {
	_, ok := keywords[strings.ToLower(s)]
	return ok
}

func IsDirective(name string) bool {
	return directives[strings.ToUpper(name)]
}

func KeywordName(s string) string {
	if k, ok := keywords[strings.ToLower(s)]; ok {
		return k
	}
	return ""
}

// RuneCount is a small helper kept here so external packages do not need
// unicode/utf8 just to count characters in a string.
func RuneCount(s string) int { return utf8.RuneCountInString(s) }

// FirstNonSpace returns the first non-whitespace rune in s, or 0.
func FirstNonSpace(s string) rune {
	for _, r := range s {
		if !unicode.IsSpace(r) {
			return r
		}
	}
	return 0
}
