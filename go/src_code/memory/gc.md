# gc过程分析

> go 1.21

## 1. 标记准备

### 1.1 gcStart函数

主要工作：

1. 检查是否达到 GC 条件
2. 异步启动对应于P数量的标记协程
3. S(STOP)TW
4. 控制标记协程数量和执行时长，使得CPU占用率趋近25%
5. 设置GC阶段为GCMark，开启混合混合写屏障
6. 标记mcache中的tiny对象
7. S(START)TW


/src/runtime/mgc.go:600

~~~go
// /src/runtime/mgc.go:531
// gcMode表示GC循环的并发程度。
type gcMode int

const (
    gcBackgroundMode gcMode = iota // concurrent GC and sweep
    gcForceMode                    // stop-the-world GC now, concurrent sweep
    gcForceBlockMode               // stop-the-world GC now and STW sweep (forced by user)
)

func gcStart(trigger gcTrigger) {
    // 获取当前 m,且不可被抢占
    mp := acquirem()
    if gp := getg(); gp == mp.g0 || mp.locks > 1 || mp.preemptoff != "" {
        releasem(mp)
        return
    }
    releasem(mp)
    mp = nil
    // 检查是否达到 GC 条件，会根据 trigger 类型作 dispatch，常见的包括堆内存大小、GC 时间间隔、手动触发的类型
    for trigger.test() && sweepone() != ^uintptr(0) {
        sweep.nbgsweep++
    }

    // 上锁
    semacquire(&work.startSema)
    // 再次检查是否达到 GC 条件
    if !trigger.test() {
        semrelease(&work.startSema)
        return
    }

    // 设置gcMode
    // 默认gcBackgroundMode，同时gc和sweep
    mode := gcBackgroundMode
    if debug.gcstoptheworld == 1 {
        mode = gcForceMode
    } else if debug.gcstoptheworld == 2 {
        mode = gcForceBlockMode
    }

    // Ok, we're doing it! Stop everybody else
    semacquire(&gcsema)
    semacquire(&worldsema)

    // 是否是用户主动触发gc
    work.userForced = trigger.kind == gcTriggerCycle

    if traceEnabled() {
        traceGCStart()
    }

    // 检查所有P是否已完成延迟的mcache刷新。
    for _, p := range allp {
        if fg := p.mcache.flushGen.Load(); fg != mheap_.sweepgen {
            println("runtime: p", p.id, "flushGen", fg, "!= sweepgen", mheap_.sweepgen)
            throw("p mcache not flushed")
        }
    }
    // 由于进入了 GC 模式，会根据 P 的数量启动多个 GC 并发标记协程，但是会先阻塞挂起，等待被唤醒
    gcBgMarkStartWorkers()
    // gcResetMarkState重置标记之前的全局状态（并发或STW），并重置所有Gs的堆栈扫描状态。
    // 在没有STW的情况下，这是安全的，因为在此期间或之后创建的任何G都将以重置状态开始。
    // 必须在系统堆栈(g0)上调用gcResetMarkState，因为它获取了堆锁
    systemstack(gcResetMarkState)
    // work属性初始化
    // 设置work.stwprocs，work.maxprocs 为 gomaxprocs
    // gomaxprocs默认等于ncpu，用户调用runtime.GOMAXPROCS(n)时会发生改变
    work.stwprocs, work.maxprocs = gomaxprocs, gomaxprocs
    if work.stwprocs > ncpu {
        // 保证work.stwprocs等于ncpu
        work.stwprocs = ncpu
    }
	
    work.heap0 = gcController.heapLive.Load()
    work.pauseNS = 0
    work.mode = mode

    now := nanotime()
    work.tSweepTerm = now
    work.pauseStart = now
    // 切换到g0，执行 Stop the world 操作
    systemstack(func() { stopTheWorldWithSema(stwGCSweepTerm) })
    // 在开始并行扫描之前完成扫描。
    systemstack(func() {
        finishsweep_m()
    })

    // clearpools before we start the GC. If we wait they memory will not be
    // reclaimed until the next GC cycle.
    clearpools()

    work.cycles.Add(1)
	
    // startCycle重置GC控制器的状态，并计算新GC周期的估计值。caller必须持有worldsema，世界必须停止。
    // 限制标记协程占用 CPU 时间片的比例为趋近 25%
    gcController.startCycle(now, int(gomaxprocs), trigger)

    // Notify the CPU limiter that assists may begin.
    gcCPULimiter.startGCTransition(true, now)

    // 在STW模式下，禁用用户Gs的调度。这也可能禁用该goroutine的调度，因此它可能会在我们重新开始世界时立即阻止。
    if mode != gcBackgroundMode {
        // schedEnableUser启用或禁用用户goroutine的计划。
        // 这不会停止已经在运行的用户goroutine，因此调用方在禁用用户goroutines时应该首先停止世界。
        schedEnableUser(false)
    }

    // 设置GC阶段为_GCmark，启用混合写屏障
    setGCPhase(_GCmark)

    gcBgMarkPrepare() // Must happen before assist enable.
    gcMarkRootPrepare()

    // 对 mcache 中的 tiny 对象进行标记
    gcMarkTinyAllocs()
    // 设置gcBlackenEnabled=1
    // 在调度过程（schedule()函数）中，调用findRunnable方法时，会检查gcBlackenEnabled != 0
    // 不为0时，就会调用gcController.findRunnableGCWorker唤醒gcBgMarkStartWorker，开始gc
    atomic.Store(&gcBlackenEnabled, 1)

    // 获取当前m,且不可被抢占
    mp = acquirem()

    // Concurrent mark.
    // 切换至 g0，重新 start the world
    // 并发标记
    systemstack(func() {
        now = startTheWorldWithSema()
        work.pauseNS += now - work.pauseStart
        work.tMark = now
        memstats.gcPauseDist.record(now - work.pauseStart)
        
        sweepTermCpu := int64(work.stwprocs) * (work.tMark - work.tSweepTerm)
        work.cpuStats.gcPauseTime += sweepTermCpu
        work.cpuStats.gcTotalTime += sweepTermCpu
        
        // Release the CPU limiter.
        gcCPULimiter.finishGCTransition(now)
    })
	
    semrelease(&worldsema)
    releasem(mp)
    
    // Make sure we block instead of returning to user code
    // in STW mode.
    if mode != gcBackgroundMode {
        Gosched()
    }
    
    semrelease(&work.startSema)
}
~~~

