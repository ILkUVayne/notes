# 内存分配器初始化

## 1. 初始化位置

schedinit函数

~~~go
func schedinit() {
    // ...
    // 内存分配器初始化
    // /src/runtime/proc.go:726
    mallocinit()
    // ...
}
~~~

## 2. mallocinit函数

内存分配器的初始化除去一些例行的检查之外，就是对堆的初始化了

/src/runtime/malloc.go:375

~~~go
// /src/runtime/mheap.go:233
var mheap_ mheap

func mallocinit() {
    // 一些涉及内存分配器的常量的检查，包括
    // heapArenaBitmapBytes, physPageSize 等等
    // var class_to_size = [_NumSizeClasses]uint16{0, 8, 16, 24, 32, 48, 64, 80, 96, 112, 128, 144, 160, 176, 192, 208, 224, 240, 256, 288, 320, 352, 384, 416, 448, 480, 512, 576, 640, 704, 768, 896, 1024, 1152, 1280, 1408, 1536, 1792, 2048, 2304, 2688, 3072, 3200, 3456, 4096, 4864, 5376, 6144, 6528, 6784, 6912, 8192, 9472, 9728, 10240, 10880, 12288, 13568, 14336, 16384, 18432, 19072, 20480, 21760, 24576, 27264, 28672, 32768}
    // _TinySizeClass = _TinySizeClass = int8(2) = 2
    // _TinySize = _TinySize = 16
    if class_to_size[_TinySizeClass] != _TinySize {
        throw("bad TinySizeClass")
    }

    if heapArenaBitmapWords&(heapArenaBitmapWords-1) != 0 {
        throw("heapArenaBitmapWords not a power of 2")
    }

    // 检查physPageSize
    // 这个值会在osinit的时候设置
    // amd64架构下，会在调用args函数时进行初始化设置
    // 即汇编启动引导最后的步骤，详细代码位置：/src/runtime/asm_amd64.s:347
    // args函数位置：/src/runtime/runtime1.go:66 具体设置代码在调用的sysargs(c, v)函数中实现
    if physPageSize == 0 {
        throw("failed to get system page size")
    }
    // physPageSize值大小相关判断
    if physPageSize > maxPhysPageSize {
        print("system page size (", physPageSize, ") is larger than maximum page size (", maxPhysPageSize, ")\n")
        throw("bad system page size")
    }
    if physPageSize < minPhysPageSize {
        print("system page size (", physPageSize, ") is smaller than minimum page size (", minPhysPageSize, ")\n")
        throw("bad system page size")
    }
    if physPageSize&(physPageSize-1) != 0 {
        print("system page size (", physPageSize, ") must be a power of 2\n")
        throw("bad system page size")
    }
    if physHugePageSize&(physHugePageSize-1) != 0 {
        print("system huge page size (", physHugePageSize, ") must be a power of 2\n")
        throw("bad system huge page size")
    }
    if physHugePageSize > maxPhysHugePageSize {
        physHugePageSize = 0
    }
    if physHugePageSize != 0 {
        for 1<<physHugePageShift != physHugePageSize {
            physHugePageShift++
        }
    }
    if pagesPerArena%pagesPerSpanRoot != 0 {
        print("pagesPerArena (", pagesPerArena, ") is not divisible by pagesPerSpanRoot (", pagesPerSpanRoot, ")\n")
        throw("bad pagesPerSpanRoot")
    }
    if pagesPerArena%pagesPerReclaimerChunk != 0 {
        print("pagesPerArena (", pagesPerArena, ") is not divisible by pagesPerReclaimerChunk (", pagesPerReclaimerChunk, ")\n")
        throw("bad pagesPerReclaimerChunk")
    }

    if minTagBits > taggedPointerBits {
        throw("taggedPointerbits too small")
    }

    // 初始化堆 mheap_
    // mheap_ 是全局变量
    mheap_.init()
    // 初始化全局变量mcache0，用于在初始化g.m.p时，即p.init
    // 详细代码查看procresize函数实现： src/runtime/proc.go:5241
    // mcache0只会与p.id==0的p绑定
    mcache0 = allocmcache()
    // 初始化锁
    lockInit(&gcBitsArenas.lock, lockRankGcBitsArenas)
    lockInit(&profInsertLock, lockRankProfInsert)
    lockInit(&profBlockLock, lockRankProfBlock)
    lockInit(&profMemActiveLock, lockRankProfMemActive)
    for i := range profMemFutureLock {
        lockInit(&profMemFutureLock[i], lockRankProfMemFuture)
    }
    lockInit(&globalAlloc.mutex, lockRankGlobalAlloc)

    // 创建初始的 arena 增长 hint
    // 初始化内存分配 arena，arena 是一段连续的内存，负责数据的内存分配
    // PtrSize是指针的大小（以字节为单位）-不安全。Sizeof（uintptr（0）），但作为理想常数。它也是机器本机单词大小的大小（即，在32位系统上为4，在64位上为8）。
    // goarch.PtrSize == 8 表示64位机器，即创建初始化64位机器
    if goarch.PtrSize == 8 {
        // 0x7f = 127
        for i := 0x7f; i >= 0; i-- {
            var p uintptr
            switch {
            case raceenabled:
                p = uintptr(i)<<32 | uintptrMask&(0x00c0<<32)
                if p >= uintptrMask&0x00e000000000 {
                    continue
                }
            case GOARCH == "arm64" && GOOS == "ios":
                p = uintptr(i)<<40 | uintptrMask&(0x0013<<28)
            case GOARCH == "arm64":
                p = uintptr(i)<<40 | uintptrMask&(0x0040<<32)
            case GOOS == "aix":
                if i == 0 {
                    continue
                }
                p = uintptr(i)<<40 | uintptrMask&(0xa0<<52)
            default:
                p = uintptr(i)<<40 | uintptrMask&(0x00c0<<32)
            }
            // 获取堆的arenaHints链表地址
            hintList := &mheap_.arenaHints
            if (!raceenabled && i > 0x3f) || (raceenabled && i > 0x5f) {
                hintList = &mheap_.userArena.arenaHints
            }
            // 通过arenaHints分配器分配一个hint
            hint := (*arenaHint)(mheap_.arenaHintAlloc.alloc())
            hint.addr = p
            // 将hint添加到arenaHints链表
            hint.next, *hintList = *hintList, hint
        }
    // else 32位机器，目前很少会有32机器了
    } else {
        const arenaMetaSize = (1 << arenaBits) * unsafe.Sizeof(heapArena{})
        meta := uintptr(sysReserve(nil, arenaMetaSize))
        if meta != 0 {
            mheap_.heapArenaAlloc.init(meta, arenaMetaSize, true)
        }
        
        procBrk := sbrk0()
        
        p := firstmoduledata.end
        if p < procBrk {
            p = procBrk
        }
        if mheap_.heapArenaAlloc.next <= p && p < mheap_.heapArenaAlloc.end {
            p = mheap_.heapArenaAlloc.end
        }
        p = alignUp(p+(256<<10), heapArenaBytes)
        arenaSizes := []uintptr{
            512 << 20,
            256 << 20,
            128 << 20,
        }
        for _, arenaSize := range arenaSizes {
            a, size := sysReserveAligned(unsafe.Pointer(p), arenaSize, heapArenaBytes)
            if a != nil {
                mheap_.arena.init(uintptr(a), size, false)
                p = mheap_.arena.end // For hint below
                break
            }
        }
        hint := (*arenaHint)(mheap_.arenaHintAlloc.alloc())
        hint.addr = p
        hint.next, mheap_.arenaHints = mheap_.arenaHints, hint
        
        userArenaHint := (*arenaHint)(mheap_.arenaHintAlloc.alloc())
        userArenaHint.addr = p
        userArenaHint.next, mheap_.userArena.arenaHints = mheap_.userArena.arenaHints, userArenaHint
    }
    // 在这里初始化内存限制，因为分配器会查看它，但我们还没有调用gcinit，我们肯定会在那之前分配内存。
    gcController.memoryLimit.Store(maxInt64)
}
~~~

