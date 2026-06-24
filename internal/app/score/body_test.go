package score

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"santiment.net/san-skills/internal/platform/exitcode"
)

func TestBodyFlagsFromData(t *testing.T) {
	b := bodyFlags{data: `{"a":1,"b":"x"}`}
	var got map[string]any
	if err := b.into(&cobra.Command{}, &got); err != nil {
		t.Fatalf("into: %v", err)
	}
	if got["a"] != float64(1) || got["b"] != "x" {
		t.Errorf("decoded body wrong: %#v", got)
	}
}

func TestBodyFlagsFromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "body.json")
	if err := os.WriteFile(path, []byte(`{"k":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	b := bodyFlags{dataFile: path}
	var got map[string]any
	if err := b.into(&cobra.Command{}, &got); err != nil {
		t.Fatalf("into: %v", err)
	}
	if got["k"] != true {
		t.Errorf("decoded body wrong: %#v", got)
	}
}

func TestBodyFlagsFromStdin(t *testing.T) {
	b := bodyFlags{dataFile: "-"}
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(`{"s":"in"}`))
	var got map[string]any
	if err := b.into(cmd, &got); err != nil {
		t.Fatalf("into: %v", err)
	}
	if got["s"] != "in" {
		t.Errorf("decoded body wrong: %#v", got)
	}
}

func TestBodyFlagsMissingIsUsageError(t *testing.T) {
	b := bodyFlags{}
	var got map[string]any
	err := b.into(&cobra.Command{}, &got)
	if err == nil {
		t.Fatal("expected error for missing body")
	}
	if code := exitcode.FromError(err); code != exitcode.Usage {
		t.Errorf("missing body should be usage error (2), got %d", code)
	}
}

func TestBodyFlagsInvalidJSONIsUsageError(t *testing.T) {
	b := bodyFlags{data: `{not valid json}`}
	var got map[string]any
	err := b.into(&cobra.Command{}, &got)
	if err == nil {
		t.Fatal("expected error for invalid JSON body")
	}
	if code := exitcode.FromError(err); code != exitcode.Usage {
		t.Errorf("invalid JSON should be usage error (2), got %d", code)
	}
}

func TestBodyFlagsFileReadError(t *testing.T) {
	b := bodyFlags{dataFile: filepath.Join(t.TempDir(), "does-not-exist.json")}
	var got map[string]any
	if err := b.into(&cobra.Command{}, &got); err == nil {
		t.Fatal("expected error reading a missing body file")
	}
}
