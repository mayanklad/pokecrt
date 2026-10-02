# Public-engine performance baseline

Measured 2 October 2026 against owner commit
`8490227efd6e7c5fd3782995acf7bfe739d4fda5`. These benchmark additions do not
change runtime behavior. This is D10a: baseline measurement and printer docs.
D10 remains in progress because the measured lookup optimization requires owner
confirmation before implementation. No v0.2 tag or publication is claimed.

## Environment and scope

- Ubuntu 24.04.3 Linux amd64, kernel 6.18.44; KVM/shared execution environment.
- AMD EPYC 9V74, nine visible logical CPUs; container quota eight CPUs,
  memory limit 8 GiB. Measurements use one Go processor (`GOMAXPROCS=1` or `-cpu=1`).
- Go 1.27.1; module minimum Go 1.27.0. CGO disabled in measured release binary.
- Dataset `8e5073aeeff3f7761e26ccdf9f068189516ae7eded9c0998a9d970b7d37e6b60`:
  1,025 species, 2,669 embedded assets; rules d06-auto-12.
- Release-style build: `-trimpath -ldflags '-s -w -X main.version=v0.2-benchmark'`.
  Snapshot verification also used `-buildvcs=false` because the verification copy
  has no Git metadata; use the same flag for a comparable binary size.
- Output goes to `/dev/null` or `io.Discard`; timing excludes terminal painting,
  downstream readers, source preparation and compilation. Truecolor is retained
  when `NO_COLOR` is empty. Nonempty `NO_COLOR=1` tests the plain path.
- Random paths use the real unbiased cryptographic selector, so sprites and
  allocation counts vary. No deterministic shortcut replaces runtime randomness.

## Fresh processes and warm filesystem cache

Each case records its first observed invocation, then five discarded warmups and
100 fresh processes. Timings include spawning and waiting, with a warm OS page
cache. P95 is the 95th sorted sample, maxima are retained to show scheduler noise.
Every invocation starts a new Go runtime; none uses a persistent application cache.

The first observation is **not a true cold-disk measurement**: compilation and
other invocations can populate page caches. We do not evict shared host caches.
True cold filesystem startup remains unmeasured and has no invented result or
budget. In-process warm measurements are recorded separately below.

| Mode / command | First observed ms | Median ms | P95 ms | Maximum ms | Peak RSS MiB |
| --- | ---: | ---: | ---: | ---: | ---: |
| color/random | 12.439 | 11.237 | 14.002 | 17.918 | 6.684 |
| color/named | 2.044 | 1.592 | 2.429 | 3.089 | 3.793 |
| color/filtered | 2.053 | 1.741 | 2.669 | 4.802 | 3.859 |
| color/variant | 2.238 | 1.566 | 2.838 | 3.936 | 3.855 |
| color/list | 10.090 | 10.483 | 14.492 | 31.062 | 5.734 |
| color/details | 1.491 | 1.374 | 1.695 | 2.656 | 3.914 |
| no_color/random | 10.351 | 9.631 | 12.304 | 13.828 | 6.559 |
| no_color/named | 1.878 | 1.557 | 4.619 | 11.173 | 3.668 |
| no_color/filtered | 1.635 | 1.465 | 2.486 | 3.878 | 3.859 |
| no_color/variant | 2.026 | 1.544 | 2.983 | 6.096 | 3.730 |
| no_color/list | 17.110 | 10.879 | 13.788 | 33.304 | 5.734 |
| no_color/details | 1.928 | 1.523 | 1.925 | 2.844 | 3.914 |

Command cases:

| Case | Arguments |
| --- | --- |
| random | `print` |
| named | `print --name charizard` |
| filtered | `print --gen 1,2 --type fire --type-any flying,dragon` |
| variant | `print --name charizard --form mega-x --shiny` |
| list | `list` (the complete compact public catalog) |
| details | `list --name eevee --details` |

RSS uses a separate native `wait4` launcher, ten fresh children per case/mode,
reporting the maximum child high-water RSS. This includes Go runtime, embedded
pages touched and heap, not only Go allocations. The native launcher avoids
mistaking Python's inherited pre-exec memory high-water mark for Pokémon runtime
memory. It does not add launcher elapsed time to the timing results above.

## Warm in-process benchmarks

Five runs per sub-benchmark, 500 ms per run, one Go processor. The table gives
median time, bytes and allocations per operation across runs. CLI benchmarks run
full parsing/query/selection/decode/render/output assembly each time; they exclude
process startup and actual terminal writes. Render benchmarks decode before the
timer. LargestArea is real Eternatus Eternamax regular artwork, 67×56 pixels.
Lookup fixtures are derived from the manifest's first/last entry, plus an absent
key; they do not assume hand-maintained positions or invent an asset.