### 1.2 启动标记协程

gcBgMarkStartWorkers函数

gcBgMarkStartWorkers方法中启动了对应于 P 数量的并发标记协程，并且通过notetsleepg的机制，使得for循环与gcBgMarkWorker内部形成联动节奏，确保每个P都能分得一个gcBgMarkWorker标记协程

/src/runtime/mgc.go:1210

~~~go
func gcBgMarkStartWorkers() {
    for gcBgMarkWorkerCount < gomaxprocs {
        go gcBgMarkWorker()
        // 挂起，等待 gcBgMarkWorker 方法中完成标记协程与 P 的绑定后唤醒
        notetsleepg(&work.bgMarkReady, -1)
        noteclear(&work.bgMarkReady)
        // The worker is now guaranteed to be added to the pool before
        // its P's next findRunnableGCWorker.
        
        gcBgMarkWorkerCount++
    }
}
~~~

gcBgMarkWorker函数

gcBgMarkWorker 方法中将g包装成一个node天添加到全局的gcBgMarkWorkerPool中，保证标记协程与P的一对一关联，并调用 gopark 方法将当前 g 挂起，等待被唤醒.

/src/runtime/mgc.go:1259

~~~go
func gcBgMarkWorker() {
    // 获取当前g
    gp := getg()
	
    gp.m.preemptoff = "GC worker init"
    node := new(gcBgMarkWorkerNode)
    gp.m.preemptoff = ""
    node.gp.set(gp)
    node.m.set(acquirem())
    // 唤醒外部的 for 循环
    notewakeup(&work.bgMarkReady)

    for {
        // 当前 g 阻塞至此，直到 gcController.findRunnableGCWorker 方法被调用，会将当前 g 唤醒
        gopark(func(g *g, nodep unsafe.Pointer) bool {
            node := (*gcBgMarkWorkerNode)(nodep)
            
            if mp := node.m.ptr(); mp != nil {
                releasem(mp)
            }
    
            // 将当前 g 包装成一个 node 添加到 gcBgMarkWorkerPool 中
            gcBgMarkWorkerPool.push(&node.node)
            return true
        }, unsafe.Pointer(node), waitReasonGCWorkerIdle, traceBlockSystemGoroutine, 0)
        // 后面的代码是被findRunnableGCWorker唤醒后的并发标记阶段
        // ...
    }
}
~~~

