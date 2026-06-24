package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"santiment.net/san-skills/internal/platform/exitcode"
)

func newTestPrinter(jsonMode bool) (*Printer, *bytes.Buffer, *bytes.Buffer) {
	var out, errBuf bytes.Buffer
	return &Printer{JSON: jsonMode, Out: &out, Err: &errBuf}, &out, &errBuf
}

func TestEmitResultSuccessJSONPassthrough(t *testing.T) {
	p, out, errBuf := newTestPrinter(true)
	body := []byte(`{"data":[1,2,3]}`)
	if code := p.EmitResult("sanr", 200, body); code != exitcode.OK {
		t.Errorf("want OK, got %d", code)
	}
	if strings.TrimSpace(out.String()) != string(body) {
		t.Errorf("json mode must pass body through verbatim, got %q", out.String())
	}
	if errBuf.Len() != 0 {
		t.Errorf("success must not write stderr, got %q", errBuf.String())
	}
}

func TestEmitResultErrorMapsExitCode(t *testing.T) {
	p, out, errBuf := newTestPrinter(true)
	code := p.EmitResult("sanr", 401, []byte(`{"message":"no auth"}`))
	if code != exitcode.Auth {
		t.Errorf("want Auth, got %d", code)
	}
	if out.Len() != 0 {
		t.Errorf("error must not write stdout, got %q", out.String())
	}
	var envelope struct {
		Error struct {
			HTTPStatus int    `json:"httpStatus"`
			Message    string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(errBuf.Bytes(), &envelope); err != nil {
		t.Fatalf("stderr must be JSON: %v (%q)", err, errBuf.String())
	}
	if envelope.Error.HTTPStatus != 401 || envelope.Error.Message != "no auth" {
		t.Errorf("unexpected error envelope: %+v", envelope.Error)
	}
}

func TestEmitResultHumanPrettyPrints(t *testing.T) {
	p, out, _ := newTestPrinter(false)
	p.EmitResult("sanr", 200, []byte(`{"a":1}`))
	if !strings.Contains(out.String(), "\n  \"a\": 1") {
		t.Errorf("human mode should indent JSON, got %q", out.String())
	}
}
