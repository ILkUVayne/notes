# 触发GC

## 1. 触发GC的事件类型

| 类型             | 触发事件                     | 校验条件       |
|----------------|--------------------------|------------|
| gcTriggerHeap  | 分配对象时触发                  | 堆已分配内存达到阈值 |
| gcTriggerTime  | 由 forcegchelper 守护协程定时触发 | 每2分钟触发一次   |
| gcTriggerCycle | 用户调用 runtime.GC 方法       | 上一轮 GC 已结束 |

在触发GC时，会通过 gcTrigger.test 方法，结合具体的触发事件类型进行触发条件的校验

/src/runtime/mgc.go:541

~~~go
type gcTrigger struct {
    kind gcTriggerKind
    now  int64  // gcTriggerTime: current time
    n    uint32 // gcTriggerCycle: cycle number to start
}

type gcTriggerKind int

const (
    // 根据堆分配内存情况，判断是否触发GC
    gcTriggerHeap gcTriggerKind = iota
    // 定时触发GC
    gcTriggerTime
    // 手动触发GC
    gcTriggerCycle
)
// 单位 nano，因此实际值为 120s = 2min
var forcegcperiod int64 = 2 * 60 * 1e9

func (t gcTrigger) test() bool {
    if !memstats.enablegc || panicking.Load() != 0 || gcphase != _GCoff {
        return false
    }
    switch t.kind {
    case gcTriggerHeap:
        // 其校验条件是判断当前堆已使用内存是否达到阈值. 此处的堆内存阈值会在上一轮GC结束时进行设定
        // 倘若堆中已使用的内存大小达到了阈值，则会真正执行 gc
        trigger, _ := gcController.trigger()
        return gcController.heapLive.Load() >= trigger
    case gcTriggerTime:
        // 每 2 min 发起一轮 gc
        if gcController.gcPercent.Load() < 0 {
            return false
        }
        lastgc := int64(atomic.Load64(&memstats.last_gc_nanotime))
        return lastgc != 0 && t.now-lastgc > forcegcperiod
    case gcTriggerCycle:
        // t.n > work.cycles, but accounting for wraparound.
        // 上一轮GC已经完成，此时能够开启新一轮GC任务
        return int32(t.n-work.cycles.Load()) > 0
    }
    return true
}
~~~

## 2. 分配对象时触发

/src/runtime/malloc.go:948

~~~go
func mallocgc(size uintptr, typ *_type, needzero bool) unsafe.Pointer {
    //...
	
    // 是否需要触发gc
    shouldhelpgc := false
	
    //...
    
    if size <= maxSmallSize {
        // 微对象 <16b 且不包含指针
        if noscan && size < maxTinySize {
            //...
            if v == 0 {
                // 倘若 mcache 中对应 spanClass 的 mspan 已满，置 true
                v, span, shouldhelpgc = c.nextFree(tinySpanClass)
            }
            //...
        } else {
            // 分配小对象
            // 根据对象大小，映射到其所属的 span 的等级(0~66）
            //...
            if v == 0 {
                // 倘若 mcache 中对应 spanClass 的 mspan 已满，置 true
                v, span, shouldhelpgc = c.nextFree(spc)
            }
            //...
        }
    } else {
		// 分配大对象
		// 分配大对象(>32kb)时候，直接触发gc
        shouldhelpgc = true
        // ...
    }
    
    //...
    
    if shouldhelpgc {
        // 尝试触发 gc，类型为 gcTriggerHeap，触发校验逻辑位于 gcTrigger.test 方法中
        if t := (gcTrigger{kind: gcTriggerHeap}); t.test() {
            gcStart(t)
        }
    }
    
    //...
    
    return x
}
~~~

## 3. 定时触发GC

runtime 包初始化的时候，即会异步开启一个守护协程，通过 for 循环 + park 的方式，循环阻塞等待被唤醒.

当被唤醒后，则会调用 gcStart 方法进入标记准备阶段，尝试开启新一轮 GC，此时触发 GC 的事件类型正是 gcTriggerTime（定时触发）

~~~go
// /src/runtime/runtime2.go:1144
var  forcegc   forcegcstate

// /src/runtime/runtime2.go:966
type forcegcstate struct {
    lock mutex
    g    *g
    idle atomic.Bool
}

// /src/runtime/proc.go:309
func init() {
    go forcegchelper()
}

