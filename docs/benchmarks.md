# Public-engine performance baseline

D10a measured 2 October 2026 against commit
`8490227efd6e7c5fd3782995acf7bfe739d4fda5`; its original baseline is retained below.
D10b applies the automatic lookup index to pushed D10a commit
`ac2deea8fdc7cab85591542cd0264103054075cb`. The index is derived from the generated
manifest, stores entry positions and is read-only after package initialization.
No handwritten inventory, new package, data change or artwork fallback is added.
Environment and reproduction commands apply to both measurements.

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

## D10a baseline: fresh processes and warm filesystem cache

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

## D10a baseline: warm in-process benchmarks

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
budgets to hide regressions. Retain the D10a baseline and D10b comparison below; the initial budgets remain
unchanged rather than being silently expanded or tightened.

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

## D10b: approved lookup index and comparison

The original two-second random-print CPU profile attributed 73.54% cumulative
sampled CPU to sprite.Lookup and key equality checks. Each availability check
scanned up to 2,669 entries. That scan was replaced with an automatically derived exact-key index on
2 October 2026.

The only production change is in internal/sprite/assets.go. The private map
associates each generated VariantKey with its manifest position. It is built
once when the process initializes and is never written afterwards. Lookup still
returns an Asset value or a zero Asset plus false. Inventory still clones the
manifest in its original order. Decode still uses exact source artwork, without
substitution. Existing generation rejects duplicates; tests cross-check every
manifest entry, missing form/gender/palette keys and returned-copy isolation.

### Fresh-process comparison

Baseline and indexed binaries use identical build flags, version text, dataset,
Go processor count and measurement method. Each row has five discarded warmups
and 100 fresh child samples before and after. Baseline was rerun sequentially
for this comparison. Help/version were also measured because package
initialization happens even when no artwork is requested. The first-observed
columns are single startup diagnostics; they are not true cold filesystem tests.
These are cold application processes on a warm filesystem cache.

| Case / mode | Before median ms | Indexed median ms | Before P95 ms | Indexed P95 ms | Before first ms | Indexed first ms |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| color/random | 10.119 | 2.945 | 12.395 | 3.677 | 10.444 | 4.301 |
| color/named | 1.539 | 1.801 | 2.051 | 2.255 | 1.721 | 2.031 |
| color/filtered | 1.623 | 1.882 | 2.054 | 2.253 | 1.525 | 1.694 |
| color/variant | 1.475 | 1.717 | 1.914 | 2.123 | 1.497 | 1.942 |
| color/list | 10.755 | 3.143 | 13.382 | 3.964 | 10.964 | 3.325 |
| color/details | 1.521 | 1.616 | 3.060 | 1.974 | 1.649 | 2.029 |
| no_color/random | 9.644 | 2.846 | 11.575 | 3.495 | 10.059 | 2.885 |
| no_color/named | 1.350 | 1.621 | 1.686 | 2.015 | 1.561 | 1.624 |
| no_color/filtered | 1.446 | 1.703 | 2.036 | 2.101 | 1.358 | 1.695 |
| no_color/variant | 1.288 | 1.622 | 1.537 | 2.026 | 1.464 | 1.805 |
| no_color/list | 10.413 | 3.152 | 12.539 | 4.055 | 10.018 | 3.000 |
| no_color/details | 1.367 | 1.651 | 1.801 | 2.499 | 2.526 | 1.883 |
| color/help | 0.898 | 1.183 | 1.083 | 1.468 | 1.306 | 1.802 |
| color/version | 0.901 | 1.198 | 1.227 | 1.517 | 0.946 | 1.201 |
| no_color/help | 0.886 | 1.251 | 1.080 | 1.709 | 0.872 | 1.341 |
| no_color/version | 0.899 | 1.211 | 1.296 | 1.500 | 1.848 | 1.240 |

