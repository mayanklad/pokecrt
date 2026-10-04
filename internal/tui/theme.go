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

type palette struct{ foreground, background, accent, muted string }

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
		return palette{foreground: fg(229, 239, 221), background: bg(6, 22, 36), accent: fg(239, 207, 109), muted: fg(111, 191, 207)}
	}
	return palette{foreground: fg(30, 54, 48), background: bg(241, 237, 220), accent: fg(132, 86, 18), muted: fg(67, 102, 83)}
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
		return "Terminal Native · default colors"
	default:
		if !m.backgroundKnown {
			return "Follow Terminal · native fallback"
		}
		if m.terminalDark {
			return "Follow Terminal · dark"
		}
		return "Follow Terminal · light"
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
