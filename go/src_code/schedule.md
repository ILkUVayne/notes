# go并发核心调度 - schedule

## 1 schedule函数

任务调度开始入口，调用findRunnable获取一个可执行任务，并调用execute执行任务

/usr/local/go_src/21/go/src/runtime/proc.go:3553

~~~go
func schedule() {
    // 获取当前运行g0绑定的m g0.m
    mp := getg().m
    
    if mp.locks != 0 {
        throw("schedule: holding locks")
    }

    // m.lockedg 会在 LockOSThread 下变为非零
    if mp.lockedg != 0 {
        stoplockedm()
        execute(mp.lockedg.ptr(), false) // Never returns.
    }
    
    // We should not schedule away from a g that is executing a cgo call,
    // since the cgo call is using the m's g0 stack.
    // m正在执行一个cgo调用,这时候不能执行调度
    if mp.incgo {
        throw("schedule: in cgo")
    }
    
top:
    // 获取m绑定的p
    pp := mp.p.ptr()
    // preempt被设置为指示该P应该尽快进入调度器（不管在其上运行什么G）。
    pp.preempt = false
    if mp.spinning && (pp.runnext != 0 || pp.runqhead != pp.runqtail) {
        throw("schedule: spinning with local work")
    }
    // 查找要执行的可运行goroutine。
    // 试图从其他P中窃取，从本地或全局队列、轮询网络中获取g。
    // tryWakeP表示返回的goroutine不正常（GC工作程序、跟踪读取器），因此调用者应该尝试唤醒P。
    // 当前的工作线程进入睡眠模式，直到获取到运行的g之后findrunnable函数才会返回
    gp, inheritTime, tryWakeP := findRunnable() // blocks until work is available
    
    if debug.dontfreezetheworld > 0 && freezing.Load() {
        lock(&deadlock)
        lock(&deadlock)
    }
	
    // 如果当前线程处于自旋状态，这个线程尝试将唤醒（创建）一个线程，并停止自旋
    if mp.spinning {
        resetspinning()
    }
    
    if sched.disable.user && !schedEnabled(gp) {
        lock(&sched.lock)
        if schedEnabled(gp) {
            unlock(&sched.lock)
        } else {
            sched.disable.runnable.pushBack(gp)
            sched.disable.n++
            unlock(&sched.lock)
            goto top
        }
    }
    
    // If about to schedule a not-normal goroutine (a GCworker or tracereader),
    // wake a P if there is one.
    // tryWakeP表示返回的goroutine不正常（GC工作程序、跟踪读取器），因此调用者应该尝试唤醒P。
    if tryWakeP {
        wakep()
    }
    if gp.lockedm != 0 {
        // 如果 g 需要 lock 到 m 上，则会将当前的 p
        // 给这个要 lock 的 g
        // 然后阻塞等待一个新的 p
        startlockedm(gp)
        goto top
    }
    // 成功获取g，开始执行任务
    execute(gp, inheritTime)
}
~~~

findRunnable函数

尝试获取一个可执行的任务g

1. 如果当前处于gc，休眠阻塞当前m不再进行调度
2. 如果需要 trace或者gc,试图调度一个对应的g
3. 每调度 61 次且全局可运行队列不为空时，就检查一次全局队列，尝试从全局队列中获取一批g放入本地队列（最多获取全局队列中的一半），保证公平性
4. 优先从本地队列中获取g，本地队列优先获取p.runnext,如果本地队列为空,尝试从全局队列中获取一批g，逻辑同步骤三
5. 全局队列未获取到g,尝试从网络io轮询器中找到准备就绪的g,把这个g变为可运行的g
6. 本地、全局以及网络io中均获取不到可执行的g时，尝试从其他p的本地队列中偷取一批g到本地队列中（最多其他某个p队列中的一半）
7. 偷取逻辑最多遍历4次，取到就会返回。前3次尝试从本地可运行队列中偷取一半g到当前p队列中，最后一次若本地队列为空，尝试偷取p.runnext
8. 所有的可能性都尝试过了,没有可运行的g,检查垃圾回收中有进行标记工作的g，如果需要，则返回相关处理的g
9. 准备解绑m和p,解绑前检查全局队列中是否又发现了任务，如果有，直接获取，并返回
10. 解绑m和p，将 p 放入 idle 链表(空闲队列)，将m切换到自旋状态，重新检查所有的队列，发现有p队列不为空，绑定m/p，重新执行获取逻辑，重新检查是否有垃圾回收中有进行标记工作的g需要运行,有则直接返回
11. 最后再次尝试检查是否存在 poll 网络的 g，如果有，则直接返回
12. 最后我们什么也没找到，暂止当前的 m 并阻塞休眠

