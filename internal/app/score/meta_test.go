package score

import (
	"encoding/json"
	"strings"
	"testing"

	"santiment.net/san-skills/internal/platform/exitcode"
)

func TestVersionCommand(t *testing.T) {
	app, out := runCmd(t, "version")
	if app.exitCode != exitcode.OK {
		t.Fatalf("version exit=%d, out=%s", app.exitCode, out)
	}
	obj, ok := decodeObject(t, out)
	if !ok {
		t.Fatalf("version output not a JSON object: %s", out)
	}
	if obj["version"] != Version {
		t.Errorf("version: want %q, got %v", Version, obj["version"])
	}
}

// describe must emit a valid catalog covering every top-level command and the
// full exit-code contract — guarding against drift between the command tree and
// the agent-facing catalog.
func TestDescribeCatalog(t *testing.T) {
	app, out := runCmd(t, "describe")
	if app.exitCode != exitcode.OK {
		t.Fatalf("describe exit=%d, out=%s", app.exitCode, out)
	}

	var doc describeDoc
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("describe output not valid JSON: %v\nout=%s", err, out)
	}

	// Every stable exit code must be documented.
	for _, code := range []string{"0", "1", "2", "3", "4", "5", "6", "7"} {
		if _, ok := doc.ExitCodes[code]; !ok {
			t.Errorf("describe exitCodes missing %q", code)
		}
	}

	// Every registered top-level command must appear in the catalog.
	got := map[string]bool{}
	for _, c := range doc.Commands {
		fields := strings.Fields(c.Path) // "score predictions" -> last field
		got[fields[len(fields)-1]] = true
	}
	want := []string{
		"version", "describe", "health", "auth", "predictions", "markets",
		"portfolio", "leaderboards", "competitions", "pairs", "prices",
		"profile", "issuers", "stakes", "events", "contracts",
	}
	for _, name := range want {
		if !got[name] {
			t.Errorf("describe catalog missing command %q", name)
		}
	}
}
