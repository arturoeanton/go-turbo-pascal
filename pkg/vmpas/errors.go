package vmpas

import "fmt"

// RuntimeError is returned by Run, RunDurable and ResumeDurable when guest
// execution fails with a Turbo Pascal-style runtime error. Code is the TP7
// error number and Message is a short human-readable description. Inspect it
// with errors.As:
//
//	if err := eng.Run(code); err != nil {
//	    var re *vmpas.RuntimeError
//	    if errors.As(err, &re) && re.Code == 200 {
//	        // division by zero or a step/time limit was hit
//	    }
//	}
type RuntimeError struct {
	Code    int    // TP7 runtime error number
	Message string // human-readable description ("" if unknown)
}

func (e *RuntimeError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("vmpas: runtime error %d", e.Code)
	}
	return fmt.Sprintf("vmpas: runtime error %d (%s)", e.Code, e.Message)
}

// runtimeErrorMessage maps a VM runtime error code to a short description. Some
// codes are shared between a classic TP7 fault and a sandbox-limit breach, so
// the text covers both.
func runtimeErrorMessage(code int) string {
	switch code {
	case 200:
		return "division by zero or step/time limit exceeded"
	case 201:
		return "range check error"
	case 202:
		return "stack overflow or call-depth limit exceeded"
	case 203:
		return "heap or output limit exceeded"
	case 204:
		return "invalid pointer operation"
	case 205:
		return "floating point overflow"
	case 215:
		return "arithmetic overflow"
	case 216:
		return "general protection fault"
	case 217:
		return "unhandled exception"
	}
	return ""
}

// newRuntimeError builds the typed error for a VM runtime error code.
func newRuntimeError(code int) *RuntimeError {
	return &RuntimeError{Code: code, Message: runtimeErrorMessage(code)}
}
