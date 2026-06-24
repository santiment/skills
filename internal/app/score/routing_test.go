package score

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"santiment.net/san-skills/internal/platform/exitcode"
	"santiment.net/san-skills/internal/platform/output"
)

// recordingServer returns a server that records request paths and replies with
// the given status and body.
func recordingServer(status int, body string) (*httptest.Server, *[]string) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	return srv, &paths
}

// runScore builds the command tree with a buffer-backed printer and runs args.
func runScore(t *testing.T, sanrURL, arenaURL string, args ...string) (*App, string) {
	t.Helper()
	app := &App{}
	var out, errBuf bytes.Buffer
	app.printer = &output.Printer{JSON: true, Out: &out, Err: &errBuf}

	full := append([]string{
		"--base-url-sanr", sanrURL,
		"--base-url-arena", arenaURL,
		"--config", filepath.Join(t.TempDir(), "absent.yaml"),
	}, args...)

	root := newRootCmd(app)
	root.SetArgs(full)
	root.SetOut(&out)
	root.SetErr(&errBuf)
	_ = root.Execute()
	return app, out.String() + errBuf.String()
}

func contains(paths []string, want string) bool {
	for _, p := range paths {
		if p == want {
			return true
		}
	}
	return false
}

func TestRoutingMarketsHitsSanrOnly(t *testing.T) {
	sanr, sanrPaths := recordingServer(200, `{"data":[]}`)
	defer sanr.Close()
	arena, arenaPaths := recordingServer(200, `{"data":[]}`)
	defer arena.Close()

	app, _ := runScore(t, sanr.URL, arena.URL, "markets", "list", "--take", "1")
	if !contains(*sanrPaths, "/v2/markets") {
		t.Errorf("markets must hit Sanr /v2/markets, sanr paths=%v", *sanrPaths)
	}
	if len(*arenaPaths) != 0 {
		t.Errorf("markets must not touch Arena, arena paths=%v", *arenaPaths)
	}
	if app.exitCode != exitcode.OK {
		t.Errorf("want exit OK, got %d", app.exitCode)
	}
}

func TestRoutingIssuersHitsArenaOnly(t *testing.T) {
	sanr, sanrPaths := recordingServer(200, `{"data":[]}`)
	defer sanr.Close()
	arena, arenaPaths := recordingServer(200, `{"data":[]}`)
	defer arena.Close()

	app, _ := runScore(t, sanr.URL, arena.URL, "issuers", "list", "--limit", "1")
	if !contains(*arenaPaths, "/v1/issuers") {
		t.Errorf("issuers must hit Arena /v1/issuers, arena paths=%v", *arenaPaths)
	}
	if len(*sanrPaths) != 0 {
		t.Errorf("issuers must not touch Sanr, sanr paths=%v", *sanrPaths)
	}
	if app.exitCode != exitcode.OK {
		t.Errorf("want exit OK, got %d", app.exitCode)
	}
}

func TestRoutingHealthHitsBoth(t *testing.T) {
	sanr, sanrPaths := recordingServer(200, `{"ok":true}`)
	defer sanr.Close()
	arena, arenaPaths := recordingServer(200, `{"ok":true}`)
	defer arena.Close()

	app, out := runScore(t, sanr.URL, arena.URL, "health")
	if !contains(*sanrPaths, "/v1/ping") || !contains(*arenaPaths, "/v1/ping") {
		t.Errorf("health must ping both, sanr=%v arena=%v", *sanrPaths, *arenaPaths)
	}
	if !strings.Contains(out, `"sanr"`) || !strings.Contains(out, `"arena"`) {
		t.Errorf("health report must mention both backends, got %s", out)
	}
	if app.exitCode != exitcode.OK {
		t.Errorf("want exit OK, got %d", app.exitCode)
	}
}

func TestExitCodeOnAuthError(t *testing.T) {
	sanr, _ := recordingServer(401, `{"message":"unauthorized"}`)
	defer sanr.Close()
	arena, _ := recordingServer(200, `{}`)
	defer arena.Close()

	app, _ := runScore(t, sanr.URL, arena.URL, "predictions", "list", "--take", "1")
	if app.exitCode != exitcode.Auth {
		t.Errorf("401 must map to exit %d, got %d", exitcode.Auth, app.exitCode)
	}
}

func TestUsageErrorExitCode(t *testing.T) {
	sanr, _ := recordingServer(200, `{}`)
	defer sanr.Close()
	arena, _ := recordingServer(200, `{}`)
	defer arena.Close()

	app, _ := runScore(t, sanr.URL, arena.URL, "predictions", "get", "not-an-int")
	if app.exitCode != exitcode.Usage {
		t.Errorf("non-int id must map to exit %d, got %d", exitcode.Usage, app.exitCode)
	}
}
