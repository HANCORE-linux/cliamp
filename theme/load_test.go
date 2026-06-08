package theme

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAllIncludesBuiltinThemes(t *testing.T) {
	// Point HOME somewhere empty so only embedded themes load.
	t.Setenv("HOME", t.TempDir())

	themes := LoadAll()
	if len(themes) == 0 {
		t.Fatal("LoadAll() returned no themes, expected built-in set")
	}

	// Check a well-known theme is present (dracula ships with the project).
	var hasDracula bool
	for _, th := range themes {
		if th.Name == "dracula" {
			hasDracula = true
			if th.Accent == "" {
				t.Error("dracula theme has empty Accent — embed/parse failed")
			}
			break
		}
	}
	if !hasDracula {
		t.Error("built-in themes missing dracula")
	}
}

func TestLoadAllSortedCaseInsensitive(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	themes := LoadAll()
	for i := 1; i < len(themes); i++ {
		a := strings.ToLower(themes[i-1].Name)
		b := strings.ToLower(themes[i].Name)
		if a > b {
			t.Errorf("themes not sorted: %q before %q", themes[i-1].Name, themes[i].Name)
		}
	}
}

func TestLoadAllPartialUserOverrideMergesOntoBuiltin(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	userDir := filepath.Join(home, ".config", "cliamp", "themes")
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// Only two fields — should MERGE onto the built-in dracula.
	partial := `accent = "#ff00ff"
fg = "#123456"
`
	if err := os.WriteFile(filepath.Join(userDir, "dracula.toml"), []byte(partial), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	themes := LoadAll()
	var got Theme
	for _, th := range themes {
		if strings.EqualFold(th.Name, "dracula") {
			got = th
			break
		}
	}
	if got.Name == "" {
		t.Fatal("dracula theme not present after merge")
	}
	// User fields take effect.
	if got.Accent != "#ff00ff" {
		t.Errorf("Accent = %q, want #ff00ff", got.Accent)
	}
	if got.FG != "#123456" {
		t.Errorf("FG = %q, want #123456", got.FG)
	}
	// Built-in dracula has bright_fg "#f8f8f2" — must survive merge.
	if got.BrightFG != "#f8f8f2" {
		t.Errorf("built-in BrightFG should survive merge: got %q, want #f8f8f2", got.BrightFG)
	}
}

func TestLoadAllAddsUserOnlyTheme(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	userDir := filepath.Join(home, ".config", "cliamp", "themes")
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	// New theme (no built-in match): all six hex fields required.
	custom := `accent = "#abcdef"
bright_fg = "#ffffff"
fg = "#cccccc"
green = "#00ff00"
yellow = "#ffff00"
red = "#ff0000"`
	if err := os.WriteFile(filepath.Join(userDir, "mytheme.toml"), []byte(custom), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	themes := LoadAll()
	var found bool
	for _, th := range themes {
		if th.Name == "mytheme" {
			found = true
			if th.Accent != "#abcdef" {
				t.Errorf("Accent = %q, want #abcdef", th.Accent)
			}
		}
	}
	if !found {
		t.Error("user theme mytheme not loaded")
	}
}

func TestLoadAllSkipsPartialThemeWithoutBuiltinMatch(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	userDir := filepath.Join(home, ".config", "cliamp", "themes")
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	// Only accent set, no built-in "broken" exists — should be rejected.
	partial := `accent = "#ff0000"`
	if err := os.WriteFile(filepath.Join(userDir, "broken.toml"), []byte(partial), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	themes := LoadAll()
	for _, th := range themes {
		if th.Name == "broken" {
			t.Fatal("partial theme without built-in match should not be loaded")
		}
	}
}

func TestLoadAllSkipsInvalidHexInMerge(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	userDir := filepath.Join(home, ".config", "cliamp", "themes")
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	// All six fields present but red is invalid hex — valid fields merge,
	// invalid field is ignored (built-in value survives).
	bad := `accent = "#ff0000"
bright_fg = "#f8f8f2"
fg = "#6272a4"
green = "#50fa7b"
yellow = "#f1fa8c"
red = "not-a-color"`
	if err := os.WriteFile(filepath.Join(userDir, "dracula.toml"), []byte(bad), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	themes := LoadAll()
	var got Theme
	for _, th := range themes {
		if strings.EqualFold(th.Name, "dracula") {
			got = th
			break
		}
	}
	if got.Name == "" {
		t.Fatal("dracula theme not found after merge")
	}
	if got.Accent != "#ff0000" {
		t.Errorf("valid Accent should merge: got %q, want #ff0000", got.Accent)
	}
	if got.Red != "#ff5555" {
		t.Errorf("invalid Red should be ignored (built-in): got %q, want #ff5555", got.Red)
	}
}

func TestLoadAllIgnoresNonTomlFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	userDir := filepath.Join(home, ".config", "cliamp", "themes")
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(userDir, "notatheme.txt"), []byte("ignored"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	// Subdirectory should also be ignored.
	if err := os.MkdirAll(filepath.Join(userDir, "nested"), 0o755); err != nil {
		t.Fatalf("MkdirAll nested: %v", err)
	}

	themes := LoadAll()
	for _, th := range themes {
		if th.Name == "notatheme" || th.Name == "nested" {
			t.Errorf("non-toml entry %q leaked into LoadAll()", th.Name)
		}
	}
}

func TestLoadAllMissingUserDir(t *testing.T) {
	// HOME points at a dir where ~/.config/cliamp/themes doesn't exist.
	t.Setenv("HOME", t.TempDir())
	themes := LoadAll()
	if len(themes) == 0 {
		t.Error("LoadAll() with missing user dir should still return built-in themes")
	}
}
