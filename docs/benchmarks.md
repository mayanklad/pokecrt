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

## D17 trainer views regression check - 3 October 2026

Go 1.27.1, Linux amd64, Intel Xeon Platinum 8370C @ 2.80 GHz; unchanged
2947-input dataset. CGO-free stripped/trimpath binary: 11,821,216 bytes
(11.274 MiB), within the approved 16 MiB trainer-inclusive budget.

The existing six public cases in color/NO_COLOR mode use GOMAXPROCS=1,
five warmups and 100 measured samples per row. Maximum short-command P95 is
4.891 ms against 8 ms; random/full-list P95 is 6.574 ms against
25 ms. These measurements follow functional/race validation, without overlapping
benchmark or dataset workloads. No fresh warm-render or native RSS result is
claimed for D17. Previous results remain dated observations; complete release
regression and representative history-growth checks remain D18 gates.

Statistics/achievement query tests establish snapshot consistency and no
mutation, not large-history performance. Network-namespace offline testing is
not claimed for this increment. Existing installed-binary tests run from an
empty directory without source artwork/cache and cover trainer browsing.

## D18 trainer CLI integration - 3 October 2026

Source baseline: `43e0a34b35a792ad268e40407aa68e863f0aa763`.
Host: Intel Xeon Platinum 8370C at 2.80 GHz, Ubuntu 24.04.3, Linux 6.18.44,
nine visible logical CPUs; Go 1.27.1, module minimum 1.27.0. Dataset and public
budgets remain unchanged. No database index or schema migration is added.

### History growth

`BenchmarkTrainerHistory` uses real first-encounter metadata/artwork and bulk
repeat fixtures with consistent history, discovery counts and XP. Each case
starts with the named history size plus one untimed repeat to settle unlocks.
Recording appends repeats during timing; reads reuse a warm repository. Results
exclude database opening, process startup, artwork rendering and formatting.
Three samples per case use `-benchtime=100ms -count=3 -cpu=1`; values below are
medians. The baseline runs the same fixture against unchanged D17 source.

| History rows | Record before / after ms | Trainer records before / after ms | Dex records before / after ms |
| ---: | ---: | ---: | ---: |
| 1,000 | 2.317 / 1.334 | 2.855 / 1.135 | 0.297 / 0.173 |
| 10,000 | 13.438 / 5.970 | 18.223 / 8.774 | 0.833 / 0.750 |
| 100,000 | 155.713 / 64.254 | 197.977 / 105.656 | 6.844 / 6.804 |

The change reuses already loaded historical regional/transformation flags and
checks distinct stored shiny variants. Both discovery tables and history are
updated atomically. Equivalence tests compare the old predicates and all goal
results across empty, repeated, shiny-first, isolated and retained-snapshot
histories. It does not replace historical evidence with current catalog metadata.
The 100,000-row medians improve by about 59%/47% for recording/trainer records.
Dex source is unchanged; its timing differences are not an optimization claim.
A repeated single-appearance fixture is a history-growth test, not a benchmark
of every possible collection. No trainer latency threshold or universal speedup
is claimed; remaining history scans still grow with history length.

Reproduce from prepared source:

```bash
go test ./internal/storage -run '^$' -bench '^BenchmarkTrainerHistory$' \
  -benchtime=100ms -count=3 -cpu=1
```

### Public warm and render checks

Existing benchmarks use three 200 ms samples, one Go processor, output discarded.
The maximum short-command median is 0.721 ms against 0.85 ms; random/list median
is at most 2.735 ms against 12 ms. Largest-area color rendering median is
0.526 ms against 0.60 ms. Its individual samples are 0.511, 0.619 and 0.526 ms;
the budget applies to the median. Terminal painting and process startup are excluded.

```bash
go test ./internal/cli ./internal/render -run '^$' \
  -bench 'Benchmark(Public|Render)$' -benchtime=200ms -count=3 -cpu=1
```

### Installed/offline integration and limits

Full tests, vet, race tests, CGO-free trainer/storage tests and all 2,947 pinned
input checks pass. Actual gameplay migration fixtures preserve profiles, active
selection, history, discoveries, XP and unlocks in successful upgrades, protected
backups and failed-upgrade rollback. Schema 002 exists only in synthetic tests.
Eight concurrent installed CLI processes preserve all committed encounters and XP.