// /src/runtime/proc.go:313
func forcegchelper() {
    forcegc.g = getg()
    lockInit(&forcegc.lock, lockRankForcegc)
    for {
        lock(&forcegc.lock)
        if forcegc.idle.Load() {
            throw("forcegc: phase error")
        }
        forcegc.idle.Store(true)
        // 令 forcegc.g 陷入被动阻塞，g 的状态会设置为 waiting，当达成 gc 条件时，g 的状态会被切换至 runnable，方法才会向下执行
        goparkunlock(&forcegc.lock, waitReasonForceGCIdle, traceBlockSystemGoroutine, 1)
        // this goroutine is explicitly resumed by sysmon
        // g 被唤醒了，则调用 gcStart 方法真正开启 gc 主流程
        if debug.gctrace > 0 {
            println("GC forced")
        }
        // Time-triggered, fully concurrent.
        gcStart(gcTrigger{kind: gcTriggerTime, now: nanotime()})
    }
}

// /src/runtime/proc.go:144
func main() {
    // ...
	if GOARCH != "wasm" { // no threads on wasm yet, so no sysmon
        systemstack(func() {
            newm(sysmon, nil, -1)
        })
    }
    // ...
}

// /src/runtime/proc.go:5515
func sysmon() {
//...

    for {
        //...
		
        // check if we need to force a GC
        // 通过 gcTrigger.test 方法检查是否需要发起 gc，触发类型为 gcTriggerTime：定时触发
        if t := (gcTrigger{kind: gcTriggerTime, now: now}); t.test() && forcegc.idle.Load() {
            lock(&forcegc.lock)
            forcegc.idle.Store(false)
            var list gList
            // 需要发起 gc，则将 forcegc.g 注入 list 中, injectglist 方法内部会执行唤醒操作
            // injectglist 方法会将g的状态改为_Grunnable,尝试将g放入全局队列并调用startm实现任务，或者放入p的本地可执行队列中
            list.push(forcegc.g)
            injectglist(&list)
            unlock(&forcegc.lock)
        }
        //...
    }
}

// /src/runtime/proc.go:3477
func injectglist(glist *gList) {
    if glist.empty() {
        return
    }
    if traceEnabled() {
        for gp := glist.head.ptr(); gp != nil; gp = gp.schedlink.ptr() {
            traceGoUnpark(gp, 0)
        }
    }

    // 将所有的g的状态变更为可执行
    head := glist.head.ptr()
    var tail *g
    qsize := 0
    for gp := head; gp != nil; gp = gp.schedlink.ptr() {
        tail = gp
        qsize++
        casgstatus(gp, _Gwaiting, _Grunnable)
    }

    // Turn the gList into a gQueue.
    var q gQueue
    q.head.set(head)
    q.tail.set(tail)
    *glist = gList{}
    // 根据n数量尝试唤醒(创建)m，执行任务的函数
    startIdle := func(n int) {
        for i := 0; i < n; i++ {
            mp := acquirem() // 获取当前m，设置不可抢占
            lock(&sched.lock)
            // 尝试获取一个空闲的p
            pp, _ := pidlegetSpinning(0)
            if pp == nil {
                unlock(&sched.lock)
                releasem(mp)
                break
            }
            // 尝试唤醒(创建)m
            startm(pp, false, true)
            unlock(&sched.lock)
            // 恢复抢占
            releasem(mp)
        }
    }

    pp := getg().m.p.ptr()
    if pp == nil {
        lock(&sched.lock)
		// 将g放入全局可执行队列
        globrunqputbatch(&q, int32(qsize))
        unlock(&sched.lock)
        startIdle(qsize)
        return
    }
    // 获取空闲p数量
    npidle := int(sched.npidle.Load())
    var globq gQueue
    var n int
    for n = 0; n < npidle && !q.empty(); n++ {
        g := q.pop()
        globq.pushBack(g)
    }
    if n > 0 {
        lock(&sched.lock)
        // 将g放入全局可执行队列
        globrunqputbatch(&globq, int32(n))
        unlock(&sched.lock)
        startIdle(n)
        qsize -= n
    }
    // q不会空，说明当前m.pp != nil 且 npidle == 0
    // 将g放入m.pp的本地队列中去
    if !q.empty() {
    runqputbatch(pp, &q, qsize)
    }
}
~~~

## 4. 手动触发GC

~~~go
func GC() {
    n := work.cycles.Load()
    gcWaitOnMark(n)
    // 触发GC
    gcStart(gcTrigger{kind: gcTriggerCycle, n: n + 1})
    
    // Wait for mark termination N+1 to complete.
    gcWaitOnMark(n + 1)
    
    for work.cycles.Load() == n+1 && sweepone() != ^uintptr(0) {
        sweep.nbgsweep++
        Gosched()
    }
    
    for work.cycles.Load() == n+1 && !isSweepDone() {
        Gosched()
    }
    
    mp := acquirem()
    cycle := work.cycles.Load()
    if cycle == n+1 || (gcphase == _GCmark && cycle == n+2) {
        mProf_PostSweep()
    }
    releasem(mp)
}
~~~