package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"github.com/mayanklad/pokecrt/internal/catalog"
	"github.com/mayanklad/pokecrt/internal/sprite"
	"github.com/mayanklad/pokecrt/internal/storage"
	"github.com/mayanklad/pokecrt/internal/trainer"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestDexFlagsAndHelpDoNotCreateState(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "missing")
	t.Setenv("POKECRT_DATA_DIR", dir)
	for _, tc := range []struct {
		args      []string
		want      string
		forbidden []string
	}{
		{[]string{"--help"}, "The summary shows", []string{"--type ", "--gender "}},
		{[]string{"list", "--help"}, "--type-any", []string{"--name ", "--gender ", "--shiny "}},
		{[]string{"show", "--help"}, "--number", []string{"--type ", "--unseen ", "--stage "}},
	} {
		var out, stderr bytes.Buffer
		if code := runDex(tc.args, &out, &stderr); code != 0 || stderr.Len() != 0 || !strings.Contains(out.String(), tc.want) {
			t.Fatal(tc.args, code, out.String(), stderr.String())
		}
		for _, v := range tc.forbidden {
			if strings.Contains(out.String(), v) {
				t.Fatalf("%v advertises unsupported flag %s", tc.args, v)
			}
		}
	}
	for _, args := range [][]string{{"list", "--unseen", "--type", "fire"}, {"list", "--seen", "--unseen"}, {"list", "--name", "charizard"}, {"list", "--form", "mega-x"}, {"show"}, {"show", "--number", "6", "--name", "charizard"}, {"show", "--number", "0"}, {"show", "--number", "6", "--form", "invented"}, {"show", "--number", "6", "--output", "sprite"}, {"list", "--type", "fire", "--type", "fire"}, {"list", "--gen", "1,,2"}, {"show", "--number", "6", "--gender", "other"}} {
		var out, err bytes.Buffer
		if code := runDex(args, &out, &err); code != 2 || out.Len() != 0 || err.Len() == 0 {
			t.Fatal(args, code, out.String(), err.String())
		}
	}
	var out, err bytes.Buffer
	if code := runDex(nil, &out, &err); code != 1 || out.Len() != 0 {
		t.Fatal(code, err.String())
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("created state", err)
	}
}
func TestDexCLICollectedAndLockedViews(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("POKECRT_DATA_DIR", dir)
	path, err := storage.ResolvePath()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	r, err := storage.Initialize(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	name, _ := trainer.ParseName("Dex Tester")
	p, err := r.CreateProfile(ctx, name, 1)
	if err != nil {
		t.Fatal(err)
	}
	s, _ := catalog.ByNumber(6)
	for _, f := range s.Forms {
		if f.ID == "mega-x" {
			s.Forms = []catalog.Form{f}
			break
		}
	}
	pool, err := trainer.NewPool([]catalog.Species{s}, func(k catalog.VariantKey) bool { _, ok := sprite.Lookup(k); return ok })
	if err != nil {
		t.Fatal(err)
	}
	choice, err := pool.Select(func(int) (int, error) { return 0, nil })
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.RecordEncounter(ctx, p.ID, choice, 10); err != nil {
		t.Fatal(err)
	}
	r.Close()
	beforeBytes, _ := os.ReadFile(path)
	beforeHash := sha256.Sum256(beforeBytes)
	checks := []struct {
		args         []string
		want, hidden string
	}{{nil, "Species: 1 / 1017", "Mega X"}, {[]string{"list", "--seen"}, "Charizard", "Charmander"}, {[]string{"list", "--unseen", "--gen", "1"}, "#004 ?????", "Charizard"}, {[]string{"show", "--number", "4", "--form", "mega-x"}, "#004 ????? - UNDISCOVERED", "Charmander"}, {[]string{"show", "--name", "charizard"}, "LOCKED FORM", "Fire / Flying"}, {[]string{"show", "--name", "charizard", "--form", "mega-x"}, "LOCKED VARIANT", "\x1b["}, {[]string{"show", "--name", "charizard", "--form", "mega-x", "--shiny"}, "Appearance: Mega X", "Charmander"}}
	for _, c := range checks {
		var out, stderr bytes.Buffer
		if code := runDex(c.args, &out, &stderr); code != 0 || !strings.Contains(out.String(), c.want) || strings.Contains(out.String(), c.hidden) {
			t.Fatalf("%v code=%d\n%s\n%s", c.args, code, out.String(), stderr.String())
		}
	}
	t.Setenv("NO_COLOR", "1")
	var out, stderr bytes.Buffer
	if code := runDex([]string{"show", "--number", "6", "--form", "mega-x", "--shiny"}, &out, &stderr); code != 0 || strings.Contains(out.String(), "\x1b[") {
		t.Fatal(code, stderr.String())
	}
	var pipeErr bytes.Buffer
	if code := runDex(nil, failingWriter{syscall.EPIPE}, &pipeErr); code != 0 || pipeErr.Len() != 0 {
		t.Fatal("broken pipe", code, pipeErr.String())
	}
	afterBytes, _ := os.ReadFile(path)
	if sha256.Sum256(afterBytes) != beforeHash {
		t.Fatal("browsing changed database bytes")
	}

}