/usr/local/go_src/21/go/src/runtime/proc.go:2891

~~~go
func findRunnable() (gp *g, inheritTime, tryWakeP bool) {
    // g0.m
    mp := getg().m
    
top:
    // g0.m.p
    pp := mp.p.ptr()
    if sched.gcwaiting.Load() {
        // 如果需要 GC，不再进行调度
        gcstopm()
        goto top
    }
    if pp.runSafePointFn != 0 {
        runSafePointFn()
    }
    
    now, pollUntil, _ := checkTimers(pp, 0)
    
    // Try to schedule the trace reader.
    // 如果正在 trace，去找 trace 的 g
    if traceEnabled() || traceShuttingDown() {
        gp := traceReader()
        if gp != nil {
            // 修改g状态为_Grunnable
            casgstatus(gp, _Gwaiting, _Grunnable)
            traceGoUnpark(gp, 0)
            return gp, false, true
        }
    }
    
    // Try to schedule a GC worker.
    // 如果正在 GC，去找 GC 的 g
    if gcBlackenEnabled != 0 {
        gp, tnow := gcController.findRunnableGCWorker(pp, now)
        if gp != nil {
            return gp, false, true
        }
        now = tnow
    }
    
    // 开始尝试获取可执行的任务g
    // pp.schedtick在每次调度程序调用时递增
    // 每调度 61 次且全局可运行队列不为空时，就检查一次全局队列，保证公平性
    if pp.schedtick%61 == 0 && sched.runqsize > 0 {
        lock(&sched.lock)
        // 从全局队列中取 g
        gp := globrunqget(pp, 1)
        unlock(&sched.lock)
        if gp != nil {
            return gp, false, false
        }
    }
    
    // Wake up the finalizer G.
    // 唤醒运行终结器的g
    if fingStatus.Load()&(fingWait|fingWake) == fingWait|fingWake {
        if gp := wakefing(); gp != nil {
            // 标记gp准备运行。
            ready(gp, 0, true)
        }
    }
    if *cgo_yield != nil {
        asmcgocall(*cgo_yield, nil)
    }
    
    // local runq
    // 尝试从本地可运行队列中获取g
    if gp, inheritTime := runqget(pp); gp != nil {
        return gp, inheritTime, false
    }
    
    // global runq
    // 尝试从全局可运行队列中获取g
    if sched.runqsize != 0 {
        lock(&sched.lock)
        gp := globrunqget(pp, 0)
        unlock(&sched.lock)
        if gp != nil {
            return gp, false, false
        }
    }
    
    // 从网络io轮询器中找到准备就绪的g,把这个g变为可运行的g
    if netpollinited() && netpollWaiters.Load() > 0 && sched.lastpoll.Load() != 0 {
        if list := netpoll(0); !list.empty() { // non-blocking
            gp := list.pop()
            // 把找到的可运行的网络io的g列表插入到全局队列
            injectglist(&list)
            casgstatus(gp, _Gwaiting, _Grunnable)
            if traceEnabled() {
                traceGoUnpark(gp, 0)
            }
            return gp, false, false
        }
    }
    
    // Spinning Ms: steal work from other Ps.
    //
    // Limit the number of spinning Ms to half the number of busy Ps.
    // This is necessary to prevent excessive CPU consumption when
    // GOMAXPROCS>>1 but the program parallelism is low.
    if mp.spinning || 2*sched.nmspinning.Load() < gomaxprocs-sched.npidle.Load() {
        if !mp.spinning {
            mp.becomeSpinning()
        }
        // 本地和全局可运行队列都不存在可运行g时，尝试从其他p的本地队列中偷取一部分g到本地队列中
        gp, inheritTime, tnow, w, newWork := stealWork(now)
        if gp != nil {
            // Successfully stole.
            return gp, inheritTime, false
        }
        // gc goto top
        if newWork {
            // There may be new timer or GC work; restart to
            // discover.
            goto top
        }
        
        now = tnow
        if w != 0 && (pollUntil == 0 || w < pollUntil) {
            // Earlier timer to wait for.
            pollUntil = w
        }
    }
    
    // 没有任何 work 可做。
    // 如果我们在 GC mark 阶段，则可以安全的扫描并 blacken 对象
    // 然后便有 work 可做，运行 idle-time 标记而非直接放弃当前的 P
    // 垃圾回收中有进行标记工作的g需要运行，调度运行标记工作的g
    if gcBlackenEnabled != 0 && gcMarkWorkAvailable(pp) && gcController.addIdleMarkWorker() {
        node := (*gcBgMarkWorkerNode)(gcBgMarkWorkerPool.pop())
        if node != nil {
            pp.gcMarkWorkerMode = gcMarkWorkerIdleMode
            gp := node.gp.ptr()
            casgstatus(gp, _Gwaiting, _Grunnable)
            if traceEnabled() {
                traceGoUnpark(gp, 0)
            }
            return gp, false, false
        }
        gcController.removeIdleMarkWorker()
    }
    
    // 如果返回了回调，而没有其他goroutine处于唤醒状态，则唤醒事件处理程序goroutine，它会暂停执行，直到触发回调。
    gp, otherReady := beforeIdle(now, pollUntil)
    if gp != nil {
        casgstatus(gp, _Gwaiting, _Grunnable)
        if traceEnabled() {
            traceGoUnpark(gp, 0)
        }
        return gp, false, false
    }
    if otherReady {
        goto top
    }
    
    // 放弃当前的 P 之前 保存allp快照
    allpSnapshot := allp
    // 快照掩码
    idlepMaskSnapshot := idlepMask
    timerpMaskSnapshot := timerpMask
    
    // return P and block
    lock(&sched.lock)
    // 进入了 gc，回到顶部暂止 m
    if sched.gcwaiting.Load() || pp.runSafePointFn != 0 {
        unlock(&sched.lock)
        goto top
    }
    // 全局队列中又发现了任务
    if sched.runqsize != 0 {
        // 赶紧偷掉返回
        gp := globrunqget(pp, 0)
        unlock(&sched.lock)
        return gp, false, false
    }
    if !mp.spinning && sched.needspinning.Load() == 1 {
        // See "Delicate dance" comment below.
        mp.becomeSpinning()
        unlock(&sched.lock)
        goto top
    }
    // 归还当前的 p
    // 解除p和m的绑定关系
    // gp.m.p = 0
    // pp.m = 0
    // pp.status = _Pidle
    if releasep() != pp {
        throw("findrunnable: wrong p")
    }
    // 将 p 放入 idle 链表(空闲队列)
    now = pidleput(pp, now)
    unlock(&sched.lock)
    
    wasSpinning := mp.spinning
    // 如果m处于自旋状态，再次尝试获取g
    if mp.spinning {
        mp.spinning = false
        if sched.nmspinning.Add(-1) < 0 {
            throw("findrunnable: negative nmspinning")
        }
        
        // Check all runqueues once again.
        pp := checkRunqsNoP(allpSnapshot, idlepMaskSnapshot)
        if pp != nil {
            acquirep(pp)
            mp.becomeSpinning()
            goto top
        }
        
        // Check for idle-priority GC work again.
        pp, gp := checkIdleGCNoP()
        if pp != nil {
            acquirep(pp)
            mp.becomeSpinning()
            
            // Run the idle worker.
            pp.gcMarkWorkerMode = gcMarkWorkerIdleMode
            casgstatus(gp, _Gwaiting, _Grunnable)
            if traceEnabled() {
                traceGoUnpark(gp, 0)
            }
            return gp, false, false
        }
        
        pollUntil = checkTimersNoP(allpSnapshot, timerpMaskSnapshot, pollUntil)
    }
    
    // Poll network until next timer.
    // 再次尝试从网络io中获取g
    if netpollinited() && (netpollWaiters.Load() > 0 || pollUntil != 0) && sched.lastpoll.Swap(0) != 0 {
        sched.pollUntil.Store(pollUntil)
        if mp.p != 0 {
            throw("findrunnable: netpoll with p")
        }
        if mp.spinning {
            throw("findrunnable: netpoll with spinning")
        }
        delay := int64(-1)
        if pollUntil != 0 {
            if now == 0 {
                now = nanotime()
            }
            delay = pollUntil - now
            if delay < 0 {
                delay = 0
            }
        }
        if faketime != 0 {
            // When using fake time, just poll.
            delay = 0
        }
        list := netpoll(delay) // block until new work is available
        // Refresh now again, after potentially blocking.
        now = nanotime()
        sched.pollUntil.Store(0)
        sched.lastpoll.Store(now)
        if faketime != 0 && list.empty() {
            // Using fake time and nothing is ready; stop M.
            // When all M's stop, checkdead will call timejump.
            stopm()
            goto top
        }
        lock(&sched.lock)
        pp, _ := pidleget(now)
        unlock(&sched.lock)
        if pp == nil {
            injectglist(&list)
        } else {
            acquirep(pp)
        if !list.empty() {
            gp := list.pop()
            injectglist(&list)
            casgstatus(gp, _Gwaiting, _Grunnable)
            if traceEnabled() {
                traceGoUnpark(gp, 0)
            }
            return gp, false, false
        }
        if wasSpinning {
            mp.becomeSpinning()
        }
        goto top
        }
    } else if pollUntil != 0 && netpollinited() {
        pollerPollUntil := sched.pollUntil.Load()
        if pollerPollUntil == 0 || pollerPollUntil > pollUntil {
            netpollBreak()
        }
    }
    // 最终未获得g,调用stopm休眠m
    stopm()
    goto top
}
~~~