| Benchmark | Median µs/op | B/op | Allocs/op |
| --- | ---: | ---: | ---: |
| Query/Standard | 7723.988 | 1,511,890 | 5,656 |
| Query/Named | 131.124 | 33,168 | 73 |
| Query/Filtered | 137.479 | 34,368 | 63 |
| Public/Color/RandomPrint | 8023.518 | 2,826,061 | 6,558 |
| Public/Color/NamedPrint | 537.923 | 216,363 | 1,922 |
| Public/Color/FilteredPrint | 568.045 | 224,432 | 2,213 |
| Public/Color/VariantPrint | 437.585 | 171,680 | 1,229 |
| Public/Color/ListCompact | 8702.533 | 2,111,513 | 14,280 |
| Public/Color/ListDetails | 351.391 | 152,367 | 622 |
| Public/NoColor/RandomPrint | 7495.812 | 2,790,859 | 6,526 |
| Public/NoColor/NamedPrint | 347.342 | 136,486 | 1,918 |
| Public/NoColor/FilteredPrint | 390.878 | 144,506 | 2,198 |
| Public/NoColor/VariantPrint | 342.182 | 130,718 | 1,226 |
| Public/NoColor/ListCompact | 8565.090 | 2,111,517 | 14,280 |
| Public/NoColor/ListDetails | 333.747 | 125,518 | 619 |
| Render/Charizard/Color | 185.689 | 72,387 | 1,727 |
| Render/Charizard/NoColor | 40.974 | 10,944 | 1,723 |
| Render/LargestArea/Color | 365.931 | 146,071 | 3,764 |
| Render/LargestArea/NoColor | 83.039 | 23,184 | 3,760 |
| Lookup/First | 0.009 | 0 | 0 |
| Lookup/Last | 13.456 | 0 | 0 |
| Lookup/Missing | 13.475 | 0 | 0 |
| Decode | 36.249 | 54,760 | 30 |

Stripped Linux amd64 binary: **6,373,536 bytes (6.078 MiB)**, with all current
metadata and accepted artwork embedded. A default unstripped developer build is
not comparable to this size. Changing version text, Go release, architecture or
build metadata can change the exact byte count.

## Initial regression budgets

These are baseline-derived review thresholds for the current dataset/build on a
comparable machine, not universal latency guarantees. Headroom covers the measured
shared-host spread and normal measurement variability. Confirm a suspected
regression with sequential reruns before changing code. A different machine or
larger audited dataset needs its own measured baseline; do not quietly inflate
budgets to hide regressions. Retain this D09 baseline after an approved optimization
and add its measured comparison before selecting tighter budgets.

| Measurement | Initial budget | Basis |
| --- | ---: | --- |
| Fresh-process random / full compact list P95, either color mode | ≤25 ms | Observed worst P95 14.492 ms, with host noise |
| Fresh-process named / filtered / variant / Eevee details P95 | ≤8 ms | Observed worst P95 4.619 ms |
| Warm random / full compact list median | ≤12 ms/op | Observed worst median 8.703 ms |
| Warm named / filtered / variant / Eevee details median | ≤0.85 ms/op | Observed worst median 0.568 ms |
| Largest bundled-area truecolor render alone median | ≤0.60 ms/op | Observed median 0.366 ms |
| Peak child RSS for the cases above | ≤10 MiB | Observed maximum 6.684 MiB |
| Stripped Linux amd64 binary, same inventory | ≤8 MiB | Observed size 6.078 MiB |

These scenarios meet the initial budgets. Unrestricted `list --details` can
legitimately render 1,025 entries; it is not covered by a single-entry detail
budget. First-observed timings and maxima are diagnostics, not P95 replacements.
Budget checks are release-review measurements, not timing assertions in unit tests.
Encounter latency at representative history sizes is deferred until encounters
and storage exist; add it at their milestone without implying it works today.
Recheck public paths at v0.3/v0.4 and the specified later performance review.

## Measured bottleneck and proposed next change

A separate two-second warm random-print CPU profile collected 4.95 seconds of
samples. `sprite.Lookup` and its key equality checks account for **73.54% cumulative
sampled CPU**; `catalog.Query` accounts for 88.08% including those calls. The query
checks artwork availability for each matching species by scanning 2,669 manifest
entries repeatedly. A last/missing lookup costs about 13.5 µs, while the first is
about 10 ns. This position-dependent cost is consistent with the source scan.

Recommendation, **pending owner confirmation**: derive an immutable exact-key
lookup index automatically from the generated manifest in the existing sprite
package. Preserve manifest inventory order, exact identity, absence behavior and
all generated source data. No per-Pokémon entries should be written by hand, no
new package is needed, and no fallback artwork should appear. Measure startup
and memory costs of building the index as well as warm improvements, then run
the full public suite. This report does not claim an unimplemented speedup.
Other possible changes are not bundled into this proposal.

