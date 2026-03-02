# SIMD byte search: `indexByteTwo` / `lastIndexByteTwo`

## What these functions do

`indexByteTwo(s []byte, b1, b2 byte) int` — returns the index of the
**first** occurrence of `b1` or `b2` in `s`, or `-1`.

`lastIndexByteTwo(s []byte, b1, b2 byte) int` — returns the index of the
**last** occurrence of `b1` or `b2` in `s`, or `-1`.

They are used by the fuzzy matching algorithm (`algo.go`) to skip ahead
during case-insensitive search. Instead of calling `bytes.IndexByte` twice
(once for lowercase, once for uppercase), a single SIMD pass finds both at
once.

## File layout

| File                  | Purpose                                                           |
| ------                | ---------                                                         |
| `indexbyte2_arm64.go` | Go declarations (`//go:noescape`) for ARM64                       |
| `indexbyte2_arm64.s`  | ARM64 NEON assembly (32-byte aligned blocks, syndrome extraction) |
| `indexbyte2_amd64.go` | Go declarations + AVX2 runtime detection for AMD64                |
| `indexbyte2_amd64.s`  | AMD64 AVX2/SSE2 assembly with CPUID dispatch                      |
| `indexbyte2_other.go` | Pure Go fallback for all other architectures                      |
| `indexbyte2_test.go`  | Unit tests, exhaustive tests, fuzz tests, and benchmarks          |

## How the SIMD implementations work

**ARM64 (NEON):**
- Broadcasts both needle bytes into NEON registers (`VMOV`).
- Processes 32-byte aligned chunks. For each chunk, compares all bytes
  against both needles (`VCMEQ`), ORs the results (`VORR`), and builds a
  64-bit syndrome with 2 bits per byte.
- `indexByteTwo` uses `RBIT` + `CLZ` to find the lowest set bit (first match).
- `lastIndexByteTwo` scans backward and uses `CLZ` on the raw syndrome to
  find the highest set bit (last match).
- Handles alignment and partial first/last blocks with bit masking.
- Adapted from Go's `internal/bytealg/indexbyte_arm64.s`.

**AMD64 (AVX2 with SSE2 fallback):**
- At init time, `cpuHasAVX2()` checks CPUID + XGETBV for AVX2 and OS YMM
  support. The result is cached in `_useAVX2`.
- **AVX2 path** (inputs >= 32 bytes, when available):
  - Broadcasts both needles via `VPBROADCASTB`.
  - Processes 32-byte blocks: `VPCMPEQB` against both needles, `VPOR`, then
    `VPMOVMSKB` to get a 32-bit mask.
  - 5 instructions per loop iteration (vs 7 for SSE2) at 2x the throughput.
  - `VZEROUPPER` before every return to avoid SSE/AVX transition penalties.
- **SSE2 fallback** (inputs < 32 bytes, or CPUs without AVX2):
  - Broadcasts via `PUNPCKLBW` + `PSHUFL`.
  - Processes 16-byte blocks: `PCMPEQB`, `POR`, `PMOVMSKB`.
  - Small inputs (<16 bytes) are handled with page-boundary-safe loads.
- Both paths use `BSFL` (forward) / `BSRL` (reverse) for bit scanning.
- Adapted from Go's `internal/bytealg/indexbyte_amd64.s`.

**Fallback (other platforms):**
- `indexByteTwo` uses two `bytes.IndexByte` calls with scope-limiting
  (search `b1` first, then limit the `b2` search to `s[:i1]`).
- `lastIndexByteTwo` uses a simple backward for loop.

## Running tests

```bash
# Unit + exhaustive tests
go test ./src/algo/ -run 'TestIndexByteTwo|TestLastIndexByteTwo' -v

# Fuzz tests (run for 10 seconds each)
go test ./src/algo/ -run '^$' -fuzz FuzzIndexByteTwo -fuzztime 10s
go test ./src/algo/ -run '^$' -fuzz FuzzLastIndexByteTwo -fuzztime 10s

# Cross-architecture: test amd64 on an arm64 Mac (via Rosetta)
GOARCH=amd64 go test ./src/algo/ -run 'TestIndexByteTwo|TestLastIndexByteTwo' -v
GOARCH=amd64 go test ./src/algo/ -run '^$' -fuzz FuzzIndexByteTwo -fuzztime 10s
GOARCH=amd64 go test ./src/algo/ -run '^$' -fuzz FuzzLastIndexByteTwo -fuzztime 10s
```

## Running micro-benchmarks

```bash
# All indexByteTwo / lastIndexByteTwo benchmarks
go test ./src/algo/ -bench 'IndexByteTwo' -benchmem

# Specific size
go test ./src/algo/ -bench 'IndexByteTwo_1000'
```

Each benchmark compares the SIMD `asm` implementation against reference
implementations (`2xIndexByte` using `bytes.IndexByte`, and a simple `loop`).

## Correctness verification

The assembly is verified by three layers of testing:

1. **Table-driven tests** — known inputs with expected outputs.
2. **Exhaustive tests** — all lengths 0–256, every match position, no-match
   cases, and both-bytes-present cases, compared against a simple loop
   reference.
3. **Fuzz tests** — randomized inputs via `testing.F`, compared against the
   same loop reference.