globrunqget函数

尝试从全局可运行队列中获取一批g，取出的g中，取出一个返回，剩余g存入调用方p的本地可运行队列中

/usr/local/go_src/21/go/src/runtime/proc.go:5992

~~~go
func globrunqget(pp *p, max int32) *g {
    assertLockHeld(&sched.lock)
    
    if sched.runqsize == 0 {
        return nil
    }
    // 获取需要取出的g的数量n
    // gomaxprocs默认等于cpu核数
    n := sched.runqsize/gomaxprocs + 1
    if n > sched.runqsize {
        n = sched.runqsize
    }
    // max表示调用方需要的g最大值
    if max > 0 && n > max {
        n = max
    }
    // n最多取全局队列的一半
    if n > int32(len(pp.runq))/2 {
        n = int32(len(pp.runq)) / 2
    }
    // 修改全局的可运行队列runqsize
    sched.runqsize -= n
    
    // 先取出一个作为返回值
    gp := sched.runq.pop()
    n--
    // 剩余取出的部分存入pp本地可运行队列中
    for ; n > 0; n-- {
        gp1 := sched.runq.pop()
        runqput(pp, gp1, false)
    }
    return gp
}
~~~

runqget函数

从本地可运行队列中获取g

/usr/local/go_src/21/go/src/runtime/proc.go:6311

