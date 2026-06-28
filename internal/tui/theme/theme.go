package theme

import (
	"charm.land/glamour/v2/ansi"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
)

var (
	Background         = adaptive("#f8f8f8", "#1c2027")
	BackgroundSubtle   = adaptive("#f3f3f3", "#21262e")
	BackgroundElevated = adaptive("#ececec", "#2a2f38")
	Foreground         = adaptive("#2a2a2a", "#cdd2da")
	ForegroundMuted    = adaptive("#686868", "#6b7280")
	ForegroundEmphasis = adaptive("#111111", "#f1f3f5")
	BorderDim          = adaptive("#ddddde", "#2c313a")
	BorderNormal       = adaptive("#c9c9ca", "#3a4049")
	Primary            = adaptive("#3b7dd8", "#56b6c2")
	Secondary          = adaptive("#7b5bb6", "#7fb3ff")
	Accent             = adaptive("#b354d4", "#56b6c2")
	Success            = adaptive("#3d9a57", "#7fd1b8")
	Warning            = adaptive("#d68c27", "#f5a742")
	Error              = adaptive("#d1383d", "#e06c75")
)

func adaptive(light, dark string) compat.AdaptiveColor {
	return compat.AdaptiveColor{Light: lipgloss.Color(light), Dark: lipgloss.Color(dark)}
}

func Base() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(Foreground)
}

func Surface() lipgloss.Style {
	return Base().Background(BackgroundSubtle)
}

func Padded() lipgloss.Style {
	return Base().Padding(0, 1)
}

func MarkdownStyle() ansi.StyleConfig {
	if compat.HasDarkBackground {
		return darkMarkdownStyle()
	}
	return lightMarkdownStyle()
}

func lightMarkdownStyle() ansi.StyleConfig {
	return ansi.StyleConfig{
		Document: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: str("#2a2a2a")}},
		Paragraph: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
			Color: str("#2a2a2a"),
		}, Margin: uintp(1)},
		BlockQuote: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
			Color:       str("#3b7dd8"),
			BlockPrefix: "▌ ",
		}, Margin: uintp(1)},
		Heading: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
			Color: str("#7b5bb6"),
			Bold:  boolp(true),
		}, Margin: uintp(1)},
		Strong: ansi.StylePrimitive{Color: str("#111111"), Bold: boolp(true)},
		Emph:   ansi.StylePrimitive{Color: str("#555555"), Italic: boolp(true)},
		Link:   ansi.StylePrimitive{Color: str("#3b7dd8"), Underline: boolp(true)},
		Code: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
			Color:           str("#3d9a57"),
			BackgroundColor: str("#e8e8e8"),
		}},
		CodeBlock: ansi.StyleCodeBlock{
			StyleBlock: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
				Color:           str("#2a2a2a"),
				BackgroundColor: str("#e8e8e8"),
			}},
			Theme: "github-light",
		},
		List:        ansi.StyleList{StyleBlock: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: str("#2a2a2a")}}, LevelIndent: 2},
		Item:        ansi.StylePrimitive{Color: str("#3b7dd8"), BlockPrefix: "• "},
		Enumeration: ansi.StylePrimitive{Color: str("#8a8a8a"), BlockPrefix: ". "},
	}
}

func darkMarkdownStyle() ansi.StyleConfig {
	return ansi.StyleConfig{
		Document: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: str("#cdd2da")}},
		Paragraph: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
			Color: str("#cdd2da"),
		}, Margin: uintp(1)},
		BlockQuote: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
			Color:       str("#5c9cf5"),
			BlockPrefix: "▌ ",
		}, Margin: uintp(1)},
		Heading: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
			Color: str("#5c9cf5"),
			Bold:  boolp(true),
		}, Margin: uintp(1)},
		Strong: ansi.StylePrimitive{Color: str("#f1f1f1"), Bold: boolp(true)},
		Emph:   ansi.StylePrimitive{Color: str("#e5c07b"), Italic: boolp(true)},
		Link:   ansi.StylePrimitive{Color: str("#56b6c2"), Underline: boolp(true)},
		Code: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
			Color:           str("#7fd88f"),
			BackgroundColor: str("#161616"),
		}},
		CodeBlock: ansi.StyleCodeBlock{
			StyleBlock: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
				Color:           str("#cfcfcf"),
				BackgroundColor: str("#111111"),
			}},
			Theme: "github-dark",
		},
		List:        ansi.StyleList{StyleBlock: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: str("#cdd2da")}}, LevelIndent: 2},
		Item:        ansi.StylePrimitive{Color: str("#56b6c2"), BlockPrefix: "• "},
		Enumeration: ansi.StylePrimitive{Color: str("#6b7280"), BlockPrefix: ". "},
	}
}

func str(v string) *string { return &v }

func boolp(v bool) *bool { return &v }

func uintp(v uint) *uint { return &v }
