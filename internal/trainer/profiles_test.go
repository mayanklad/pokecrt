package trainer

import (
	"errors"
	"strings"
	"testing"
)

func TestNameNormalizationAndFolding(t *testing.T) {
	for _, tc := range []struct{ raw, display, key string }{
		{"  Professor Oak  ", "Professor Oak", "professor oak"},
		{"Élodie", "Élodie", "élodie"},
		{"E\u0301LODIE", "ÉLODIE", "élodie"},
		{"Straße", "Straße", "strasse"},
		{"STRASSE", "STRASSE", "strasse"},
		{"Σίσυφος", "Σίσυφος", "σίσυφοσ"},
		{"雪", "雪", "雪"},
		{"../Oak'; DROP TABLE trainers;--", "../Oak'; DROP TABLE trainers;--", "../oak'; drop table trainers;--"},
		{"--help", "--help", "--help"},
		{strings.Repeat("雪", 32), strings.Repeat("雪", 32), strings.Repeat("雪", 32)},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			n, err := ParseName(tc.raw)
			if err != nil || n.Display() != tc.display || n.Key() != tc.key {
				t.Fatalf("got %q/%q, %v; want %q/%q", n.Display(), n.Key(), err, tc.display, tc.key)
			}
		})
	}
}

func TestInvalidNames(t *testing.T) {
	for _, raw := range []string{"", "   ", strings.Repeat("雪", 33), "A\tB", "A\nB", "A\rB", "A\x00B", "A\x1bB", "A\u0085B", "A\u2028B", "A\u2029B", string([]byte{0xff})} {
		if _, err := ParseName(raw); !errors.Is(err, ErrInvalidName) {
			t.Errorf("accepted %q: %v", raw, err)
		}
	}
	// The limit counts trimmed input code points, before NFC composition.
	if _, err := ParseName(strings.Repeat("e\u0301", 17)); !errors.Is(err, ErrInvalidName) {
		t.Fatal("input longer than 32 code points accepted")
	}
}
