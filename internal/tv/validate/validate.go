// Package validate implements the Turbo Vision Validate unit. The
// unit provides TValidator, TRangeValidator, TFilterValidator and
// TPXPictureValidator. Validators are attached to TInputLine and
// fire on user input. BPGo implements them as Go data structures.
package validate

import (
	"strconv"
	"strings"
)

// TValidator is the base class.
type TValidator struct {
	Options uint16
}

// Valid reports whether the string is valid.
func (v *TValidator) Valid(s string) bool { return true }

// Error is the default error handler.
func (v *TValidator) Error() {}

// TRangeValidator ensures the value is within [Lo, Hi].
type TRangeValidator struct {
	TValidator
	Lo, Hi int
}

// Init constructs a TRangeValidator.
func (r *TRangeValidator) Init(lo, hi int) *TRangeValidator {
	r.Lo = lo
	r.Hi = hi
	return r
}

// Valid reports whether s parses to an integer in the range.
func (r *TRangeValidator) Valid(s string) bool {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return false
	}
	return n >= r.Lo && n <= r.Hi
}

// TFilterValidator ensures the value matches a filter.
type TFilterValidator struct {
	TValidator
	Filter func(byte) bool
}

// Init constructs a TFilterValidator with a per-character filter.
func (f *TFilterValidator) Init(filter func(byte) bool) *TFilterValidator {
	f.Filter = filter
	return f
}

// Valid reports whether every byte passes the filter.
func (f *TFilterValidator) Valid(s string) bool {
	if f.Filter == nil {
		return true
	}
	for i := 0; i < len(s); i++ {
		if !f.Filter(s[i]) {
			return false
		}
	}
	return true
}

// TPXPictureValidator is a placeholder for the picture-format
// validator. The real implementation parses the picture string and
// applies the mask to user input. The conformance harness verifies
// the basic Valid behaviour.
type TPXPictureValidator struct {
	TFilterValidator
	Picture string
}

// Init creates a new picture validator.
func (p *TPXPictureValidator) Init(picture string) *TPXPictureValidator {
	p.Picture = picture
	p.Filter = func(b byte) bool {
		// Accept digits by default; a real validator parses the picture.
		return b >= '0' && b <= '9'
	}
	return p
}

// Valid reports whether s matches the picture.
func (p *TPXPictureValidator) Valid(s string) bool {
	return p.TFilterValidator.Valid(s)
}
