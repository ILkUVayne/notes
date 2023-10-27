# 通信原语 channel

> go 1.21

## 1. channel结构

实现 Channel 的结构就是一个 mutex 锁加上一个环状缓存、 一个发送方队列和一个接收方队列

~~~go
// src/runtime/chan.go:33
type hchan struct {
    qcount   uint           // 队列中的所有数据数
    dataqsiz uint           // 环形队列的大小
    buf      unsafe.Pointer // 指向大小为 dataqsiz 的数组
    elemsize uint16         // 元素大小
    closed   uint32         // 是否关闭
    elemtype *_type         // 元素类型 
    sendx    uint           // 发送索引
    recvx    uint           // 接收索引
    recvq    waitq          // recv 等待列表，即（ <-ch ）
    sendq    waitq          // send 等待列表，即（ ch<- ）
    
    // lock protects all fields in hchan, as well as several
    // fields in sudogs blocked on this channel.
    //
    // Do not change another G's status while holding this lock
    // (in particular, do not ready a G), as this can deadlock
    // with stack shrinking.
    lock mutex
}
// src/runtime/chan.go:54
type waitq struct { // 等待队列 sudog 双向队列
    first *sudog
    last  *sudog
}

// src/runtime/runtime2.go:355
// sudog表示等待列表中的g，例如用于在通道上发送/接收。 
// sudog是必要的，因为g↔ 同步对象关系是多对多的。一个g可能在许多等待列表上，因此一个g可以有许多sudog；许多gs可能在同一个同步对象上等待，因此一个对象可能有许多sudog。 
// sudog是从一个特殊的池中分配的。使用acquireSudog和releaseSudog来分配和释放它们
type sudog struct {
    // 以下字段受此sudog正在阻塞的通道的hchan.lock保护。对于参与通道操作的sudog，shrinkstack依赖于此。
    g *g
    
    next *sudog // 后指针
    prev *sudog // 前指针
    // data element (may point to stack)
    // 数据元素（可能指向堆栈）
    elem unsafe.Pointer 
    
    // 以下字段永远不会同时访问。
    // 对于channel，waitlink只能由g访问。
    // 对于信号量，所有字段（包括上面的字段）只有在持有semaRoot锁时才能访问。
    
    acquiretime int64
    releasetime int64
    ticket      uint32
    
    // isSelect表示g正在参与select，因此g.selectDone必须是CAS才能赢得唤醒比赛。
    isSelect bool
    
    // success表示信道c上的通信是否成功。
    // 如果goroutine是因为通过通道c传递值而被唤醒的，则为true；
    // 如果是因为c关闭而被唤醒，则为false。
    success bool
    
    parent   *sudog // semaRoot binary tree
    waitlink *sudog // g.waiting list or semaRoot
    waittail *sudog // semaRoot
    c        *hchan // channel
}
~~~

## 2. channel创建

编译如下代码：

~~~go
func main() {
    c := make(chan int)
    println(c)

}
~~~

编译结果：

