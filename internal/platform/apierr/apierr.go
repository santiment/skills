// Package apierr provides a normalized error type for non-2xx API responses,
// shared by every CLI in the family. It extracts a human message and an
// optional application code from common JSON error shapes.
package apierr

import (
	"encoding/json"
	"fmt"
)

// Error is a normalized representation of a failed API call.
type Error struct {
	// Backend is the logical backend name that produced the error (e.g. "sanr").
	Backend string
	// HTTPStatus is the HTTP status code returned by the server.
	HTTPStatus int
	// Code is an application-level error code, if the body exposed one.
	Code string
	// Message is the best human-readable message extracted from the body.
	Message string
	// Body is the raw response body, preserved for --json passthrough.
	Body []byte
}

func (e *Error) Error() string {
	switch {
	case e.Code != "" && e.Message != "":
		return fmt.Sprintf("%s: %s (%s, HTTP %d)", e.Backend, e.Message, e.Code, e.HTTPStatus)
	case e.Message != "":
		return fmt.Sprintf("%s: %s (HTTP %d)", e.Backend, e.Message, e.HTTPStatus)
	default:
		return fmt.Sprintf("%s: request failed with HTTP %d", e.Backend, e.HTTPStatus)
	}
}

// FromResponse builds an *Error from a backend name, status code and raw body,
// best-effort extracting a message and code from common JSON error envelopes.
func FromResponse(backend string, status int, body []byte) *Error {
	e := &Error{Backend: backend, HTTPStatus: status, Body: body}

	// Try a few common shapes: {"message": "..."} / {"error": "..."} /
	// {"error": {"message": "...", "code": "..."}} / {"statusCode":..,"message":..}.
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(body, &probe); err != nil {
		return e
	}
	if msg, ok := scalarString(probe["message"]); ok {
		e.Message = msg
	}
	if code, ok := scalarString(probe["code"]); ok {
		e.Code = code
	}
	if e.Message == "" {
		if nested, ok := probe["error"]; ok {
			if msg, ok := scalarString(nested); ok {
				e.Message = msg
			} else {
				var inner struct {
					Message string `json:"message"`
					Code    string `json:"code"`
				}
				if json.Unmarshal(nested, &inner) == nil {
					if inner.Message != "" {
						e.Message = inner.Message
					}
					if inner.Code != "" && e.Code == "" {
						e.Code = inner.Code
					}
				}
			}
		}
	}
	return e
}

func scalarString(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 {
		return "", false
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", false
	}
	return s, true
}
