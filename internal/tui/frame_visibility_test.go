package tui

import (
	"github.com/charmbracelet/x/ansi"
	"github.com/mayanklad/pokecrt/internal/trainer"
	"strings"
	"testing"
)

func TestInformationControlsRequireOverflowAndStayInline(t *testing.T) {
	for _, section := range []int{2, 3} {
		for _, size := range [][2]int{{40, 12}, {64, 24}, {120, 40}, {190, 50}} {
			m := trainerReviewModel(size[0], size[1])
			m.section = section
			if section == 3 {
				m = achievementControlsModel(size[0], size[1])
			}
			rows := strings.Split(ansi.Strip(m.View().Content), "\n")
			for _, control := range m.activityControls() {
				panel := -1
				switch control.id {
				case 40, 42, 43:
					panel = 0
				case 41, 44, 45:
					panel = 1
				}
				if panel >= 0 {
					if m.activityPanelMaxScroll(panel) == 0 {
						t.Fatal("unnecessary panel control")
					}
					if control.y != m.dexGeometry().y+5+m.activityBodyHeight()-1 {
						t.Fatal("control outside frame")
					}
				}
				if panel >= 0 || control.id == 22 || control.id == 16 || control.id == 17 {
					cells := []rune(rows[control.y])
					if cells[control.x-1] == ' ' || cells[control.x+control.w] == ' ' {
						t.Fatalf("%v section %d: interrupted frame beside %d", size, section, control.id)
					}
				}
			}
		}
	}
	m := trainerReviewModel(190, 60)
	for _, c := range m.activityControls() {
		if c.id >= 40 && c.id <= 45 {
			t.Fatal("controls on fitting content")
		}
	}
}

func TestTrainerFormsHaveNoHelpTabStop(t *testing.T) {
	for _, screen := range []setupScreen{profilesScreen, createScreen} {
		m := activityModel()
		m.width, m.height, m.screen, m.focus = 40, 16, screen, 0
		for range 80 {
			m.moveFocus(1)
			if m.focus == 90 {
				t.Fatal("hidden Help tab stop")
			}
		}
		if strings.Contains(ansi.Strip(m.View().Content), "Help") {
			t.Fatal("redundant Help")
		}
	}
}

func TestTrainerFormButtonFocusAndLabels(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {40, 16}, {64, 24}, {120, 40}} {
		for _, screen := range []setupScreen{createScreen, profilesScreen} {
			m := activityModel()
			m.width, m.height, m.screen = size[0], size[1], screen
			for _, mono := range []bool{false, true} {
				m.noColor = mono
				ids := []int{1, 2, 3, 4, 5, 6}
				if screen == createScreen {
					ids = append(ids, 7)
				}
				for _, id := range ids {
					m.focus = id
					v := ansi.Strip(m.View().Content)
					if strings.Count(v, "▶") != 1 {
						t.Fatalf("%v screen %d focus %d mono %v: ambiguous focus", size, screen, id, mono)
					}
				}
			}
		}
	}
}

func TestTrainerActiveMarkerSurvivesOtherSelection(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {64, 24}, {120, 40}} {
		m := setupModel(t)
		m.width, m.height, m.screen = size[0], size[1], profilesScreen
		m.snapshot.Entries = []trainer.Profile{{Name: "Mayank", Active: true}, {Name: "Trainer2"}}
		m.selected = 1
		m.focus = 0
		v := ansi.Strip(m.View().Content)
		if size[1] > 12 && !strings.Contains(v, "● Mayank") {
			t.Fatal("active dot lost when another trainer selected")
		}
		if !strings.Contains(v, "▶   Trainer2") {
			t.Fatal("selected trainer pointer missing")
		}
		if strings.Count(v, "●") > 1 {
			t.Fatal("selection also uses active marker")
		}
		rows := strings.Split(v, "\n")
		if size == [2]int{40, 12} && !strings.Contains(rows[3], "2–2 of 2") {
			t.Fatal("list counter not in header")
		}
		if size == [2]int{40, 12} && []rune(rows[5])[19] != ' ' {
			t.Fatal("list counter overlaps action row")
		}
	}
}
