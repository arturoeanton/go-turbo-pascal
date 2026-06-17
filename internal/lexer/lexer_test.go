package lexer

import (
	"strings"
	"testing"
)

func mustTokenize(t *testing.T, src string) []Token {
	t.Helper()
	l := New(src)
	if len(l.Errors()) > 0 {
		t.Fatalf("lex errors: %v", l.Errors())
	}
	return l.Tokens()
}

func TestKeywords(t *testing.T) {
	src := "and array asm begin case const constructor destructor div do downto else end file for function goto if implementation in inherited inline interface label mod nil not object of or packed procedure program record repeat set shl shr string then to type unit until uses var while with xor"
	toks := mustTokenize(t, src)
	if len(toks) != len(strings.Fields(src))+1 { // +EOF
		t.Fatalf("expected %d tokens, got %d", len(strings.Fields(src))+1, len(toks))
	}
	if toks[len(toks)-1].Kind != TokEOF {
		t.Errorf("expected EOF at end")
	}
}

func TestNilIsKeyword(t *testing.T) {
	toks := mustTokenize(t, "nil NIL Nil")
	for i, tok := range toks[:3] {
		if tok.Kind != TokKeyword || tok.Lower != "nil" {
			t.Errorf("token %d: expected nil keyword, got %+v", i, tok)
		}
	}
}

func TestMixedCase(t *testing.T) {
	toks := mustTokenize(t, "BEGIN BeGiN begin")
	for i, tok := range toks[:3] {
		if tok.Kind != TokKeyword || tok.Lower != "begin" {
			t.Errorf("token %d: expected keyword begin, got %v", i, tok)
		}
	}
}

func TestIdentifiersCaseInsensitive(t *testing.T) {
	toks := mustTokenize(t, "MyVar MYVAR myvar")
	if toks[0].Lower != toks[1].Lower || toks[1].Lower != toks[2].Lower {
		t.Errorf("identifiers should compare equal case-insensitively: %v", toks[:3])
	}
}

func TestHexIntegers(t *testing.T) {
	toks := mustTokenize(t, "$1A $FF $100")
	if toks[0].Hex != 0x1A || toks[1].Hex != 0xFF || toks[2].Hex != 0x100 {
		t.Errorf("hex values: %+v", toks[:3])
	}
}

func TestDecIntegers(t *testing.T) {
	toks := mustTokenize(t, "0 1 42 65535")
	for i, v := range []int64{0, 1, 42, 65535} {
		if toks[i].Int != v {
			t.Errorf("int %d: expected %d, got %d", i, v, toks[i].Int)
		}
	}
}

func TestRealLiterals(t *testing.T) {
	toks := mustTokenize(t, "3.14 0.5 1.0e3 2E-2")
	if toks[0].Kind != TokReal || toks[0].Real != 3.14 {
		t.Errorf("real 0: %+v", toks[0])
	}
	if toks[3].Kind != TokReal || toks[3].Real != 0.02 {
		t.Errorf("real 3: %+v", toks[3])
	}
}

func TestStringWithControlChars(t *testing.T) {
	toks := mustTokenize(t, "'hello' 'a#13b'#10'c' '#0'")
	// Expected: STRING("hello"), STRING("a\rb"), INT(#10), STRING("c"), STRING("\x00"), EOF
	if toks[0].Kind != TokString || toks[0].Str != "hello" {
		t.Errorf("string 0: %+v", toks[0])
	}
	if toks[1].Kind != TokString || toks[1].Str != "a\rb" {
		t.Errorf("string 1: %+v", toks[1])
	}
	if toks[2].Kind != TokInt || toks[2].Int != 10 {
		t.Errorf("int: %+v", toks[2])
	}
	if toks[3].Kind != TokString || toks[3].Str != "c" {
		t.Errorf("string 2: %+v", toks[3])
	}
	if toks[4].Kind != TokString || toks[4].Str != "\x00" {
		t.Errorf("string 3: %+v", toks[4])
	}
}

func TestStringEscapedQuote(t *testing.T) {
	toks := mustTokenize(t, "'it''s'")
	if toks[0].Str != "it's" {
		t.Errorf("string: %+v", toks[0])
	}
}

func TestUnterminatedString(t *testing.T) {
	l := New("'oops")
	if len(l.Errors()) == 0 {
		t.Error("expected error for unterminated string")
	}
}

func TestCommentsAndDirectives(t *testing.T) {
	toks := mustTokenize(t, "begin {comment} (* paren *) {$R+} (*$R-*) end")
	// Expect: BEGIN, END, EOF (comments and directives are filtered)
	if toks[0].Lower != "begin" || toks[len(toks)-2].Lower != "end" {
		t.Errorf("keyword placement wrong: %+v", toks)
	}
}

func TestUnterminatedComment(t *testing.T) {
	l := New("(* oops")
	if len(l.Errors()) == 0 {
		t.Error("expected error for unterminated (* *)")
	}
}

func TestOperators(t *testing.T) {
	toks := mustTokenize(t, "+ - * / = <> < <= > >= @ ^ ( ) [ ] .. , ; : . :=")
	want := []TokenKind{TokOp, TokOp, TokOp, TokOp, TokEqual, TokOp, TokOp, TokOp, TokOp, TokOp, TokAt, TokCaret, TokLParen, TokRParen, TokLBracket, TokRBracket, TokRange, TokComma, TokSemicolon, TokColon, TokPeriod, TokAssign, TokEOF}
	if len(toks) != len(want) {
		t.Fatalf("expected %d tokens, got %d", len(want), len(toks))
	}
	for i, k := range want {
		if toks[i].Kind != k {
			t.Errorf("token %d: expected %v, got %v", i, k, toks[i].Kind)
		}
	}
}

func TestPositions(t *testing.T) {
	toks := mustTokenize(t, "a\nb")
	if toks[0].Line != 1 || toks[0].Col != 1 {
		t.Errorf("a: %d/%d", toks[0].Line, toks[0].Col)
	}
	if toks[1].Line != 2 || toks[1].Col != 1 {
		t.Errorf("b: %d/%d", toks[1].Line, toks[1].Col)
	}
}

func TestNumberVsRange(t *testing.T) {
	toks := mustTokenize(t, "1..2 3 0.5")
	if toks[0].Kind != TokInt || toks[1].Kind != TokRange {
		t.Errorf("1..2 should be int range, got %v %v", toks[0].Kind, toks[1].Kind)
	}
	if toks[2].Kind != TokInt {
		t.Errorf("2 should be int, got %v", toks[2].Kind)
	}
	if toks[3].Kind != TokInt {
		t.Errorf("3 should be int, got %v", toks[3].Kind)
	}
	if toks[4].Kind != TokReal {
		t.Errorf("0.5 should be real, got %v", toks[4].Kind)
	}
}