~~~go
func runqget(pp *p) (gp *g, inheritTime bool) {
    // If there's a runnext, it's the next G to run.
    next := pp.runnext
    // If the runnext is non-0 and the CAS fails, it could only have been stolen by another P,
    // because other Ps can race to set runnext to 0, but only the current P can set it to non-0.
    // Hence, there's no need to retry this CAS if it fails.
    // 如果pp.runnext不为空，直接取出pp.runnext并返回
    if next != 0 && pp.runnext.cas(next, 0) {
        return next.ptr(), true
    }
    
    for {
        h := atomic.LoadAcq(&pp.runqhead) // load-acquire, synchronize with other consumers
        t := pp.runqtail
        // 本地队列是空，返回 nil
        if t == h {
            return nil, false
        }
        // 从本地队列中以 cas 方式拿一个
        gp := pp.runq[h%uint32(len(pp.runq))].ptr()
        if atomic.CasRel(&pp.runqhead, h, h+1) { // cas-release, commits consume
            return gp, false
        }
    }
}
~~~

stealWork函数

尝试从其他p的本地队列中获取偷取g,总共遍历4次，为保证公平性，遍历开始的位置随机，若前3次都未能偷取到g，则在第4次遍历时，如果本地队列任然为空，会尝试偷取p.runnext