~~~bash
$ go tool compile -S test.go
main.main STEXT size=72 args=0x0 locals=0x20 funcid=0x0 align=0x0
        0x0000 00000 (/home/luoyang/code/go_demo/src_test/test.go:7)    TEXT    main.main(SB), ABIInternal, $32-0
        0x0000 00000 (/home/luoyang/code/go_demo/src_test/test.go:7)    CMPQ    SP, 16(R14)
        0x0004 00004 (/home/luoyang/code/go_demo/src_test/test.go:7)    PCDATA  $0, $-2
        0x0004 00004 (/home/luoyang/code/go_demo/src_test/test.go:7)    JLS     65
        0x0006 00006 (/home/luoyang/code/go_demo/src_test/test.go:7)    PCDATA  $0, $-1
        0x0006 00006 (/home/luoyang/code/go_demo/src_test/test.go:7)    PUSHQ   BP
        0x0007 00007 (/home/luoyang/code/go_demo/src_test/test.go:7)    MOVQ    SP, BP
        0x000a 00010 (/home/luoyang/code/go_demo/src_test/test.go:7)    SUBQ    $24, SP
        0x000e 00014 (/home/luoyang/code/go_demo/src_test/test.go:7)    FUNCDATA        $0, gclocals·J5F+7Qw7O7ve2QcWC7DpeQ==(SB)
        0x000e 00014 (/home/luoyang/code/go_demo/src_test/test.go:7)    FUNCDATA        $1, gclocals·CnDyI2HjYXFz19SsOj98tw==(SB)
        0x000e 00014 (/home/luoyang/code/go_demo/src_test/test.go:8)    LEAQ    type:chan int(SB), AX
        0x0015 00021 (/home/luoyang/code/go_demo/src_test/test.go:8)    XORL    BX, BX
        0x0017 00023 (/home/luoyang/code/go_demo/src_test/test.go:8)    PCDATA  $1, $0
        0x0017 00023 (/home/luoyang/code/go_demo/src_test/test.go:8)    CALL    runtime.makechan(SB)
        0x001c 00028 (/home/luoyang/code/go_demo/src_test/test.go:8)    MOVQ    AX, main.c+16(SP)
        0x0021 00033 (/home/luoyang/code/go_demo/src_test/test.go:9)    PCDATA  $1, $1
        0x0021 00033 (/home/luoyang/code/go_demo/src_test/test.go:9)    CALL    runtime.printlock(SB)
        0x0026 00038 (/home/luoyang/code/go_demo/src_test/test.go:9)    MOVQ    main.c+16(SP), AX
        0x002b 00043 (/home/luoyang/code/go_demo/src_test/test.go:9)    PCDATA  $1, $0
        0x002b 00043 (/home/luoyang/code/go_demo/src_test/test.go:9)    CALL    runtime.printpointer(SB)
        0x0030 00048 (/home/luoyang/code/go_demo/src_test/test.go:9)    CALL    runtime.printnl(SB)
        0x0035 00053 (/home/luoyang/code/go_demo/src_test/test.go:9)    CALL    runtime.printunlock(SB)
        0x003a 00058 (/home/luoyang/code/go_demo/src_test/test.go:11)   ADDQ    $24, SP
        0x003e 00062 (/home/luoyang/code/go_demo/src_test/test.go:11)   POPQ    BP
        ...
~~~

从编译结果能发现，make(chan int)实际调用的是runtime.makechan(SB)

makechan函数

创建一个 Channel 最重要的操作就是创建 hchan 以及分配所需的 buf 大小的内存空间。

src/runtime/chan.go:72

~~~go
// 将 hchan 的大小对齐
// src/runtime/chan.go:29
hchanSize = unsafe.Sizeof(hchan{}) + uintptr(-int(unsafe.Sizeof(hchan{}))&(maxAlign-1))

func makechan(t *chantype, size int) *hchan {
    elem := t.Elem
    
    // compiler checks this but be safe.
    // 编译器检查type是否有效
    if elem.Size_ >= 1<<16 {
        throw("makechan: invalid channel element type")
    }
    // 检查hchan大小是否对齐
    if hchanSize%maxAlign != 0 || elem.Align_ > maxAlign {
        throw("makechan: bad alignment")
    }
    // MulUintptr返回a*b以及乘法是否溢出。在受支持的平台上，这是编译器降低的内在值。
    // 检查elem.Size_ * uintptr(size) ，即检查确认 channel 的容量不会溢出
    // mem = elem.Size_ * uintptr(size)，即channel的容量
    // 无缓冲channel， mem = 0
    mem, overflow := math.MulUintptr(elem.Size_, uintptr(size))
    if overflow || mem > maxAlloc-hchanSize || size < 0 {
        panic(plainError("makechan: size out of range"))
    }
    
    // 当存储在buf中的元素不包含指针时，Hchan不包含GC感兴趣的指针。
    // buf指向相同的分配，elemtype是持久的。
    // SudoG是从其拥有的线程中引用的，因此无法收集它们。
    var c *hchan
    switch {
        // mem == 0表示无缓冲channel
        case mem == 0:
        // 队列或元素大小为零
        // 在堆上为c分配hchanSize大小的内存
        c = (*hchan)(mallocgc(hchanSize, nil, true))
        // 获取c.buf起始位置的指针，初始化c.buf
        c.buf = c.raceaddr()
        // 元素不包含指针
        case elem.PtrBytes == 0:
        // 在一个调用中分配 hchan 和 buf
        c = (*hchan)(mallocgc(hchanSize+mem, nil, true))
        c.buf = add(unsafe.Pointer(c), hchanSize)
        default:
        // 元素包含指针。
        c = new(hchan)
        c.buf = mallocgc(mem, elem, true)
    }
    // 初始化c
    c.elemsize = uint16(elem.Size_)
    c.elemtype = elem
    c.dataqsiz = uint(size)
    lockInit(&c.lock, lockRankHchan)
    
    if debugChan {
        print("makechan: chan=", c, "; elemsize=", elem.Size_, "; dataqsiz=", size, "\n")
    }
    return c
}
~~~