For color output, random printing improves from 10.119 to 2.945 ms median
(3.44×), and full compact listing from 10.755 to 3.143 ms (3.42×). Several
short commands become about 0.1–0.3 ms slower because the index is built at every
process start. This is a measured tradeoff, not a claim that every command gets
faster. Process results include index construction; warm benchmarks below do not.

### Warm comparison

The original five-run 500-ms D10a results are compared with five indexed runs
using the same command and environment. Reported values are medians. Ordinary
host variation affects rendering/named-command measurements. Random artwork
changes bytes/allocations slightly; the index does not change per-call data
ownership or output assembly.

| Benchmark | Before µs/op | Indexed µs/op | Indexed B/op | Indexed allocs/op |
| --- | ---: | ---: | ---: | ---: |
| Query/Standard | 7723.988 | 1096.384 | 1,511,891 | 5,656 |
| Query/Named | 131.124 | 129.886 | 33,168 | 73 |
| Query/Filtered | 137.479 | 133.403 | 34,368 | 63 |
| Public/Color/RandomPrint | 8023.518 | 1963.117 | 2,823,404 | 6,518 |
| Public/Color/NamedPrint | 537.923 | 544.350 | 216,364 | 1,922 |
| Public/Color/FilteredPrint | 568.045 | 604.706 | 224,217 | 2,201 |
| Public/Color/VariantPrint | 437.585 | 483.238 | 171,681 | 1,229 |
| Public/Color/ListCompact | 8702.533 | 2073.147 | 2,111,528 | 14,280 |
| Public/Color/ListDetails | 351.391 | 367.278 | 152,368 | 622 |
| Public/NoColor/RandomPrint | 7495.812 | 1879.288 | 2,790,914 | 6,531 |
| Public/NoColor/NamedPrint | 347.342 | 369.445 | 136,487 | 1,918 |
| Public/NoColor/FilteredPrint | 390.878 | 403.647 | 144,608 | 2,201 |
| Public/NoColor/VariantPrint | 342.182 | 342.987 | 130,719 | 1,226 |
| Public/NoColor/ListCompact | 8565.090 | 2211.598 | 2,111,527 | 14,280 |
| Public/NoColor/ListDetails | 333.747 | 322.956 | 125,518 | 619 |
| Render/Charizard/Color | 185.689 | 179.988 | 72,387 | 1,727 |
| Render/Charizard/NoColor | 40.974 | 43.625 | 10,944 | 1,723 |
| Render/LargestArea/Color | 365.931 | 438.452 | 146,071 | 3,764 |
| Render/LargestArea/NoColor | 83.039 | 89.841 | 23,184 | 3,760 |
| Lookup/First | 0.009 | 0.031 | 0 | 0 |
| Lookup/Last | 13.456 | 0.028 | 0 | 0 |
| Lookup/Missing | 13.475 | 0.021 | 0 | 0 |
| Decode | 36.249 | 36.402 | 54,760 | 30 |

The formerly cheap first entry also pays hashing cost: about 31 ns indexed.
Last/missing entries now take about 28/21 ns instead of about 13.5 µs.
Full standard query falls from 7.724 to 1.096 ms. Renderer and source PNGs are
unchanged; differences in their measured times are not renderer optimizations.

### Resource comparison

RSS is independently measured with the native launcher, ten fresh children per
row/build. Resident-page/allocator variation means differences are not a direct
measurement of map payload size. The index persists for the process lifetime.

| Case / mode | Before peak RSS MiB | Indexed peak RSS MiB |
| --- | ---: | ---: |
| color/random | 6.684 | 6.809 |
| color/named | 3.793 | 4.430 |
| color/filtered | 3.859 | 4.496 |
| color/variant | 3.855 | 4.492 |
| color/list | 5.734 | 5.996 |
| color/details | 3.914 | 4.426 |
| no_color/random | 6.559 | 6.809 |
| no_color/named | 3.668 | 4.430 |
| no_color/filtered | 3.859 | 4.371 |
| no_color/variant | 3.730 | 4.367 |
| no_color/list | 5.609 | 5.996 |
| no_color/details | 3.914 | 4.426 |