### 1.3 Stop the world

主要工作：

1. 取锁：sched.lock
2. 将 sched.gcwaiting 标识置为 1，后续的调度流程见其标识，都会阻塞挂起
3. 抢占所有g，并将 P 的状态置为 syscall
4. 将所有P的状态改为 stop
5. 倘若部分任务无法抢占，则等待其完成后再进行抢占
6. 调用方法worldStopped收尾，世界停止了

/src/runtime/proc.go:1315

~~~go
func stopTheWorldWithSema(reason stwReason) {
    if traceEnabled() {
        traceSTWStart(reason)
    }
    gp := getg()
	
    if gp.m.locks > 0 {
        throw("stopTheWorld: holding locks")
    }
    // 全局调度锁
    lock(&sched.lock)
    sched.stopwait = gomaxprocs
    // 此标识置 1，之后所有的调度都会阻塞等待
    sched.gcwaiting.Store(true)
    // 发送抢占信息抢占所有 G，后将 p 状态置为 syscall
    preemptall()
    // stop current P
    gp.m.p.ptr().status = _Pgcstop // Pgcstop is only diagnostic.
    sched.stopwait--
    // 把所有 p 的状态置为 stop
    for _, pp := range allp {
        s := pp.status
        if s == _Psyscall && atomic.Cas(&pp.status, s, _Pgcstop) {
            if traceEnabled() {
                traceGoSysBlock(pp)
                traceProcStop(pp)
            }
            pp.syscalltick++
            sched.stopwait--
        }
    }
    // 把空闲 p 的状态置为 stop
    now := nanotime()
    for {
        pp, _ := pidleget(now)
        if pp == nil {
            break
        }
        pp.status = _Pgcstop
        sched.stopwait--
    }
    wait := sched.stopwait > 0
    unlock(&sched.lock)
    
    // 倘若有 p 无法被抢占，则阻塞直到将其统统抢占完成
    if wait {
        for {
            // 等待100us，尝试重新抢占
            if notetsleep(&sched.stopnote, 100*1000) {
                noteclear(&sched.stopnote)
                break
            }
            preemptall()
        }
    }
    
    // sanity checks
    bad := ""
    if sched.stopwait != 0 {
        bad = "stopTheWorld: not stopped (stopwait != 0)"
    } else {
        for _, pp := range allp {
            if pp.status != _Pgcstop {
                bad = "stopTheWorld: not stopped (status != _Pgcstop)"
            }
        }
    }
    if freezing.Load() {
        lock(&deadlock)
        lock(&deadlock)
    }
    if bad != "" {
        throw(bad)
    }
    // stop the world
    worldStopped()
}
~~~

### 1.4 控制标记协程频率

gcStart方法中，还会通过gcController.startCycle方法，将标记协程对CPU的占用率控制在 25% 左右. 此时，根据P的数量是否能被4整除，分为两种处理方式：

倘若P的个数能被4整除，则简单将标记协程的数量设置为P/4
倘若P的个数不能被4整除，则通过控制标记协程执行时长的方式，来使全局标记协程对CPU的使用率趋近于25%

src/runtime/mgcpacer.go:384

~~~go
// 目标：标记协程对CPU的使用率维持在25%的水平
const gcBackgroundUtilization = 0.25


