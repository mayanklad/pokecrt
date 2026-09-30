package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func fixtureLock(data []byte) sourceLock {
	digest := sha256.Sum256(data)
	return sourceLock{
		SchemaVersion: 1,
		Sources: []source{{
			ID: "fixture", Repository: "example/dataset",
			Revision: strings.Repeat("a", 40),
			Terms:    "Fixture only", Attribution: "Test fixture",
			Files: []sourceFile{{Path: "input.txt", Size: int64(len(data)), SHA256: hex.EncodeToString(digest[:])}},
		}},
	}
}

func TestRejectInvalidLocks(t *testing.T) {
	tests := []struct {
		name string
		edit func(*sourceLock)
	}{
		{"floating revision", func(l *sourceLock) { l.Sources[0].Revision = "main" }},
		{"path traversal", func(l *sourceLock) { l.Sources[0].Files[0].Path = "../outside" }},
		{"absolute path", func(l *sourceLock) { l.Sources[0].Files[0].Path = "/outside" }},
		{"missing hash", func(l *sourceLock) { l.Sources[0].Files[0].SHA256 = "" }},
		{"missing terms", func(l *sourceLock) { l.Sources[0].Terms = "" }},
		{"duplicate source", func(l *sourceLock) { l.Sources = append(l.Sources, l.Sources[0]) }},
		{"duplicate file", func(l *sourceLock) { l.Sources[0].Files = append(l.Sources[0].Files, l.Sources[0].Files[0]) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lock := fixtureLock([]byte("expected"))
			test.edit(&lock)
			if err := validateLock(lock); err == nil {
				t.Fatal("invalid source lock accepted")
			}
		})
	}
}

func TestOfflineCheckAndCorruptCache(t *testing.T) {
	cache := t.TempDir()
	filename := filepath.Join(cache, "fixture", "input.txt")
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte("expected"), 0o600); err != nil {
		t.Fatal(err)
	}
	lock := fixtureLock([]byte("expected"))
	count, err := syncInputs(context.Background(), lock, cache, false, nil)
	if err != nil || count != 1 {
		t.Fatalf("offline check: count=%d error=%v", count, err)
	}
	if err := os.WriteFile(filename, []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = syncInputs(context.Background(), lock, cache, true, nil)
	if err == nil || !strings.Contains(err.Error(), "SHA-256 mismatch") {
		t.Fatalf("corruption was not rejected: %v", err)
	}
	data, err := os.ReadFile(filename)
	if err != nil || string(data) != "tampered" {
		t.Fatalf("corrupt cache was modified: data=%q error=%v", data, err)
	}
}

func TestMissingCacheCheckCreatesNothing(t *testing.T) {
	cache := filepath.Join(t.TempDir(), "absent")
	if _, err := syncInputs(context.Background(), fixtureLock([]byte("expected")), cache, false, nil); err == nil {
		t.Fatal("missing cached input accepted")
	}
	if _, err := os.Stat(cache); !os.IsNotExist(err) {
		t.Fatalf("check created cache: %v", err)
	}
}

