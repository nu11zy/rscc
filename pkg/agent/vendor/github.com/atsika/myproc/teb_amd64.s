//func GetTEBptr() uintptr
TEXT ·GetTEBptr(SB), $0-8
    MOVQ 	0x30(GS), AX
    MOVQ	AX, ret+0(FP)
    RET