## 3. 堆初始化

init函数

/src/runtime/mheap.go:752

~~~go
func (h *mheap) init() {
    // 初始化锁
    lockInit(&h.lock, lockRankMheap)
    lockInit(&h.speciallock, lockRankMheapSpecial)
    // 初始化堆中各个组件的分配器
    // span的分配器
    h.spanalloc.init(unsafe.Sizeof(mspan{}), recordspan, unsafe.Pointer(h), &memstats.mspan_sys)
    // mcache的分配器
    h.cachealloc.init(unsafe.Sizeof(mcache{}), nil, nil, &memstats.mcache_sys)
    // specialfinalizer的分配器
    h.specialfinalizeralloc.init(unsafe.Sizeof(specialfinalizer{}), nil, nil, &memstats.other_sys)
    // specialprofile的分配器  
    h.specialprofilealloc.init(unsafe.Sizeof(specialprofile{}), nil, nil, &memstats.other_sys)
    // specialReachable的分配器  
    h.specialReachableAlloc.init(unsafe.Sizeof(specialReachable{}), nil, nil, &memstats.other_sys)
    // specialPinCounter的分配器 
    h.specialPinCounterAlloc.init(unsafe.Sizeof(specialPinCounter{}), nil, nil, &memstats.other_sys)
    // arenaHints的分配器
    h.arenaHintAlloc.init(unsafe.Sizeof(arenaHint{}), nil, nil, &memstats.other_sys)
    
    // 不对 mspan 的分配清零，后台扫描可以通过分配它来并发的检查一个 span
    // 因此 span 的 sweepgen 在释放和重新分配时候能存活，从而可以防止后台扫描
    // 不正确的将其从 0 进行 CAS。
    //
    // 因为 mspan 不包含堆指针，因此它是安全的
    h.spanalloc.zero = false
    // 遍历h.central，初始化h.central[i].mcentral
    // len(h.central) == 136
    for i := range h.central {
        h.central[i].mcentral.init(spanClass(i))
    }
    // 初始化h.pages
    h.pages.init(&h.lock, &memstats.gcMiscSys, false)
}
~~~