Maximum indexed peak RSS is 6.809 MiB. Short command rows increase by roughly
0.5–0.8 MiB; full random/list rows grow less because previous transient
allocations already contribute to their peak. Both stripped comparison binaries
are 6,373,536 bytes (6.078 MiB); unchanged file size is an observed aligned binary
result, not a promise that additional code always has no size cost.
All indexed scenarios remain within the initial EPYC-environment budgets above.
No extra runtime optimization or budget change is included.

### Intel i7 laptop verification: before and after

The D10a and D10b laptop benchmark output uses Linux amd64, Intel Core i7-10750H
2.60 GHz, five 500-ms runs and `-cpu=1`. Both logs show successful tests and
benchmarks and the same dataset. Go version, kernel and power settings were not
included; these observations are not a fully qualified machine-specific budget.
They measure warm in-process work, excluding process startup and index construction.

| Benchmark | Before ms/op | Indexed ms/op | Before / indexed |
| --- | ---: | ---: | ---: |
| Public/Color/RandomPrint | 12.705 | 3.274 | 3.88× |
| Public/Color/ListCompact | 13.394 | 3.791 | 3.53× |
| Query/Standard | 12.538 | 1.800 | 6.96× |
| Public/Color/NamedPrint | 0.829 | 0.814 | 1.02× |
| Public/Color/FilteredPrint | 0.914 | 0.891 | 1.03× |
| Public/Color/VariantPrint | 0.687 | 0.702 | 0.98× |
| Public/Color/ListDetails | 0.566 | 0.599 | 0.94× |

Laptop late/missing exact lookups fall from median 20.227/20.285 µs to
56.92/39.46 ns. Named and other small paths show ordinary run-to-run differences;
only the repeated-lookup paths demonstrate the intended large improvement.
Do not apply EPYC absolute thresholds directly to the Intel i7 laptop. Retain its
local before/after reference and record environment/power settings for future
release comparisons.

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


## D12 trainer dependency review

D12 is measured against `30d0e76935cfff337c3909f6c5b4aa98eb8652ba`,
using sequential baseline/current builds on the same EPYC 9V74 Linux amd64
host, Go 1.27.1, CGO disabled, `-trimpath -ldflags "-s -w"`, version `dev`
and `-buildvcs=false`. The dataset and embedded artwork are identical.
SQLite and Unicode support are linked for trainer commands; public commands
still perform no database/path work. These measurements include dependency
initialization and executable loading.

Each fresh-process row has five warmups and 100 measured samples, with output
discarded and `GOMAXPROCS=1`. Peak RSS uses the native launcher described above,
ten independent processes per case and mode. Measurements are host-specific;
filesystem caches are warm and startup timing is not terminal painting latency.

| Case / mode | D11 median ms | D12 median ms | D11 P95 ms | D12 P95 ms | D12 peak RSS MiB |
| --- | ---: | ---: | ---: | ---: | ---: |
| color/random | 3.059 | 3.041 | 3.591 | 3.683 | 8.910 |
| color/named | 1.837 | 2.014 | 2.198 | 2.348 | 6.406 |
| color/filtered | 1.967 | 2.154 | 2.345 | 2.563 | 6.535 |
| color/variant | 1.826 | 1.827 | 2.174 | 2.464 | 6.656 |
| color/list | 3.297 | 3.162 | 3.908 | 3.958 | 8.031 |
| color/details | 1.600 | 1.773 | 2.043 | 2.361 | 6.340 |
| no_color/random | 2.659 | 2.756 | 3.400 | 3.239 | 8.785 |
| no_color/named | 1.559 | 1.907 | 2.003 | 2.599 | 6.406 |
| no_color/filtered | 1.607 | 2.091 | 2.031 | 2.648 | 6.410 |
| no_color/variant | 1.487 | 1.875 | 2.135 | 2.454 | 6.531 |
| no_color/list | 3.040 | 3.407 | 4.594 | 4.117 | 8.031 |
| no_color/details | 1.494 | 1.957 | 1.969 | 2.304 | 6.340 |

