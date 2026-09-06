// Package huskerr define errores amigables para el usuario: un mensaje
// claro, una sugerencia de remediación y, opcionalmente, la causa técnica
// original. La causa y el stack de creación solo se muestran con --verbose
// o HUSK_DEBUG=1; el resto del tiempo el usuario solo ve mensaje + sugerencia.
package huskerr

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
)

// Error es un error accionable: qué falló y qué hacer al respecto.
type Error struct {
	Msg   string
	Hint  string
	Cause error

	stack []uintptr
}

// New crea un *Error, capturando el stack de creación para diagnóstico.
// cause puede ser nil cuando el error no envuelve una causa técnica (por
// ejemplo, un chequeo de negocio que simplemente no se cumple).
func New(msg, hint string, cause error) *Error {
	const depth = 32
	pcs := make([]uintptr, depth)
	n := runtime.Callers(2, pcs)
	return &Error{Msg: msg, Hint: hint, Cause: cause, stack: pcs[:n]}
}

func (e *Error) Error() string { return e.Msg }

// Unwrap permite que errors.Is/errors.As atraviesen la causa original.
func (e *Error) Unwrap() error { return e.Cause }

// StackString renderiza el stack de creación del error, una entrada por
// frame de llamada.
func (e *Error) StackString() string {
	if len(e.stack) == 0 {
		return ""
	}
	frames := runtime.CallersFrames(e.stack)
	var b strings.Builder
	for {
		frame, more := frames.Next()
		fmt.Fprintf(&b, "  %s\n      %s:%d\n", frame.Function, frame.File, frame.Line)
		if !more {
			break
		}
	}
	return b.String()
}

// Debug reporta si deben mostrarse causa y stack: con --verbose o con la
// variable de entorno HUSK_DEBUG=1.
func Debug(verbose bool) bool {
	return verbose || os.Getenv("HUSK_DEBUG") == "1"
}

// PrintFriendly escribe una representación legible de err en w. Si err es un
// *huskerr.Error, se muestra "Error: <msg>" y "Sugerencia: <hint>"; la causa
// y el stack solo se agregan cuando Debug(verbose) es true. Para cualquier
// otro error se hace un fallback simple.
func PrintFriendly(w io.Writer, err error, verbose bool) {
	var he *Error
	if errors.As(err, &he) {
		fmt.Fprintf(w, "Error: %s\n", he.Msg)
		if he.Hint != "" {
			fmt.Fprintf(w, "Sugerencia: %s\n", he.Hint)
		}
		if Debug(verbose) {
			if he.Cause != nil {
				fmt.Fprintf(w, "\nCausa: %v\n", he.Cause)
			}
			if s := he.StackString(); s != "" {
				fmt.Fprintf(w, "\nStack:\n%s", s)
			}
		}
		return
	}
	fmt.Fprintf(w, "Error: %v\n", err)
}