## 3. 发送数据 chan <- v

通过编译获取发送数据实际函数

~~~go
func main() {
    c := make(chan int)
    c <- 1
    println(<-c)
}
~~~

编译结果：

~~~bash
$ go tool compile -S test.go
main.main STEXT size=107 args=0x0 locals=0x28 funcid=0x0 align=0x0
        0x0000 00000 (/home/luoyang/code/go_demo/src_test/test.go:7)    TEXT    main.main(SB), ABIInternal, $40-0
        0x0000 00000 (/home/luoyang/code/go_demo/src_test/test.go:7)    CMPQ    SP, 16(R14)
        0x0004 00004 (/home/luoyang/code/go_demo/src_test/test.go:7)    PCDATA  $0, $-2
        0x0004 00004 (/home/luoyang/code/go_demo/src_test/test.go:7)    JLS     100
        0x0006 00006 (/home/luoyang/code/go_demo/src_test/test.go:7)    PCDATA  $0, $-1
        0x0006 00006 (/home/luoyang/code/go_demo/src_test/test.go:7)    PUSHQ   BP
        0x0007 00007 (/home/luoyang/code/go_demo/src_test/test.go:7)    MOVQ    SP, BP
        0x000a 00010 (/home/luoyang/code/go_demo/src_test/test.go:7)    SUBQ    $32, SP
        0x000e 00014 (/home/luoyang/code/go_demo/src_test/test.go:7)    FUNCDATA        $0, gclocals·J5F+7Qw7O7ve2QcWC7DpeQ==(SB)
        0x000e 00014 (/home/luoyang/code/go_demo/src_test/test.go:7)    FUNCDATA        $1, gclocals·CnDyI2HjYXFz19SsOj98tw==(SB)
        0x000e 00014 (/home/luoyang/code/go_demo/src_test/test.go:9)    LEAQ    type:chan int(SB), AX
        0x0015 00021 (/home/luoyang/code/go_demo/src_test/test.go:9)    XORL    BX, BX
        0x0017 00023 (/home/luoyang/code/go_demo/src_test/test.go:9)    PCDATA  $1, $0
        0x0017 00023 (/home/luoyang/code/go_demo/src_test/test.go:9)    CALL    runtime.makechan(SB)
        0x001c 00028 (/home/luoyang/code/go_demo/src_test/test.go:9)    MOVQ    AX, main.c+24(SP)
        0x0021 00033 (/home/luoyang/code/go_demo/src_test/test.go:10)   LEAQ    main..stmp_0(SB), BX
        0x0028 00040 (/home/luoyang/code/go_demo/src_test/test.go:10)   PCDATA  $1, $1
        0x0028 00040 (/home/luoyang/code/go_demo/src_test/test.go:10)   CALL    runtime.chansend1(SB)
        0x002d 00045 (/home/luoyang/code/go_demo/src_test/test.go:11)   MOVQ    $0, main..autotmp_1+16(SP)
        0x0036 00054 (/home/luoyang/code/go_demo/src_test/test.go:11)   MOVQ    main.c+24(SP), AX
        0x003b 00059 (/home/luoyang/code/go_demo/src_test/test.go:11)   LEAQ    main..autotmp_1+16(SP), BX
        0x0040 00064 (/home/luoyang/code/go_demo/src_test/test.go:11)   PCDATA  $1, $0
        0x0040 00064 (/home/luoyang/code/go_demo/src_test/test.go:11)   CALL    runtime.chanrecv1(SB)
        0x0045 00069 (/home/luoyang/code/go_demo/src_test/test.go:11)   CALL    runtime.printlock(SB)
        0x004a 00074 (/home/luoyang/code/go_demo/src_test/test.go:11)   MOVQ    main..autotmp_1+16(SP), AX
        0x004f 00079 (/home/luoyang/code/go_demo/src_test/test.go:11)   CALL    runtime.printint(SB)
        0x0054 00084 (/home/luoyang/code/go_demo/src_test/test.go:11)   CALL    runtime.printnl(SB)
        0x0059 00089 (/home/luoyang/code/go_demo/src_test/test.go:11)   CALL    runtime.printunlock(SB)
        0x005e 00094 (/home/luoyang/code/go_demo/src_test/test.go:13)   ADDQ    $32, SP
        0x0062 00098 (/home/luoyang/code/go_demo/src_test/test.go:13)   POPQ    BP
        ...