func TestFetchVerifiesBeforeWriting(t *testing.T) {
	for _, body := range []string{"expected", "tampered", "oversized input"} {
		t.Run(body, func(t *testing.T) {
			client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: r}, nil
			})}
			filename := filepath.Join(t.TempDir(), "downloads", "input.txt")
			file := fixtureLock([]byte("expected")).Sources[0].Files[0]
			err := fetchFile(context.Background(), client, "https://example.test/input", filename, file)
			if body != "expected" {
				if err == nil {
					t.Fatal("bad download accepted")
				}
				if _, err := os.Stat(filepath.Dir(filename)); !os.IsNotExist(err) {
					t.Fatalf("bad download created output: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(filename)
			if err != nil || string(data) != body {
				t.Fatalf("download: data=%q error=%v", data, err)
			}
			entries, err := os.ReadDir(filepath.Dir(filename))
			if err != nil || len(entries) != 1 {
				t.Fatalf("temporary files left behind: entries=%v error=%v", entries, err)
			}
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestDeveloperInvocationRequiresExplicitPaths(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if status := run(context.Background(), []string{"--check"}, &stdout, &stderr); status != 2 {
		t.Fatalf("status=%d; stderr=%q", status, stderr.String())
	}
}

func TestAssetPinningRequiresRealPNGAndRecordsRawHash(t *testing.T) {
	data := encodeFixturePNG(t, false)
	for _, tc := range []struct {
		name   string
		status int
		body   []byte
		valid  bool
	}{
		{"pinned PNG", 200, data, true}, {"not found", 404, data, false}, {"HTML", 200, []byte("<html>error</html>"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(tc.status); w.Write(tc.body) }))
			defer server.Close()
			filename := filepath.Join(t.TempDir(), "asset.png")
			file, e := pinAsset(context.Background(), server.Client(), server.URL, filename, "pokemon/regular/fixture.png")
			if !tc.valid {
				if e == nil {
					t.Fatal("accepted invalid artwork")
				}
				if _, e := os.Stat(filename); !os.IsNotExist(e) {
					t.Fatal("persisted invalid artwork")
				}
				return
			}
			if e != nil {
				t.Fatal(e)
			}
			if file.SHA256 != digest(data) || file.Size != int64(len(data)) {
				t.Fatal("pin does not describe raw bytes")
			}
			got, e := os.ReadFile(filename)
			if e != nil || !bytes.Equal(got, data) {
				t.Fatal("pin changed input bytes")
			}
		})
	}
}

func TestExplicitAssetLockUpdatePreservesExistingPins(t *testing.T) {
	for _, fail := range []bool{false, true} {
		lock, cache, m := automaticFixture(t)
		before, _ := json.Marshal(lock)
		png := encodeFixturePNG(t, false)
		client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if !strings.Contains(r.URL.Path, "/"+strings.Repeat("b", 40)+"/pokemon/") {
				t.Errorf("request did not retain pinned revision: %s", r.URL)
			}
			status := 200
			if fail {
				status = 404
			}
			return &http.Response{StatusCode: status, Body: io.NopCloser(bytes.NewReader(png)), Header: make(http.Header)}, nil
		})}
		updated, e := updateAssetLock(context.Background(), client, lock, cache, m)
		after, _ := json.Marshal(lock)
		if !bytes.Equal(before, after) {
			t.Fatal("mutated caller's original pins")
		}
		if fail {
			if e == nil {
				t.Fatal("accepted failed download")
			}
			continue
		}
		if e != nil {
			t.Fatal(e)
		}
		var oldSrc, newSrc source
		for _, s := range lock.Sources {
			if s.ID == "pokesprite-v2" {
				oldSrc = s
			}
		}
		for _, s := range updated.Sources {
			if s.ID == "pokesprite-v2" {
				newSrc = s
			}
		}
		if len(newSrc.Files) != len(oldSrc.Files)+2 {
			t.Fatal("missing palette pins or repinned existing asset")
		}
		for _, old := range oldSrc.Files {
			found := false
			for _, current := range newSrc.Files {
				if current.Path == old.Path {
					found = true
					if current != old {
						t.Fatal("changed an existing integrity pin")
					}
				}
			}
			if !found {
				t.Fatal("dropped existing input")
			}
		}
	}
}