The package executable is extracted into an empty directory and checked with
Linux seccomp denying socket creation and network operations. A negative-control
socket call fails with EPERM. Public commands create no trainer state; isolated
profile creation, every encounter mode, Dex locks, achievements, reopening,
read-only bytes, installation, shell piping and corrupt-state independence pass.
Network namespaces are unavailable on this host; no namespace success is claimed.

Native C fork/exec/wait4 measures peak child RSS over ten samples per public case
and color mode. The archive-extracted binary reaches 8.950 MiB against 10 MiB.
An initial verification-file reading reached 13.637 MiB. Copying the identical
binary removed that discrepancy; extracted and installed-style copies remain
within budget. This file/runtime measurement sensitivity is retained rather than
interpreted as a source regression or proof that every host will match the result.
Release binary size is 11,821,216 bytes (11.274 MiB) against the approved 16 MiB
trainer-inclusive budget.

Real Bash piping and installation are verified. Fastfetch is not installed on
this benchmark host; raw-input piping was subsequently verified on a separate
Linux amd64 desktop with zsh and kitty. JSONC command-raw configuration examples
are checked against official documentation, without claiming a runtime test.
Cold filesystem startup and platform binaries beyond Linux amd64 are unverified.

Fresh-process measurements use the extracted release executable, six public
cases in color/plain modes, five discarded warmups and 100 samples per case.
Timing uses posix_spawn/wait4, GOMAXPROCS=1 and /dev/null output. Short-command
P95 is at most 4.311 ms (8 ms budget); random/list P95 at most 7.050 ms (25 ms).
The filesystem cache is warm; the first invocation is not a cold-disk claim.

The subsequent player-facing wording/layout review changes no queries, artwork,
selection or rewards. Full tests/vet/race and CGO-free trainer/storage checks pass.
The same 12-case fresh-process method gives short-command P95 <= 7.660 ms and
random/list P95 <= 9.300 ms. The stripped binary is 11,821,216 bytes (11.274 MiB).
Warm/render measurements above precede this presentation-only review; those
paths are unchanged and no new warm/render result is claimed.

## D19 Adventure Menu shell — 4 October 2026

Baseline: `0fc3efd8feefd56be46bdbcd981773e42c097d4e`; candidate adds
Bubble Tea v2.0.10, responsive shell controls and async read-only profile status.
AMD EPYC 9V74, Linux 6.18.44, Go 1.27.1, Linux amd64; same unchanged dataset
and 2,669 accepted embedded assets. Both measured binaries use CGO_ENABLED=0,
`-buildvcs=false -trimpath -ldflags '-s -w'` and version `dev`.

Fresh-process method: GOMAXPROCS=1, five discarded warmups and 100 samples per
case/mode, stdout to /dev/null. P95 is sorted sample 95. Each binary is measured
sequentially across the six existing cases in color and NO_COLOR=1. These are
warm-filesystem spawn-and-exit observations, not cold-disk timings. RSS uses ten
additional executions per case through a native fork/exec/wait4 launcher,
avoiding Python's inherited pre-exec high-water RSS.

| Case / mode | Baseline P95 ms | D19 P95 ms | D19 peak RSS MiB |
| --- | ---: | ---: | ---: |
| color/random | 4.999 | 4.049 | 9.742 |
| color/named | 2.614 | 2.565 | 7.363 |
| color/filtered | 3.687 | 2.631 | 7.367 |
| color/variant | 2.871 | 2.381 | 7.426 |
| color/list | 4.839 | 4.708 | 8.863 |
| color/details | 2.510 | 2.500 | 7.113 |
| plain/random | 4.259 | 3.842 | 9.617 |
| plain/named | 2.511 | 2.325 | 7.238 |
| plain/filtered | 2.446 | 2.373 | 7.242 |
| plain/variant | 2.229 | 2.843 | 7.426 |
| plain/list | 4.581 | 4.988 | 8.863 |
| plain/details | 2.344 | 2.608 | 7.113 |

The short cases remain under 8 ms, random/full-list under 25 ms and public
RSS under 10 MiB. Baseline stripped size is 11,821,216 bytes (11.274 MiB);
D19 is 13,250,720 bytes (12.637 MiB), within the approved 16 MiB limit.
No public performance threshold is changed.

Warm public/render review: three 500 ms runs per sub-benchmark, one processor.
The largest median across named/filtered/variant/details cases is 0.519 ms
(0.85 ms limit), random/full-list 2.199 ms (12 ms limit), and largest-area
truecolor render 0.364 ms (0.60 ms limit). The shared host and scheduler affect
these observations; they are not a universal input-to-display latency promise.