The worst observed short-command P95 is 2.648 ms against the initial 8 ms
threshold; random/full-list P95 is at most 4.117 ms against 25 ms. Peak RSS
is 8.910 MiB against 10 MiB. These public startup/memory scenarios pass the
existing thresholds; individual timing differences include shared-host noise.

The stripped executable grows from 6,373,536 bytes (6.078 MiB) to 11,579,552
bytes (11.043 MiB), about 5 MiB added by linked SQLite/Unicode/trainer support.
The initial 8 MiB threshold remains the historical public-only binary baseline.
The v0.3 trainer-inclusive stripped Linux amd64 binary has a 16 MiB review
threshold, established on 2 October 2026 to account for the linked SQLite
engine and remaining trainer functionality. D12's 11.043 MiB binary meets it.
All public latency, rendering and RSS thresholds remain unchanged. Recheck size
and public performance before v0.3 publication; the size allowance is not a
latency exemption. No alternate SQLite driver, external database executable,
second PokéCRT binary or runtime download is introduced.

Warm checks use three 200-ms runs per case, `-cpu=1`, with existing
BenchmarkPublic/BenchmarkRender cases and discarded output. They exclude
process startup and SQLite initialization; they exercise unchanged public paths.

| Warm scenario | D12 median ms/op | Existing threshold ms/op |
| --- | ---: | ---: |
| Public/Color/RandomPrint | 1.755 | 12.00 |
| Public/Color/NamedPrint | 0.514 | 0.85 |
| Public/Color/FilteredPrint | 0.549 | 0.85 |
| Public/Color/VariantPrint | 0.428 | 0.85 |
| Public/Color/ListCompact | 1.965 | 12.00 |
| Public/Color/ListDetails | 0.342 | 0.85 |
| Public/NoColor/RandomPrint | 1.704 | 12.00 |
| Public/NoColor/NamedPrint | 0.336 | 0.85 |
| Public/NoColor/FilteredPrint | 0.378 | 0.85 |
| Public/NoColor/VariantPrint | 0.320 | 0.85 |
| Public/NoColor/ListCompact | 1.975 | 12.00 |
| Public/NoColor/ListDetails | 0.306 | 0.85 |
| Render/LargestArea/Color | 0.365 | 0.60 |

All listed warm scenarios remain within their existing thresholds. These
short review runs supplement the retained five-run D10 baselines; they do
not replace those measurements or establish trainer encounter latency.


## D13 public-path check

D13's internal encounter foundation is checked against source
`3a01880b76b2de76f708a154749cec906c282e4d`. It introduces no public encounter
command, package initialization work, migration or dependency. With the same
Go 1.27.1/EPYC environment, CGO-disabled stripped build and dataset as D12,
the binary remains 11,579,552 bytes (11.043 MiB), within the v0.3 16 MiB limit.

A sequential fresh-process check uses the same six public cases, both color
modes, five warmups and 100 samples per row described above. Worst short-case
P95 is 2.926 ms (8 ms limit); random/full-list P95 is 4.931 ms
(25 ms limit). These measurements include executable startup and remain within
existing thresholds. Warm/RSS rows above remain D12 observations; they are not
relabeled as D13 measurements. Encounter latency at representative history sizes
will be measured with completed progression before v0.3 publication.


## D14 public-path check

D14 adds internal XP and achievement evaluation against source
`b4a0c1014300bc8084ed295136ad0891e91bb884`. Targets are initialized only when
recording an encounter; public commands retain their storage-independent path.
On the same Linux amd64 EPYC/Go 1.27.1 environment, the six existing fresh-process
cases in color/NO_COLOR modes use five warmups and 100 samples per row.
Short-command P95 is at most 3.012 ms against 8 ms; random/list P95 is at most
4.979 ms against 25 ms. The CGO-disabled stripped binary remains 11,579,552 bytes
(11.043 MiB), below the approved trainer-inclusive 16 MiB threshold.
Prior warm/render/RSS results retain their original milestone labels. No new
encounter latency or history-growth performance result is claimed here.


