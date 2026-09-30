package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

type sourceLock struct {
	SchemaVersion int      `json:"schema_version"`
	Sources       []source `json:"sources"`
}

type source struct {
	ID          string       `json:"id"`
	Repository  string       `json:"repository"`
	Revision    string       `json:"revision"`
	Terms       string       `json:"terms"`
	Attribution string       `json:"attribution"`
	Files       []sourceFile `json:"files"`
}

type sourceFile struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

func readLock(filename string) (sourceLock, error) {
	var lock sourceLock
	file, err := os.Open(filename)
	if err != nil {
		return lock, fmt.Errorf("open source lock: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return lock, err
	}
	if info.Size() > 1<<20 {
		return lock, fmt.Errorf("source lock exceeds 1 MiB")
	}
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&lock); err != nil {
		return lock, fmt.Errorf("decode source lock: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return lock, fmt.Errorf("source lock must contain exactly one JSON document")
	}
	return lock, validateLock(lock)
}

func validateLock(lock sourceLock) error {
	if lock.SchemaVersion != 1 || len(lock.Sources) == 0 {
		return fmt.Errorf("source lock requires schema_version 1 and at least one source")
	}
	idPattern := regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	repoPattern := regexp.MustCompile(`^[A-Za-z0-9_-]+/[A-Za-z0-9_.-]+$`)
	seenSources := make(map[string]bool)
	for _, source := range lock.Sources {
		if !idPattern.MatchString(source.ID) || seenSources[source.ID] {
			return fmt.Errorf("invalid or duplicate source ID %q", source.ID)
		}
		seenSources[source.ID] = true
		if !repoPattern.MatchString(source.Repository) || !validHex(source.Revision, 20) {
			return fmt.Errorf("source %s requires a repository and full lowercase commit SHA", source.ID)
		}
		if strings.TrimSpace(source.Terms) == "" || strings.TrimSpace(source.Attribution) == "" || len(source.Files) == 0 {
			return fmt.Errorf("source %s requires terms, attribution, and files", source.ID)
		}
		seenFiles := make(map[string]bool)
		for _, file := range source.Files {
			if file.Path == "." || file.Path == ".." || strings.HasPrefix(file.Path, "../") || path.IsAbs(file.Path) || path.Clean(file.Path) != file.Path || strings.ContainsAny(file.Path, "\\:\x00") || seenFiles[file.Path] {
				return fmt.Errorf("source %s: invalid or duplicate file path %q", source.ID, file.Path)
			}
			seenFiles[file.Path] = true
			if file.Size <= 0 || file.Size > 16<<20 || !validHex(file.SHA256, 32) {
				return fmt.Errorf("source %s: file %s requires a size and lowercase SHA-256", source.ID, file.Path)
			}
		}
	}
	return nil
}

func validHex(value string, byteCount int) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == byteCount && value == strings.ToLower(value)
}

func syncInputs(ctx context.Context, lock sourceLock, cache string, fetch bool, client *http.Client, options ...progressOptions) (count int, err error) {
	if err := validateLock(lock); err != nil {
		return 0, err
	}
	total := 0
	for _, src := range lock.Sources {
		total += len(src.Files)
	}
	progress := newTransferProgress("Inputs", total, options)
	defer func() { progress.finish(err) }()
	for _, source := range lock.Sources {
		for _, file := range source.Files {
			if err := ctx.Err(); err != nil {
				return count, err
			}
			filename := filepath.Join(cache, source.ID, filepath.FromSlash(file.Path))
			data, err := os.ReadFile(filename)
			downloaded := os.IsNotExist(err) && fetch
			if downloaded {
				url := "https://raw.githubusercontent.com/" + source.Repository + "/" + source.Revision + "/" + file.Path
				if err := fetchFile(ctx, client, url, filename, file); err != nil {
					return count, fmt.Errorf("%s/%s: %w", source.ID, file.Path, err)
				}
			} else {
				if err != nil {
					return count, fmt.Errorf("read %s: %w", filename, err)
				}
				if err := verifyBytes(file, data); err != nil {
					return count, fmt.Errorf("%s: %w; remove this cached file before fetching again", filename, err)
				}
			}
			count++
			progress.record(source.ID+"/"+file.Path, downloaded)
		}
	}
	return count, nil
}