### 3.1 分配器

mheap结构

从mheap结构结构能发现，调用init函数初始化的分配器全是fixalloc。

linearAlloc 是一个基于线性分配策略的分配器，从mallocinit函数中能够看出它只作为 mheap_.heapArenaAlloc 和 mheap_.arena 在 32 位系统上使用

/src/runtime/mheap.go:202

~~~go
type mheap struct {
    ...
    lock mutex
    pages pageAlloc // page allocation data structure
    ...

    heapArenaAlloc linearAlloc
    
    ...
    
    arena linearAlloc
    ...
    
    central [numSpanClasses]struct {
        mcentral mcentral
        pad      [(cpu.CacheLinePadSize - unsafe.Sizeof(mcentral{})%cpu.CacheLinePadSize) % cpu.CacheLinePadSize]byte
    }
    
    spanalloc              fixalloc // allocator for span*
    cachealloc             fixalloc // allocator for mcache*
    specialfinalizeralloc  fixalloc // allocator for specialfinalizer*
    specialprofilealloc    fixalloc // allocator for specialprofile*
    specialReachableAlloc  fixalloc // allocator for specialReachable
    specialPinCounterAlloc fixalloc // allocator for specialPinCounter
    speciallock            mutex    // lock for special record allocators.
    arenaHintAlloc         fixalloc // allocator for arenaHints
    
    ...
}
~~~

#### 3.1.1 fixalloc

fixalloc 是一个基于自由列表的固定大小的分配器。其核心原理是将若干未分配的内存块连接起来， 将未分配的区域的第一个字为指向下一个未分配区域的指针使用。Go 的主分配堆中 malloc（span、cache、treap、finalizer、profile、arena hint 等） 均 围绕它为实体进行固定分配和回收