## D15 encounter command and public regression check

D15 exposes the encounter command and five output modes against source
`223ff220a36e69df8da89600f4bfc083a0a34922`. Its formatter consumes only a committed
result. Eligible progress intersects discoveries with inventory derived from
exact accepted artwork; historical counts remain separate. Help and invalid
invocations bypass storage, and public print/list/help/version do not open it.

Measurements use the same Linux amd64 AMD EPYC 9V74, Go 1.27.1 environment and
CGO-disabled, trimpath, stripped build as the prior checks. Size is 11,665,568
bytes (11.125 MiB), within the approved trainer-inclusive 16 MiB threshold.
Fresh-process timing retains six cases, color/NO_COLOR, GOMAXPROCS=1, five
warmups and 100 samples per row. Short-command P95 is at most 3.093 ms (8 ms
limit); random/full-list P95 is at most 4.589 ms (25 ms limit). Separate native
wait4 RSS measurements use ten fresh children per case/mode: maximum 8.950 MiB
against 10 MiB. Python launcher RSS is not used as child runtime memory.

Warm checks use existing BenchmarkPublic/BenchmarkRender cases, three 200-ms
runs per case, -cpu=1 and discarded output. Median ms/op results are:

| Warm scenario | D15 median ms/op | Existing threshold ms/op |
| --- | ---: | ---: |
| Public/Color/RandomPrint | 1.847 | 12.00 |
| Public/Color/NamedPrint | 0.496 | 0.85 |
| Public/Color/FilteredPrint | 0.536 | 0.85 |
| Public/Color/VariantPrint | 0.411 | 0.85 |
| Public/Color/ListCompact | 2.030 | 12.00 |
| Public/Color/ListDetails | 0.357 | 0.85 |
| Public/NoColor/RandomPrint | 1.738 | 12.00 |
| Public/NoColor/NamedPrint | 0.345 | 0.85 |
| Public/NoColor/FilteredPrint | 0.381 | 0.85 |
| Public/NoColor/VariantPrint | 0.324 | 0.85 |
| Public/NoColor/ListCompact | 2.119 | 12.00 |
| Public/NoColor/ListDetails | 0.323 | 0.85 |
| Render/LargestArea/Color | 0.366 | 0.60 |

All existing public timing/rendering/memory/size thresholds pass. These results
are environment-specific and do not establish encounter latency at growing
history sizes; that remains a v0.3 publication check. Installed-binary tests
exercise the command away from source assets/cache, all output modes, truecolor/
NO_COLOR and post-commit closed pipes. A separate network-namespace isolation
attempt was unavailable because this environment denies user namespace mapping;
no new network-isolated encounter result is claimed.

## Achievement expansion after D15

This increment adds 16 approved themed goals to the ordered registry against
source `0d3c8b67638c398635a51da90d4807c6555956f3`, supporting 50 feasible current
achievements. Evaluation still uses the existing atomic encounter transaction;
public dispatch, renderer, catalog and sprite source files are unchanged.
No new dependency, migration, generated dataset or source file is introduced.

This check runs on **Intel Xeon Platinum 8370C at 2.80 GHz**, nine visible CPUs,
Ubuntu 24.04.3 LTS, Linux 6.18.44, Go 1.27.1, Linux amd64. The hardware differs
from the earlier EPYC D15 measurement; prior values retain their original labels.
The CGO-disabled trimpath stripped executable is 11,694,240 bytes (11.152 MiB),
within the approved 16 MiB trainer-inclusive threshold. Dataset remains
`8e5073aeeff3f7761e26ccdf9f068189516ae7eded9c0998a9d970b7d37e6b60`.

