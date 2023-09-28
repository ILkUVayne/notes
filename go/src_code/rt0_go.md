# ams_amd64.s中rt0_go方法分析

> 使用dlv 工具调式代码，方便了解执行步骤

rt0_go方法即为go程序的实际入口，主要功能是完成Go程序启动时的所有初始化工作

## 1.开始debug

~~~bash
# 添加断点runtime.rt0_go
$ dlv debug
Type 'help' for list of commands.
(dlv) b runtime.rt0_go
Breakpoint 1 set at 0x463680 for runtime.rt0_go() /usr/local/go_src/21/go/src/runtime/asm_amd64.s:161
~~~

## 2.copy argc和argv 到AX和BX寄存器

~~~
MOVQ	DI, AX		// argc
MOVQ	SI, BX		// argv
# 向下移动SP寄存器
SUBQ	$(5*8), SP		// 3args 2auto
ANDQ	$~15, SP
# 把argc和argv分别赋值到SP+24 SP+32
MOVQ	AX, 24(SP)
MOVQ	BX, 32(SP)
~~~

## 3.初始化g0

g0 m0是全局变量，在执行rt0_go是已被初始化
/usr/local/go_src/21/go/src/runtime/proc.go:113
~~~
var (
	m0           m
	g0           g
	mcache0      *mcache
	raceprocctx0 uintptr
	raceFiniLock mutex
)
~~~

next到初始化g0位置

~~~bash
(dlv) n
> runtime.rt0_go() /usr/local/go_src/21/go/src/runtime/asm_amd64.s:170 (PC: 0x463698)
Warning: debugging optimized function
   165:         MOVQ    AX, 24(SP)
   166:         MOVQ    BX, 32(SP)
   167:
   168:         // create istack out of the given (operating system) stack.
   169:         // _cgo_init may update stackguard.
=> 170:         MOVQ    $runtime·g0(SB), DI
   171:         LEAQ    (-64*1024)(SP), BX
   172:         MOVQ    BX, g_stackguard0(DI)
   173:         MOVQ    BX, g_stackguard1(DI)
   174:         MOVQ    BX, (g_stack+stack_lo)(DI)
   175:         MOVQ    SP, (g_stack+stack_hi)(DI)
~~~

代码分析

~~~
# 将g0地址赋值给DI寄存器（前面已经分析了，g0是全局变量已被初始化分配了地址）
MOVQ	$runtime·g0(SB), DI
# SP向下移动64*1024字节，并获取地址，赋值给BX寄存器
LEAQ	(-64*1024)(SP), BX
# BX = *SP-64*1024
# g0.stackguard0 = BX
MOVQ	BX, g_stackguard0(DI)
# g0.stackguard1 = BX
MOVQ	BX, g_stackguard1(DI)
# g0.stack.lo = BX
MOVQ	BX, (g_stack+stack_lo)(DI)
# g0.stack.hi = SP
MOVQ	SP, (g_stack+stack_hi)(DI)
~~~

查看g0值，验证

~~~
(dlv) p g0
runtime.g {
        // 140728952698464 - 140728952632928 = 65,536 = 64*1024
        stack: runtime.stack {lo: 140728952632928, hi: 140728952698464},
        stackguard0: 140728952632928,
        stackguard1: 140728952632928,
        _panic: *runtime._panic nil,
        _defer: *runtime._defer nil,
        m: *runtime.m nil,
        sched: runtime.gobuf {sp: 0, pc: 0, g: 0, ctxt: unsafe.Pointer(0x0), ret: 0, lr: 0, bp: 0},
        // ...
        gcAssistBytes: 0,}
~~~

### 4.设置线程的本地存储

~~~
    LEAQ	runtime·m0+m_tls(SB), DI
    CALL	runtime·settls(SB)
    
    // store through it, to make sure it works
    get_tls(BX)
    MOVQ	$0x123, g(BX)
    MOVQ	runtime·m0+m_tls(SB), AX
    CMPQ	AX, $0x123
    JEQ 2(PC)
    CALL	runtime·abort(SB)
ok:
    // set the per-goroutine and per-mach "registers"
    get_tls(BX)
    LEAQ	runtime·g0(SB), CX
    MOVQ	CX, g(BX)
    LEAQ	runtime·m0(SB), AX
    
    // save m->g0 = g0
    MOVQ	CX, m_g0(AX)
    // save m0 to g0->m
    MOVQ	AX, g_m(CX)