结构

/src/runtime/mfixalloc.go:31

~~~go
// fixalloc 是一个简单的固定大小对象的自由表内存分配器。
// Malloc 使用围绕 sysAlloc 的 fixalloc 来管理其 MCache 和 MSpan 对象。
//
// fixalloc.alloc 返回的内存默认为零，但调用者可以通过将 zero 标志设置为 false
// 来自行负责将分配归零。如果这部分内存永远不包含堆指针，则这样的操作是安全的。
//
// 调用方负责锁定 fixalloc 调用。调用方可以在对象中保持状态，
// 但当释放和重新分配时第一个字会被破坏。
//
// 考虑使 fixalloc 的类型变为 go:notinheap.
type fixalloc struct {
    size   uintptr
    first  func(arg, p unsafe.Pointer) // 首次调用时返回 p
    arg    unsafe.Pointer
    list   *mlink // 可复用的被回收的内存链表
    chunk  uintptr // 使用 uintptr 而非 unsafe.Pointer 来避免 write barrier
    nchunk uint32  // 当前chunk中剩余的字节
    nalloc uint32  // 新chunks的大小（字节）
    inuse  uintptr // 正在使用字节
    stat   *sysMemStat
    zero   bool // 归零的分配
}
~~~

##### 3.1.1.1 init

/src/runtime/mfixalloc.go:56

~~~go
type mlink struct {
    _    sys.NotInHeap
    next *mlink
}

func (f *fixalloc) init(size uintptr, first func(arg, p unsafe.Pointer), arg unsafe.Pointer, stat *sysMemStat) {
    // _FixAllocChunk = 16 << 10 = 16384
    // Chunk最大值不能超过16384
	if size > _FixAllocChunk {
        throw("runtime: fixalloc size too large")
    }
    // size 不能小于unsafe.Sizeof(mlink{})
    // unsafe.Sizeof(v) 返回v占用的内存大小（字节）
    if min := unsafe.Sizeof(mlink{}); size < min {
        size = min
    }
    // 初始化fixalloc的属性值
    f.size = size
    f.first = first
    f.arg = arg
    f.list = nil
    f.chunk = 0
    f.nchunk = 0
    f.nalloc = uint32(_FixAllocChunk / size * size) // Round _FixAllocChunk down to an exact multiple of size to eliminate tail waste
    f.inuse = 0
    f.stat = stat
    f.zero = true
}
~~~

##### 3.1.1.2 分配

fixalloc 基于自由表策略进行实现，分为两种情况：

1. 存在被释放、可复用的内存
2. 不存在可复用的内存

对于第一种情况，也就是在运行时内存被释放，但这部分内存并不会被立即回收给操作系统， 我们直接从自由表f.list中获得即可，但需要注意按需将这部分内存进行清零操作。

对于第二种情况，我们直接向操作系统申请固定大小的内存，然后扣除分配的大小即可。

/src/runtime/mfixalloc.go:76