Fresh-process timing uses the same six public scenarios, color/NO_COLOR,
GOMAXPROCS=1, five warmups and 100 samples per row. Maximum short-command P95
is 3.459 ms against 8 ms; random/full-list P95 is 6.254 ms against 25 ms.
Separate native wait4 measurements use ten fresh children per scenario/mode;
maximum peak RSS is 9.020 MiB against 10 MiB.

Warm public measurements retain three 200-ms runs per case, -cpu=1 and discarded
output. Median ms/op results on this host are:

| Warm scenario | Median ms/op | Existing threshold ms/op |
| --- | ---: | ---: |
| Public/Color/RandomPrint | 2.571 | 12.00 |
| Public/Color/NamedPrint | 0.714 | 0.85 |
| Public/Color/FilteredPrint | 0.748 | 0.85 |
| Public/Color/VariantPrint | 0.579 | 0.85 |
| Public/Color/ListCompact | 2.730 | 12.00 |
| Public/Color/ListDetails | 0.443 | 0.85 |
| Public/NoColor/RandomPrint | 2.519 | 12.00 |
| Public/NoColor/NamedPrint | 0.433 | 0.85 |
| Public/NoColor/FilteredPrint | 0.478 | 0.85 |
| Public/NoColor/VariantPrint | 0.538 | 0.85 |
| Public/NoColor/ListCompact | 3.262 | 12.00 |
| Public/NoColor/ListDetails | 0.388 | 0.85 |

The initial combined public/render package benchmark measured largest-area color
rendering at 0.776 ms, exceeding 0.60 ms. The renderer and all catalog/sprite Go
files match D15 byte-for-byte. To resolve that result rather than waive it,
isolated same-host renderer checks used three 500-ms runs per build, -cpu=1,
with decode outside the timed loop. D15 baseline median is 0.509 ms; expanded
build median is 0.510 ms. Both retain 146,071 B/op and 3,764 allocations/op and
meet the existing 0.60 ms budget. This longer isolated check avoids concurrent
package benchmark work; the initial higher observation is retained here.
No rendering optimization or threshold relaxation is made in this increment.

Correctness tests cover all 50 unique attainable definitions, new threshold
boundaries, unsupported-goal suppression, exact historical form evidence,
concurrent one-time awards, per-trainer isolation and rollback after a new themed
insert. Installed binary modes and post-commit output failure checks continue
to pass. No new network-namespace isolation or history-growth encounter latency
result is claimed; those remain checks before v0.3 publication.

## D16 private Pokédex regression check - 3 October 2026

Go 1.27.1, Linux amd64, Intel Xeon Platinum 8370C @ 2.80 GHz; dataset
`8e5073aeeff3f7761e26ccdf9f068189516ae7eded9c0998a9d970b7d37e6b60`.
CGO-free stripped/trimpath binary: 11,784,352 bytes (11.238 MiB), within
the approved 16 MiB trainer-inclusive limit.

An initial startup run overlapping other validation reached short-command P95
14.948 ms. A subsequent run reached 9.326 ms. Both exceed the 8 ms budget and
are retained as observations, not passing evidence. After other validation
finished, the isolated six-case color/NO_COLOR startup run (GOMAXPROCS=1,
five warmups and 100 measured samples per row) measured maximum short-command
P95 4.392 ms and random/full-list P95 10.010 ms, within 8/25 ms.
No timing budget or public command implementation was changed.

An overlapping warm run also exceeded named/filtered/variant/list budgets.
The isolated follow-up uses three 500 ms repetitions on one Go CPU. Median
warm short cases are at most 0.686 ms against 0.85 ms; random/full-list at most
3.225 ms against 12 ms. Isolated render measurement, three 500 ms repetitions,
measures largest-area color median 0.589 ms against 0.60 ms. Native wait4 child
RSS (ten repetitions per case/mode) reaches 9.039 MiB against 10 MiB.

These are public-path regression checks, not representative large-history Dex
or encounter latency claims. Network-namespace testing remains unavailable in
this execution environment as recorded for D15. Release offline and
representative trainer-history performance gates remain required before v0.3.