/usr/local/go_src/21/go/src/runtime/proc.go:3260

~~~go
func stealWork(now int64) (gp *g, inheritTime bool, rnow, pollUntil int64, newWork bool) {
    pp := getg().m.p.ptr()
    
    ranTimer := false
    
    const stealTries = 4
    // 偷取操作最多遍历4次，偷取成功直接返回
    // 为保证公平，遍历的起点随机
    for i := 0; i < stealTries; i++ {
        stealTimersOrRunNextG := i == stealTries-1
        
        for enum := stealOrder.start(fastrand()); !enum.done(); enum.next() {
            // 已经进入了 GC,直接返回
			if sched.gcwaiting.Load() {
                // GC work may be available.
                return nil, false, now, pollUntil, true
            }
            // 获取当前遍历到的p
            p2 := allp[enum.position()]
            // 当前遍历到的p等于传入的p（即需要偷取g的p）时，continue
            if pp == p2 {
                continue
            }
            // 最后一次遍历，且遍历到的p是空闲状态，调用checkTimers，可能会使gp存在可执行任务
            if stealTimersOrRunNextG && timerpMask.read(enum.position()) {
                tnow, w, ran := checkTimers(p2, now)
                now = tnow
                if w != 0 && (pollUntil == 0 || w < pollUntil) {
                    pollUntil = w
                }
                if ran {
                    if gp, inheritTime := runqget(pp); gp != nil {
                        return gp, inheritTime, now, pollUntil, ranTimer
                    }
                    ranTimer = true
                }
            }
            
            // Don't bother to attempt to steal if p2 is idle.
            // 遍历到的p不是空闲状态，尝试偷取g
            if !idlepMask.read(enum.position()) {
                if gp := runqsteal(pp, p2, stealTimersOrRunNextG); gp != nil {
                    return gp, false, now, pollUntil, ranTimer
                }
            }
        }
    }
    
    return nil, false, now, pollUntil, ranTimer
}
~~~

runqsteal函数

尝试从p2中偷取g,不存在则返回nil. 当stealRunNextG==true（即最后一次遍历）时，本地队列为空时，会尝试偷取p2.runnext

/usr/local/go_src/21/go/src/runtime/proc.go:6432

~~~go
func runqsteal(pp, p2 *p, stealRunNextG bool) *g {
    t := pp.runqtail
    n := runqgrab(p2, &pp.runq, t, stealRunNextG)
    // n==0表示p2不存在可偷取的g
    if n == 0 {
        return nil
    }
    // 减去需要返回的g
    n--
    // 获取需要返回的g
    gp := pp.runq[(t+n)%uint32(len(pp.runq))].ptr()
    // n == 0表示获取的是p2.runnext，这时候只获取了一个，直接返回
    if n == 0 {
        return gp
    }
    h := atomic.LoadAcq(&pp.runqhead) // load-acquire, synchronize with consumers
    if t-h+n >= uint32(len(pp.runq)) {
        throw("runqsteal: runq overflow")
    }
    atomic.StoreRel(&pp.runqtail, t+n) // store-release, makes the item available for consumption
    return gp
}
~~~

runqgrab函数

从pp的可运行队列中抓取一批g到batch(pp.runq)中,即存入当前p的本地可运行队列

/usr/local/go_src/21/go/src/runtime/proc.go:6377

