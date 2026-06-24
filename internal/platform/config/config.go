// Package config loads effective settings for the score CLI by layering, in
// order of decreasing precedence: command-line flags, environment variables,
// the config file profile, and built-in defaults. It also persists cached
// auth tokens back to the config file (the login cache).
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Default backend base URLs for the Santiment Score product.
const (
	DefaultSanrBaseURL  = "https://api.sanr.app"
	DefaultArenaBaseURL = "https://api-arena.santiment.net"
	DefaultProfile      = "default"
)

// Environment variable names (documented in SKILL.md).
const (
	EnvConfig       = "SCORE_CONFIG"
	EnvProfile      = "SCORE_PROFILE"
	EnvSanrBaseURL  = "SANR_BASE_URL"
	EnvArenaBaseURL = "ARENA_BASE_URL"
	EnvSanrToken    = "SANR_TOKEN"
	EnvArenaAPIKey  = "ARENA_API_KEY"
)

// Profile is one named set of backend settings in the config file.
type Profile struct {
	SanrBaseURL  string `yaml:"sanr_base_url,omitempty"`
	ArenaBaseURL string `yaml:"arena_base_url,omitempty"`
	SanrToken    string `yaml:"sanr_token,omitempty"`
	ArenaAPIKey  string `yaml:"arena_api_key,omitempty"`
}

// File is the on-disk config structure.
type File struct {
	CurrentProfile string              `yaml:"current_profile,omitempty"`
	Profiles       map[string]*Profile `yaml:"profiles,omitempty"`
}

// Overrides carries flag values supplied on the command line. Empty strings
// mean "not set" and fall through to lower-precedence layers.
type Overrides struct {
	ConfigPath   string
	Profile      string
	SanrBaseURL  string
	ArenaBaseURL string
	Token        string
	APIKey       string
}

// Settings is the resolved, effective configuration the app runs with.
type Settings struct {
	ProfileName  string
	ConfigPath   string
	SanrBaseURL  string
	ArenaBaseURL string
	SanrToken    string
	ArenaAPIKey  string
}

// DefaultPath returns the config file path, honoring SCORE_CONFIG and
// XDG_CONFIG_HOME, defaulting to ~/.config/score/config.yaml.
func DefaultPath() string {
	if p := os.Getenv(EnvConfig); p != "" {
		return p
	}
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "score", "config.yaml")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".score", "config.yaml")
	}
	return filepath.Join(home, ".config", "score", "config.yaml")
}

// Load reads and parses the config file. A missing file is not an error: it
// returns an empty File so first-run usage works without setup.
func Load(path string) (*File, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &File{Profiles: map[string]*Profile{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	var f File
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	if f.Profiles == nil {
		f.Profiles = map[string]*Profile{}
	}
	return &f, nil
}

// Resolve layers overrides > env > file profile > defaults into Settings.
func Resolve(ov Overrides) (*Settings, error) {
	path := firstNonEmpty(ov.ConfigPath, os.Getenv(EnvConfig), DefaultPath())
	file, err := Load(path)
	if err != nil {
		return nil, err
	}

	profileName := firstNonEmpty(ov.Profile, os.Getenv(EnvProfile), file.CurrentProfile, DefaultProfile)
	prof := file.Profiles[profileName]
	if prof == nil {
		prof = &Profile{}
	}

	// One token authenticates both backends: it is sent as the Sanr bearer token
	// and as the Arena x-api-key. So the Arena key falls back to the Sanr token
	// when no Arena-specific value is set — a single `auth login` (or SANR_TOKEN)
	// is enough for every command. An explicit --api-key / ARENA_API_KEY still
	// overrides, for the rare case the two differ.
	sanrToken := firstNonEmpty(ov.Token, os.Getenv(EnvSanrToken), prof.SanrToken)
	s := &Settings{
		ProfileName: profileName,
		ConfigPath:  path,
		SanrBaseURL: firstNonEmpty(ov.SanrBaseURL, os.Getenv(EnvSanrBaseURL),
			prof.SanrBaseURL, DefaultSanrBaseURL),
		ArenaBaseURL: firstNonEmpty(ov.ArenaBaseURL, os.Getenv(EnvArenaBaseURL),
			prof.ArenaBaseURL, DefaultArenaBaseURL),
		SanrToken:   sanrToken,
		ArenaAPIKey: firstNonEmpty(ov.APIKey, os.Getenv(EnvArenaAPIKey), prof.ArenaAPIKey, sanrToken),
	}
	return s, nil
}

// SaveTokens caches the Sanr API token into the named profile of the config
// file, creating the file and parent directory if needed. The same token also
// authenticates Arena, so this single value is all the CLI needs to persist.
func SaveTokens(path, profile, token string) error {
	if profile == "" {
		profile = DefaultProfile
	}
	file, err := Load(path)
	if err != nil {
		return err
	}
	prof := file.Profiles[profile]
	if prof == nil {
		prof = &Profile{}
		file.Profiles[profile] = prof
	}
	prof.SanrToken = token
	if file.CurrentProfile == "" {
		file.CurrentProfile = profile
	}
	return write(path, file)
}

func write(path string, file *File) error {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create config dir: %w", err)
		}
	}
	raw, err := yaml.Marshal(file)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