~~~go
func (f *fixalloc) alloc() unsafe.Pointer {
    // fixalloc 的个字段必须先被 init
    if f.size == 0 {
        print("runtime: use of FixAlloc_Alloc before FixAlloc_Init\n")
        throw("runtime: internal error")
    }
    // 如果 f.list 不是 nil, 则说明还存在已经释放、可复用的内存，直接将其分配
    if f.list != nil {
        // 取出 f.list
        v := unsafe.Pointer(f.list)
        // 并将其指向下一段区域
        f.list = f.list.next
        // 增加使用的(分配)大小
        f.inuse += f.size
        // 如果需要对内存清零，则对取出的内存执行初始化
        // f.zero默认都为true,除了h.spanalloc
        // mheap.init中单独设置了h.spanalloc.zero = false
        if f.zero {
            // memclrNoHeapPointers 用于清理不包含堆指针的内存区块,从v的地址开始的f.size个字节
            // memclrNoHeapPointers使用汇编实现
            // amd64架构：src/runtime/memclr_amd64.s:13
            memclrNoHeapPointers(v, f.size)
        }
        // 返回分配的内存
        return v
    }
    // f.list 中没有可复用的内存
    // 如果此时 nchunk 不足以分配一个 size
    if uintptr(f.nchunk) < f.size {
        // 则向操作系统申请内存，大小为 16 << 10 pow(2,14),即2的14次幂 16384
        // align = 0,则使用默认值8，即以8字节对齐
        // align必须是2的幂
        f.chunk = uintptr(persistentalloc(uintptr(f.nalloc), 0, f.stat))
        // 更新当前chunk中剩余的字节
        f.nchunk = f.nalloc
    }
    // 获取申请好的内存地址
    v := unsafe.Pointer(f.chunk)
    // first 只有在 fixalloc 作为 spanalloc 时候，才会被设置为 recordspan
    // 通过mheap.init函数能发现，只有span的分配器才会传入first，其他分配器都是nil
    if f.first != nil {
        // 用于为 heap.allspans 添加新的 span
        f.first(f.arg, v)
    }
    f.chunk = f.chunk + f.size
    // 更新当前chunk中剩余的字节，扣除并保留 size 大小的空间
    f.nchunk -= uint32(f.size)
    // 更新正在使用字节
    f.inuse += f.size
    return v
}
~~~

##### 3.1.1.3 回收

/src/runtime/mfixalloc.go:106

直接将回收的地址指针放回到自由表f.list中即可

~~~go
func (f *fixalloc) free(p unsafe.Pointer) {
    // 更新正在使用字节,减去回收的大小
    f.inuse -= f.size
    // 将要释放的内存地址作为 mlink 指针插入到 f.list 内，完成回收
    v := (*mlink)(p)
    v.next = f.list
    f.list = v
}
~~~

## 4 分配mcache

mcache会在p初始化的时候被初始化，并绑定到p上

结构：

/src/runtime/mcache.go:19

~~~go
type mcache struct {
    _ sys.NotInHeap
    
    // 下面的成员在每次 malloc 时都会被访问
    // 因此将它们放到一起来利用缓存的局部性原理
    nextSample uintptr // 分配这么多字节后触发堆样本
    scanAlloc  uintptr // 分配的可扫描堆的字节数
    
    // 没有指针的微小对象的分配器缓存。
    // 请参考 malloc.go 中的 "小型分配器" 注释。
    //
    // tiny 指向当前 tiny 块的起始位置，或当没有 tiny 块时候为 nil
    // tiny 是一个堆指针。由于 mcache 在非 GC 内存中，我们通过在
    // mark termination 期间在 releaseAll 中清除它来处理它。
    tiny       uintptr
    tinyoffset uintptr
    tinyAllocs uintptr 
    
    // 其余部分并不是在每个malloc上都可以访问的。
    
    alloc [numSpanClasses]*mspan // 用来分配的 spans，由 spanClass 索引
    
    stackcache [_NumStackOrders]stackfreelist
    
    flushGen atomic.Uint32
}
~~~

### 4.1 分配

allocmcache函数

从 mheap 上分配一个 mcache。 由于 mheap 是全局的，因此在分配期必须对其进行加锁，而分配通过 fixAlloc 组件完成

/src/runtime/mcache.go:85

~~~go
// 虚拟的MSpan，不包含任何对象。
var emptymspan mspan

func allocmcache() *mcache {
    var c *mcache
    // 切换到g0栈执行
    systemstack(func() {
        lock(&mheap_.lock)
        // 使用mcache分配器分配一个mcache
        c = (*mcache)(mheap_.cachealloc.alloc())
        // 设置c.flushGen == mheap_.sweepgen
        c.flushGen.Store(mheap_.sweepgen)
        unlock(&mheap_.lock)
    })
    // len(c.alloc) == 136
    for i := range c.alloc {
        // 暂时指向虚拟的 mspan 中
        c.alloc[i] = &emptymspan
    }
    // 返回下一个采样点，是服从泊松过程的随机数
    c.nextSample = nextSample()
    return c
}
~~~