~~~go
func runqgrab(pp *p, batch *[256]guintptr, batchHead uint32, stealRunNextG bool) uint32 {
    for {
        h := atomic.LoadAcq(&pp.runqhead) // load-acquire, synchronize with other consumers
        t := atomic.LoadAcq(&pp.runqtail) // load-acquire, synchronize with the producer
        n := t - h
        // 获取pp中的一半数量
        n = n - n/2
        // 如果pp可运行队列为空
        if n == 0 {
            // stealRunNextG==true表示当前为最后一次循环
            if stealRunNextG {
                // Try to steal from pp.runnext.
                // 尝试偷取下一个可运行的g pp.runnext
                if next := pp.runnext; next != 0 {
                    if pp.status == _Prunning {
                        if GOOS != "windows" && GOOS != "openbsd" && GOOS != "netbsd" {
                            usleep(3)
                        } else {
                            osyield()
                        }
                    }
                    if !pp.runnext.cas(next, 0) {
                        continue
                    }
                    // 将next放入batch，返回存入数量1
                    batch[batchHead%uint32(len(batch))] = next
                    return 1
                }
            }
            // 当前不是最后一次循环，直接返回
            return 0
        }
        if n > uint32(len(pp.runq)/2) { // read inconsistent h and t
            continue
        }
        // 尝试偷取其现有的一半 g，并且返回实际偷取的数量.
        for i := uint32(0); i < n; i++ {
            g := pp.runq[(h+i)%uint32(len(pp.runq))]
            batch[(batchHead+i)%uint32(len(batch))] = g
        }
        if atomic.CasRel(&pp.runqhead, h, h+n) { // cas-release, commits consume
            return n
        }
    }
}
~~~

execute函数

在当前 M 上调度 gp。 如果 inheritTime 为 true，则 gp 继承剩余的时间片。否则从一个新的时间片开始

/usr/local/go_src/21/go/src/runtime/proc.go:2847

~~~go
func execute(gp *g, inheritTime bool) {
    // g0.m
    mp := getg().m
    
    if goroutineProfile.active {
        // 确保gp已经将其堆栈写入goroutine概要文件，就像goroutine分析器第一次停止世界时一样。
        tryRecordGoroutineProfile(gp, osyield)
    }
    
    // Assign gp.m before entering _Grunning so running Gs have an
    // M.
    // 设置当前m运行用户程序的g位gp,即将执行用户程序的g和m绑定
    mp.curg = gp
    // 设置执行用户程序的g(gp)的m为mp，这样gp和当前的m相互绑定
    gp.m = mp
    // 将gp的状态从_Grunnable修改为_Grunning,意味着gp马上会得到执行
    casgstatus(gp, _Grunnable, _Grunning)
    gp.waitsince = 0
    // 设置gp的抢占标志为false
    gp.preempt = false
    gp.stackguard0 = gp.stack.lo + stackGuard
    if !inheritTime {
        mp.p.ptr().schedtick++
    }
    
    // Check whether the profiler needs to be turned on or off.
    hz := sched.profilehz
    if mp.profilehz != hz {
        setThreadCPUProfiler(hz)
    }
    
    if traceEnabled() {
        // GoSysExit has to happen when we have a P, but before GoStart.
        // So we emit it here.
        if gp.syscallsp != 0 {
            traceGoSysExit()
        }
        traceGoStart()
    }
    // gogo将从g0栈切换到执行用户程序的gp,真正开始执行用户程序
    gogo(&gp.sched)
}
~~~

gogo函数

gogo函数是用汇编语言实现,切换g0栈到需要执行的g,切换原理就是将要运行g的调度信息g.sched从内存中恢复到CPU寄存器，设置SP和IP等寄存器的值，跳转到要运行的位置开始执行指令。总之一句话，gogo函数完成了从g0到用户g的切换，即CPU执行权的转让以及栈的切换

/usr/local/go_src/21/go/src/runtime/asm_amd64.s:404

~~~plan9_x86
TEXT runtime·gogo(SB), NOSPLIT, $0-8
    // 0(FP)表示第一个参数，即buf=&gp.sched
    // BX=buf
    MOVQ	buf+0(FP), BX		// gobuf
    // DX=buf.g=&gp.sched.g
    MOVQ	gobuf_g(BX), DX
    MOVQ	0(DX), CX		// make sure g != nil
    JMP	gogo<>(SB)