~~~

从编译结果能发现，发送数据时，实际调用：runtime.chansend1(SB)函数，同时接受数据实际调用runtime.chanrecv1(SB)函数

chansend1函数

chansend1函数实际调用chansend函数

src/runtime/chan.go:144

~~~go
func chansend1(c *hchan, elem unsafe.Pointer) {
    chansend(c, elem, true, getcallerpc())
}
~~~

chansend函数

发送过程包含三个步骤：

1. 持有锁
2. 入队，拷贝要发送的数据
3. 释放锁

步骤二存在三种情况：

1. 是否有正在阻塞的接收方，存在则直接发送
2. 不存在，找到是否有空余的缓存，是则存入
3. 以上均不满足，阻塞直到被唤醒

src/runtime/chan.go:160

~~~go
func chansend(c *hchan, ep unsafe.Pointer, block bool, callerpc uintptr) bool {
    // 如果一个 Channel 为零值（比如没有初始化）
    // 当向 nil channel 发送数据时，会调用 gopark
    // 而 gopark 会将当前的 Goroutine 休眠，从而发生死锁崩溃
	if c == nil {
        if !block {
            return false
        }
        gopark(nil, nil, waitReasonChanSendNilChan, traceBlockForever, 2)
        throw("unreachable")
    }
    
    if debugChan {
        print("chansend: chan=", c, "\n")
    }
    
    if raceenabled {
        racereadpc(c.raceaddr(), callerpc, abi.FuncPCABIInternal(chansend))
    }
    // 在不获取锁的情况下检查失败的非阻塞操作。
    if !block && c.closed == 0 && full(c) {
        return false
    }
    
    var t0 int64
    if blockprofilerate > 0 {
        t0 = cputicks()
    }
    
    lock(&c.lock)
    // 持有锁之前我们已经检查了锁的状态，
    // 但这个状态可能在持有锁之前、该检查之后发生变化，
    // 因此还需要再检查一次 channel 的状态
    if c.closed != 0 {
        unlock(&c.lock)
        panic(plainError("send on closed channel"))
    }
    // 1. channel 上有阻塞的接收方，取出接收等待队列的sg，直接发送
    // 若存在，sg = c.recvq.first
    if sg := c.recvq.dequeue(); sg != nil {
        send(c, sg, ep, func() { unlock(&c.lock) }, 3)
        return true
    }
    // 2. 判断 channel 中缓存是否有剩余空间
    // 有剩余空间，存入 c.buf
    if c.qcount < c.dataqsiz {
        // c.buf的大小即chan的大小 例如make(chan int,10),则size=10
        // 获取要存入c.buf的地址指针，即c.buf+c.sendx的地址，类似于数组或则槽,理解为c.buf[c.buf+c.sendx]
        qp := chanbuf(c, c.sendx)
        if raceenabled {
            racenotify(c, c.sendx, nil)
        }
        // 将要发送的数据拷贝到 buf 中
        // ep即为发送的数据，将ep保存到qp位置中，理解为c.buf[c.buf+c.sendx] = ep
        typedmemmove(c.elemtype, qp, ep)
        // 更新发送索引
        c.sendx++
        // c.buf是一个环形缓存，故c.sendx == c.dataqsiz表示这个环存满了，发送索引又到了起始位置
        // 如果 c.sendx 索引越界则设为 0
        if c.sendx == c.dataqsiz {
            c.sendx = 0
        }
        // 完成存入，记录增加的数据，解锁
        c.qcount++
        unlock(&c.lock)
        return true
    }
    
    if !block {
        unlock(&c.lock)
        return false
    }
    
    // 3. 既找不到接收方，buf 也已经存满，阻塞在 channel 上，等待接收方接收数据
    // 获取当前运行g
    gp := getg()
    // 从当前p的sudogcache中获取一个sudog
    mysg := acquireSudog()
    mysg.releasetime = 0
    if t0 != 0 {
        mysg.releasetime = -1
    }
    // mysg保存当前发送方要发送的数据
    mysg.elem = ep
    mysg.waitlink = nil
    // mysg绑定当前g
    mysg.g = gp
    mysg.isSelect = false
    // mysg绑定当前chan
    mysg.c = c
    // gp和mysg绑定
    gp.waiting = mysg
    gp.param = nil
    // 将mysg放入chan的send等待列表
    c.sendq.enqueue(mysg)
    gp.parkingOnChan.Store(true)
    // 被动调度
    // 将当前的 g 从调度队列移出
    gopark(chanparkcommit, unsafe.Pointer(&c.lock), waitReasonChanSend, traceBlockChanSend, 2)
    // 因为调度器在停止当前 g 的时候会记录运行现场，当恢复阻塞的发送操作时候，会从此处继续开始执行
    KeepAlive(ep)
    
    // 有人把我们唤醒了
    if mysg != gp.waiting {
        throw("G waiting list is corrupted")
    }
    gp.waiting = nil
    gp.activeStackChans = false
    closed := !mysg.success
    gp.param = nil
    if mysg.releasetime > 0 {
        blockevent(mysg.releasetime-t0, 2)
    }
    // 取消与之前阻塞的 channel 的关联
    mysg.c = nil
    // 将mysg放回p.sudogcache
    // g被唤醒时mysg已被接收方从c.sendq中取出，所以只需需要将之放回p.sudogcache即可
    releaseSudog(mysg)
    if closed {
        if c.closed == 0 {
            throw("chansend: spurious wakeup")
        }
        panic(plainError("send on closed channel"))
    }
    return true
}
~~~