由于运行时提供了采样过程堆分析的支持， 由于我们的采样的目标是平均每个 MemProfileRate 字节对分配进行采样， 显然，在整个时间线上的分配情况应该是完全随机分布的，这是一个泊松过程。 因此最佳的采样点应该是服从指数分布 exp(MemProfileRate) 的随机数，其中 MemProfileRate 为均值。

~~~go
// MemProfileRate 是一个公共变量，可以在用户态代码进行修改
// /src/runtime/mprof.go:595
var MemProfileRate int = 512 * 1024

// /src/runtime/malloc.go:1370
func nextSample() uintptr {
    if GOOS == "plan9" {
        // Plan 9 doesn't support floating point in note handler.
        if g := getg(); g == g.m.gsignal {
            return nextSampleNoFP()
        }
    }
    
    return uintptr(fastexprand(MemProfileRate))
}
~~~

### 4.2 释放

releaseAll函数

由于 mcache 从非 GC 内存上进行分配，因此出现的任何堆指针都必须进行特殊处理。 所以在释放前，需要调用 mcache.releaseAll 将堆指针进行处理

/src/runtime/mcache.go:259

~~~go
func (c *mcache) releaseAll() {
    // 刷新scanAlloc
    scanAlloc := int64(c.scanAlloc)
    c.scanAlloc = 0
    
    sg := mheap_.sweepgen
    dHeapLive := int64(0)
    for i := range c.alloc {
        s := c.alloc[i]
        if s != &emptymspan {
            slotsUsed := int64(s.allocCount) - int64(s.allocCountBeforeCache)
            s.allocCountBeforeCache = 0
            
            // 根据分配的内容调整smallAllocCount。
            stats := memstats.heapStats.acquire()
            atomic.Xadd64(&stats.smallAllocCount[spanClass(i).sizeclass()], slotsUsed)
            memstats.heapStats.release()
            
            // 在不一致的内部统计中调整实际分配。
            // 我们之前假设已分配整个跨度
            gcController.totalAlloc.Add(slotsUsed * int64(s.elemsize))
            
            if s.sweepgen != sg+1 {
                dHeapLive -= int64(uintptr(s.nelems)-uintptr(s.allocCount)) * int64(s.elemsize)
            }
            
            // 将 span 归还
            mheap_.central[i].mcentral.uncacheSpan(s)
            c.alloc[i] = &emptymspan
        }
    }
    // 清空 tinyalloc 池.
    c.tiny = 0
    c.tinyoffset = 0
    
    // 刷新 tinyAllocs.
    stats := memstats.heapStats.acquire()
    atomic.Xadd64(&stats.tinyAllocCount, int64(c.tinyAllocs))
    c.tinyAllocs = 0
    memstats.heapStats.release()
    
    // 更新heapLive和heapScan。
    gcController.update(dHeapLive, scanAlloc)
}

func freemcache(c *mcache) {
    systemstack(func() {
        // 归还 span
        c.releaseAll()
        // 释放 stack
        stackcache_clear(c)
        lock(&mheap_.lock)
        // 将 mcache 释放
        mheap_.cachealloc.free(unsafe.Pointer(c))
        unlock(&mheap_.lock)
    })
}
~~~

## 5 总结

内存初始化步骤：

1. 调度器的初始化schedinit函数中调用mallocinit()开始内存初始化
2. 首先进行一些涉及内存分配器的常量的检查，然后开始调用mheap_.init()初始化堆
3. mheap_.init初始化堆中各个组件的分配器，初始化h.central数组中的mcentral，初始化h.pages
4. 堆初始化完成后，调用allocmcache从堆中分配一个mcache0，用于在schedinit函数之后调用procresize初始化p时与p.id==0的p绑定，mcache0被绑定后就会被销毁
5. 然后初始化内存分配 arena（64位操作系统），最后初始化内存限制，然后内存初始化结束，然后继续schedinit函数的后续其他初始化操作