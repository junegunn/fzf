#include "textflag.h"

// func indexByteTwo(s []byte, b1, b2 byte) int
//
// Returns the index of the first occurrence of b1 or b2 in s, or -1.
// Uses ARM64 NEON to search for both bytes in a single pass over the data.
// Adapted from Go's internal/bytealg/indexbyte_arm64.s (single-byte version).
TEXT ·indexByteTwo(SB),NOSPLIT,$0-40
	MOVD	s_base+0(FP), R0
	MOVD	s_len+8(FP), R2
	MOVBU	b1+24(FP), R1
	MOVBU	b2+25(FP), R7
	MOVD	$ret+32(FP), R8

	// Core algorithm:
	// For each 32-byte chunk we calculate a 64-bit syndrome value,
	// with two bits per byte. We compare against both b1 and b2,
	// OR the results, then use the same syndrome extraction as
	// Go's IndexByte.

	CBZ	R2, fail
	MOVD	R0, R11
	// Magic constant 0x40100401 allows us to identify which lane matches.
	// Each byte in the group of 4 gets a distinct bit: 1, 4, 16, 64.
	MOVD	$0x40100401, R5
	VMOV	R1, V0.B16    // V0 = splat(b1)
	VMOV	R7, V7.B16    // V7 = splat(b2)
	// Work with aligned 32-byte chunks
	BIC	$0x1f, R0, R3
	VMOV	R5, V5.S4
	ANDS	$0x1f, R0, R9
	AND	$0x1f, R2, R10
	BEQ	loop

	// Input string is not 32-byte aligned. Process the first
	// aligned 32-byte block and mask off bytes before our start.
	VLD1.P	(R3), [V1.B16, V2.B16]
	SUB	$0x20, R9, R4
	ADDS	R4, R2, R2
	// Compare against both needles
	VCMEQ	V0.B16, V1.B16, V3.B16  // b1 vs first 16 bytes
	VCMEQ	V7.B16, V1.B16, V8.B16  // b2 vs first 16 bytes
	VORR	V8.B16, V3.B16, V3.B16  // combine
	VCMEQ	V0.B16, V2.B16, V4.B16  // b1 vs second 16 bytes
	VCMEQ	V7.B16, V2.B16, V9.B16  // b2 vs second 16 bytes
	VORR	V9.B16, V4.B16, V4.B16  // combine
	// Build syndrome
	VAND	V5.B16, V3.B16, V3.B16
	VAND	V5.B16, V4.B16, V4.B16
	VADDP	V4.B16, V3.B16, V6.B16
	VADDP	V6.B16, V6.B16, V6.B16
	VMOV	V6.D[0], R6
	// Clear the irrelevant lower bits
	LSL	$1, R9, R4
	LSR	R4, R6, R6
	LSL	R4, R6, R6
	// The first block can also be the last
	BLS	masklast
	// Have we found something already?
	CBNZ	R6, tail

loop:
	VLD1.P	(R3), [V1.B16, V2.B16]
	SUBS	$0x20, R2, R2
	// Compare against both needles, OR results
	VCMEQ	V0.B16, V1.B16, V3.B16
	VCMEQ	V7.B16, V1.B16, V8.B16
	VORR	V8.B16, V3.B16, V3.B16
	VCMEQ	V0.B16, V2.B16, V4.B16
	VCMEQ	V7.B16, V2.B16, V9.B16
	VORR	V9.B16, V4.B16, V4.B16
	// If we're out of data we finish regardless of the result
	BLS	end
	// Fast check: OR both halves and check for any match
	VORR	V4.B16, V3.B16, V6.B16
	VADDP	V6.D2, V6.D2, V6.D2
	VMOV	V6.D[0], R6
	CBZ	R6, loop

end:
	// Found something or out of data — build full syndrome
	VAND	V5.B16, V3.B16, V3.B16
	VAND	V5.B16, V4.B16, V4.B16
	VADDP	V4.B16, V3.B16, V6.B16
	VADDP	V6.B16, V6.B16, V6.B16
	VMOV	V6.D[0], R6
	// Only mask for the last block
	BHS	tail

masklast:
	// Clear irrelevant upper bits
	ADD	R9, R10, R4
	AND	$0x1f, R4, R4
	SUB	$0x20, R4, R4
	NEG	R4<<1, R4
	LSL	R4, R6, R6
	LSR	R4, R6, R6

tail:
	CBZ	R6, fail
	RBIT	R6, R6
	SUB	$0x20, R3, R3
	CLZ	R6, R6
	ADD	R6>>1, R3, R0
	SUB	R11, R0, R0
	MOVD	R0, (R8)
	RET

fail:
	MOVD	$-1, R0
	MOVD	R0, (R8)
	RET

