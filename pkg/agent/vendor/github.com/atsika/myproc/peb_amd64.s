// https://github.com/C-Sto/BananaPhone/blob/master/pkg/BananaPhone/asm_x64.s

//func GetPEBptr() uintptr
TEXT ·GetPEBptr(SB), $0-8
    MOVQ 	0x60(GS), AX
    MOVQ	AX, ret+0(FP)
    RET
