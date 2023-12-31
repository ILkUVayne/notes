#include "textflag.h"

// 定义int
// Id = 9527
DATA ·Id+0(SB)/1,$0x37
DATA ·Id+1(SB)/1,$0x25
DATA ·Id+2(SB)/1,$0x00
DATA ·Id+3(SB)/1,$0x00
DATA ·Id+4(SB)/1,$0x00
DATA ·Id+5(SB)/1,$0x00
DATA ·Id+6(SB)/1,$0x00
DATA ·Id+7(SB)/1,$0x00
GLOBL ·Id(SB),NOPTR,$8

// 定义字符串
// Name = lyer
// go 字符串结构体
// type reflect.StringHeader struct {
//      Data uintptr
//      Len int
// }

// 方法一
// String
DATA ·NameData+0(SB)/8,$"lyer"
GLOBL ·NameData(SB),NOPTR,$8

// StringHeader
// Data
DATA ·Name+0(SB)/8,$·NameData(SB)
// Len
DATA ·Name+8(SB)/8,$4
GLOBL ·Name(SB),NOPTR,$16


// ·Name内存长度24字节 前16字节表示reflect.StringHeader 后8字节存放字符串
// 前8字节表示Data 指向后8字节$·Name+16(SB)
DATA ·Name2+0(SB)/8,$·Name2+16(SB)
// 中8字节表示字符串长度，这里是4
DATA ·Name2+8(SB)/8,$4
// 后8字节存放字符串
DATA ·Name2+16(SB)/8,$"lyer"
GLOBL ·Name2(SB),NOPTR,$24

// Name = Hello world!
DATA ·text<>+0(SB)/8,$"Hello wo"
DATA ·text<>+8(SB)/8,$"rld!"
GLOBL ·text<>(SB),NOPTR,$16
// Data
DATA ·HelloWorld+0(SB)/8,$·text<>(SB)
// Len
DATA ·HelloWorld+8(SB)/8,$12
GLOBL ·HelloWorld(SB),NOPTR,$16

// slice HelloWorld1 []string

// type reflect.SliceHeader struct {
//      Data uintptr
//      Len int
//      Cap int
// }

// Data
DATA ·HelloWorld1+0(SB)/8,$·text<>(SB)
// Len
DATA ·HelloWorld1+8(SB)/8,$12
// Cap
DATA ·HelloWorld1+16(SB)/8,$16
GLOBL ·HelloWorld1(SB),NOPTR,$24


// func Hello()
TEXT ·Hello(SB),$16-0
MOVQ ·helloWorld+0(SB), AX
MOVQ AX, 0(SP)
MOVQ ·helloWorld+8(SB), BX
MOVQ BX, 8(SP)
CALL runtime·printstring(SB)
CALL runtime·printnl(SB)
RET

// func Swap(a,b int) (int, int)
TEXT ·Swap(SB),$0-32
MOVQ a+0(FP), AX
MOVQ b+8(FP), BX
MOVQ BX, ret0+16(FP)
MOVQ AX, ret1+24(FP)
RET
