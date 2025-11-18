// +build amd64

#include "textflag.h"

// func checksumAVX2(data []byte) uint64
TEXT ·checksumAVX2(SB), NOSPLIT, $0-32
    MOVQ data+0(FP), SI      // SI = data pointer
    MOVQ data+8(FP), CX      // CX = length
    XORQ AX, AX              // AX = sum (accumulator)

    // Check if we have at least 32 bytes
    CMPQ CX, $32
    JL   scalar_checksum

    // Zero out AVX2 accumulator
    VPXOR Y0, Y0, Y0         // Y0 = 0 (main accumulator)
    VPXOR Y1, Y1, Y1         // Y1 = 0 (secondary accumulator)

avx2_loop:
    CMPQ CX, $32
    JL   avx2_remainder

    // Load 32 bytes
    VMOVDQU (SI), Y2

    // Unpack bytes to words and accumulate
    VPXOR Y3, Y3, Y3
    VPUNPCKLBW Y3, Y2, Y4    // Low 16 bytes -> 16 words
    VPUNPCKHBW Y3, Y2, Y5    // High 16 bytes -> 16 words

    VPADDW Y4, Y0, Y0        // Add to accumulator
    VPADDW Y5, Y1, Y1

    ADDQ $32, SI
    SUBQ $32, CX
    JMP  avx2_loop

avx2_remainder:
    // Combine Y0 and Y1
    VPADDW Y1, Y0, Y0

    // Horizontal sum of Y0
    VEXTRACTI128 $1, Y0, X1  // Extract high 128 bits
    VPADDW X1, X0, X0        // Add high and low

    // Sum 8 words in X0
    VPSRLDQ $8, X0, X1
    VPADDW X1, X0, X0
    VPSRLDQ $4, X0, X1
    VPADDW X1, X0, X0
    VPSRLDQ $2, X0, X1
    VPADDW X1, X0, X0

    // Extract result to AX
    VMOVQ X0, AX
    ANDQ $0xFFFF, AX

    // Process remaining bytes with scalar code
    JMP scalar_checksum

scalar_checksum:
    // Process remaining bytes in 2-byte chunks
    XORQ DX, DX              // DX = temp register

scalar_loop:
    CMPQ CX, $2
    JL   last_byte

    MOVWLZX (SI), DX         // Load 2 bytes
    ADDQ DX, AX
    ADDQ $2, SI
    SUBQ $2, CX
    JMP  scalar_loop

last_byte:
    CMPQ CX, $1
    JNE  done

    MOVBLZX (SI), DX         // Load 1 byte
    SHLQ $8, DX              // Shift to high byte
    ADDQ DX, AX

done:
    MOVQ AX, ret+24(FP)
    RET

// func checksumSSE2(data []byte) uint64
TEXT ·checksumSSE2(SB), NOSPLIT, $0-32
    MOVQ data+0(FP), SI      // SI = data pointer
    MOVQ data+8(FP), CX      // CX = length
    XORQ AX, AX              // AX = sum

    // Check if we have at least 16 bytes
    CMPQ CX, $16
    JL   sse2_scalar

    // Zero out SSE2 accumulator
    PXOR X0, X0              // X0 = main accumulator
    PXOR X1, X1              // X1 = secondary accumulator

sse2_loop:
    CMPQ CX, $16
    JL   sse2_remainder

    // Load 16 bytes
    MOVOU (SI), X2

    // Unpack and accumulate
    PXOR X3, X3
    MOVOA X2, X4
    PUNPCKLBW X3, X4         // Low 8 bytes -> 8 words
    PUNPCKHBW X3, X2         // High 8 bytes -> 8 words

    PADDW X4, X0
    PADDW X2, X1

    ADDQ $16, SI
    SUBQ $16, CX
    JMP  sse2_loop

sse2_remainder:
    // Combine accumulators
    PADDW X1, X0

    // Horizontal sum
    MOVOA X0, X1
    PSRLDQ $8, X1
    PADDW X1, X0
    MOVOA X0, X1
    PSRLDQ $4, X1
    PADDW X1, X0
    MOVOA X0, X1
    PSRLDQ $2, X1
    PADDW X1, X0

    // Extract result
    MOVQ X0, AX
    ANDQ $0xFFFF, AX

sse2_scalar:
    // Process remaining bytes
    XORQ DX, DX

sse2_scalar_loop:
    CMPQ CX, $2
    JL   sse2_last_byte

    MOVWLZX (SI), DX
    ADDQ DX, AX
    ADDQ $2, SI
    SUBQ $2, CX
    JMP  sse2_scalar_loop

sse2_last_byte:
    CMPQ CX, $1
    JNE  sse2_done

    MOVBLZX (SI), DX
    SHLQ $8, DX
    ADDQ DX, AX

sse2_done:
    MOVQ AX, ret+24(FP)
    RET

// func hasAVX2() bool
TEXT ·hasAVX2(SB), NOSPLIT, $0-1
    // Check CPUID for AVX2 support
    MOVQ $7, AX
    MOVQ $0, CX
    CPUID
    SHRQ $5, BX              // Bit 5 of EBX indicates AVX2
    ANDQ $1, BX
    MOVB BX, ret+0(FP)
    RET

// func hasSSE2() bool
TEXT ·hasSSE2(SB), NOSPLIT, $0-1
    // Check CPUID for SSE2 support
    MOVQ $1, AX
    CPUID
    SHRQ $26, DX             // Bit 26 of EDX indicates SSE2
    ANDQ $1, DX
    MOVB DX, ret+0(FP)
    RET
