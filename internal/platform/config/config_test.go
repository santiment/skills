package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePrecedence(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	yaml := `current_profile: default
profiles:
  default:
    sanr_base_url: https://file.sanr
    sanr_token: file-token
    arena_api_key: file-key
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv(EnvSanrToken, "env-token")

	// Flag (--token) should win over env, which wins over file.
	s, err := Resolve(Overrides{ConfigPath: cfgPath, Token: "flag-token"})
	if err != nil {
		t.Fatal(err)
	}
	if s.SanrToken != "flag-token" {
		t.Errorf("token: want flag-token, got %q", s.SanrToken)
	}
	// No flag -> env wins over file.
	s, _ = Resolve(Overrides{ConfigPath: cfgPath})
	if s.SanrToken != "env-token" {
		t.Errorf("token: want env-token, got %q", s.SanrToken)
	}
	// File value used when no flag/env.
	if s.SanrBaseURL != "https://file.sanr" {
		t.Errorf("sanrBaseURL: want file value, got %q", s.SanrBaseURL)
	}
	// Default used when nothing set.
	if s.ArenaBaseURL != DefaultArenaBaseURL {
		t.Errorf("arenaBaseURL: want default, got %q", s.ArenaBaseURL)
	}
	if s.ArenaAPIKey != "file-key" {
		t.Errorf("arenaApiKey: want file-key, got %q", s.ArenaAPIKey)
	}
}

func TestResolveMissingFileUsesDefaults(t *testing.T) {
	s, err := Resolve(Overrides{ConfigPath: filepath.Join(t.TempDir(), "absent.yaml")})
	if err != nil {
		t.Fatalf("missing file should not error: %v", err)
	}
	if s.SanrBaseURL != DefaultSanrBaseURL || s.ArenaBaseURL != DefaultArenaBaseURL {
		t.Errorf("expected default base URLs, got %q / %q", s.SanrBaseURL, s.ArenaBaseURL)
	}
	if s.ProfileName != DefaultProfile {
		t.Errorf("expected default profile, got %q", s.ProfileName)
	}
}

func TestSaveTokensRoundTrip(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "nested", "config.yaml")
	if err := SaveTokens(cfgPath, "default", "tok-123", "refresh-456"); err != nil {
		t.Fatal(err)
	}
	s, err := Resolve(Overrides{ConfigPath: cfgPath})
	if err != nil {
		t.Fatal(err)
	}
	if s.SanrToken != "tok-123" {
		t.Errorf("token not persisted, got %q", s.SanrToken)
	}
	if s.SanrRefreshToken != "refresh-456" {
		t.Errorf("refresh not persisted, got %q", s.SanrRefreshToken)
	}
	// File must be created with restrictive permissions.
	info, err := os.Stat(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("config perms: want 0600, got %o", perm)
	}
}
