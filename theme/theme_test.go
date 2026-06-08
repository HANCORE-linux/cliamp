package theme

import (
	"strings"
	"testing"
)

func TestDefault(t *testing.T) {
	d := Default()
	if d.Name != DefaultName {
		t.Errorf("Name = %q, want %q", d.Name, DefaultName)
	}
	if !d.IsDefault() {
		t.Error("IsDefault() should be true for default theme")
	}
}

func TestIsDefault(t *testing.T) {
	tests := []struct {
		name  string
		theme Theme
		want  bool
	}{
		{"empty hex values", Theme{Name: "Default"}, true},
		{"has accent", Theme{Name: "Custom", Accent: "#ff0000"}, false},
		{"has green", Theme{Name: "Custom", Green: "#00ff00"}, false},
		{"has bright fg", Theme{Name: "Custom", BrightFG: "#ffffff"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.theme.IsDefault(); got != tt.want {
				t.Errorf("IsDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParse(t *testing.T) {
	input := `# Solarized Dark theme
accent = "#268bd2"
bright_fg = "#93a1a1"
fg = "#839496"
green = "#859900"
yellow = "#b58900"
red = "#dc322f"
`
	th, err := Parse("solarized-dark", strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if th.Name != "solarized-dark" {
		t.Errorf("Name = %q, want solarized-dark", th.Name)
	}
	if th.Accent != "#268bd2" {
		t.Errorf("Accent = %q, want #268bd2", th.Accent)
	}
	if th.BrightFG != "#93a1a1" {
		t.Errorf("BrightFG = %q, want #93a1a1", th.BrightFG)
	}
	if th.FG != "#839496" {
		t.Errorf("FG = %q, want #839496", th.FG)
	}
	if th.Green != "#859900" {
		t.Errorf("Green = %q, want #859900", th.Green)
	}
	if th.Yellow != "#b58900" {
		t.Errorf("Yellow = %q, want #b58900", th.Yellow)
	}
	if th.Red != "#dc322f" {
		t.Errorf("Red = %q, want #dc322f", th.Red)
	}
}

func TestParseSkipsComments(t *testing.T) {
	input := `# comment
accent = "#ff0000"
# another comment
`
	th, err := Parse("test", strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if th.Accent != "#ff0000" {
		t.Errorf("Accent = %q, want #ff0000", th.Accent)
	}
}

func TestParseEmpty(t *testing.T) {
	th, err := Parse("empty", strings.NewReader(""))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if !th.IsDefault() {
		t.Error("empty parse should produce default-like theme")
	}
}

func TestParseStripsQuotes(t *testing.T) {
	// Both single and double quotes should be stripped
	input := `accent = '#ff0000'
fg = "#00ff00"
`
	th, err := Parse("test", strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if th.Accent != "#ff0000" {
		t.Errorf("Accent = %q, want #ff0000", th.Accent)
	}
	if th.FG != "#00ff00" {
		t.Errorf("FG = %q, want #00ff00", th.FG)
	}
}

func TestParsedThemeNotDefault(t *testing.T) {
	input := `accent = "#ff0000"`
	th, err := Parse("custom", strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if th.IsDefault() {
		t.Error("theme with accent should not be IsDefault()")
	}
}

func TestValidHex(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"#fff", true},
		{"#FFf", true},
		{"#aabbcc", true},
		{"#AABBCC", true},
		{"#aabbccdd", true},
		{"", false},
		{"aabbcc", false},
		{"#xyz", false},
		{"#12", false},
		{"#1234567", false},
		{"#gggggg", false},
	}
	for _, tt := range tests {
		got := validHex(tt.s)
		if got != tt.want {
			t.Errorf("validHex(%q) = %v, want %v", tt.s, got, tt.want)
		}
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		theme   Theme
		wantErr bool
		errMsg  string
	}{
		{
			name:    "all valid fields",
			theme:   Theme{"test", "#bd93f9", "#f8f8f2", "#6272a4", "#50fa7b", "#f1fa8c", "#ff5555"},
			wantErr: false,
		},
		{
			name:    "partial fields",
			theme:   Theme{"partial", "#ff0000", "", "", "", "", ""},
			wantErr: true,
		},
		{
			name:    "all empty",
			theme:   Theme{"empty", "", "", "", "", "", ""},
			wantErr: true,
		},
		{
			name:    "invalid hex one field",
			theme:   Theme{"bad", "#ff0000", "#f8f8f2", "#6272a4", "#50fa7b", "#f1fa8c", "not-a-color"},
			wantErr: true,
		},
		{
			name:    "error contains field names",
			theme:   Theme{"broken", "#ff0000", "", "", "", "", ""},
			wantErr: true,
			errMsg:  "bright_fg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.theme.Validate()
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.errMsg != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestMerge(t *testing.T) {
	builtin := Theme{"dracula", "#bd93f9", "#f8f8f2", "#6272a4", "#50fa7b", "#f1fa8c", "#ff5555"}

	tests := []struct {
		name  string
		src   Theme
		check func(t *testing.T, dst Theme)
	}{
		{
			name: "partial override",
			src:  Theme{"dracula", "#ff00ff", "", "#123456", "", "", ""},
			check: func(t *testing.T, dst Theme) {
				if dst.Accent != "#ff00ff" {
					t.Errorf("Accent = %q, want #ff00ff", dst.Accent)
				}
				if dst.FG != "#123456" {
					t.Errorf("FG = %q, want #123456", dst.FG)
				}
				if dst.BrightFG != "#f8f8f2" {
					t.Errorf("BrightFG should survive: got %q", dst.BrightFG)
				}
				if dst.Green != "#50fa7b" {
					t.Errorf("Green should survive: got %q", dst.Green)
				}
				if dst.Yellow != "#f1fa8c" {
					t.Errorf("Yellow should survive: got %q", dst.Yellow)
				}
				if dst.Red != "#ff5555" {
					t.Errorf("Red should survive: got %q", dst.Red)
				}
			},
		},
		{
			name: "ignores invalid hex",
			src:  Theme{"dracula", "#aa0000", "", "", "", "", "not-a-color"},
			check: func(t *testing.T, dst Theme) {
				if dst.Accent != "#aa0000" {
					t.Errorf("valid Accent should merge: got %q", dst.Accent)
				}
				if dst.Red != "#ff5555" {
					t.Errorf("invalid Red should be ignored: got %q, want #ff5555", dst.Red)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dst := builtin
			merge(&dst, tt.src)
			tt.check(t, dst)
		})
	}
}