// func lastIndexByteTwo(s []byte, b1, b2 byte) int
//
// Returns the index of the last occurrence of b1 or b2 in s, or -1.
// Scans backward using ARM64 NEON.
TEXT ·lastIndexByteTwo(SB),NOSPLIT,$0-40
	MOVD	s_base+0(FP), R0
	MOVD	s_len+8(FP), R2
	MOVBU	b1+24(FP), R1
	MOVBU	b2+25(FP), R7
	MOVD	$ret+32(FP), R8

	CBZ	R2, lfail
	MOVD	R0, R11          // save base
	ADD	R0, R2, R12      // R12 = end = base + len
	MOVD	$0x40100401, R5
	VMOV	R1, V0.B16       // V0 = splat(b1)
	VMOV	R7, V7.B16       // V7 = splat(b2)
	VMOV	R5, V5.S4

	// Align: find the aligned block containing the last byte
	SUB	$1, R12, R3
	BIC	$0x1f, R3, R3    // R3 = start of aligned block containing last byte

	// --- Process tail block ---
	VLD1	(R3), [V1.B16, V2.B16]
	VCMEQ	V0.B16, V1.B16, V3.B16
	VCMEQ	V7.B16, V1.B16, V8.B16
	VORR	V8.B16, V3.B16, V3.B16
	VCMEQ	V0.B16, V2.B16, V4.B16
	VCMEQ	V7.B16, V2.B16, V9.B16
	VORR	V9.B16, V4.B16, V4.B16
	VAND	V5.B16, V3.B16, V3.B16
	VAND	V5.B16, V4.B16, V4.B16
	VADDP	V4.B16, V3.B16, V6.B16
	VADDP	V6.B16, V6.B16, V6.B16
	VMOV	V6.D[0], R6

	// Mask upper bits (bytes past end of slice)
	// tail_bytes = end - R3 (1..32)
	SUB	R3, R12, R10     // R10 = tail_bytes
	MOVD	$64, R4
	SUB	R10<<1, R4, R4   // R4 = 64 - 2*tail_bytes
	LSL	R4, R6, R6
	LSR	R4, R6, R6

	// Is this also the head block?
	CMP	R11, R3          // R3 - R11
	BLO	lmaskfirst       // R3 < base: head+tail in same block
	BEQ	ltailonly         // R3 == base: single aligned block

	// R3 > base: more blocks before this one
	CBNZ	R6, llast
	B	lbacksetup

ltailonly:
	// Single block, already masked upper bits
	CBNZ	R6, llast
	B	lfail

lmaskfirst:
	// Mask lower bits (bytes before start of slice)
	SUB	R3, R11, R4      // R4 = base - R3
	LSL	$1, R4, R4
	LSR	R4, R6, R6
	LSL	R4, R6, R6
	CBNZ	R6, llast
	B	lfail

lbacksetup:
	SUB	$0x20, R3

lbackloop:
	VLD1	(R3), [V1.B16, V2.B16]
	VCMEQ	V0.B16, V1.B16, V3.B16
	VCMEQ	V7.B16, V1.B16, V8.B16
	VORR	V8.B16, V3.B16, V3.B16
	VCMEQ	V0.B16, V2.B16, V4.B16
	VCMEQ	V7.B16, V2.B16, V9.B16
	VORR	V9.B16, V4.B16, V4.B16
	// Quick check: any match in this block?
	VORR	V4.B16, V3.B16, V6.B16
	VADDP	V6.D2, V6.D2, V6.D2
	VMOV	V6.D[0], R6

	// Is this a head block? (R3 < base)
	CMP	R11, R3
	BLO	lheadblock

	// Full block (R3 >= base)
	CBNZ	R6, lbackfound
	// More blocks?
	BEQ	lfail            // R3 == base, no more
	SUB	$0x20, R3
	B	lbackloop

lbackfound:
	// Build full syndrome
	VAND	V5.B16, V3.B16, V3.B16
	VAND	V5.B16, V4.B16, V4.B16
	VADDP	V4.B16, V3.B16, V6.B16
	VADDP	V6.B16, V6.B16, V6.B16
	VMOV	V6.D[0], R6
	B	llast

lheadblock:
	// R3 < base. Build full syndrome if quick check had a match.
	CBZ	R6, lfail
	VAND	V5.B16, V3.B16, V3.B16
	VAND	V5.B16, V4.B16, V4.B16
	VADDP	V4.B16, V3.B16, V6.B16
	VADDP	V6.B16, V6.B16, V6.B16
	VMOV	V6.D[0], R6
	// Mask lower bits
	SUB	R3, R11, R4      // R4 = base - R3
	LSL	$1, R4, R4
	LSR	R4, R6, R6
	LSL	R4, R6, R6
	CBZ	R6, lfail

llast:
	// Find last match: highest set bit in syndrome
	// Syndrome has bit 2i set for matching byte i.
	// CLZ gives leading zeros; byte_offset = (63 - CLZ) / 2.
	CLZ	R6, R6
	MOVD	$63, R4
	SUB	R6, R4, R6       // R6 = 63 - CLZ = bit position
	LSR	$1, R6            // R6 = byte offset within block
	ADD	R3, R6, R0        // R0 = absolute address
	SUB	R11, R0, R0       // R0 = slice index
	MOVD	R0, (R8)
	RET

lfail:
	MOVD	$-1, R0
	MOVD	R0, (R8)
	RET