## Reproduce the benchmarks

Prepare pinned assets using the normal dataset command first. From the project
root, with Go on PATH:

```bash
go test ./internal/... -run '^$' \
  -bench 'Benchmark(Public|Query|Lookup|Decode|Render)$' \
  -benchmem -benchtime=500ms -count=5 -cpu=1

CGO_ENABLED=0 go build -buildvcs=false -trimpath \
  -ldflags '-s -w -X main.version=v0.2-benchmark' \
  -o bin/pokecrt-bench ./cmd/pokecrt
wc -c < bin/pokecrt-bench

# Profile separately from timed baseline runs.
go test ./internal/cli -run '^$' \
  -bench '^BenchmarkPublic/Color/RandomPrint$' -benchtime=2s -cpu=1 \
  -cpuprofile=/tmp/pokecrt-random.cpu -o /tmp/pokecrt-cli.test
go tool pprof -top /tmp/pokecrt-cli.test /tmp/pokecrt-random.cpu
```

For fresh-process timing on Linux with Python 3, run the following from the
project root. `os.posix_spawn` and `wait4` avoid background workers; stderr must be
empty and status zero. Repeat sequentially without other benchmark/build jobs.

```bash
python3 - <<'PYTHON'
import json, os, pathlib, statistics, tempfile, time
binary = pathlib.Path('bin/pokecrt-bench').resolve()
cases = {
 'random': ['print'],
 'named': ['print', '--name', 'charizard'],
 'filtered': ['print', '--gen', '1,2', '--type', 'fire', '--type-any', 'flying,dragon'],
 'variant': ['print', '--name', 'charizard', '--form', 'mega-x', '--shiny'],
 'list': ['list'],
 'details': ['list', '--name', 'eevee', '--details'],
}
def invoke(args, env):
 with open(os.devnull, 'wb') as sink, tempfile.TemporaryFile() as error_file:
  start = time.perf_counter_ns()
  pid = os.posix_spawn(str(binary), [str(binary), *args], env, file_actions=[
   (os.POSIX_SPAWN_DUP2, sink.fileno(), 1),
   (os.POSIX_SPAWN_DUP2, error_file.fileno(), 2),
  ])
  _, status, usage = os.wait4(pid, 0)
  elapsed = (time.perf_counter_ns() - start) / 1e6
  error_file.seek(0); error = error_file.read().decode()
  if os.waitstatus_to_exitcode(status) or error: raise RuntimeError((status, error))
 return elapsed
results = {}
for mode in ('color', 'no_color'):
 env = dict(os.environ, GOMAXPROCS='1', NO_COLOR='' if mode == 'color' else '1')
 for name, args in cases.items():
  first = invoke(args, env)
  for _ in range(5): invoke(args, env)
  samples = [invoke(args, env) for _ in range(100)]
  times = sorted(samples)
  results[mode+'/'+name] = dict(first_ms=first, median_ms=statistics.median(times), p95_ms=times[94], max_ms=max(times), samples=100)
print(json.dumps(results, indent=2))
PYTHON
```

For independent peak child RSS on Linux, use a small native launcher and a C
compiler. These are temporary measurement helpers, not additional project files
or runtime dependencies. Linux `ru_maxrss` is in KiB. Repeat each case ten times,
with `GOMAXPROCS=1` and the same color settings; take the maximum and divide by
1,024 for MiB. Example:

```bash
cat > /tmp/pokecrt-rss.c <<'C'
#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <sys/resource.h>
#include <sys/wait.h>
#include <unistd.h>
int main(int argc, char **argv) {
    if (argc < 2) return 2;
    pid_t pid = fork();
    if (pid < 0) { perror("fork"); return 1; }
    if (pid == 0) { execv(argv[1], argv + 1); perror("execv"); _exit(127); }
    int status;
    struct rusage usage;
    if (wait4(pid, &status, 0, &usage) < 0) { perror("wait4"); return 1; }
    if (!WIFEXITED(status) || WEXITSTATUS(status)) return 1;
    fprintf(stderr, "%ld\n", usage.ru_maxrss);
    return 0;
}
C
cc -O2 /tmp/pokecrt-rss.c -o /tmp/pokecrt-rss
for sample in $(seq 1 10); do
  GOMAXPROCS=1 NO_COLOR= /tmp/pokecrt-rss "$PWD/bin/pokecrt-bench" print > /dev/null
done
# Substitute each command from the case table; repeat with NO_COLOR=1.
```

Before v0.2 publication, finish the approved D10 work, apply and test the ZIP,
review real terminal artwork, and prepare/smoke-test the release archive with the
existing packaging workflow. Only the owner tags and publishes the release.