func (c *gcControllerState) startCycle(markStartTime int64, procs int, trigger gcTrigger) {
    // ...
    // P 的个数 * 0.25
    totalUtilizationGoal := float64(procs) * gcBackgroundUtilization
    // P 的个数 * 0.25 后四舍五入取整
    dedicatedMarkWorkersNeeded := int64(totalUtilizationGoal + 0.5)
    utilError := float64(dedicatedMarkWorkersNeeded)/totalUtilizationGoal - 1
    const maxUtilError = 0.3
    // 倘若 P 的个数不能被 4 整除
    if utilError < -maxUtilError || utilError > maxUtilError {
        if float64(dedicatedMarkWorkersNeeded) > totalUtilizationGoal {
            // Too many dedicated workers.
            dedicatedMarkWorkersNeeded--
        }
        // 计算出每个 P 需要额外执行标记任务的时间片比例
        c.fractionalUtilizationGoal = (totalUtilizationGoal - float64(dedicatedMarkWorkersNeeded)) / float64(procs)
    } else {
        // 倘若 P 的个数可以被 4 整除，则无需控制执行时长
        c.fractionalUtilizationGoal = 0
    }
    
    // In STW mode, we just want dedicated workers.
    if debug.gcstoptheworld > 0 {
        dedicatedMarkWorkersNeeded = int64(procs)
        c.fractionalUtilizationGoal = 0
    }
    // ...
}
~~~

### 1.5 设置写屏障

调用setGCPhase方法，标志GC正式进入并发标记（GCmark）阶段.

src/runtime/mgc.go:234

~~~go
func setGCPhase(x uint32) {
    atomic.Store(&gcphase, x)
    // 在_GCMark和_GCMarkTermination阶段中，会启用混合写屏障.
    writeBarrier.needed = gcphase == _GCmark || gcphase == _GCmarktermination
    writeBarrier.enabled = writeBarrier.needed
}
~~~

### 1.6 标记Tiny对象

遍历所有的P，对mcache中的Tiny对象分别调用greyobject方法进行置灰

src/runtime/mgcmark.go:1586

~~~go
func gcMarkTinyAllocs() {
    assertWorldStopped()
    // 遍历全局的p列表
    for _, p := range allp {
        c := p.mcache
        if c == nil || c.tiny == 0 {
            continue
        }
        _, span, objIndex := findObject(c.tiny, 0, 0)
        gcw := &p.gcw
        greyobject(c.tiny, 0, 0, span, gcw, objIndex)
    }
}
~~~

### 1.6 Start the world

startTheWorldWithSema与stopTheWorldWithSema形成对偶. 该方法会重新恢复世界的生机，将所有P唤醒. 倘若缺少M，则构造新的M为P补齐

src/runtime/proc.go:1433

~~~go
func startTheWorldWithSema() int64 {
    assertWorldStopped()
    // 获取m,不可抢占
    mp := acquirem()
    // 是否存在网络io中准备就绪的g
    if netpollinited() {
        list := netpoll(0) // non-blocking
        // 放入全局或者本地可执行队列中
        injectglist(&list)
    }
    lock(&sched.lock)
    // 获取gomaxprocs，默认等于ncpu，可能被用户修改
    procs := gomaxprocs
    if newprocs != 0 {
        procs = newprocs
        newprocs = 0
    }
    // 根据procs修改p的数量
    p1 := procresize(procs)
    sched.gcwaiting.Store(false)
    if sched.sysmonwait.Load() {
        sched.sysmonwait.Store(false)
        notewakeup(&sched.sysmonnote)
    }
    unlock(&sched.lock)
    // 重启世界
    worldStarted()
    // 遍历所有p,并唤醒
    for p1 != nil {
        p := p1
        p1 = p1.link.ptr()
        if p.m != 0 {
            mp := p.m.ptr()
            p.m = 0
            if mp.nextp != 0 {
                throw("startTheWorld: inconsistent mp->nextp")
            }
            mp.nextp.set(p)
            notewakeup(&mp.park)
        } else {
            // Start M to run P.  Do not start another M below.
            newm(nil, p, -1)
        }
    }

    // Capture start-the-world time before doing clean-up tasks.
    startTime := nanotime()
    if traceEnabled() {
        traceSTWDone()
    }

    // 尝试唤醒新的m执行任务
    wakep()
    // 恢复可抢占
    releasem(mp)
    return startTime
}
~~~

