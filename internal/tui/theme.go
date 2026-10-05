package tui

import (
	"fmt"
	"strings"
)

type Appearance string

const (
	FollowTerminal Appearance = "follow-terminal"
	Dark           Appearance = "dark"
	Light          Appearance = "light"
	TerminalNative Appearance = "terminal-native"
)

var appearances = []Appearance{Dark, Light, FollowTerminal, TerminalNative}
var appearanceNames = []string{"Dark", "Light", "Follow Terminal", "Terminal Native"}

func ParseAppearance(s string) (Appearance, error) {
	a := Appearance(s)
	for _, valid := range appearances {
		if a == valid {
			return a, nil
		}
	}
	return "", fmt.Errorf("unknown appearance %q; use dark, light, follow-terminal or terminal-native", s)
}

type palette struct{ foreground, background, accent, muted, gold string }

func (m Model) palette() palette {
	if m.noColor {
		return palette{}
	}
	native := m.appearance == TerminalNative || (m.appearance == FollowTerminal && !m.backgroundKnown)
	if native {
		return palette{accent: "\x1b[1m", muted: ""}
	}
	dark := m.appearance == Dark || (m.appearance == FollowTerminal && m.terminalDark)
	if dark {
		return palette{foreground: fg(247, 241, 214), background: bg(2, 19, 33), accent: fg(244, 230, 181), muted: fg(73, 213, 236), gold: fg(249, 201, 80)}
	}
	return palette{foreground: fg(34, 52, 62), background: bg(246, 245, 237), accent: fg(95, 116, 125), muted: fg(20, 102, 117), gold: fg(133, 89, 19)}
}
func fg(r, g, b int) string { return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b) }
func bg(r, g, b int) string { return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b) }
func (m Model) appearanceStatus() string {
	switch m.appearance {
	case Dark:
		return "Dark"
	case Light:
		return "Light"
	case TerminalNative:
		return "Terminal Native   default colors"
	default:
		if !m.backgroundKnown {
			return "Follow Terminal   native fallback"
		}
		if m.terminalDark {
			return "Follow Terminal   dark"
		}
		return "Follow Terminal   light"
	}
}
func clean(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 32 || r == 127 || (r >= 0x80 && r <= 0x9f) || r == '\u2028' || r == '\u2029' {
			return ' '
		}
		return r
	}, s)
}

func (m Model) lightPalette() bool {
	return m.appearance == Light || m.appearance == FollowTerminal && m.backgroundKnown && !m.terminalDark
}
func (m Model) nativePalette() bool {
	return m.appearance == TerminalNative || m.appearance == FollowTerminal && !m.backgroundKnown
}

// Purpose colours accompany text and markers; they never replace them.
func (m Model) buttonColours(label string) (surface, edge, lower string) {
	if m.noColor {
		return "", "", ""
	}
	label = strings.ToLower(label)
	family := 0
	switch {
	case strings.Contains(label, "quit"):
		family = 4
	case strings.Contains(label, "back") || strings.Contains(label, "cancel") || label == "index":
		family = 1
	case strings.Contains(label, "appearance") || strings.Contains(label, "theme") || strings.Contains(label, "default"):
		family = 2
	case strings.Contains(label, "refresh") || label == "read":
		family = 3
	}
	darkEdges := [][3]int{{99, 202, 183}, {151, 174, 186}, {183, 157, 224}, {118, 180, 226}, {227, 141, 145}}
	lightEdges := [][3]int{{25, 110, 97}, {73, 91, 106}, {111, 73, 158}, {40, 101, 151}, {155, 66, 78}}
	edgeRGB := darkEdges[family]
	if m.lightPalette() {
		edgeRGB = lightEdges[family]
	}
	edge = fg(edgeRGB[0], edgeRGB[1], edgeRGB[2])
	surface = edge
	lower = fg(edgeRGB[0]*3/4, edgeRGB[1]*3/4, edgeRGB[2]*3/4)
	return
}

func (m Model) controlColours(id int, label string) (string, string, string) {
	if m.settings && id >= 0 && id <= 4 {
		label = "Appearance"
	}
	return m.buttonColours(label)
}
