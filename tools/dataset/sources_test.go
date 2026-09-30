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