TEXT gogo<>(SB), NOSPLIT, $0
    get_tls(CX)
    // 把g放入到tls[0],即把要运行的g的指针放进线程本地存储，后面的代码可以通过本地线程存储
    // 获取到当前正在执行的goroutine的地址，在这之前，本地线程存储中存放的是g0的地址
    MOVQ	DX, g(CX)
    MOVQ	DX, R14		// set the g register
    // SP=buf.sp=&gp.sched.sp，即把CPU的栈顶寄存器SP设置为gp.sched.sp，成功完成从
    // g0栈切换到gp栈
    MOVQ	gobuf_sp(BX), SP	// restore SP
    // 恢复调度上下文到CPU对应的寄存器
    // 将系统调用的返回值放入到AX寄存器中
    MOVQ	gobuf_ret(BX), AX
    MOVQ	gobuf_ctxt(BX), DX
    MOVQ	gobuf_bp(BX), BP
    // 前面已经将调度相关的值都放入到CPU的寄存器中了，将gp.sched中的值清空，这样可以减轻gc的工作量
    MOVQ	$0, gobuf_sp(BX)	// clear to help garbage collector
    MOVQ	$0, gobuf_ret(BX)
    MOVQ	$0, gobuf_ctxt(BX)
    MOVQ	$0, gobuf_bp(BX)
    // 把gp.sched.pc的值放入到BX寄存器，对于main goroutine，sched.pc中存储的是runtime包
    // 中main()函数的地址
    MOVQ	gobuf_pc(BX), BX
    // JMP把BX寄存器中的地址值放入到CPU的IP寄存器中，然后CPU跳转到该地址的位置开始执行指令
    // 即跳转到main()函数执行代码
    JMP	BX
~~~

goexit函数

gogo函数切换g0栈到需要执行的g，执行完g的fn后，会执行goexit，处理收尾工作

/usr/local/go_src/21/go/src/runtime/asm_amd64.s:1649

~~~plan9_x86
TEXT runtime·goexit(SB),NOSPLIT|TOPFRAME|NOFRAME,$0-0
    BYTE	$0x90	// NOP
    CALL	runtime·goexit1(SB)	// does not return
    // traceback from goexit1 must hit code range of goexit
    BYTE	$0x90	// NOP
~~~

goexit1函数

调用了mcall函数，mcall是汇编实现的。mcall函数的功能是从当前用户程序g切换到g0上运行，然后在g0栈上执行goexit0函数。概括起来，mcall完成两个主要逻辑：

1. 保存当前的g的调度信息到内存中，通过当前的g，找到与它绑定的m,在通过m找到m中保存的g0，然后将g0保存到tls中，修改CPU寄存器的值为g0栈的内容
2. 切换到g0栈，执行goexit0函数

/usr/local/go_src/21/go/src/runtime/proc.go:3850

~~~go
func goexit1() {
    // 这里是检查data race逻辑
    if raceenabled {
        racegoend()
    }
    if traceEnabled() {
        traceGoEnd()
    }
    // 开始收尾工作
    mcall(goexit0)
}
~~~

goexit0函数

goexit0函数把gp(用户程序g)状态从_Grunning修改为_Gdead，然后清理gp对象中保存内容，其次通过函数dropg解除gp和m之间的绑定关系，然后将gp放入到P的freeg队列中缓存起来，以便后续复用，最后调用schedule，进行新一轮调度.

/usr/local/go_src/21/go/src/runtime/proc.go:3861

