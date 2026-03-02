#include "textflag.h"

// func cpuHasAVX2() bool
//
// Checks CPUID and XGETBV for AVX2 + OS YMM support.
TEXT ·cpuHasAVX2(SB),NOSPLIT,$0-1
	MOVQ	BX, R8          // save BX (callee-saved, clobbered by CPUID)

	// Check max CPUID leaf >= 7
	MOVL	$0, AX
	CPUID
	CMPL	AX, $7
	JL	cpuid_no

	// Check OSXSAVE (CPUID.1:ECX bit 27)
	MOVL	$1, AX
	CPUID
	TESTL	$(1<<27), CX
	JZ	cpuid_no

	// Check AVX2 (CPUID.7.0:EBX bit 5)
	MOVL	$7, AX
	MOVL	$0, CX
	CPUID
	TESTL	$(1<<5), BX
	JZ	cpuid_no

	// Check OS YMM state support via XGETBV
	MOVL	$0, CX
	BYTE	$0x0F; BYTE $0x01; BYTE $0xD0  // XGETBV → EDX:EAX
	ANDL	$6, AX          // bits 1 (XMM) and 2 (YMM)
	CMPL	AX, $6
	JNE	cpuid_no

	MOVQ	R8, BX          // restore BX
	MOVB	$1, ret+0(FP)
	RET

cpuid_no:
	MOVQ	R8, BX
	MOVB	$0, ret+0(FP)
	RET

// func indexByteTwo(s []byte, b1, b2 byte) int
//
// Returns the index of the first occurrence of b1 or b2 in s, or -1.
// Uses AVX2 (32 bytes/iter) when available, SSE2 (16 bytes/iter) otherwise.
TEXT ·indexByteTwo(SB),NOSPLIT,$0-40
	MOVQ	s_base+0(FP), SI
	MOVQ	s_len+8(FP), BX
	MOVBLZX	b1+24(FP), AX
	MOVBLZX	b2+25(FP), CX
	LEAQ	ret+32(FP), R8

	TESTQ	BX, BX
	JEQ	fwd_failure

	// Try AVX2 for inputs >= 32 bytes
	CMPQ	BX, $32
	JLT	fwd_sse2
	CMPB	·_useAVX2(SB), $1
	JNE	fwd_sse2

	// ====== AVX2 forward search ======
	MOVD	AX, X0
	VPBROADCASTB	X0, Y0       // Y0 = splat(b1)
	MOVD	CX, X1
	VPBROADCASTB	X1, Y1       // Y1 = splat(b2)

	MOVQ	SI, DI
	LEAQ	-32(SI)(BX*1), AX    // AX = last valid 32-byte chunk
	JMP	fwd_avx2_entry

fwd_avx2_loop:
	VMOVDQU	(DI), Y2
	VPCMPEQB	Y0, Y2, Y3
	VPCMPEQB	Y1, Y2, Y4
	VPOR	Y3, Y4, Y3
	VPMOVMSKB	Y3, DX
	BSFL	DX, DX
	JNZ	fwd_avx2_success
	ADDQ	$32, DI

fwd_avx2_entry:
	CMPQ	DI, AX
	JB	fwd_avx2_loop

	// Last 32-byte chunk (may overlap with previous)
	MOVQ	AX, DI
	VMOVDQU	(AX), Y2
	VPCMPEQB	Y0, Y2, Y3
	VPCMPEQB	Y1, Y2, Y4
	VPOR	Y3, Y4, Y3
	VPMOVMSKB	Y3, DX
	BSFL	DX, DX
	JNZ	fwd_avx2_success

	MOVQ	$-1, (R8)
	VZEROUPPER
	RET

fwd_avx2_success:
	SUBQ	SI, DI
	ADDQ	DX, DI
	MOVQ	DI, (R8)
	VZEROUPPER
	RET

	// ====== SSE2 forward search (< 32 bytes or no AVX2) ======

