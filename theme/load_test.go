package theme

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAllIncludesBuiltinThemes(t *testing.T) {
	t.Setenv("CLIAMP_CONFIG_DIR", filepath.Join(t.TempDir(), "empty"))

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
	t.Setenv("CLIAMP_CONFIG_DIR", filepath.Join(t.TempDir(), "empty"))

	themes := LoadAll()
	for i := 1; i < len(themes); i++ {
		a := strings.ToLower(themes[i-1].Name)
		b := strings.ToLower(themes[i].Name)
		if a > b {
			t.Errorf("themes not sorted: %q before %q", themes[i-1].Name, themes[i].Name)
		}
	}
}

func TestLoadAllUserThemeScenarios(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		fileBody string
		check    func(t *testing.T, themes []Theme)
	}{
		{
			name:     "partial merge onto builtin",
			fileName: "dracula.toml",
			fileBody: "accent = \"#ff00ff\"\nfg = \"#123456\"\n",
			check: func(t *testing.T, themes []Theme) {
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
				if got.Accent != "#ff00ff" {
					t.Errorf("Accent = %q, want #ff00ff", got.Accent)
				}
				if got.FG != "#123456" {
					t.Errorf("FG = %q, want #123456", got.FG)
				}
				if got.BrightFG != "#f8f8f2" {
					t.Errorf("built-in BrightFG should survive: got %q, want #f8f8f2", got.BrightFG)
				}
			},
		},
		{
			name:     "full standalone theme accepted",
			fileName: "mytheme.toml",
			fileBody: `accent = "#abcdef"
bright_fg = "#ffffff"
fg = "#cccccc"
green = "#00ff00"
yellow = "#ffff00"
red = "#ff0000"`,
			check: func(t *testing.T, themes []Theme) {
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
			},
		},
		{
			name:     "partial without builtin match skipped",
			fileName: "broken.toml",
			fileBody: "accent = \"#ff0000\"",
			check: func(t *testing.T, themes []Theme) {
				for _, th := range themes {
					if th.Name == "broken" {
						t.Fatal("partial theme without built-in match should not be loaded")
					}
				}
			},
		},
		{
			name:     "invalid hex in merge ignored",
			fileName: "dracula.toml",
			fileBody: `accent = "#ff0000"
bright_fg = "#f8f8f2"
fg = "#6272a4"
green = "#50fa7b"
yellow = "#f1fa8c"
red = "not-a-color"`,
			check: func(t *testing.T, themes []Theme) {
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
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configDir := filepath.Join(t.TempDir(), ".config", "cliamp")
			t.Setenv("CLIAMP_CONFIG_DIR", configDir)
			userDir := filepath.Join(configDir, "themes")
			if err := os.MkdirAll(userDir, 0o755); err != nil {
				t.Fatalf("MkdirAll: %v", err)
			}
			if err := os.WriteFile(filepath.Join(userDir, tt.fileName), []byte(tt.fileBody), 0o644); err != nil {
				t.Fatalf("WriteFile: %v", err)
			}

			tt.check(t, LoadAll())
		})
	}
}

func TestLoadAllIgnoresNonTomlFiles(t *testing.T) {
	configDir := filepath.Join(t.TempDir(), ".config", "cliamp")
	t.Setenv("CLIAMP_CONFIG_DIR", configDir)
	userDir := filepath.Join(configDir, "themes")
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
	t.Setenv("CLIAMP_CONFIG_DIR", filepath.Join(t.TempDir(), "empty"))
	themes := LoadAll()
	if len(themes) == 0 {
		t.Error("LoadAll() with missing user dir should still return built-in themes")
	}
}