send函数

src/runtime/chan.go:294

~~~go
func send(c *hchan, sg *sudog, ep unsafe.Pointer, unlockf func(), skip int) {
    // 是否检测数据竞争状态 go build -race
    if raceenabled {
        if c.dataqsiz == 0 {
            racesync(c, sg)
        } else {
            racenotify(c, c.recvx, nil)
            racenotify(c, c.recvx, sg)
            c.recvx++
            if c.recvx == c.dataqsiz {
                c.recvx = 0
            }
            c.sendx = c.recvx // c.sendx = (c.sendx+1) % c.dataqsiz
        }
    }
    // 如果sg.elem不等于nil,即elem数据元素指向某个堆栈
    // 则调用sendDirect将数据直接写入接收方的执行栈
    if sg.elem != nil {
        sendDirect(c.elemtype, sg, ep)
        // 重置sg.elem为nil
        sg.elem = nil
    }
    // 唤醒gp,即当前被阻塞的接受方sg.g
    gp := sg.g
    unlockf()
    gp.param = unsafe.Pointer(sg)
    sg.success = true
    if sg.releasetime != 0 {
        sg.releasetime = cputicks()
    }
    // 将 gp 作为下一个立即被执行的 Goroutine
    goready(gp, skip+1)
}

// src/runtime/chan.go:335
func sendDirect(t *_type, sg *sudog, src unsafe.Pointer) {
    dst := sg.elem
    typeBitsBulkBarrier(t, uintptr(dst), uintptr(src), t.Size_)
    // 直接写入接收方的执行栈！
    memmove(dst, src, t.Size_)
}
~~~

## 4. 接收数据  <- chan

上面以及得到，接受数据实际调用chanrecv1函数

chanrecv1函数

src/runtime/chan.go:441

~~~go
// v := <-c
func chanrecv1(c *hchan, elem unsafe.Pointer) {
    chanrecv(c, elem, true)
}

//go:nosplit
// v,ok := <-c
func chanrecv2(c *hchan, elem unsafe.Pointer) (received bool) {
    _, received = chanrecv(c, elem, true)
    return
}
~~~

chanrecv函数

主要步骤：

1. 上锁
2. 从缓存中出队，拷贝要接收的数据
3. 解锁

步骤二详细流程：

1. 如果 Channel 已被关闭，且 Channel 没有数据，立刻返回
2. 如果存在正在阻塞的发送方，说明缓存已满，从缓存队头取一个数据，再恢复一个阻塞的发送方
3. 否则，检查缓存，如果缓存中仍有数据，则从缓存中读取，读取过程会将队列中的数据拷贝一份到接收方的执行栈中
4. 没有能接受的数据，阻塞当前的接收方 Goroutine

