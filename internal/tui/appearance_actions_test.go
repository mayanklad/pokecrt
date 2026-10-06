package tui

import (
	"context"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
)

func TestAppearanceActionLabelsAcrossPagesAndSizes(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {40, 16}, {47, 20}, {48, 20}, {56, 24}, {64, 24}, {100, 24}, {120, 40}} {
		for _, mono := range []bool{false, true} {
			for page := 0; page < 7; page++ {
				m := trainerReviewModel(size[0], size[1])
				m.noColor = mono
				switch page {
				case 0:
					m.screen = mainScreen
					m.focus = 4
				case 1:
					m.section = 1
					m.focus = 13
				case 2:
					m.section = 2
					m.focus = 13
				case 3:
					m.section = 3
					m.focus = 13
				case 4:
					m.screen = createScreen
					m.focus = 6
				case 5:
					m.screen = profilesScreen
					m.focus = 6
				case 6:
					m = dexModel(t, collectedDex())
					m.width, m.height = size[0], size[1]
					m.noColor = mono
					m.focus = 7
				}
				for _, mode := range appearances {
					m.appearance = mode
					view := ansi.Strip(m.View().Content)
					if !strings.Contains(view, "Appearance") || strings.Contains(view, "Theme") {
						t.Fatalf("%v page%d mono%v: inconsistent action\n%s", size, page, mono, view)
					}
					if strings.Count(view, "▶") != 1 {
						t.Fatalf("%v page%d: missing focus", size, page)
					}
				}
			}
		}
	}
}

func TestTownYouMarkerHasNoSideBars(t *testing.T) {
	for _, size := range [][2]int{{64, 13}, {90, 22}, {120, 30}} {
		m := New(context.Background(), nil, Dark, true)
		c := newCanvas(size[0], size[1])
		m.paintScene(c, 0, 0, size[0], size[1])
		row := strings.Split(ansi.Strip(c.content(m.palette())), "\n")[size[1]/2]
		marker := strings.Index(row, "◆ YOU")
		if marker < 0 {
			t.Fatal("missing location")
		}
		if strings.Contains(row, "│ ◆ YOU │") || strings.Contains(row, "| ◆ YOU |") {
			t.Fatal("boxed location marker")
		}
	}
}

func TestWrappedFooterNavigationAndMouseTargets(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {40, 13}, {40, 16}, {47, 20}, {40, 24}, {48, 20}, {56, 24}, {120, 40}} {
		for page := 0; page < 6; page++ {
			m := trainerReviewModel(size[0], size[1])
			ids := []int{12, 13, 11, 14}
			switch page {
			case 0:
				m = dexModel(t, collectedDex())
				m.width, m.height = size[0], size[1]
				ids = []int{6, 7, 5, 8}
			case 1:
				m.section = 1
			case 2:
				m.section = 2
			case 3:
				m.section = 3
			case 4:
				m.screen = createScreen
				ids = []int{1, 2, 6, 7}
			case 5:
				m.screen = dexSearchScreen
				ids = []int{1, 2, 6, 7}
			}
			var controls []dexControl
			if page == 0 {
				controls = m.dexControls()
			} else if page < 4 {
				controls = m.activityControls()
			} else {
				controls = m.createFocusTargets()
			}
			actions := map[int]dexControl{}
			for _, c := range controls {
				for _, id := range ids {
					if c.id == id {
						actions[id] = c
					}
				}
			}
			for _, id := range ids {
				c, ok := actions[id]
				if !ok {
					t.Fatalf("missing action %v page%d id%d", size, page, id)
				}
				if c.y < 7 {
					t.Fatalf("action left footer %v page%d id%d row%d", size, page, id, c.y)
				}
				if c.x < 1 || c.x+c.w >= m.width || c.y >= m.height-2 {
					t.Fatalf("action outside frame %v page%d id%d", size, page, id)
				}
				m.focus = id
				canvas := newCanvas(m.width, m.height)
				if page == 0 {
					m.paintDex(canvas)
				} else if page < 4 {
					m.paintActivity(canvas)
				} else {
					m.paintCreate(canvas)
				}
				hit := false
				for _, h := range canvas.hits {
					if h.id == id && h.y == c.y {
						hit = true
					}
				}
				if !hit {
					t.Fatalf("missing action mouse target %v page%d id%d", size, page, id)
				}
			}
			if !compactActionRows(m.width) {
				continue
			}
			cases := []struct {
				from int
				dir  string
				want int
			}{{ids[0], "right", ids[1]}, {ids[1], "left", ids[0]}, {ids[0], "down", ids[2]}, {ids[1], "down", ids[3]}, {ids[2], "up", ids[0]}, {ids[3], "up", ids[1]}, {ids[1], "right", ids[1]}, {ids[2], "left", ids[2]}, {ids[3], "down", ids[3]}}
			for _, route := range cases {
				next := m
				next.focus = route.from
				if page == 0 {
					next.navigateDexControl(route.dir)
				} else if page < 4 {
					next.activityMove(route.dir)
				} else {
					next.navigateSetup(route.dir)
				}
				if next.focus != route.want {
					t.Fatalf("%v page%d %d %s -> %d want%d", size, page, route.from, route.dir, next.focus, route.want)
				}
			}
		}
	}
}

func TestFormFooterFitsBetweenCompactAndOutlinedWidths(t *testing.T) {
	for width := 40; width <= 100; width++ {
		for _, height := range []int{12, 16, 24, 32} {
			for _, screen := range []setupScreen{createScreen, dexSearchScreen} {
				m := trainerReviewModel(width, height)
				m.screen = screen
				for _, id := range []int{1, 2, 6, 7} {
					m.focus = id
					c := newCanvas(width, height)
					m.paintCreate(c)
					found := false
					for _, h := range c.hits {
						if h.id == id {
							found = true
							if h.x < 1 || h.x+h.width >= width {
								t.Fatalf("clipped form action %dx%d id%d", width, height, id)
							}
						}
					}
					if !found {
						t.Fatal("missing form action")
					}
					rows := strings.Split(ansi.Strip(c.content(m.palette())), "\n")
					for _, row := range rows {
						if width > 88 || height > 28 {
							continue
						}
						if !strings.HasSuffix(row, "│") && !strings.HasSuffix(row, "╮") && !strings.HasSuffix(row, "╯") && !strings.HasSuffix(row, "┤") {
							t.Fatalf("form overwrites frame %dx%d", width, height)
						}
					}
				}
			}
		}
	}
}