func verifyBytes(file sourceFile, data []byte) error {
	if int64(len(data)) != file.Size {
		return fmt.Errorf("size mismatch: got %d bytes, expected %d", len(data), file.Size)
	}
	digest := sha256.Sum256(data)
	if hex.EncodeToString(digest[:]) != file.SHA256 {
		return fmt.Errorf("SHA-256 mismatch")
	}
	return nil
}

func fetchFile(ctx context.Context, client *http.Client, url, filename string, file sourceFile) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned HTTP %d", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, file.Size+1))
	if err != nil {
		return fmt.Errorf("read download: %w", err)
	}
	if err := verifyBytes(file, data); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(filename), ".download-*")
	if err != nil {
		return err
	}
	defer os.Remove(temporary.Name())
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporary.Name(), filename)
}

// updateAssetLock is an explicit developer action. Existing pins are immutable;
// new paths come only from the derived, quality-gated inventory.
func updateAssetLock(ctx context.Context, client *http.Client, lock sourceLock, cache string, m mappingConfig, options ...progressOptions) (resultLock sourceLock, err error) {
	resolved, err := deriveMappings(lock, cache, m)
	if err != nil {
		return lock, err
	}
	index := -1
	for i, src := range lock.Sources {
		if src.ID == "pokesprite-v2" {
			index = i
		}
	}
	if index < 0 {
		return lock, fmt.Errorf("artwork source missing")
	}
	src := lock.Sources[index]
	existing := map[string]bool{}
	for _, file := range src.Files {
		existing[file.Path] = true
	}
	var paths []string
	for _, asset := range resolved.Assets {
		if !existing[asset.Path] {
			paths = append(paths, asset.Path)
			existing[asset.Path] = true
		}
	}
	sort.Strings(paths)
	progress := newTransferProgress("New artwork", len(paths), options)
	defer func() { progress.finish(err) }()
	type result struct {
		file sourceFile
		err  error
	}
	results := make([]result, len(paths))
	jobs := make(chan int)
	var workers sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for i := range jobs {
				file, e := pinAsset(ctx, client, "https://raw.githubusercontent.com/"+src.Repository+"/"+src.Revision+"/"+paths[i], filepath.Join(cache, src.ID, filepath.FromSlash(paths[i])), paths[i])
				results[i] = result{file, e}
				if e == nil {
					progress.record(src.ID+"/"+paths[i], true)
				}
			}
		}()
	}
	for i := range paths {
		jobs <- i
	}
	close(jobs)
	workers.Wait()
	for i, r := range results {
		if r.err != nil {
			return lock, fmt.Errorf("pin %s: %w", paths[i], r.err)
		}
	}
	// Copy the slice so callers' existing lock records cannot be modified.
	lock.Sources = append([]source(nil), lock.Sources...)
	lock.Sources[index].Files = append([]sourceFile(nil), src.Files...)
	for _, r := range results {
		lock.Sources[index].Files = append(lock.Sources[index].Files, r.file)
	}
	sort.Slice(lock.Sources[index].Files, func(i, j int) bool { return lock.Sources[index].Files[i].Path < lock.Sources[index].Files[j].Path })
	return lock, validateLock(lock)
}