~~~

#### 4.1查看设置前，g0和m0的值

g0

~~~bash
(dlv) p g0
runtime.g {
        // ...
        m: *runtime.m nil,
        sched: runtime.gobuf {sp: 0, pc: 0, g: 0, ctxt: unsafe.Pointer(0x0), ret: 0, lr: 0, bp: 0},
        // ...
        gcAssistBytes: 0,}
        
# g0.m = nil
~~~

m0

~~~bash
(dlv) p m0
runtime.m {
        g0: *runtime.g nil,
        // ...
        }
# m0.g0 = nil
~~~

#### 4.2查看设置后，g0和m0的值

g0

~~~bash
(dlv) p g0
runtime.g {
        stack: runtime.stack {lo: 140728952632928, hi: 140728952698464},
        stackguard0: 140728952632928,
        stackguard1: 140728952632928,
        _panic: *runtime._panic nil,
        _defer: *runtime._defer nil,
        m: *runtime.m {
                g0: *(*runtime.g)(0x52fe60),
                // ...
                },
        sched: runtime.gobuf {sp: 0, pc: 0, g: 0, ctxt: unsafe.Pointer(0x0), ret: 0, lr: 0, bp: 0},
        // ...
        }
~~~

m0 

~~~bash
(dlv) p m0
runtime.m {
        g0: *runtime.g {
                stack: (*runtime.stack)(0x52fe60),
                stackguard0: 140728952632928,
                stackguard1: 140728952632928,
                _panic: *runtime._panic nil,
                _defer: *runtime._defer nil,
                m: *(*runtime.m)(0x530240),
                // ...
                },
        // ...
        }
~~~

查看前后g0 m0的值能发现，g0中的m与m0中的g已经建立互相的关联引用

### 5.接下来进入go进程的核心处理流程

~~~
// 下面语句处理操作系统传递过来的参数
// 即方法开始时的参数操作
// 将argc从内存搬到AX存储器中
MOVL	24(SP), AX		// copy argc
// 将argc搬到SP+0的位置，即栈顶位置
MOVL	AX, 0(SP)
// 将argv从内存搬到AX寄存器中
MOVQ	32(SP), AX		// copy argv
// 将argv搬到SP+8的位置
MOVQ	AX, 8(SP)
// 调用runtime.args
CALL	runtime·args(SB)
// 调用osinit函数，获取CPU的核数，存放在全局变量ncpu中，供后面的调度时使用
CALL	runtime·osinit(SB)
// 调用schedinit进行调度的初始化
CALL	runtime·schedinit(SB)

// create a new goroutine to start program
// runtime·mainPC就是runtime.main函数，这里是将runtime·mainPC的地址赋值给AX
MOVQ	$runtime·mainPC(SB), AX		// entry
// 将AX入栈，作为newproc的fn参数
PUSHQ	AX
// 调用runtime.newproc方法 runtime/proc.go:4477
// 创建goroutine,用来运行runtime.main函数
// runtime.main函数中会执行的逻辑：
//    1.新建一个线程执行sysmon函数，sysmon的工作是系统后台监控，通过轮询的方式判断是否需要进行垃圾回收和调度抢占
//    2.启动gc清扫的goroutine
//    3.执行runtime包的初始化，main包以及main包import的所有包的初始化
//    4.最终执行main.mian函数，完成后调用exit(0)系统调用退出进程
CALL	runtime·newproc(SB)
POPQ	AX

// start this M
// 调用runtime.mstart启动工作线程m,进入调度系统
// 在进行一些设置和检测后
// 执行调度方法schedule()，开始协程调度，该函数不会返回
// 这里就会调度刚创建的runtime.main协程，并运行
CALL	runtime·mstart(SB)

CALL	runtime·abort(SB)	// mstart should never return
RET
~~~

### 6.总结

整个初始化过程就是go调度器（g0,m0,p）的初始化过程，初始化g0,m的线程本地存储，创建runtime.main协程，主要是调用runtime.proc方法，这个方法会创建实际运行runtime.main方法的协程，初始化协程属性以及p的队列处理，最后启动工作线程m,进入调度系统

系统在初始化后，主要就是围绕schedule()调度逻辑