Wide shell View benchmark, three runs on one processor at 120×36: 358,497,
364,927 and 352,072 ns/op; median 0.358 ms, 316,481 B/op and 1,153 allocs/op.
This includes frame assembly and neutral motif rendering, not physical terminal
painting or storage. It establishes an initial shell measurement, not a final
TUI budget. There is no application animation loop.

A native wait4 launcher observed the Dark shell in a pseudo-terminal for one
second of startup plus three seconds idle: 41.213 ms total user+system CPU,
11,636 KiB peak RSS, zero output bytes during the idle interval. CPU/RSS include
startup and shutdown, not isolated steady-state consumption. This TUI RSS is
reported separately from the public-command 10 MiB threshold.

Verification: fresh full tests, vet, race tests, CGO-free tui/trainer/storage
tests and all 2,947 pinned inputs pass. Model tests check resize bounds, every
shell control's mouse target, focus/activation, NO_COLOR, theme replies/fallback,
async/stale results and unchanged/missing/corrupt trainer databases. Real PTY
process checks cover 120×36, 100×32, 80×24/16, 48×22, 40×12 and 32×8,
arrows, mouse-only appearance/quit, Ctrl+C, resize and a simulated light
background reply. Terminal attributes, alternate screen and mouse reporting
restore on exit; first-run storage stays absent.

PTY frames were inspected at wide, compact and appearance-settings sizes.
Real-emulator transparency, native theme notifications and terminal/multiplexer
compatibility still need the owner's terminal checks; unit/PTY tests do not
establish those. Network namespace isolation could not be repeated: unshare
failed to write uid_map in this environment. No new offline-isolation claim is
made. The shell performs no HTTP/network operation at runtime.

### D19 visual revision: Pokédex device

The owner selected the device after terminal-rendered review, then rejected its
pixel Poké Ball. The shipped revision uses a clean question mark and text,
with a bounded centered stage and filled focus bar. The previous D19 tables
above describe the initial shell measurements; they are retained as historical
samples, not fresh measurements of this revision.

Same CGO-free, stripped build flags: 13,246,624 bytes (12.633 MiB), below 16 MiB.
Revised 120×36 View assembly: 539,489 / 1,114,590 / 817,305 ns/op; median
0.817 ms, approximately 397.7 kB/op and 2,468 allocations/op, shared-host
GOMAXPROCS=8. This is frame construction, not end-to-end key response. The
interface has no decorative tick or continuous animation. Wide/compact,
190×50, light, short-window, native, NO_COLOR, resize, arrows and mouse appearance
runs verify restoration and absent first-run state. Public CLI/render paths are
unchanged by this visual revision; their prior measurements are not rerun or
represented as newly measured. Theme/translucency still need real-terminal review.


## D20 setup and appearance persistence — 4 October 2026

Source baseline: D19 `742ac3248883ccacf396c1533deff0ebb65ad560`. Host:
Linux amd64, Intel Xeon Platinum 8370C, Go 1.27.1. Both comparison binaries
use CGO_ENABLED=0, -buildvcs=false, -trimpath and -ldflags '-s -w', version dev,
and the same bundled dataset. No module or generated-data changes.
The final candidate is 13,308,064 bytes (12.692 MiB), within 16 MiB.

Fresh processes: GOMAXPROCS=1, five warmups and 100 samples per command/mode,
stdout discarded; native wait4 launcher used for ten RSS samples. Warm filesystem
cache, not true cold startup or physical terminal drawing.

| Case / mode | D19 P95 ms | D20 P95 ms | D20 peak RSS MiB |
| --- | ---: | ---: | ---: |
| color/random | 9.835 | 8.303 | 9.863 |
| color/named | 6.057 | 5.565 | 7.359 |
| color/filtered | 3.985 | 13.367 | 7.363 |
| color/variant | 4.436 | 15.497 | 7.484 |
| color/list | 22.005 | 23.722 | 8.984 |
| color/details | 8.833 | 4.835 | 7.234 |
| plain/random | 17.499 | 18.559 | 9.613 |
| plain/named | 11.494 | 5.459 | 7.234 |
| plain/filtered | 15.230 | 11.333 | 7.238 |
| plain/variant | 5.434 | 6.220 | 7.484 |
| plain/list | 15.288 | 7.379 | 8.984 |
| plain/details | 4.359 | 4.262 | 7.234 |

