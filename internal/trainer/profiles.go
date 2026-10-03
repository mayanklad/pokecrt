// Package trainer defines trainer identities and domain rules independent of
// SQL, command parsing and presentation.
package trainer

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

var (
	ErrInvalidName   = errors.New("invalid trainer name")
	ErrDuplicateName = errors.New("a trainer with that name already exists")
	ErrNotFound      = errors.New("trainer not found")
	ErrNoActive      = errors.New("no active trainer")
)

// Name can only be constructed through ParseName. Display spelling is NFC;
// lookup keys are NFC-normalized Unicode case folds, never filenames.
type Name struct{ display, key string }

func (n Name) Display() string { return n.display }
func (n Name) Key() string     { return n.key }

func ParseName(raw string) (Name, error) {
	if !utf8.ValidString(raw) {
		return Name{}, fmt.Errorf("%w: use valid UTF-8", ErrInvalidName)
	}
	trimmed := strings.TrimSpace(raw)
	if n := utf8.RuneCountInString(trimmed); n < 1 || n > 32 {
		return Name{}, fmt.Errorf("%w: use 1–32 Unicode characters, excluding leading/trailing spaces", ErrInvalidName)
	}
	for _, r := range trimmed {
		if unicode.IsControl(r) || r == '\u2028' || r == '\u2029' {
			return Name{}, fmt.Errorf("%w: controls, tabs, newlines and line separators are not allowed", ErrInvalidName)
		}
	}
	display := norm.NFC.String(trimmed)
	return Name{display: display, key: norm.NFC.String(cases.Fold().String(display))}, nil
}

type Profile struct {
	ID          int64
	Name        string
	CreatedAtMS int64
	Active      bool
}

// Profiles is the repository contract for profile operations. Selection changes
// never create encounters or discoveries. Lifecycle is managed by the caller.
type Profiles interface {
	CreateProfile(context.Context, Name, int64) (Profile, error)
	UseProfile(context.Context, Name) (Profile, error)
	ListProfiles(context.Context) ([]Profile, error)
	FindProfile(context.Context, Name) (Profile, error)
	ActiveProfile(context.Context) (Profile, error)
}
