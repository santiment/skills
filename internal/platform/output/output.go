// Package output renders command results in a stable, machine-friendly way for
// every CLI in the family. In --json mode the raw API body is passed through
// unchanged (so agents see exactly what the backend returned); in human mode
// JSON is pretty-printed. Errors always go to stderr; results to stdout.
package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"santiment.net/san-skills/internal/platform/apierr"
	"santiment.net/san-skills/internal/platform/exitcode"
)

// Printer renders results and errors. Construct it with New.
type Printer struct {
	JSON bool
	Out  io.Writer
	Err  io.Writer
}

// New returns a Printer writing to os.Stdout/os.Stderr.
func New(jsonMode bool) *Printer {
	return &Printer{JSON: jsonMode, Out: os.Stdout, Err: os.Stderr}
}

// EmitResult renders a raw API response. It returns the stable exit code the
// process should use. Non-2xx responses are rendered as errors on stderr.
func (p *Printer) EmitResult(backend string, status int, body []byte) int {
	if status >= 400 {
		return p.EmitError(apierr.FromResponse(backend, status, body))
	}
	if len(body) == 0 {
		return exitcode.OK
	}
	if p.JSON {
		p.writeRaw(p.Out, body)
		return exitcode.OK
	}
	p.writePretty(body)
	return exitcode.OK
}

// EmitError renders any error (transport, API, or usage) on stderr and returns
// the stable exit code for it.
func (p *Printer) EmitError(err error) int {
	code := exitcode.FromError(err)
	if p.JSON {
		p.writeErrorJSON(err)
	} else {
		_, _ = fmt.Fprintf(p.Err, "error: %s\n", err.Error())
	}
	return code
}

// EmitValue renders an arbitrary Go value as JSON on stdout (used by commands
// that synthesize their own payload, e.g. health and describe).
func (p *Printer) EmitValue(v any) error {
	enc := json.NewEncoder(p.Out)
	if !p.JSON {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(v)
}

func (p *Printer) writeErrorJSON(err error) {
	payload := map[string]any{"message": err.Error()}
	var apiErr *apierr.Error
	if e, ok := err.(*apierr.Error); ok {
		apiErr = e
	}
	if apiErr != nil {
		payload["httpStatus"] = apiErr.HTTPStatus
		if apiErr.Code != "" {
			payload["code"] = apiErr.Code
		}
		if apiErr.Backend != "" {
			payload["backend"] = apiErr.Backend
		}
		if apiErr.Message != "" {
			payload["message"] = apiErr.Message
		}
	}
	enc := json.NewEncoder(p.Err)
	_ = enc.Encode(map[string]any{"error": payload})
}

func (p *Printer) writeRaw(w io.Writer, body []byte) {
	_, _ = w.Write(body)
	if len(body) > 0 && body[len(body)-1] != '\n' {
		_, _ = io.WriteString(w, "\n")
	}
}

func (p *Printer) writePretty(body []byte) {
	var buf bytes.Buffer
	if err := json.Indent(&buf, body, "", "  "); err != nil {
		p.writeRaw(p.Out, body) // not JSON; print as-is
		return
	}
	p.writeRaw(p.Out, buf.Bytes())
}
