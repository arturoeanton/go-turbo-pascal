// Package strings implements the Strings unit runtime. Strings in
// TP7/BP7 are PChar (null-terminated) and the unit provides classic
// C-style helpers: StrCat, StrComp, StrIComp, StrLCat, etc. The
// implementation uses a safe byte buffer with an explicit length to
// match the BP7 semantics.
package strings

import "strings"

func StrCat(dest, source []byte) []byte {
	end := -1
	for i, b := range dest {
		if b == 0 {
			end = i
			break
		}
	}
	if end < 0 {
		end = len(dest)
	}
	for _, b := range source {
		if end >= len(dest) {
			break
		}
		if b == 0 {
			break
		}
		dest[end] = b
		end++
	}
	if end < len(dest) {
		dest[end] = 0
	}
	return dest
}

func StrComp(s1, s2 []byte) int {
	return strings.Compare(string(s1), string(s2))
}

func StrIComp(s1, s2 []byte) int {
	if strings.EqualFold(string(s1), string(s2)) {
		return 0
	}
	return 1
}

func StrLen(s []byte) int {
	for i, b := range s {
		if b == 0 {
			return i
		}
	}
	return len(s)
}

func StrEnd(s []byte) []byte {
	for i, b := range s {
		if b == 0 {
			return s[i:]
		}
	}
	return s[len(s):]
}

func StrCopy(dest, source []byte) []byte {
	for i := 0; i < len(dest); i++ {
		if i >= len(source) || source[i] == 0 {
			dest[i] = 0
			break
		}
		dest[i] = source[i]
	}
	return dest
}

func StrECopy(dest, source []byte) []byte {
	for i := 0; i < len(dest); i++ {
		if i >= len(source) || source[i] == 0 {
			dest[i] = 0
			return dest[i:]
		}
		dest[i] = source[i]
	}
	dest[len(dest)-1] = 0
	return dest[len(dest)-1:]
}

func StrLCat(dest, source []byte, maxLen int) []byte {
	end := -1
	for i, b := range dest {
		if b == 0 {
			end = i
			break
		}
	}
	if end < 0 {
		end = len(dest)
	}
	for _, b := range source {
		if end >= maxLen-1 {
			break
		}
		if b == 0 {
			break
		}
		dest[end] = b
		end++
	}
	if end < maxLen {
		dest[end] = 0
	}
	return dest
}

func StrLComp(s1, s2 []byte, maxLen int) int {
	return strings.Compare(string(stripNull(s1, maxLen)), string(stripNull(s2, maxLen)))
}

func StrLIComp(s1, s2 []byte, maxLen int) int {
	a := stripNull(s1, maxLen)
	b := stripNull(s2, maxLen)
	if string(a) == string(b) {
		return 0
	}
	if strings.EqualFold(string(a), string(b)) {
		return 0
	}
	if string(a) < string(b) {
		return -1
	}
	return 1
}

func StrLCopy(dest, source []byte, maxLen int) []byte {
	for i := 0; i < maxLen-1 && i < len(dest); i++ {
		if i >= len(source) || source[i] == 0 {
			dest[i] = 0
			break
		}
		dest[i] = source[i]
	}
	if maxLen-1 < len(dest) {
		dest[maxLen-1] = 0
	}
	return dest
}

func StrLower(s []byte) []byte {
	for i, b := range s {
		if b == 0 {
			break
		}
		if b >= 'A' && b <= 'Z' {
			s[i] = b + ('a' - 'A')
		}
	}
	return s
}

func StrUpper(s []byte) []byte {
	for i, b := range s {
		if b == 0 {
			break
		}
		if b >= 'a' && b <= 'z' {
			s[i] = b - ('a' - 'A')
		}
	}
	return s
}

func StrMove(dest, source []byte, count int) []byte {
	// Overlap-safe move (memmove semantics).
	if len(dest) == 0 || len(source) == 0 || count <= 0 {
		return dest
	}
	c := count
	if c > len(dest) {
		c = len(dest)
	}
	if c > len(source) {
		c = len(source)
	}
	if len(dest) > 0 && len(source) > 0 {
		copy(dest[:c], source[:c])
	}
	return dest
}

func StrNew(s []byte) []byte {
	if s == nil {
		return nil
	}
	n := StrLen(s) + 1
	out := make([]byte, n)
	copy(out, s[:n])
	return out
}

func StrDispose(s []byte) {}

func StrPas(s []byte) string {
	return string(stripNull(s, len(s)))
}

func StrPCopy(dest []byte, str string) []byte {
	for i := 0; i < len(dest); i++ {
		if i >= len(str) {
			dest[i] = 0
			break
		}
		dest[i] = str[i]
	}
	return dest
}

func StrPos(sub, s []byte) []byte {
	if len(sub) == 0 || len(s) == 0 {
		return nil
	}
	idx := strings.Index(string(s), string(stripNull(sub, len(sub))))
	if idx < 0 {
		return nil
	}
	return s[idx:]
}

func StrScan(s []byte, ch byte) []byte {
	for i, b := range s {
		if b == 0 {
			return nil
		}
		if b == ch {
			return s[i:]
		}
	}
	return nil
}

func StrRScan(s []byte, ch byte) []byte {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == 0 {
			continue
		}
		if s[i] == ch {
			return s[i:]
		}
	}
	return nil
}

func stripNull(s []byte, maxLen int) []byte {
	for i := 0; i < maxLen && i < len(s); i++ {
		if s[i] == 0 {
			return s[:i]
		}
	}
	return s[:maxLen]
}