## 2. 并发标记

在并发标记之前，已经将创建了用于并发标记的协程，此时gc协程还处于阻塞状态，等待被调度

### 2.1 调度标记协程

schedule函数

GMP调度的主干方法schedule中，会通过g0调用findRunnable方法P寻找下一个可执行的协程，找到后会调用execute方法，内部完成由g0->g的切换，真正执行用户协程中的任务

/src/runtime/proc.go:3553

~~~go
func schedule() {
    // ...
	gp, inheritTime, tryWakeP := findRunnable()
    // ...
    execute(gp, inheritTime)
}
~~~

findRunnable函数

检查全局标识gcBlackenEnabled发现当前开启GC模式时，会调用 gcControllerState.findRunnableGCWorker唤醒并取得标记协程

src/runtime/proc.go:2891

~~~go
func findRunnable() (gp *g, inheritTime, tryWakeP bool) {
    // ...
    
    // gcBlackenEnabled在gc标记准备阶段被置为1
    // 如果正在 GC，去找 GC 的 g
    if gcBlackenEnabled != 0 {
        gp, tnow := gcController.findRunnableGCWorker(pp, now)
        if gp != nil {
            return gp, false, true
        }
        now = tnow
    }
    
    // ...
}
~~~

findRunnableGCWorker函数

findRunnableGCWorker函数会从全局的标记协程池 gcBgMarkWorkerPool获取到一个封装了标记协程的node. 并通过gcControllerState中 dedicatedMarkWorkersNeeded、fractionalUtilizationGoal等字段标识判定标记协程的标记模式，然后将标记协程状态由_Gwaiting唤醒为_Grunnable，并返回给 g0 用于执行

src/runtime/mgcpacer.go:731

~~~go
func (c *gcControllerState) findRunnableGCWorker(pp *p, now int64) (*g, int64) {
    if gcBlackenEnabled == 0 {
        throw("gcControllerState.findRunnable: blackening not enabled")
    }
    
    if now == 0 {
        now = nanotime()
    }
    if gcCPULimiter.needUpdate(now) {
        gcCPULimiter.update(now)
    }
    // 保证当前 pp 是可以调度标记协程的，每个 p 只能执行一个标记协程
    if !gcMarkWorkAvailable(pp) {
        return nil, now
    }

    // 从全局标记协程池子 gcBgMarkWorkerPool 中取出 g
    node := (*gcBgMarkWorkerNode)(gcBgMarkWorkerPool.pop())
    if node == nil {
        return nil, now
    }
    
    decIfPositive := func(val *atomic.Int64) bool {
    for {
        v := val.Load()
        if v <= 0 {
            return false
        }
        
        if val.CompareAndSwap(v, v-1) {
            return true
        }
    }
    }
    // 确认标记的模式
    if decIfPositive(&c.dedicatedMarkWorkersNeeded) {
        pp.gcMarkWorkerMode = gcMarkWorkerDedicatedMode
    } else if c.fractionalUtilizationGoal == 0 {
        // No need for fractional workers.
        gcBgMarkWorkerPool.push(&node.node)
        return nil, now
    } else {
        delta := now - c.markStartTime
        if delta > 0 && float64(pp.gcFractionalMarkTime)/float64(delta) > c.fractionalUtilizationGoal {
            // Nope. No need to run a fractional worker.
            gcBgMarkWorkerPool.push(&node.node)
            return nil, now
        }
        // Run a fractional worker.
        pp.gcMarkWorkerMode = gcMarkWorkerFractionalMode
    }

    // 将标记协程的状态置为 runnable，填了 gcBgMarkWorker 方法中 gopark 操作留下的坑
    gp := node.gp.ptr()
    casgstatus(gp, _Gwaiting, _Grunnable)
    if traceEnabled() {
        traceGoUnpark(gp, 0)
    }
    return gp, now
}
~~~