Maximum public RSS is 9.863 MiB,
within 10 MiB. Random/full-list P95 is at most
23.722 ms,
within 25 ms. Short-command P95 is at most
15.497 ms,
**above the unchanged 8 ms threshold**. D19 itself reaches
15.230 ms
on this host. This run does not establish a passing startup gate or attribute
the noise to D20; controlled/local timing remains required. No threshold is raised.

Warm in-process benchmarks, one processor and three 300 ms samples per case:
worst short-command median 0.703 ms (0.85 ms limit), random/full-list
median 6.333 ms (12 ms limit), largest-area color render median
0.515 ms (0.60 ms limit). All meet those warm-work budgets.
Wide View at 120×36, three GOMAXPROCS=8 samples: 748,252 / 714,803 / 848,931
ns/op; median 0.748 ms, about 398.2 kB/op and 2,470 allocations/op. This is
frame assembly, not end-to-end response latency.

A startup + three-second idle + shutdown PTY sample used
49.842 ms total CPU and 11.484 MiB
peak RSS; idle output was 0 bytes. Session CPU is not
isolated idle CPU. The TUI memory figure is separate from the public-path cap.
Appearance file parsing is bounded to 256 bytes; no JSON dependency, decorative
ticks or animation is added. Supported Follow Terminal queries remain deliberate.

Full tests/vet/race and CGO-free UI/domain/storage checks pass. PTY checks cover
first-run wide/compact/40×12, invalid and Unicode input, cancel, mouse-only
creation/cursor edit/quit, separate additional-profile activation, selection,
flag/saved-theme precedence, resize, Native/NO_COLOR and restoration. State checks
find no gameplay records after setup/switching. Model tests cover long lists,
profiles without an active trainer, duplicate names, corrupt state, blocked/cancelled
requests, stale results, atomic saves, concurrent saves, invalid/future/symlinked
settings and late-load precedence. Actual desktop/terminal translucency and
multiplexer compatibility remain manual checks; no new network-isolation claim.


## D21 initial browse prototype measurements (superseded layout)

Owner baseline: `5ad62fd5d1e896eec24cb360e685038052f7e176` (D20). Go 1.27.1,
Linux amd64, Intel Xeon Platinum 8370C shared host. CGO_ENABLED=0 with
-buildvcs=false -trimpath and -ldflags '-s -w'. No added dependency or global
Dex initialization. Public measurements used the browse candidate before the
final focus/resize-only corrections; final binary size is listed separately.

Final stripped binary: 13,385,888 bytes (12.766 MiB), below 16 MiB.
Measured D20 binary: 13,316,256 bytes.

Fresh public process checks: five warmups, 100 timed samples per case/mode,
ten native wait4 RSS samples, GOMAXPROCS=1; output discarded.

| Case / mode | D20 P95 ms | D21 P95 ms | D21 peak RSS MiB |
| --- | ---: | ---: | ---: |
| color/random | 10.498 | 13.595 | 9.684 |
| color/named | 10.024 | 7.036 | 7.305 |
| color/filtered | 6.327 | 6.204 | 7.309 |
| color/variant | 5.268 | 13.107 | 7.367 |
| color/list | 8.995 | 14.381 | 8.930 |
| color/details | 3.995 | 5.302 | 7.180 |
| plain/random | 13.899 | 8.362 | 9.559 |
| plain/named | 13.221 | 4.978 | 7.180 |
| plain/filtered | 14.263 | 4.577 | 7.184 |
| plain/variant | 11.988 | 5.128 | 7.367 |
| plain/list | 20.520 | 10.242 | 8.930 |
| plain/details | 9.873 | 6.561 | 7.180 |

Maximum public RSS: 9.684 MiB (10 MiB limit). Random/full-list P95: at most
14.381 ms (25 ms limit). Short-command P95: at most 13.107 ms, **above the
unchanged 8 ms threshold**; D20 also exceeds that threshold in this run.
Shared-host scheduling prevents a clean latency clearance; no threshold is relaxed.

Warm command checks, three 300 ms samples and one processor: worst short median
0.789 ms (0.85 ms limit), random/full-list median 3.099 ms (12 ms limit).
Largest-area truecolor renderer median was 0.601 ms, slightly above the 0.600 ms
limit. A follow-up three one-second-sample run reached 0.918 ms; the renderer
source is unchanged. A same-host D20 comparison gave a 0.720 ms median (also
over the threshold). These render measurements do **not** clear the render gate.
A comparable owner-machine D20/D21 check remains part of D23 performance review.