src/runtime/chan.go:457

~~~go
func chanrecv(c *hchan, ep unsafe.Pointer, block bool) (selected, received bool) {
    if debugChan {
        print("chanrecv: chan=", c, "\n")
    }
    // nil channel，同 send，会导致两个 Goroutine 的死锁
    if c == nil {
        if !block {
            return
        }
        gopark(nil, nil, waitReasonChanReceiveNilChan, traceBlockForever, 2)
        throw("unreachable")
    }
    
    // 在不获取锁的情况下检查失败的非阻塞操作。同send
    if !block && empty(c) {
        if atomic.Load(&c.closed) == 0 {
            return
        }
        if empty(c) {
            if raceenabled {
                raceacquire(c.raceaddr())
            }
            if ep != nil {
                typedmemclr(c.elemtype, ep)
            }
            return true, false
        }
    }
    
    var t0 int64
    if blockprofilerate > 0 {
        t0 = cputicks()
    }
    
    lock(&c.lock)
    // 1. channel 已经 close，且 channel 中没有数据，则直接返回
    if c.closed != 0 {
        if c.qcount == 0 {
            if raceenabled {
                raceacquire(c.raceaddr())
            }
            unlock(&c.lock)
            if ep != nil {
                // 清空ep
                typedmemclr(c.elemtype, ep)
            }
            return true, false
        }
    } else {
        // 2. channel 上有阻塞的发送方，取出c.sendq中的sg,直接接收
        if sg := c.sendq.dequeue(); sg != nil {
            recv(c, sg, ep, func() { unlock(&c.lock) }, 3)
            return true, true
        }
    }
    // 3. channel 的 buf 不空
    if c.qcount > 0 {
        // 直接从接收队列中接收，即c.buf[c.buf+c.recvx]
        qp := chanbuf(c, c.recvx)
        if raceenabled {
            racenotify(c, c.recvx, nil)
        }
        if ep != nil {
            // 取出qp,即将qp对应的值（c.buf[c.buf+c.recvx]），拷贝到ep
            typedmemmove(c.elemtype, ep, qp)
        }
        // 清空qp的值，因为该值已经拷贝到ep
        typedmemclr(c.elemtype, qp)
        // 更新接收方索引
        c.recvx++
        // 索引到达临界值，重置索引
        if c.recvx == c.dataqsiz {
            c.recvx = 0
        }
        // 更新buf中的数量
        c.qcount--
        unlock(&c.lock)
        return true, true
    }
    
    if !block {
        unlock(&c.lock)
        return false, false
    }

    // 4. 没有数据可以接收，阻塞当前 Goroutine,同发送阻塞逻辑
    // 当前g
    gp := getg()
    // 从p.sudogcache中获取一个sudog
    mysg := acquireSudog()
    mysg.releasetime = 0
    if t0 != 0 {
        mysg.releasetime = -1
    }
    mysg.elem = ep
    mysg.waitlink = nil
    gp.waiting = mysg
    mysg.g = gp
    mysg.isSelect = false
    mysg.c = c
    gp.param = nil
    c.recvq.enqueue(mysg)
    gp.parkingOnChan.Store(true)
    // 阻塞，等待被唤醒
    gopark(chanparkcommit, unsafe.Pointer(&c.lock), waitReasonChanReceive, traceBlockChanRecv, 2)
    
    // 被唤醒
    if mysg != gp.waiting {
        throw("G waiting list is corrupted")
    }
    gp.waiting = nil
    gp.activeStackChans = false
    if mysg.releasetime > 0 {
        blockevent(mysg.releasetime-t0, 2)
    }
    success := mysg.success
    gp.param = nil
    mysg.c = nil
    releaseSudog(mysg)
    return true, success
}
~~~

recv函数

从recv函数实现，可以看出，当为无缓冲channel时，无缓冲 Channel 的接收方会先从发送方栈拷贝数据后，发送方才会被放回调度队列中，等待重新调度。

src/runtime/chan.go:615