### 2.2 并发标记

标记协程被唤醒后，主线又重新拉回到gcBgMarkWorker方法中，此时会根据findRunnableGCWorker方法中预设的标记模式，调用gcDrain方法开始执行并发标记工作.

标记模式包含以下三种：

- gcMarkWorkerDedicatedMode：专一模式. 需要完整执行完标记任务，不可被抢占
- gcMarkWorkerFractionalMode：分时模式. 当标记协程执行时长达到一定比例后，可以被抢占
- gcMarkWorkerIdleMode: 空闲模式. 随时可以被抢占

在执行专一模式时，会先以可被抢占的模式尝试执行，倘若真的被用户协程抢占，则会先将当前P本地队列的用户协程投放到全局g队列中，再将标记模式改为不可抢占模式. 这样设计的优势是，通过负载均衡的方式，减少当前P下用户协程的等待时长，提高用户体验.

在gcDrain方法中，有两个核心的gcDrainFlags控制着标记协程的运行风格：

- gcDrainIdle：空闲模式，随时可被抢占
- gcDrainFractional：分时模式，执行一定比例的时长后可被抢占

gcBgMarkWorker函数

/src/runtime/mgc.go:1259

~~~go
// /src/runtime/mgc.go:247
type gcMarkWorkerMode int

const (
    // 指示下一个计划的G没有开始工作，并且该模式应该被忽略。
    gcMarkWorkerNotWorker gcMarkWorkerMode = iota
    
    // 专一模式. 需要完整执行完标记任务，不可被抢占
    gcMarkWorkerDedicatedMode
    
    // 分时模式. 当标记协程执行时长达到一定比例后，可以被抢占
    gcMarkWorkerFractionalMode
    
    // 空闲模式. 随时可以被抢占
    gcMarkWorkerIdleMode
)

// src/runtime/mgcmark.go:1005
type gcDrainFlags int

const (
    gcDrainUntilPreempt gcDrainFlags = 1 << iota
    gcDrainFlushBgCredit
    gcDrainIdle
    gcDrainFractional
)