~~~go
// goexit0函数是在g0上执行的，入参gp是用户程序g
func goexit0(gp *g) {
    // g0
    mp := getg().m
    pp := mp.p.ptr()
    // 将gp的状态从_Grunning修改为_Gdead
    casgstatus(gp, _Grunning, _Gdead)
    gcController.addScannableStack(pp, -int64(gp.stack.hi-gp.stack.lo))
    if isSystemGoroutine(gp, false) {
        sched.ngsys.Add(-1)
    }
    // 清理gp对象中保存内容
    gp.m = nil
    locked := gp.lockedm != 0
    gp.lockedm = 0
    mp.lockedg = 0
    gp.preemptStop = false
    gp.paniconfault = false
    gp._defer = nil // should be true already but just in case.
    gp._panic = nil // non-nil for Goexit during panic. points at stack-allocated data.
    gp.writebuf = nil
    gp.waitreason = waitReasonZero
    gp.param = nil
    gp.labels = nil
    gp.timer = nil
    // 如果当前在进行垃圾回收，将gp赋值标记的信息刷新到全局信用池中
    if gcBlackenEnabled != 0 && gp.gcAssistBytes > 0 {
        // Flush assist credit to the global pool. This gives
        // better information to pacing if the application is
        // rapidly creating an exiting goroutines.
        assistWorkPerByte := gcController.assistWorkPerByte.Load()
        scanCredit := int64(assistWorkPerByte * float64(gp.gcAssistBytes))
        gcController.bgScanCredit.Add(scanCredit)
        gp.gcAssistBytes = 0
    }
    // 解除gp和m之间的绑定关系
    // 指将当前 g 的 m 置空、将当前 m 的 g 置空，从而完成解绑
    dropg()
    
    if GOARCH == "wasm" { // no threads yet on wasm
        gfput(pp, gp)
        schedule() // never returns
    }
    
    if mp.lockedInt != 0 {
        print("invalid m->lockedInt = ", mp.lockedInt, "\n")
        throw("internal lockOSThread error")
    }
    // 将gp放入到p的freeg队列中，以便下次可以复用，不用new一个g对象，避免重新申请内存
    gfput(pp, gp)
    if locked {
        // The goroutine may have locked this thread because
        // it put it in an unusual kernel state. Kill it
        // rather than returning it to the thread pool.
        
        // Return to mstart, which will release the P and exit
        // the thread.
        if GOOS != "plan9" { // See golang.org/issue/22227.
            gogo(&mp.g0.sched)
        } else {
            // Clear lockedExt on plan9 since we may end up re-using
            // this thread.
            mp.lockedExt = 0
        }
    }
    // 再次调用schedule，进行新一轮调度
    schedule()
}
~~~

## 2 总结

schedule函数是go任务调度的入口

schedule函数的执行流程：

1. 函数会在调用findRunnable函数后进入休眠，直到获取到一个可执行任务g,然后是findRunnable主要流程
2. 首先判断是否已经调度了61次，若是，在全局可执行队列不为空的前提下，优先从全局队列中获取一批g放入当前p的本地队列，并返回
3. 否则优先从p的本地队列中获取g（优先p.runnext）返回
4. 本地队列为空时，尝试从全局队列中获取一批g返回
5. 全局队列中也获取不到，尝试从io轮询器中找到准备就绪的g,把这个g变为可运行的g返回
6. 以上均获取不到时，尝试从其他p的本地队列中偷取一批g到本地队列
7. 偷取逻辑最多尝试4次，并且最后一次会额外尝试偷取对应p.runnext
8. 尝试偷取失败后，会进行当前m和p的解绑，然后再一次重新尝试查询全局、io轮询器、其他p中是否有新的g，若存在，重新执行获取逻辑并返回
9. 最后我们什么也没找到，暂止当前的 m 并阻塞休眠
10. 当findRunnable函数获取到可执行的g后，最终执行execute函数，执行任务
11. execute函数绑定当前m和需要执行的g,修改g状态到_Grunning，调用gogo函数，并g设置对应的寄存器信息，然后将g0栈切换到对应g,最终执行g.fn
12. g.fn执行完成后，会执行goexit函数（可以理解为是goexit函数调用g.fn,具体实现是在newproc1中伪造成了goexit函数调用g.fn proc.go:4532）
13. goexit函数由汇编实现，实际调用goexit1函数，goexit1函数mcall(goexit0)函数
14. mcall是汇编实现，保存当前的g的调度信息到内存中，由用户g切换到g0,在g0栈执行goexit0
15. goexit0函数修改g状态为_Gdead，清理g对象中的内容，通过函数dropg解除g和m之间的绑定关系，然后将g放入到P的freeg队列中缓存起来，以便后续复用
16. 最后调用schedule，进行新一轮调度