~~~go
func recv(c *hchan, sg *sudog, ep unsafe.Pointer, unlockf func(), skip int) {
    // c.dataqsiz == 0表示size为0，即无缓冲channel
	if c.dataqsiz == 0 {
        if raceenabled {
            racesync(c, sg)
        }
        if ep != nil {
            // 直接从对方的栈进行拷贝,即将sg.elem的值拷贝到ep中
            recvDirect(c.elemtype, sg, ep)
        }
    } else {
        // 有缓冲channel
        // 获取接受队列索引地址，这个地址指向c.buf中的某个位置，类似于c.buf[c.buf+c.recvx]
        // 结合前面的发送代码能发c.recvx和c.sendx是组合使用的，公用c.buf环形缓存
        qp := chanbuf(c, c.recvx)
        if raceenabled {
            racenotify(c, c.recvx, nil)
            racenotify(c, c.recvx, sg)
        }
        // 从接收队列拷贝数据到接收方
        // 将.buf[c.buf+c.recvx]拷贝到ep
        if ep != nil {
            typedmemmove(c.elemtype, ep, qp)
        }
        // 从发送方拷贝数据到接收队列
        // 将当前阻塞的发送方的栈内值拷贝到接收方（c.buf[c.buf+c.recvx]）队列中
        typedmemmove(c.elemtype, qp, sg.elem)
        // 更新c.recvx索引
        c.recvx++
        // 如果c.recvx达到最大值，重置c.recvx
        if c.recvx == c.dataqsiz {
            c.recvx = 0
        }
        // 同步更新发送方索引
        c.sendx = c.recvx // c.sendx = (c.sendx+1) % c.dataqsiz
    }
    // 唤醒阻塞的发送方，放回调度队列，等待重新调度
    // sg.elem值置为空，因为该值已被拷贝到接收方队列中，即c.buf[c.buf+c.recvx]
    // 严格来说sg.elem是被放入了c.buf缓存中
    sg.elem = nil
    gp := sg.g
    unlockf()
    gp.param = unsafe.Pointer(sg)
    sg.success = true
    if sg.releasetime != 0 {
        sg.releasetime = cputicks()
    }
    // 重新唤醒sg.g，即发送方被阻塞的g,等待新的调度
    goready(gp, skip+1)
}
~~~

## 5. Channel 的关闭

close实际调用的是closechan，可以通过编译汇编得到

closechan函数

当 Channel 关闭时，我们必须让所有阻塞的接收方重新被调度，让所有的发送方也重新被调度，这时候 的实现先将 Goroutine 统一添加到一个列表中（需要锁），然后逐个地进行复始（不需要锁）

src/runtime/chan.go:357

~~~go
func closechan(c *hchan) {
    if c == nil {
        panic(plainError("close of nil channel"))
    }
    
    lock(&c.lock)
    if c.closed != 0 {
        unlock(&c.lock)
        panic(plainError("close of closed channel"))
    }
    
    if raceenabled {
        callerpc := getcallerpc()
        racewritepc(c.raceaddr(), callerpc, abi.FuncPCABIInternal(closechan))
        racerelease(c.raceaddr())
    }
    
    c.closed = 1
    // 声明一个gList，用来存放阻塞在 Channel 的 g 
    var glist gList
    
    // 释放所有的接收方
    for {
        // 获取接收方阻塞的g
        sg := c.recvq.dequeue()
        if sg == nil {
            break
        }
        if sg.elem != nil {
            // 清零
            typedmemclr(c.elemtype, sg.elem)
            sg.elem = nil
        }
        if sg.releasetime != 0 {
            sg.releasetime = cputicks()
        }
        gp := sg.g
        gp.param = unsafe.Pointer(sg)
        sg.success = false
        if raceenabled {
            raceacquireg(gp, c.raceaddr())
        }
        glist.push(gp)
    }

    // 释放所有的发送方
    for {
        // 获取发送方阻塞的g
        sg := c.sendq.dequeue()
        if sg == nil {
            break
        }
        sg.elem = nil
        if sg.releasetime != 0 {
            sg.releasetime = cputicks()
        }
        gp := sg.g
        gp.param = unsafe.Pointer(sg)
        sg.success = false
        if raceenabled {
            raceacquireg(gp, c.raceaddr())
        }
        glist.push(gp)
    }
    unlock(&c.lock)

    // 就绪所有的 G
    // 遍历glist中的所有阻塞接收方和发送方的g,依次唤醒，让他重新被调度
    for !glist.empty() {
        gp := glist.pop()
        gp.schedlink = 0
        goready(gp, 3)
    }
}
~~~