func gcBgMarkWorker() {
    // 获取当前g
    gp := getg()
	
    gp.m.preemptoff = "GC worker init"
    node := new(gcBgMarkWorkerNode)
    gp.m.preemptoff = ""
    node.gp.set(gp)
    node.m.set(acquirem())
    // 唤醒外部的 for 循环
    notewakeup(&work.bgMarkReady)

    for {
        // gopark
        // ...
		
        // 被findRunnableGCWorker唤醒后的并发标记阶段
        // ...
        node.m.set(acquirem())
        pp := gp.m.p.ptr() // P can't change with preemption disabled.
        
        if gcBlackenEnabled == 0 {
            println("worker mode", pp.gcMarkWorkerMode)
            throw("gcBgMarkWorker: blackening not enabled")
        }
        
        if pp.gcMarkWorkerMode == gcMarkWorkerNotWorker {
            throw("gcBgMarkWorker: mode not set")
        }
        
        startTime := nanotime()
        pp.gcMarkWorkerStartTime = startTime
        var trackLimiterEvent bool
        if pp.gcMarkWorkerMode == gcMarkWorkerIdleMode {
            trackLimiterEvent = pp.limiterEvent.start(limiterEventIdleMarkWork, startTime)
        }
        
        decnwait := atomic.Xadd(&work.nwait, -1)
        if decnwait == work.nproc {
            println("runtime: work.nwait=", decnwait, "work.nproc=", work.nproc)
            throw("work.nwait was > work.nproc")
        }
        // 根据不同的运作模式，执行 gcDrain 方法：
        systemstack(func() {
            casGToWaiting(gp, _Grunning, waitReasonGCWorkerActive)
            switch pp.gcMarkWorkerMode {
            default:
                throw("gcBgMarkWorker: unexpected gcMarkWorkerMode")
            case gcMarkWorkerDedicatedMode:
                // 专一模式
                // gcDrainUntilPreempt|gcDrainFlushBgCredit => 1|2 == 3 == gcDrainIdle
                gcDrain(&pp.gcw, gcDrainUntilPreempt|gcDrainFlushBgCredit)
                if gp.preempt {
                    if drainQ, n := runqdrain(pp); n > 0 {
                        lock(&sched.lock)
                        globrunqputbatch(&drainQ, int32(n))
                        unlock(&sched.lock)
                    }
                }
                gcDrain(&pp.gcw, gcDrainFlushBgCredit)
            case gcMarkWorkerFractionalMode:
                // 分时模式
                gcDrain(&pp.gcw, gcDrainFractional|gcDrainUntilPreempt|gcDrainFlushBgCredit)
            case gcMarkWorkerIdleMode:
                // 空闲模式
                gcDrain(&pp.gcw, gcDrainIdle|gcDrainUntilPreempt|gcDrainFlushBgCredit)
            }
            casgstatus(gp, _Gwaiting, _Grunning)
        })
        
        // Account for time and mark us as stopped.
        now := nanotime()
        duration := now - startTime
        gcController.markWorkerStop(pp.gcMarkWorkerMode, duration)
        if trackLimiterEvent {
            pp.limiterEvent.stop(limiterEventIdleMarkWork, now)
        }
        if pp.gcMarkWorkerMode == gcMarkWorkerFractionalMode {
            atomic.Xaddint64(&pp.gcFractionalMarkTime, duration)
        }
		
        incnwait := atomic.Xadd(&work.nwait, +1)
        if incnwait > work.nproc {
            println("runtime: p.gcMarkWorkerMode=", pp.gcMarkWorkerMode,
            "work.nwait=", incnwait, "work.nproc=", work.nproc)
            throw("work.nwait > work.nproc")
        }
		
        pp.gcMarkWorkerMode = gcMarkWorkerNotWorker
		
        if incnwait == work.nproc && !gcMarkWorkAvailable(nil) {
            releasem(node.m.ptr())
            node.m.set(nil)
            
            gcMarkDone()
        }
    }
}
~~~

### 2.3 标记流程

gcDrain 方法是并发标记阶段的核心方法：

- 在空闲模式（idle）和分时模式（fractional）下，会提前设好 check 函数（pollWork 和 pollFractionalWorkerExit）
- 标记根对象
- 循环从gcw缓存队列中取出灰色对象，执行scanObject方法进行扫描标记
- 定期检查check 函数，判断标记流程是否应该被打断

src/runtime/mgcmark.go:1036

