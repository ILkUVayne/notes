# go初始化之执行前的准备 - mstart方法

> go 1.21

## 1 mstart函数

mstart新m的入口点，实际方法使用汇编实现

/usr/local/go_src/21/go/src/runtime/proc.go:1520

~~~go
// mstart is the entry-point for new Ms.
// It is written in assembly, uses ABI0, is marked TOPFRAME, and calls mstart0.
func mstart()
~~~

mstart函数的实际实现，调用mstart0方法

/usr/local/go_src/21/go/src/runtime/asm_amd64.s:393

~~~plan9_x86
TEXT runtime·mstart(SB),NOSPLIT|TOPFRAME|NOFRAME,$0
    CALL	runtime·mstart0(SB)
    RET // not reached
~~~

## 2 mstart0方法

mstart新m的实际入口点

/usr/local/go_src/21/go/src/runtime/proc.go:1530

~~~go
func mstart0() {
    // 获取当前运行的g,初始时为g0
    gp := getg()
    // 初始化g堆栈
    // 初始时，g0的stack.lo已完成初始化，它不等于0
    osStack := gp.stack.lo == 0
    if osStack {
        // Initialize stack bounds from system stack.
        // Cgo may have left stack size in stack.hi.
        // minit may update the stack bounds.
        //
        // Note: these bounds may not be very accurate.
        // We set hi to &size, but there are things above
        // it. The 1024 is supposed to compensate this,
        // but is somewhat arbitrary.
        size := gp.stack.hi
        if size == 0 {
        size = 8192 * sys.StackGuardMultiplier
        }
        gp.stack.hi = uintptr(noescape(unsafe.Pointer(&size)))
        gp.stack.lo = gp.stack.hi - size + 1024
    }
    // Initialize stack guard so that we can start calling regular
    // Go code.
    // 初始化堆栈保护
    gp.stackguard0 = gp.stack.lo + stackGuard
    // This is the g0, so we can also call go:systemstack
    // functions, which check stackguard1.
    gp.stackguard1 = gp.stackguard0
    // 为调度做一些处理工作，并调用schedule开始任务调度
    mstart1()
    
    // Exit this thread.
	// 正常情况下不会走到退出线程逻辑，因为调用调度函数schedule后，该函数不会返回
    if mStackIsSystemAllocated() {
        // Windows, Solaris, illumos, Darwin, AIX and Plan 9 always system-allocate
        // the stack, but put it in gp.stack before mstart,
        // so the logic above hasn't set osStack yet.
        osStack = true
    }
    // 退出线程
    mexit(osStack)
}
~~~

## 3 mstart1函数

~~~go
func mstart1() {
    // 初始时，_g_为m0中的g0,其他情况下_g_也是各个m的g0
    gp := getg()
    
    if gp != gp.m.g0 {
        throw("bad runtime·mstart")
    }
    
    // Set up m.g0.sched as a label returning to just
    // after the mstart1 call in mstart0 above, for use by goexit0 and mcall.
    // We're never coming back to mstart1 after we call schedule,
    // so other calls can reuse the current frame.
    // And goexit0 does a gogo that needs to return from mstart1
    // and let mstart0 exit the thread.
    // getcallerpc()获取mstart1执行完的返回地址
    // getcallersp()获取调用mstart1时的栈顶地址
    // 保存g0调度信息
    gp.sched.g = guintptr(unsafe.Pointer(gp))
    gp.sched.pc = getcallerpc()
    gp.sched.sp = getcallersp()
    
    asminit()
    // 初始化m,主要设置线程的备用信号堆栈和掩码
    minit()
    
    // Install signal handlers; after minit so that minit can
    // prepare the thread to be able to handle the signals.
    // 如果当前的m是m0,执行mstartm0操作
	if gp.m == &m0 {
        // 对于初始m,需要设置系统信号量的处理函数
        mstartm0()
    }
    // 执行启动函数
    // 对于m0是没有mstartfn函数，对其他m如果有起始任务函数，则需要执行，比如sysmon函数
    // newm函数创建新m时，会设置m0.mstartfn
    // 具体是newm函数调用allocm创建m对象时，设置mstartfn
    if fn := gp.m.mstartfn; fn != nil {
        fn()
    }
    // 如果当前g的m不是m0,它现在还没有p，需要获取一个p, m0已经绑定了allp[0]，所以不用关心m0
    if gp.m != &m0 {
        // 完成_g_.m和p的互相绑定
        acquirep(gp.m.nextp.ptr())
        gp.m.nextp = 0
    }
    // 调用调度函数schedule，该函数不会返回
    schedule()
}
~~~