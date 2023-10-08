# golang 程序（初始化）入口

> 版本：go 1.21 平台：amd64 linux

## 对应平台的go程序实际入口

### 1.编写一个简单的go程序

~~~go
package main

func main() {
	println("Hello world!")
}
~~~

### 2.编译可执行文件

~~~bash
# -gcflags "-N -l" 参数关闭编译器代码优化和函数内联，避免断点和单步执⾏⽆法准确对应源码⾏，避免⼩函数和局部变量被优化掉
$ go build -gcflags "-N -l" -o entry entry.go
~~~

### 3.使用gdb调试

> gdb tips:设置 断点了若发现有多个断点（即可能存在多个同名函数等）时，可使用 (gdb) i b 命令查看每个断点信息

~~~bash
$ gdb entry 
...
(gdb) info file
Local exec file:
        
        Entry point: 0x455e40
...        
        
# Entry point: 0x455e40 即为入口点
# 设置断点

(gdb) b *0x455e40
Breakpoint 1 at 0x455e40: file /usr/local/go_src/21/go/src/runtime/rt0_linux_amd64.s, line 8.

# /usr/local/go_src/21/go/src/runtime 为我安装的golang源码路径
# rt0_linux_amd64.s 即为该断点的汇编代码路径（不同的平台的实际汇编文件会不一样）
# 用你熟悉的方式打开rt0_linux_amd64.s
~~~

rt0_linux_amd64.s

~~~cgo
#include "textflag.h"

TEXT _rt0_amd64_linux(SB),NOSPLIT,$-8
        JMP     _rt0_amd64(SB)

TEXT _rt0_amd64_linux_lib(SB),NOSPLIT,$0
        JMP     _rt0_amd64_lib(SB)
~~~

发现汇编代码中函数实际跳转的是_rt0_amd64(SB)方法，继续设置断点
~~~bash
(gdb) b _rt0_amd64
Breakpoint 2 at 0x454200: file /usr/local/go_src/21/go/src/runtime/asm_amd64.s, line 16.
# 发现该函数在/usr/local/go_src/21/go/src/runtime/asm_amd64.s 16行
~~~

asm_amd64.s 16行代码

~~~cgo
    ...
TEXT _rt0_amd64(SB),NOSPLIT,$-8
	MOVQ	0(SP), DI	// argc
	LEAQ	8(SP), SI	// argv
	JMP	runtime·rt0_go(SB)
    ...
~~~

该方法设置了arg后，跳转到了runtime·rt0_go(SB)方法，设置断点

~~~bash
# 注意汇编函数runtime·rt0_go(SB)的中点需要改成下点，才能正常设置断点
(gdb) b runtime.rt0_go
Breakpoint 3 at 0x454220: file /usr/local/go_src/21/go/src/runtime/asm_amd64.s, line 161.
# 找到runtime.rt0_go方法的在同文件161行
~~~

runtime.rt0_go方法

~~~cgo
TEXT runtime·rt0_go(SB),NOSPLIT|NOFRAME|TOPFRAME,$0
    ...

    CALL	runtime·args(SB)
    CALL	runtime·osinit(SB)
    CALL	runtime·schedinit(SB)
    // create a new goroutine to start program
    MOVQ	$runtime·mainPC(SB), AX		// entry
    PUSHQ	AX
    CALL	runtime·newproc(SB)
    POPQ	AX
    
    // start this M
    CALL	runtime·mstart(SB)
    
    CALL	runtime·abort(SB)	// mstart should never return
    RET
	
    ...

// mainPC is a function value for runtime.main, to be passed to newproc.
// The reference to runtime.main is made via ABIInternal, since the
// actual function (not the ABI0 wrapper) is needed by newproc.
DATA	runtime·mainPC+0(SB)/8,$runtime·main<ABIInternal>(SB)
GLOBL	runtime·mainPC(SB),RODATA,$8
~~~

阅读源码可发现runtime.rt0_go方法即为go程序的实际入口（初始化起点）

### 4.初始化步骤

~~~
# 整理命令行参数
CALL	runtime·args(SB)
# CPU Core 数量
CALL	runtime·osinit(SB)
# 运⾏时环境初始化构造
CALL	runtime·schedinit(SB)

# 创建mian goroutine协程，执行runtime中的mian(runtime.main中进行一系列初始化操作后，最终再执行main.main即我们编写的entry.go)
MOVQ	$runtime·mainPC(SB), AX		// entry
PUSHQ	AX
# 创建协程函数，即golang中的 go关键字， 实现代码在runtime.proc中
CALL	runtime·newproc(SB)
POPQ	AX

// 让当前线程执行mian goroutine
CALL	runtime·mstart(SB)

CALL	runtime·abort(SB)	// mstart should never return
RET
~~~

### 5.结论 

go 程序的起点并非我们编写的main.main函数，而是runtime/asm_amd64.s.runtime.rt0_go（amd64）,在经过汇编代码的一系列初始化运行时环境之后，才会实际开始执行main.main