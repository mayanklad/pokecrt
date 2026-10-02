package cli

import (
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/mayanklad/pokecrt/internal/catalog"
)

type rootOptions struct{ help, version bool }

// onceValue lets FlagSet parse values while enforcing one logical occurrence.
// Aliases share this wrapper, so --help and -h cannot bypass the duplicate rule.
type onceValue struct {
	flag.Value
	name          string
	seen, boolean bool
}

func (v *onceValue) String() string {
	if v == nil || v.Value == nil {
		return ""
	}
	return v.Value.String()
}

func (v *onceValue) Set(s string) error {
	if v.seen {
		return fmt.Errorf("flag --%s may appear only once", v.name)
	}
	v.seen = true
	s = strings.TrimSpace(s)
	if v.boolean {
		s = strings.ToLower(s)
		if s != "true" && s != "false" {
			return fmt.Errorf("flag --%s requires true or false", v.name)
		}
	} else if s == "" {
		return fmt.Errorf("flag --%s cannot be empty", v.name)
	}
	return v.Value.Set(s)
}
func (v *onceValue) IsBoolFlag() bool { return v.boolean }
func trackFlag(fs *flag.FlagSet, name string) {
	f := fs.Lookup(name)
	boolean := false
	if b, ok := f.Value.(interface{ IsBoolFlag() bool }); ok {
		boolean = b.IsBoolFlag()
	}
	f.Value = &onceValue{Value: f.Value, name: name, boolean: boolean}
}
func parseFlags(fs *flag.FlagSet, args []string) error {
	if err := parseFlagSet(fs, args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected positional argument %q", fs.Arg(0))
	}
	return nil
}

func parseFlagSet(fs *flag.FlagSet, args []string) error {
	// Public long flags use --; FlagSet otherwise also accepts -name.
	// Inspect flag spellings only, never string values or positionals.
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" || arg == "-" || !strings.HasPrefix(arg, "-") {
			break
		}
		spelling, _, hasValue := strings.Cut(arg, "=")
		if strings.HasPrefix(spelling, "-") && !strings.HasPrefix(spelling, "--") && spelling != "-h" {
			return fmt.Errorf("unknown flag %q", spelling)
		}
		f := fs.Lookup(strings.TrimLeft(spelling, "-"))
		if f != nil && !hasValue {
			boolean, ok := f.Value.(interface{ IsBoolFlag() bool })
			if !ok || !boolean.IsBoolFlag() {
				i++ // FlagSet owns parsing and validation of this value.
			}
		}
	}
	return fs.Parse(args)
}
func parseRootFlags(args []string) (rootOptions, error) {
	var o rootOptions
	fs := flag.NewFlagSet("pokecrt", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.BoolVar(&o.help, "help", false, "Show help")
	trackFlag(fs, "help")
	fs.Var(fs.Lookup("help").Value, "h", "Show help")
	fs.BoolVar(&o.version, "version", false, "Show version")
	trackFlag(fs, "version")
	err := parseFlags(fs, args)
	return o, err
}

type selectorValues struct {
	gen, types, typeAny, color, stage string
	selection                         catalog.Selection
}

func (v *selectorValues) register(fs *flag.FlagSet) {
	fs.StringVar(&v.selection.Name, "name", "", "Exact species name or generated alias")
	fs.StringVar(&v.gen, "gen", "", "Introduction generations, comma-separated")
	fs.StringVar(&v.types, "type", "", "All selected-form types, comma-separated")
	fs.StringVar(&v.typeAny, "type-any", "", "Any selected-form types, comma-separated")
	fs.StringVar(&v.color, "color", "", "Species Pokédex colors, comma-separated")
	fs.StringVar(&v.stage, "stage", "", "Evolution stages, comma-separated")
	fs.StringVar(&v.selection.Form, "form", "standard", "Exact form slug")
	fs.StringVar(&v.selection.Gender, "gender", "", "Distinct visual gender: male or female")
	fs.BoolVar(&v.selection.Shiny, "shiny", false, "Select shiny artwork")
	fs.BoolVar(&v.selection.Legendary, "legendary", false, "Require source legendary status")
	fs.BoolVar(&v.selection.Mythical, "mythical", false, "Require source mythical status")
	fs.BoolVar(&v.selection.Baby, "baby", false, "Require source baby status")
	for _, name := range []string{"name", "gen", "type", "type-any", "color", "stage", "form", "gender", "shiny", "legendary", "mythical", "baby"} {
		trackFlag(fs, name)
	}
}
func splitValues(raw, name string) ([]string, error) {
	if raw == "" {
		return nil, nil
	}
	values := strings.Split(raw, ",")
	for i, s := range values {
		values[i] = strings.ToLower(strings.TrimSpace(s))
		if values[i] == "" {
			return nil, fmt.Errorf("empty --%s list element", name)
		}
	}
	return values, nil
}
func numericValues(raw, name string) ([]int, error) {
	values, err := splitValues(raw, name)
	if err != nil {
		return nil, err
	}
	result := []int{}
	for _, s := range values {
		for _, r := range s {
			if r < '0' || r > '9' {
				return nil, fmt.Errorf("--%s requires positive decimal integers", name)
			}
		}
		n, err := strconv.Atoi(s)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("invalid --%s value %q", name, s)
		}
		result = append(result, n)
	}
	return result, nil
}
func (v selectorValues) validate() (catalog.Selection, error) {
	q := v.selection
	var err error
	if q.Generations, err = numericValues(v.gen, "gen"); err != nil {
		return q, err
	}
	if q.Stages, err = numericValues(v.stage, "stage"); err != nil {
		return q, err
	}
	if q.Types, err = splitValues(v.types, "type"); err != nil {
		return q, err
	}
	if q.TypesAny, err = splitValues(v.typeAny, "type-any"); err != nil {
		return q, err
	}
	if q.Colors, err = splitValues(v.color, "color"); err != nil {
		return q, err
	}
	return catalog.ValidateSelection(q)
}
