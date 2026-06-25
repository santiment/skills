package hyperhandler

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"santiment.net/san-skills/internal/platform/exitcode"
	"santiment.net/san-skills/internal/platform/output"
)

// run executes the command tree with args against a buffer-backed printer and
// returns the recorded exit code plus stdout/stderr. It runs fully offline:
// describe/validate/the mainnet guard never touch the network.
func run(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	app := &App{printer: &output.Printer{JSON: true, Out: &out, Err: &errb}}
	root := newRootCmd(app)
	root.SetArgs(args)
	root.SetOut(&out)
	root.SetErr(&errb)
	if err := root.ExecuteContext(context.Background()); err != nil {
		// usage errors from cobra itself surface here, not via app.exitCode.
		return exitcode.Usage, out.String(), errb.String()
	}
	return app.exitCode, out.String(), errb.String()
}

func TestDescribeSucceeds(t *testing.T) {
	code, out, _ := run(t, "describe", "--json")
	if code != exitcode.OK {
		t.Fatalf("describe exit = %d, want 0", code)
	}
	for _, want := range []string{`"name":"hyperhandler"`, `"exitCodes"`, `"path":"hyperhandler exec"`} {
		if !strings.Contains(out, want) {
			t.Errorf("describe output missing %q", want)
		}
	}
}

func TestValidateValidSignal(t *testing.T) {
	// Inject a signal via a temp file so the test does not depend on stdin.
	f := t.TempDir() + "/sig.json"
	writeFile(t, f, `{"pair":"BTC","side":"long","order_type":"limit","entry_price":67500,"size":0.1,"leverage":5,"stop_loss":66000}`)
	code, out, _ := run(t, "validate", "--signal", f, "--json")
	if code != exitcode.OK {
		t.Fatalf("validate exit = %d, want 0", code)
	}
	if !strings.Contains(out, `"valid":true`) {
		t.Errorf("expected valid:true, got %s", out)
	}
}

func TestMainnetGuardBlocksExec(t *testing.T) {
	f := t.TempDir() + "/sig.json"
	writeFile(t, f, `{"pair":"BTC","side":"long","order_type":"market","size":0.1,"leverage":5}`)
	// exec on mainnet without --confirm must refuse with exit 2 and send nothing.
	code, _, errb := run(t, "exec", "--signal", f, "--network", "mainnet", "--json")
	if code != exitcode.Usage {
		t.Fatalf("mainnet exec without --confirm exit = %d, want 2", code)
	}
	if !strings.Contains(errb, "--confirm") {
		t.Errorf("expected a --confirm message, got %s", errb)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