func TestTransferProgressDistinguishesDownloadsAndVerifiedCache(t *testing.T) {
	data := []byte("pinned fixture\n")
	lock := fixtureLock(data)
	cache := t.TempDir()
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(data)), Header: make(http.Header)}, nil
	})}
	var logs bytes.Buffer
	n, e := syncInputs(context.Background(), lock, cache, true, client, progressOptions{Writer: &logs, Verbose: true})
	if e != nil || n != 1 {
		t.Fatalf("download: %d %v", n, e)
	}
	text := logs.String()
	if !strings.Contains(text, "downloaded fixture/input.txt") || !strings.Contains(text, "1/1 | downloaded 1 | cached 0 | complete") {
		t.Fatalf("download progress: %q", text)
	}
	if strings.ContainsAny(text, "\r\x1b") {
		t.Fatal("redirected logs contain terminal controls")
	}
	logs.Reset()
	n, e = syncInputs(context.Background(), lock, cache, false, client, progressOptions{Writer: &logs, Verbose: true})
	if e != nil || n != 1 || !strings.Contains(logs.String(), "cached fixture/input.txt") || !strings.Contains(logs.String(), "downloaded 0 | cached 1 | complete") {
		t.Fatalf("cache verification progress: %d %v %q", n, e, logs.String())
	}
	if e := os.WriteFile(filepath.Join(cache, "fixture", "input.txt"), []byte("corrupt"), 0600); e != nil {
		t.Fatal(e)
	}
	logs.Reset()
	n, e = syncInputs(context.Background(), lock, cache, false, client, progressOptions{Writer: &logs})
	if e == nil || n != 0 || !strings.Contains(e.Error(), "input.txt") || !strings.Contains(logs.String(), "0/1 | downloaded 0 | cached 0 | failed") {
		t.Fatalf("corrupt cache was counted as verified: %d %v %q", n, e, logs.String())
	}
}

func TestProgressOnStderrPreservesDatasetStdout(t *testing.T) {
	data := []byte("fixture\n")
	lock := fixtureLock(data)
	dir := t.TempDir()
	cache := filepath.Join(dir, "cache")
	fixtureInput(t, &lock, cache, "fixture", "input.txt", data)
	encoded, e := json.Marshal(lock)
	if e != nil {
		t.Fatal(e)
	}
	filename := filepath.Join(dir, "sources.json")
	if e := os.WriteFile(filename, encoded, 0600); e != nil {
		t.Fatal(e)
	}
	for _, verbose := range []bool{false, true} {
		var stdout, stderr bytes.Buffer
		args := []string{"--sources", filename, "--cache", cache, "--check"}
		if verbose {
			args = append(args, "--verbose")
		}
		if status := run(context.Background(), args, &stdout, &stderr); status != 0 {
			t.Fatalf("status=%d stderr=%q", status, stderr.String())
		}
		if stdout.String() != "Verified 1 pinned inputs.\n" {
			t.Fatalf("stdout changed: %q", stdout.String())
		}
		if !strings.Contains(stderr.String(), "cached 1 | complete") {
			t.Fatalf("missing stderr summary: %q", stderr.String())
		}
		if strings.Contains(stderr.String(), "cached fixture/input.txt") != verbose {
			t.Fatalf("verbose routing: %q", stderr.String())
		}
	}
}

func TestConcurrentInteractiveProgressIsSerializedAndFinalized(t *testing.T) {
	var logs bytes.Buffer
	p := newTransferProgress("New artwork", 80, []progressOptions{{Writer: &logs, Interactive: true}})
	var workers sync.WaitGroup
	for w := 0; w < 8; w++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for i := 0; i < 10; i++ {
				p.record("fixture.png", true)
			}
		}()
	}
	workers.Wait()
	// Exercise the heartbeat before finishing; it must not claim completions.
	p.mu.Lock()
	p.spinner++
	p.display(false, "")
	p.mu.Unlock()
	p.finish(nil)
	text := logs.String()
	if !strings.Contains(text, "New artwork [") || !strings.Contains(text, "80/80 | downloaded 80 | cached 0 | complete\n") || !strings.Contains(text, "\r") {
		t.Fatalf("interactive progress: %q", text)
	}
	select {
	case <-p.stopped:
	default:
		t.Fatal("heartbeat did not stop")
	}
}
