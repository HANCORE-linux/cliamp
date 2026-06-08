// Package theme handles loading and parsing color themes from TOML files.
package theme

import (
	"bufio"
	"cmp"
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"cliamp/internal/appdir"
)

//go:embed themes/*.toml
var builtinThemes embed.FS

// DefaultName is the display name for the built-in ANSI fallback theme.
const DefaultName = "Default - Terminal colors"

// Theme holds a named color scheme with hex color values.
type Theme struct {
	Name     string
	Accent   string // hex
	BrightFG string
	FG       string
	Green    string
	Yellow   string
	Red      string
}

// IsDefault returns true if this is the sentinel default theme (no hex values).
func (t Theme) IsDefault() bool {
	return t.Accent == "" && t.Green == "" && t.BrightFG == ""
}

// validHex reports whether s is a valid hex color like "#fff" or "#aabbcc".
func validHex(s string) bool {
	if len(s) < 4 || s[0] != '#' {
		return false
	}
	for _, r := range s[1:] {
		if !(r >= '0' && r <= '9') && !(r >= 'a' && r <= 'f') && !(r >= 'A' && r <= 'F') {
			return false
		}
	}
	n := len(s) - 1
	return n == 3 || n == 6 || n == 8
}

// Validate checks that all six color fields are non-empty hex values.
func (t Theme) Validate() error {
	missing := make([]string, 0, 6)
	if !validHex(t.Accent) {
		missing = append(missing, "accent")
	}
	if !validHex(t.BrightFG) {
		missing = append(missing, "bright_fg")
	}
	if !validHex(t.FG) {
		missing = append(missing, "fg")
	}
	if !validHex(t.Green) {
		missing = append(missing, "green")
	}
	if !validHex(t.Yellow) {
		missing = append(missing, "yellow")
	}
	if !validHex(t.Red) {
		missing = append(missing, "red")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing or invalid hex fields: %s", strings.Join(missing, ", "))
	}
	return nil
}

// merge applies every non-empty valid-hex field from src onto dst.
// Fields with invalid hex values are silently ignored so that a corrupt
// or partial user file only changes the colours it explicitly sets.
func merge(dst *Theme, src Theme) {
	if validHex(src.Accent) {
		dst.Accent = src.Accent
	}
	if validHex(src.BrightFG) {
		dst.BrightFG = src.BrightFG
	}
	if validHex(src.FG) {
		dst.FG = src.FG
	}
	if validHex(src.Green) {
		dst.Green = src.Green
	}
	if validHex(src.Yellow) {
		dst.Yellow = src.Yellow
	}
	if validHex(src.Red) {
		dst.Red = src.Red
	}
}

// Default returns a sentinel "Default" theme with empty hex values,
// signaling that ANSI fallback colors should be used.
func Default() Theme {
	return Theme{Name: DefaultName}
}

// Parse reads flat TOML key=value lines from r and returns a Theme.
// Uses the same manual parsing approach as config/config.go.
func Parse(name string, r io.Reader) (Theme, error) {
	t := Theme{Name: name}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		val = strings.Trim(val, `"'`)

		switch key {
		case "accent":
			t.Accent = val
		case "bright_fg":
			t.BrightFG = val
		case "fg":
			t.FG = val
		case "red":
			t.Red = val
		case "yellow":
			t.Yellow = val
		case "green":
			t.Green = val
		}
	}
	return t, scanner.Err()
}

// LoadAll loads built-in themes and user custom themes from
// ~/.config/cliamp/themes/*.toml. User themes override built-in
// themes with the same name. Returns a sorted list.
func LoadAll() []Theme {
	themes := make(map[string]Theme)

	// Load embedded built-in themes (lower priority).
	loadBuiltin(themes)

	// Load user custom themes (override built-in if same name).
	dir, err := appdir.Dir()
	if err == nil {
		loadUserDir(filepath.Join(dir, "themes"), themes)
	}

	// Sort by name.
	result := make([]Theme, 0, len(themes))
	for _, t := range themes {
		result = append(result, t)
	}
	slices.SortFunc(result, func(a, b Theme) int {
		return cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return result
}

// loadBuiltin parses the embedded theme TOML files.
func loadBuiltin(themes map[string]Theme) {
	entries, err := builtinThemes.ReadDir("themes")
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".toml") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".toml")
		f, err := builtinThemes.Open("themes/" + e.Name())
		if err != nil {
			continue
		}
		t, err := Parse(name, f)
		f.Close()
		if err != nil {
			continue
		}
		themes[strings.ToLower(name)] = t
	}
}

// loadUserDir loads themes from ~/.config/cliamp/themes/*.toml.
//
// If a user file matches a built-in theme name its fields are merged onto
// the built-in, so that a partial override (e.g. only "accent = ...")
// changes only that colour.  Themes without a built-in match require all
// six hex fields to pass validation.  Invalid hex values in either mode are
// silently ignored — the corresponding built-in (or zero) value survives.
func loadUserDir(dir string, themes map[string]Theme) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".toml") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".toml")
		path := filepath.Join(dir, e.Name())
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		t, err := Parse(name, f)
		f.Close()
		if err != nil {
			continue
		}
		key := strings.ToLower(name)
		if _, exists := themes[key]; exists {
			// Merge onto the existing (built-in) theme.
			existing := themes[key]
			merge(&existing, t)
			themes[key] = existing
		} else {
			// New theme (no built-in match): all six fields required.
			if err := t.Validate(); err != nil {
				continue
			}
			themes[key] = t
		}
	}
}