fwd_sse2:
	// Broadcast b1 into X0
	MOVD	AX, X0
	PUNPCKLBW	X0, X0
	PUNPCKLBW	X0, X0
	PSHUFL	$0, X0, X0

	// Broadcast b2 into X4
	MOVD	CX, X4
	PUNPCKLBW	X4, X4
	PUNPCKLBW	X4, X4
	PSHUFL	$0, X4, X4

	CMPQ	BX, $16
	JLT	fwd_small

	MOVQ	SI, DI
	LEAQ	-16(SI)(BX*1), AX
	JMP	fwd_sseloopentry

fwd_sseloop:
	MOVOU	(DI), X1
	MOVOU	X1, X2
	PCMPEQB	X0, X1
	PCMPEQB	X4, X2
	POR	X2, X1
	PMOVMSKB	X1, DX
	BSFL	DX, DX
	JNZ	fwd_ssesuccess
	ADDQ	$16, DI

fwd_sseloopentry:
	CMPQ	DI, AX
	JB	fwd_sseloop

	// Search the last 16-byte chunk (may overlap)
	MOVQ	AX, DI
	MOVOU	(AX), X1
	MOVOU	X1, X2
	PCMPEQB	X0, X1
	PCMPEQB	X4, X2
	POR	X2, X1
	PMOVMSKB	X1, DX
	BSFL	DX, DX
	JNZ	fwd_ssesuccess

fwd_failure:
	MOVQ	$-1, (R8)
	RET

fwd_ssesuccess:
	SUBQ	SI, DI
	ADDQ	DX, DI
	MOVQ	DI, (R8)
	RET

fwd_small:
	// Check if loading 16 bytes from SI would cross a page boundary
	LEAQ	16(SI), AX
	TESTW	$0xff0, AX
	JEQ	fwd_endofpage

	MOVOU	(SI), X1
	MOVOU	X1, X2
	PCMPEQB	X0, X1
	PCMPEQB	X4, X2
	POR	X2, X1
	PMOVMSKB	X1, DX
	BSFL	DX, DX
	JZ	fwd_failure
	CMPL	DX, BX
	JAE	fwd_failure
	MOVQ	DX, (R8)
	RET

fwd_endofpage:
	MOVOU	-16(SI)(BX*1), X1
	MOVOU	X1, X2
	PCMPEQB	X0, X1
	PCMPEQB	X4, X2
	POR	X2, X1
	PMOVMSKB	X1, DX
	MOVL	BX, CX
	SHLL	CX, DX
	SHRL	$16, DX
	BSFL	DX, DX
	JZ	fwd_failure
	MOVQ	DX, (R8)
	RET

// func lastIndexByteTwo(s []byte, b1, b2 byte) int
//
// Returns the index of the last occurrence of b1 or b2 in s, or -1.
// Uses AVX2 (32 bytes/iter) when available, SSE2 (16 bytes/iter) otherwise.
TEXT ·lastIndexByteTwo(SB),NOSPLIT,$0-40
	MOVQ	s_base+0(FP), SI
	MOVQ	s_len+8(FP), BX
	MOVBLZX	b1+24(FP), AX
	MOVBLZX	b2+25(FP), CX
	LEAQ	ret+32(FP), R8

	TESTQ	BX, BX
	JEQ	back_failure

	// Try AVX2 for inputs >= 32 bytes
	CMPQ	BX, $32
	JLT	back_sse2
	CMPB	·_useAVX2(SB), $1
	JNE	back_sse2

	// ====== AVX2 backward search ======
	MOVD	AX, X0
	VPBROADCASTB	X0, Y0
	MOVD	CX, X1
	VPBROADCASTB	X1, Y1

	// DI = start of last 32-byte chunk
	LEAQ	-32(SI)(BX*1), DI

back_avx2_loop:
	CMPQ	DI, SI
	JBE	back_avx2_first

	VMOVDQU	(DI), Y2
	VPCMPEQB	Y0, Y2, Y3
	VPCMPEQB	Y1, Y2, Y4
	VPOR	Y3, Y4, Y3
	VPMOVMSKB	Y3, DX
	BSRL	DX, DX
	JNZ	back_avx2_success
	SUBQ	$32, DI
	JMP	back_avx2_loop