func pinAsset(ctx context.Context, client *http.Client, url, filename, assetPath string) (sourceFile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return sourceFile{}, err
	}
	response, err := client.Do(req)
	if err != nil {
		return sourceFile{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return sourceFile{}, fmt.Errorf("HTTP %d", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, (16<<20)+1))
	if err != nil {
		return sourceFile{}, err
	}
	if len(data) == 0 || len(data) > 16<<20 {
		return sourceFile{}, fmt.Errorf("invalid asset size")
	}
	if _, _, _, err := normalizePNG(data); err != nil {
		return sourceFile{}, err
	}
	file := sourceFile{Path: assetPath, Size: int64(len(data)), SHA256: digest(data)}
	if err := atomicWrite(filename, data); err != nil {
		return sourceFile{}, err
	}
	return file, nil
}

func atomicWrite(filename string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(filename), ".dataset-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Chmod(f.Name(), 0644); err != nil {
		return err
	}
	return os.Rename(f.Name(), filename)
}

// Progress is presentation only; it never changes hashes, lock records or stdout.
type progressOptions struct {
	Writer      io.Writer
	Verbose     bool
	Interactive bool
}

type transferProgress struct {
	mu                                                   sync.Mutex
	options                                              progressOptions
	phase                                                string
	total, completed, downloaded, cached, spinner, width int
	lastReport                                           time.Time
	lastCompleted                                        int
	stop                                                 chan struct{}
	stopped                                              chan struct{}
}

func newTransferProgress(phase string, total int, options []progressOptions) *transferProgress {
	if len(options) == 0 || options[0].Writer == nil {
		return nil
	}
	p := &transferProgress{options: options[0], phase: phase, total: total, stop: make(chan struct{}), stopped: make(chan struct{})}
	p.display(false, "")
	go func() {
		defer close(p.stopped)
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-p.stop:
				return
			case <-ticker.C:
				p.mu.Lock()
				if p.options.Interactive || time.Since(p.lastReport) >= 5*time.Second {
					p.spinner++
					p.display(false, "")
				}
				p.mu.Unlock()
			}
		}
	}()
	return p
}

func (p *transferProgress) record(filename string, downloaded bool) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.completed++
	kind := "cached"
	if downloaded {
		p.downloaded++
		kind = "downloaded"
	} else {
		p.cached++
	}
	if p.options.Verbose {
		p.clear()
		fmt.Fprintf(p.options.Writer, "%s: %s %s\n", p.phase, kind, filename)
		if p.options.Interactive {
			p.display(false, "")
		}
	} else if !p.options.Interactive && p.completed < p.total && p.completed-p.lastCompleted >= 250 {
		p.display(false, "")
	}
}

func (p *transferProgress) clear() {
	if p.options.Interactive && p.width > 0 {
		fmt.Fprintf(p.options.Writer, "\r%s\r", strings.Repeat(" ", p.width))
		p.width = 0
	}
}

// Caller holds the mutex, or invokes display before starting the ticker.
func (p *transferProgress) display(final bool, status string) {
	p.lastReport = time.Now()
	p.lastCompleted = p.completed
	line := fmt.Sprintf("%s: %d/%d | downloaded %d | cached %d", p.phase, p.completed, p.total, p.downloaded, p.cached)
	if p.options.Interactive && !final {
		filled := 0
		if p.total > 0 {
			filled = 12 * p.completed / p.total
		}
		line = fmt.Sprintf("%s [%s%s] %c %d/%d | downloaded %d | cached %d", p.phase, strings.Repeat("#", filled), strings.Repeat("-", 12-filled), "|/-\\"[p.spinner%4], p.completed, p.total, p.downloaded, p.cached)
	}
	if status != "" {
		line += " | " + status
	}
	if p.options.Interactive {
		p.clear()
		fmt.Fprint(p.options.Writer, line)
		p.width = len(line)
		if final {
			fmt.Fprintln(p.options.Writer)
			p.width = 0
		}
	} else {
		fmt.Fprintln(p.options.Writer, line)
	}
}

func (p *transferProgress) finish(err error) {
	if p == nil {
		return
	}
	close(p.stop)
	<-p.stopped
	p.mu.Lock()
	defer p.mu.Unlock()
	status := "complete"
	if err != nil {
		status = "failed"
	}
	p.display(true, status)
}
