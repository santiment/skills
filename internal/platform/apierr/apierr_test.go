package apierr

import "testing"

func TestFromResponseShapes(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		wantMsg  string
		wantCode string
	}{
		{name: "message", body: `{"message":"boom"}`, wantMsg: "boom"},
		{name: "message+code", body: `{"message":"boom","code":"E1"}`, wantMsg: "boom", wantCode: "E1"},
		{name: "error-scalar", body: `{"error":"oops"}`, wantMsg: "oops"},
		{name: "error-nested", body: `{"error":{"message":"deep","code":"E2"}}`, wantMsg: "deep", wantCode: "E2"},
		{name: "message-wins-over-error", body: `{"message":"top","error":"ignored"}`, wantMsg: "top"},
		{name: "malformed-json", body: `not json at all`, wantMsg: ""},
		{name: "empty-body", body: ``, wantMsg: ""},
		{name: "unrelated-shape", body: `{"foo":"bar"}`, wantMsg: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := FromResponse("sanr", 400, []byte(tc.body))
			if e.Backend != "sanr" {
				t.Errorf("backend not preserved: %q", e.Backend)
			}
			if e.HTTPStatus != 400 {
				t.Errorf("status not preserved: %d", e.HTTPStatus)
			}
			if string(e.Body) != tc.body {
				t.Errorf("body not preserved: %q", e.Body)
			}
			if e.Message != tc.wantMsg {
				t.Errorf("message: want %q, got %q", tc.wantMsg, e.Message)
			}
			if e.Code != tc.wantCode {
				t.Errorf("code: want %q, got %q", tc.wantCode, e.Code)
			}
		})
	}
}

func TestErrorString(t *testing.T) {
	cases := []struct {
		name string
		err  *Error
		want string
	}{
		{
			name: "code+message",
			err:  &Error{Backend: "sanr", HTTPStatus: 422, Code: "E1", Message: "bad"},
			want: "sanr: bad (E1, HTTP 422)",
		},
		{
			name: "message-only",
			err:  &Error{Backend: "arena", HTTPStatus: 404, Message: "missing"},
			want: "arena: missing (HTTP 404)",
		},
		{
			name: "neither",
			err:  &Error{Backend: "sanr", HTTPStatus: 500},
			want: "sanr: request failed with HTTP 500",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.err.Error(); got != tc.want {
				t.Errorf("Error(): want %q, got %q", tc.want, got)
			}
		})
	}
}