back_avx2_first:
	// First 32 bytes (DI <= SI, load from SI)
	VMOVDQU	(SI), Y2
	VPCMPEQB	Y0, Y2, Y3
	VPCMPEQB	Y1, Y2, Y4
	VPOR	Y3, Y4, Y3
	VPMOVMSKB	Y3, DX
	BSRL	DX, DX
	JNZ	back_avx2_firstsuccess

	MOVQ	$-1, (R8)
	VZEROUPPER
	RET

back_avx2_success:
	SUBQ	SI, DI
	ADDQ	DX, DI
	MOVQ	DI, (R8)
	VZEROUPPER
	RET

back_avx2_firstsuccess:
	MOVQ	DX, (R8)
	VZEROUPPER
	RET

	// ====== SSE2 backward search (< 32 bytes or no AVX2) ======

back_sse2:
	// Broadcast b1 into X0
	MOVD	AX, X0
	PUNPCKLBW	X0, X0
	PUNPCKLBW	X0, X0
	PSHUFL	$0, X0, X0

	// Broadcast b2 into X4
	MOVD	CX, X4
	PUNPCKLBW	X4, X4
	PUNPCKLBW	X4, X4
	PSHUFL	$0, X4, X4

	CMPQ	BX, $16
	JLT	back_small

	// DI = start of last 16-byte chunk
	LEAQ	-16(SI)(BX*1), DI

back_sseloop:
	CMPQ	DI, SI
	JBE	back_ssefirst

	MOVOU	(DI), X1
	MOVOU	X1, X2
	PCMPEQB	X0, X1
	PCMPEQB	X4, X2
	POR	X2, X1
	PMOVMSKB	X1, DX
	BSRL	DX, DX
	JNZ	back_ssesuccess
	SUBQ	$16, DI
	JMP	back_sseloop

back_ssefirst:
	// First 16 bytes (DI <= SI, load from SI)
	MOVOU	(SI), X1
	MOVOU	X1, X2
	PCMPEQB	X0, X1
	PCMPEQB	X4, X2
	POR	X2, X1
	PMOVMSKB	X1, DX
	BSRL	DX, DX
	JNZ	back_ssefirstsuccess

back_failure:
	MOVQ	$-1, (R8)
	RET

back_ssesuccess:
	SUBQ	SI, DI
	ADDQ	DX, DI
	MOVQ	DI, (R8)
	RET

back_ssefirstsuccess:
	// DX = byte offset from base
	MOVQ	DX, (R8)
	RET

back_small:
	// Check page boundary
	LEAQ	16(SI), AX
	TESTW	$0xff0, AX
	JEQ	back_endofpage

	MOVOU	(SI), X1
	MOVOU	X1, X2
	PCMPEQB	X0, X1
	PCMPEQB	X4, X2
	POR	X2, X1
	PMOVMSKB	X1, DX
	// Mask to first BX bytes: keep bits 0..BX-1
	MOVL	$1, AX
	MOVL	BX, CX
	SHLL	CX, AX
	DECL	AX
	ANDL	AX, DX
	BSRL	DX, DX
	JZ	back_failure
	MOVQ	DX, (R8)
	RET

back_endofpage:
	// Load 16 bytes ending at base+n
	MOVOU	-16(SI)(BX*1), X1
	MOVOU	X1, X2
	PCMPEQB	X0, X1
	PCMPEQB	X4, X2
	POR	X2, X1
	PMOVMSKB	X1, DX
	// Bits correspond to bytes [base+n-16, base+n).
	// We want original bytes [0, n), which are bits [16-n, 16).
	// Mask: keep bits (16-n) through 15.
	MOVL	$16, CX
	SUBL	BX, CX
	SHRL	CX, DX
	SHLL	CX, DX
	BSRL	DX, DX
	JZ	back_failure
	// DX is the bit position in the loaded chunk.
	// Original byte index = DX - (16 - n) = DX + n - 16
	ADDL	BX, DX
	SUBL	$16, DX
	MOVQ	DX, (R8)
	RET