120×40 Pokédex with collected Mega X shiny artwork: three 300 ms one-processor
View samples 1.294/1.383/2.558 ms, median 1.383 ms, approximately 442.4 kB/op
and 1,964 allocations/op. This measures frame construction, not input-to-display
latency. Sprite decode/render occurs in a worker only when selection changes;
View imports the renderer's SGR cells and preserves original source proportions.

Dark-mode active Pokédex idle sample: three seconds, 0 emitted bytes and 0 CPU
ticks at the host's clock resolution; process peak RSS 8,344 KiB (8.148 MiB).
This single sample does not guarantee zero resource use in every session.
Follow Terminal retains its deliberate focused background-color checks.

Full tests/vet/race and CGO-free build pass. Terminal cases cover keyboard/mouse
search and exact variant selection, uncollected form/palette locks, anonymous
evolution navigation, fresh creation, 40×12 and compact/wide/resize, Native,
NO_COLOR, live Follow Terminal and restoration. The seeded test database remains
byte-identical across browsing; trainer-switch tests verify no cross-profile
collection retention. Automated frame/hit tests exercise all modes and search
keyboard pages. Multiplexer and real terminal translucency remain owner checks;
no new network-isolation verification is claimed.


## D21 approved device revision measurements

Measured on Intel Xeon Platinum 8573C against committed D20 (`5ad62fd5d1e896eec24cb360e685038052f7e176`), using stripped CGO-disabled builds, GOMAXPROCS=1, five warmups and 100 fresh-process samples per public command/mode, plus ten peak-RSS samples. These results are not directly comparable to the earlier host.

- Binary: 13,422,752 bytes (12.80 MiB), below 16 MiB; D20: 13,316,256 bytes.
- Public peak RSS: at most 9,836 KiB, below 10 MiB.
- Random/full-list P95: at most 6.126 ms, below 25 ms.
- Short-command P95: at most 8.143 ms (color exact-variant command), narrowly above the 8 ms target. Other short cases ranged 3.347–4.034 ms. This gate remains unresolved; no blanket performance pass is claimed.
- Device View benchmark: three runs, median 1.040 ms/frame, 552,676 B/op and 2,859 allocations/op. This measures frame construction, not end-to-end terminal latency.
- Warm-command and largest-render budgets were not remeasured for this layout revision; previous renderer findings remain unresolved.
- Seventeen PTY scenarios passed: keyboard/mouse browsing, compact and minimum layouts, safe search, exact appearance locks, evolution locks, resizing, NO_COLOR, Native, and live Follow Terminal changes. Terminal state was restored and the seeded database remained byte-identical. Full tests and vet passed; full race checks passed during the redesign and affected TUI/CLI race checks passed after final code changes.


## D22 activity view measurements

Intel Xeon Platinum 8573C, Go 1.27.1, Linux amd64; three 300 ms runs, one CPU.
View construction uses a 120×40 frame; encounter history and locked-goal
benchmarks contain 50 rows/goals. Trainer uses the empty collection state.
These measure frame construction, not terminal end-to-end latency.

| View | Median ns/op | B/op | Allocations/op |
| --- | ---: | ---: | ---: |
| Encounter history | 1,149,215 | 504,492 | 4,084 |
| Trainer | 765,412 | 388,805 | 1,800 |
| Achievements | 915,892 | 468,331 | 3,213 |

Stripped CGO-disabled binary: 13,508,768 bytes (12.88 MiB), below 16 MiB.
Navigation/scrolling runs no database work and prepares no new sprite. Worker
reads and explicit encounters run outside Update/View. No animation/timer was
added; the existing Follow Terminal focused polling remains unchanged. Screen
transition clears happen once per transition rather than every frame.

Public command startup/RSS and renderer budgets were not remeasured in D22.
The D21 8.143 ms short-command P95 and previous warm/render budget findings
remain open for the resource review; no blanket resource-budget pass is claimed.

D22 terminal verification: 21 running PTY scenarios passed, including mouse-only
encounter/history entry navigation, compact/minimum layouts, NO_COLOR artwork,
resize, live Follow Terminal replies and appearance return. All exits restored
terminal state. The five explicit test encounters added exactly five database
rows; the separate read-only browsing/theme/profile-chooser cases left the
seeded database byte-identical. Real terminal translucency and multiplexer
compatibility remain manual checks for D23.
