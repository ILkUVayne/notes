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
    // 获取当前运行的g0
    gp := getg()
    // 初始化g0堆栈
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

为调度做一些处理工作，并调用schedule开始任务调度

/usr/local/go_src/21/go/src/runtime/proc.go:1573

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
    // gp.sched保存了goroutine的调度信息
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

asminit函数

由汇编实现，amd64架构下是一个空函数

/usr/local/go_src/21/go/src/runtime/asm_amd64.s:389

~~~plan9_x86
TEXT runtime·asminit(SB),NOSPLIT,$0-0
	// No per-thread init.
	RET
~~~

acquirep函数

M 与 P 的绑定过程只是简单的将 P 链表中的 P ，保存到 M 中的 P 指针上。 绑定前，P 的状态一定是 _Pidle，绑定后 P 的状态一定为 _Prunning

/usr/local/go_src/21/go/src/runtime/proc.go:5326

~~~go
func acquirep(pp *p) {
    // Do the part that isn't allowed to have write barriers.
    wirep(pp)
    
    // Have p; write barriers now allowed.
    
    // Perform deferred mcache flush before this P can allocate
    // from a potentially stale mcache.
    pp.mcache.prepareForSweep()
    
    if traceEnabled() {
        traceProcStart()
    }
}

func wirep(pp *p) {
    gp := getg()
    
    if gp.m.p != 0 {
        throw("wirep: already in go")
    }
    // 检查 m 是否正常，并检查要获取的 p 的状态
    if pp.m != 0 || pp.status != _Pidle {
        id := int64(0)
        if pp.m != 0 {
            id = pp.m.ptr().id
        }
        print("wirep: p->m=", pp.m, "(", id, ") p->status=", pp.status, "\n")
        throw("wirep: invalid p state")
    }
    // 将 p 绑定到 m，p 和 m 互相引用
    // gp.m.p = pp
    gp.m.p.set(pp)
    // pp.m = gp.m
    pp.m.set(gp.m)
    pp.status = _Prunning
}
~~~

## 4 m的暂止和复始

stopm函数

当 M 需要被暂止时，可能（因为还有其他暂止 M 的方法）会执行该调用。 此调用会将 M 进行暂止，并阻塞到它被复始时。这一过程就是工作线程的暂止和复始。

/usr/local/go_src/21/go/src/runtime/proc.go:2520

~~~go
func stopm() {
    gp := getg()
    
    if gp.m.locks != 0 {
        throw("stopm holding locks")
    }
    if gp.m.p != 0 {
        throw("stopm holding p")
    }
    if gp.m.spinning {
        throw("stopm spinning")
    }
    // 将 m 放回到 空闲列表中，因为我们马上就要暂止了
    lock(&sched.lock)
    mput(gp.m)
    unlock(&sched.lock)
    // 在此阻塞，直到被唤醒
    mPark()
    // 此时已经被复始，说明有任务要执行
    // 立即 acquire P
    acquirep(gp.m.nextp.ptr())
    gp.m.nextp = 0
}

func mPark() {
    gp := getg()
    // 暂止当前的 M，在此阻塞，直到被唤醒，等待被其他工作线程唤醒
    // 调用notewakeup唤醒m
    // 如调用startm函数时，sched.midle空闲队列存在m,则取出并调用notewakeup唤醒m proc.go:2649
    notesleep(&gp.m.park)
    // 清除暂止的 note
    noteclear(&gp.m.park)
}

func notesleep(n *note) {
    gp := getg()
    // 必须在g0上执行
    if gp != gp.m.g0 {
        throw("notesleep not on g0")
    }
    ns := int64(-1)
    if *cgo_yield != nil {
        // Sleep for an arbitrary-but-moderate interval to poll libc interceptors.
        ns = 10e6
    }
    for atomic.Load(key32(&n.key)) == 0 {
        gp.m.blocked = true
        // 休眠时间不超过ns，ns<0表示永远休眠
        // futexsleep调用c系统调用futex实现线程阻塞
        // 超过休眠时间时间还未被唤醒（ns>0），则会会被定时任务唤醒
        futexsleep(key32(&n.key), 0, ns)
        if *cgo_yield != nil {
            asmcgocall(*cgo_yield, nil)
        }
        gp.m.blocked = false
    }
}
~~~

## 4 总结

mstart函数是在go程序初始化完成后（或者新m启动时）的入口函数，主要是检查（初始化）m的堆栈信息，设置信号，g0的调度信息。完成后最终调用schedule()函数，正式开始程序的任务调度