~~~go
func gcDrain(gcw *gcWork, flags gcDrainFlags) {
    if !writeBarrier.needed {
        throw("gcDrain phase incorrect")
    }
    
    gp := getg().m.curg
    // 模式标记
    preemptible := flags&gcDrainUntilPreempt != 0
    flushBgCredit := flags&gcDrainFlushBgCredit != 0
    idle := flags&gcDrainIdle != 0
    
    initScanWork := gcw.heapScanWork
	
    checkWork := int64(1<<63 - 1)
    var check func() bool
    if flags&(gcDrainIdle|gcDrainFractional) != 0 {
        checkWork = initScanWork + drainCheckThreshold
        if idle {
            check = pollWork
        } else if flags&gcDrainFractional != 0 {
            check = pollFractionalWorkerExit
        }
    }
    
    // 倘若根对象还未标记完成，则先进行根对象标记
    if work.markrootNext < work.markrootJobs {
        // Stop if we're preemptible or if someone wants to STW.
        for !(gp.preempt && (preemptible || sched.gcwaiting.Load())) {
            job := atomic.Xadd(&work.markrootNext, +1) - 1
            if job >= work.markrootJobs {
                break
            }
            // 标记根对象
            markroot(gcw, job, flushBgCredit)
            if check != nil && check() {
                goto done
            }
        }
    }
    
    // 遍历队列，进行对象标记.
    // Stop if we're preemptible or if someone wants to STW.
    for !(gp.preempt && (preemptible || sched.gcwaiting.Load())) {
        if work.full == 0 {
            gcw.balance()
        }
        // 尝试从 p 本地队列中获取灰色对象，无锁
        b := gcw.tryGetFast()
        if b == 0 {
            // 尝试从全局队列中获取灰色对象，加锁
            b = gcw.tryGet()
            if b == 0 {
                // 刷新写屏障缓存,将p的写屏障缓存刷新的gcw
                wbBufFlush()
                b = gcw.tryGet()
            }
        }
        if b == 0 {
            // 已无对象需要标记
            break
        }
        // 进行对象的标记，并顺延指针进行后续对象的扫描
        scanobject(b, gcw)
		
        if gcw.heapScanWork >= gcCreditSlack {
            gcController.heapScanWork.Add(gcw.heapScanWork)
            if flushBgCredit {
                gcFlushBgCredit(gcw.heapScanWork - initScanWork)
                initScanWork = 0
            }
            checkWork -= gcw.heapScanWork
            gcw.heapScanWork = 0
            
            if checkWork <= 0 {
                checkWork += drainCheckThreshold
                if check != nil && check() {
                    break
                }
            }
        }
    }
    
done:
    // Flush remaining scan work credit.
    if gcw.heapScanWork > 0 {
        gcController.heapScanWork.Add(gcw.heapScanWork)
        if flushBgCredit {
            gcFlushBgCredit(gcw.heapScanWork - initScanWork)
        }
        gcw.heapScanWork = 0
    }
}
~~~

wbBufFlush函数

在混合写屏障机制中，核心是会将需要置灰的对象添加到当前P的wbBuf缓存中. 随后在并发标记缺灰、标记终止前置检查等时机会执行wbBufFlush1方法，批量地将wbBuf中的对象释放出来进行置灰，保证达到预期的效果

wbBufFlush1方法中涉及了对象置灰操作，其包含了在对应mspan的bitmap中打点标记以及将对象添加到gcw队列两步

src/runtime/mwbbuf.go:166

~~~go
func wbBufFlush() {
    
    if getg().m.dying > 0 {
        getg().m.p.ptr().wbBuf.discard()
        return
    }
    
    systemstack(func() {
        wbBufFlush1(getg().m.p.ptr())
    })
}

func wbBufFlush1(pp *p) {
    // 获取当前 P 通过屏障机制缓存的指针
    start := uintptr(unsafe.Pointer(&pp.wbBuf.buf[0]))
    n := (pp.wbBuf.next - start) / unsafe.Sizeof(pp.wbBuf.buf[0])
    ptrs := pp.wbBuf.buf[:n]
	
    pp.wbBuf.next = 0
    
    if useCheckmark {
        // Slow path for checkmark mode.
        for _, ptr := range ptrs {
            shade(ptr)
        }
        pp.wbBuf.reset()
        return
    }
    // 将缓存的指针作标记，添加到 gcw 队列
    gcw := &pp.gcw
    pos := 0
    for _, ptr := range ptrs {
        if ptr < minLegalPointer {
            continue
        }
        obj, span, objIndex := findObject(ptr, 0, 0)
        if obj == 0 {
            continue
        }
        
        mbits := span.markBitsForIndex(objIndex)
        if mbits.isMarked() {
            continue
        }
        mbits.setMarked()
        
        // 标记span
        arena, pageIdx, pageMask := pageIndexOf(span.base())
        if arena.pageMarks[pageIdx]&pageMask == 0 {
            atomic.Or8(&arena.pageMarks[pageIdx], pageMask)
        }
        
        if span.spanclass.noscan() {
            gcw.bytesMarked += uint64(span.elemsize)
            continue
        }
        ptrs[pos] = obj
        pos++
    }

    // 所有缓存对象入队
    gcw.putBatch(ptrs[:pos])
    
    pp.wbBuf.reset()
}
~~~