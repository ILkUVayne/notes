# 对象分配流程

> go 1.21

## 1. 分配方式

编译器会优先尝试在栈上分配内存，因为这样不需要gc来回收，性能更佳，若无法在栈上分配（如所需内存太大等）则需要在堆上分配内存，new(T)、&T{}、make(T,size)最终最终都会殊途同归步入 mallocgc 方法中

尝试编译以下代码：

~~~go
package main

type largeobj struct {
    arr [1 << 20]byte
}

func main() {

    v1 := new(largeobj)
    v2 := &largeobj{}
    v3 := make([]int, 1<<20)
    println(v1, v2, v3)
}
~~~

编译结果：

~~~bash
$ go tool compile -S test.go
main.main STEXT size=165 args=0x0 locals=0x38 funcid=0x0 align=0x0
        0x0000 00000 (/home/luoyang/code/go_demo/src_test/test.go:7)    TEXT    main.main(SB), ABIInternal, $56-0
        0x0000 00000 (/home/luoyang/code/go_demo/src_test/test.go:7)    CMPQ    SP, 16(R14)
        0x0004 00004 (/home/luoyang/code/go_demo/src_test/test.go:7)    PCDATA  $0, $-2
        0x0004 00004 (/home/luoyang/code/go_demo/src_test/test.go:7)    JLS     150
        0x000a 00010 (/home/luoyang/code/go_demo/src_test/test.go:7)    PCDATA  $0, $-1
        0x000a 00010 (/home/luoyang/code/go_demo/src_test/test.go:7)    PUSHQ   BP
        0x000b 00011 (/home/luoyang/code/go_demo/src_test/test.go:7)    MOVQ    SP, BP
        0x000e 00014 (/home/luoyang/code/go_demo/src_test/test.go:7)    SUBQ    $48, SP
        0x0012 00018 (/home/luoyang/code/go_demo/src_test/test.go:7)    FUNCDATA        $0, gclocals·3CgL1OMj4PK20UKKkS8Bfw==(SB)
        0x0012 00018 (/home/luoyang/code/go_demo/src_test/test.go:7)    FUNCDATA        $1, gclocals·yYudLF4U+CnNIuwQq6MIGQ==(SB)
        0x0012 00018 (/home/luoyang/code/go_demo/src_test/test.go:9)    LEAQ    type:main.largeobj(SB), AX
        0x0019 00025 (/home/luoyang/code/go_demo/src_test/test.go:9)    PCDATA  $1, $0
        0x0019 00025 (/home/luoyang/code/go_demo/src_test/test.go:9)    CALL    runtime.newobject(SB)
        0x001e 00030 (/home/luoyang/code/go_demo/src_test/test.go:9)    MOVQ    AX, main.v1+32(SP)
        0x0023 00035 (/home/luoyang/code/go_demo/src_test/test.go:10)   LEAQ    type:main.largeobj(SB), AX
        0x002a 00042 (/home/luoyang/code/go_demo/src_test/test.go:10)   PCDATA  $1, $1
        0x002a 00042 (/home/luoyang/code/go_demo/src_test/test.go:10)   CALL    runtime.newobject(SB)
        0x002f 00047 (/home/luoyang/code/go_demo/src_test/test.go:10)   MOVQ    AX, main.v2+24(SP)
        0x0034 00052 (/home/luoyang/code/go_demo/src_test/test.go:11)   MOVL    $1048576, BX
        0x0039 00057 (/home/luoyang/code/go_demo/src_test/test.go:11)   MOVQ    BX, CX
        0x003c 00060 (/home/luoyang/code/go_demo/src_test/test.go:11)   LEAQ    type:int(SB), AX
        0x0043 00067 (/home/luoyang/code/go_demo/src_test/test.go:11)   PCDATA  $1, $2
        0x0043 00067 (/home/luoyang/code/go_demo/src_test/test.go:11)   CALL    runtime.makeslice(SB)
        0x0048 00072 (/home/luoyang/code/go_demo/src_test/test.go:11)   MOVQ    AX, main..autotmp_7+40(SP)
        0x004d 00077 (/home/luoyang/code/go_demo/src_test/test.go:12)   PCDATA  $1, $3
        0x004d 00077 (/home/luoyang/code/go_demo/src_test/test.go:12)   CALL    runtime.printlock(SB)
        0x0052 00082 (/home/luoyang/code/go_demo/src_test/test.go:12)   MOVQ    main.v1+32(SP), AX
        0x0057 00087 (/home/luoyang/code/go_demo/src_test/test.go:12)   PCDATA  $1, $4
        0x0057 00087 (/home/luoyang/code/go_demo/src_test/test.go:12)   CALL    runtime.printpointer(SB)
        0x005c 00092 (/home/luoyang/code/go_demo/src_test/test.go:12)   NOP
        0x0060 00096 (/home/luoyang/code/go_demo/src_test/test.go:12)   CALL    runtime.printsp(SB)
        0x0065 00101 (/home/luoyang/code/go_demo/src_test/test.go:12)   MOVQ    main.v2+24(SP), AX
        0x006a 00106 (/home/luoyang/code/go_demo/src_test/test.go:12)   PCDATA  $1, $5
        0x006a 00106 (/home/luoyang/code/go_demo/src_test/test.go:12)   CALL    runtime.printpointer(SB)
        0x006f 00111 (/home/luoyang/code/go_demo/src_test/test.go:12)   CALL    runtime.printsp(SB)
        0x0074 00116 (/home/luoyang/code/go_demo/src_test/test.go:12)   MOVQ    main..autotmp_7+40(SP), AX
        0x0079 00121 (/home/luoyang/code/go_demo/src_test/test.go:12)   MOVL    $1048576, BX
        0x007e 00126 (/home/luoyang/code/go_demo/src_test/test.go:12)   MOVQ    BX, CX
        0x0081 00129 (/home/luoyang/code/go_demo/src_test/test.go:12)   PCDATA  $1, $0
        0x0081 00129 (/home/luoyang/code/go_demo/src_test/test.go:12)   CALL    runtime.printslice(SB)
        0x0086 00134 (/home/luoyang/code/go_demo/src_test/test.go:12)   CALL    runtime.printnl(SB)
        0x008b 00139 (/home/luoyang/code/go_demo/src_test/test.go:12)   CALL    runtime.printunlock(SB)
        0x0090 00144 (/home/luoyang/code/go_demo/src_test/test.go:13)   ADDQ    $48, SP
        0x0094 00148 (/home/luoyang/code/go_demo/src_test/test.go:13)   POPQ    BP
        0x0095 00149 (/home/luoyang/code/go_demo/src_test/test.go:13)   RET
~~~

从编译结果能发现：

new(largeobj) -> v1 -> runtime.newobject(SB)

&largeobj{} -> v2 -> runtime.newobject(SB)

make([]int, 1<<20) -> v3 -> runtime.makeslice(SB)

newobject函数

/src/runtime/malloc.go:1323

~~~go
func newobject(typ *_type) unsafe.Pointer {
    return mallocgc(typ.Size_, typ, true)
}
~~~

makeslice函数

/src/runtime/slice.go:88

~~~go
func makeslice(et *_type, len, cap int) unsafe.Pointer {
    mem, overflow := math.MulUintptr(et.Size_, uintptr(cap))
    if overflow || mem > maxAlloc || len < 0 || len > cap {
        mem, overflow := math.MulUintptr(et.Size_, uintptr(len))
        if overflow || mem > maxAlloc || len < 0 {
            panicmakeslicelen()
        }
        panicmakeslicecap()
    }
    
    return mallocgc(mem, et, true)
}
~~~

从newobject和makeslice方法能发现，分配堆内存均是调用mallocgc实现

## 2. mallocgc分配内存

根据size（需要分配的内存大小），分三种方式分配内存

1. 微对象且不包含指针 <16b
2. 小对象 <=32kb
3. 大对象 >32kb

go堆内存分配的统一入口：

/src/runtime/malloc.go:948

~~~go
var zerobase uintptr

func mallocgc(size uintptr, typ *_type, needzero bool) unsafe.Pointer {
    if gcphase == _GCmarktermination {
        throw("mallocgc called with gcphase == _GCmarktermination")
    }
    // 需要分配的内存size==0时，均返回zerobase的地址
    if size == 0 {
        return unsafe.Pointer(&zerobase)
    }
    
    lockRankMayQueueFinalizer()
    
    userSize := size
    // 开启了内存错误检查 go build -asan
    if asanenabled {
        size += computeRZlog(size)
    }
    // debug malloc
    if debug.malloc {
        if debug.sbrk != 0 {
            align := uintptr(16)
            if typ != nil {
                if size&7 == 0 {
                    align = 8
                } else if size&3 == 0 {
                    align = 4
                } else if size&1 == 0 {
                    align = 2
                } else {
                    align = 1
                }
            }
            return persistentalloc(size, align, &memstats.other_sys)
        }
        
        if inittrace.active && inittrace.id == getg().goid {
            inittrace.allocs += 1
        }
    }
    // 计算在gc阶段（gcBlackenEnabled != 0）分配的辅助标记内存大小
    assistG := deductAssistCredit(size)
    
    // 获取当前m，并且当前m不能被抢占
    // 不能被抢占原因acquirem函数将m.locks++，src/runtime/preempt.go:286 canPreemptM方法会判断是否可抢占
    mp := acquirem()
    if mp.mallocing != 0 {
        throw("malloc deadlock")
    }
    if mp.gsignal == getg() {
        throw("malloc during signal")
    }
    // 表示m正在内存分配
    mp.mallocing = 1
    // 是否需要触发gc
    shouldhelpgc := false
    dataSize := userSize
    // 获取当前m.p.mcache
    c := getMCache(mp)
    if c == nil {
        throw("mallocgc called without a P or outside bootstrapping")
    }
    var span *mspan
    // 声明分配的内存的地址
    var x unsafe.Pointer
    // 判断需要申请的内存是否包含指针，标识 gc 时是否需要展开扫描，noscan==true 不包含 反之包含
    noscan := typ == nil || typ.PtrBytes == 0
    delayedZeroing := false
    // 根据size（需要分配的内存大小），分三种方式分配内存
    //    1. 微对象且不包含指针 <16b
    //    2. 小对象 <=32kb
    //    3. 大对象 >32kb
    //
    // 小对象或者微对象分配 <=32kb
    if size <= maxSmallSize {
        // 微对象 <16b 且不包含指针
        if noscan && size < maxTinySize {
            // tiny 内存块中的空闲内存偏移值，即从 offset 往后有空闲位置
            off := c.tinyoffset
            // 内存对齐
            // 如果大小为 5 ~ 8 B，size 会被调整为 8 b，此时 8 & 7 == 0，会走进此分支
            if size&7 == 0 {
                // 将 offset 补齐到 8 b 倍数的位置
                off = alignUp(off, 8)
            // 如果是32位系统，且size==12时，走这个分支
            } else if goarch.PtrSize == 4 && size == 12 {
                // 将 offset 补齐到 8 b 倍数的位置
                off = alignUp(off, 8)
            // 如果大小为 3 ~ 4 b，size 会被调整为 4 b，此时 4 & 3 == 0，会走进此分支
            } else if size&3 == 0 {
                // 将 offset 补齐到 4 b 倍数的位置
                off = alignUp(off, 4)
            // 如果大小为 1 ~ 2 b，size 会被调整为 2 b，此时 2 & 1 == 0，会走进此分支
            } else if size&1 == 0 {
                // 将 offset 补齐到 2 b 倍数的位置
                off = alignUp(off, 2)
            }
            // 如果当前 tiny 内存块空间还够用，则直接分配并返回
            if off+size <= maxTinySize && c.tiny != 0 {
                // 分配空间
                // c.tiny + off 即获取空闲内存的开始地址
                x = unsafe.Pointer(c.tiny + off)
                // 更新空闲内存偏移值
                c.tinyoffset = off + size
                // 更新tiny内存分配的对象数量
                c.tinyAllocs++
                // 更新mp.mallocing的值，表示内存分配结束
                mp.mallocing = 0
                // 恢复m可抢占
                releasem(mp)
                return x
            }
            // 当前 tiny 内存块空间不够，分配一个新的 tiny 内存块
            span = c.alloc[tinySpanClass]
            // 尝试从 mCache 中获取
            v := nextFreeFast(span)
            if v == 0 {
                // 从 mCache 中获取失败，则从 mCentral 或者 mHeap 中获取进行兜底
                v, span, shouldhelpgc = c.nextFree(tinySpanClass)
            }
            // 获取分配的内存地址
            x = unsafe.Pointer(v)
            // 初始化
            (*[2]uint64)(x)[0] = 0
            (*[2]uint64)(x)[1] = 0
            if !raceenabled && (size < c.tinyoffset || c.tiny == 0) {
                c.tiny = uintptr(x)
                c.tinyoffset = size
            }
            size = maxTinySize
        } else {
            // 分配小对象
            // 根据对象大小，映射到其所属的 span 的等级(0~66）
            var sizeclass uint8
            if size <= smallSizeMax-8 {
                sizeclass = size_to_class8[divRoundUp(size, smallSizeDiv)]
            } else {
                sizeclass = size_to_class128[divRoundUp(size-smallSizeMax, largeSizeDiv)]
            }
            // 对应 span 等级下，分配给每个对象的空间大小(0~32KB)
            size = uintptr(class_to_size[sizeclass])
            // 创建 spanClass 标识，其中前 7 位对应为 span 的等级(0~66)，最后标识表示了这个对象 gc 时是否需要扫描
            spc := makeSpanClass(sizeclass, noscan)
            // 获取 mcache 中的 span
            span = c.alloc[spc]
            // 尝试从mcache的span中获取内存
            v := nextFreeFast(span)
            if v == 0 {
                // mcache 分配空间失败，则通过 mcentral、mheap 兜底
                v, span, shouldhelpgc = c.nextFree(spc)
            }
            // 分配空间 
            x = unsafe.Pointer(v)
            if needzero && span.needzero != 0 {
                // 清理内存区块
                memclrNoHeapPointers(x, size)
            }
        }
    } else {
		// 分配大对象
        shouldhelpgc = true
        // 从 mheap 中获取 0 号 span
        span = c.allocLarge(size, noscan)
        span.freeindex = 1
        span.allocCount = 1
        size = span.elemsize
        // 分配空间  
        x = unsafe.Pointer(span.base())
        if needzero && span.needzero != 0 {
            if noscan {
                delayedZeroing = true
            } else {
                memclrNoHeapPointers(x, size)
            }
        }
    }
    
    if !noscan {
        var scanSize uintptr
        heapBitsSetType(uintptr(x), size, dataSize, typ)
        if dataSize > typ.Size_ {
            if typ.PtrBytes != 0 {
                scanSize = dataSize - typ.Size_ + typ.PtrBytes
            }
        } else {
            scanSize = typ.PtrBytes
        }
        c.scanAlloc += scanSize
    }
    
    publicationBarrier()
    span.freeIndexForScan = span.freeindex
    
    if gcphase != _GCoff {
        // GC期间新分配的对象，会被直接置黑
        gcmarknewobject(span, uintptr(x), size)
    }
    
    if raceenabled {
        racemalloc(x, size)
    }
    
    if msanenabled {
        msanmalloc(x, size)
    }
    
    if asanenabled {
        rzBeg := unsafe.Add(x, userSize)
        asanpoison(rzBeg, size-userSize)
        asanunpoison(x, userSize)
    }
    
    if rate := MemProfileRate; rate > 0 {
        // Note cache c only valid while m acquired; see #47302
        if rate != 1 && size < c.nextSample {
            c.nextSample -= size
        } else {
            profilealloc(mp, x, size)
        }
    }
    mp.mallocing = 0
    releasem(mp)
    
    if delayedZeroing {
        if !noscan {
            throw("delayed zeroing on data that may contain pointers")
        }
        memclrNoHeapPointersChunked(size, x) // This is a possible preemption point: see #47302
    }
    
    if debug.malloc {
        if debug.allocfreetrace != 0 {
            tracealloc(x, size, typ)
        }
        
        if inittrace.active && inittrace.id == getg().goid {
            inittrace.bytes += uint64(size)
        }
    }
    
    if assistG != nil {
        assistG.gcAssistBytes -= int64(size - dataSize)
    }
    
    if shouldhelpgc {
        if t := (gcTrigger{kind: gcTriggerHeap}); t.test() {
            gcStart(t)
        }
    }
    
    if raceenabled && noscan && dataSize < maxTinySize {
        x = add(x, size-dataSize)
    }
    
    return x
}
~~~

## 3. 微对象内存分配

微对象判断条件：noscan && size < maxTinySize

1. noscan表示不包含指针
2. maxTinySize == 16b , 即分配的内存小于16b

### 3.1 线程本地缓存（p.mcache）

1. mcache 是每个 P 独有的缓存，因此交互无锁 
2. mcache 将每种 spanClass 等级的 mspan 各缓存了一个，总数为 2（nocan 维度） * 68（大小维度）= 136
3. mcache 中还有一个为对象分配器 tiny allocator，用于处理小于 16B 对象的内存分配，在 3.3 小节中详细展开.

从mcache结构能看出，tiny用于微对象的内存分配，alloc用于小对象的内存分配

src/runtime/mcache.go:19

~~~go
type mcache struct {
    _ sys.NotInHeap
    
    // 下面的成员在每次 malloc 时都会被访问
    // 因此将它们放到一起来利用缓存的局部性原理
    nextSample uintptr // 分配这么多字节后触发堆样本
    scanAlloc  uintptr // 分配的可扫描堆的字节数
    
    // 没有指针的微小对象的分配器缓存。
    // tiny 指向当前 tiny 块的起始位置，或当没有 tiny 块时候为 nil
    // tiny 是一个堆指针。由于 mcache 在非 GC 内存中，我们通过在
    // mark termination 期间在 releaseAll 中清除它来处理它。
    tiny       uintptr
    tinyoffset uintptr
    // tinyAllocs是拥有此mcache的P执行的微小分配的数量。
    tinyAllocs uintptr
    
    // The rest is not accessed on every malloc.
    // mcache 中缓存的 mspan，每种 spanClass 各一个
    alloc [numSpanClasses]*mspan
    
    stackcache [_NumStackOrders]stackfreelist
    
    flushGen atomic.Uint32
}
~~~

### 3.2 分配步骤

每个 P 独有的 mache 会有个微对象分配器，基于 offset 线性移动的方式对微对象进行分配，每 16B 成块，对象依据其大小，会向上取整为 2 的整数次幂进行空间补齐，然后进入分配流程.

1. 向上取整为 2 的整数次幂进行空间补齐
2. 当前 tiny 内存块空间还够用，则直接分配并返回
3. 当前 tiny 内存块空间不够，分配一个新的 tiny 内存块,分配新内存的顺序：p.mcache->mCentral->mheap->向操作系统申请

~~~go
noscan := typ == nil || typ.ptrdata == 0
// ...
    if noscan && size < maxTinySize {
        off := c.tinyoffset
        // 向上取整为 2 的整数次幂进行空间补齐
        if size&7 == 0 {
            off = alignUp(off, 8)
        } else if goarch.PtrSize == 4 && size == 12 {
            off = alignUp(off, 8)
        } else if size&3 == 0 {
            off = alignUp(off, 4)
        } else if size&1 == 0 {
            off = alignUp(off, 2)
        }
        // 当前 tiny 内存块空间还够用，则直接分配并返回
        if off+size <= maxTinySize && c.tiny != 0 {
            x = unsafe.Pointer(c.tiny + off)
            c.tinyoffset = off + size
            c.tinyAllocs++
            mp.mallocing = 0
            releasem(mp)
            return x
        }
        // 当前 tiny 内存块空间不够，分配一个新的 tiny 内存块
        span = c.alloc[tinySpanClass]
        v := nextFreeFast(span)
        if v == 0 {
            v, span, shouldhelpgc = c.nextFree(tinySpanClass)
        }
        x = unsafe.Pointer(v)
        (*[2]uint64)(x)[0] = 0
        (*[2]uint64)(x)[1] = 0
        // 看看我们是否需要根据剩余可用空间量替换现有的小块
        if !raceenabled && (size < c.tinyoffset || c.tiny == 0) {
            c.tiny = uintptr(x)
            c.tinyoffset = size
        }
        size = maxTinySize
    }
//...
~~~

### 3.3 申请新的内存

当tiny不足以分配时，会尝试以这个顺序 **p.mcache->mCentral->mheap->向操作系统申请** 一次申请内存

#### 3.3.1 尝试从p.mcache分配内存

nextFreeFast函数

src/runtime/malloc.go:888

~~~go
func nextFreeFast(s *mspan) gclinkptr {
    // 返回s.allocCache（bitmap）末尾零位数
    theBit := sys.TrailingZeros64(s.allocCache) // Is there a free object in the allocCache?
    // 超过了说明没可用的了
	if theBit < 64 {
        result := s.freeindex + uintptr(theBit)
        // 超过了表明无可用的
        if result < s.nelems {
            // 可能有可用的
            freeidx := result + 1
            if freeidx%64 == 0 && freeidx != s.nelems {
                // 不是最后一个，且整除64，返回0
                return 0
            }
            // 真的有可用的
            // 更新一下
            s.allocCache >>= uint(theBit + 1)
            s.freeindex = freeidx
            s.allocCount++
            return gclinkptr(result*s.elemsize + s.base())
        }
    }
    return 0
}
~~~

#### 3.3.2 尝试从mCentral分配内存

当线程本地缓存的对应mspan中不足以分内新的内存时，尝试从mCentral->mheap->向操作系统申请分配内存

nextFree函数

src/runtime/malloc.go:915

~~~go
func (c *mcache) nextFree(spc spanClass) (v gclinkptr, s *mspan, shouldhelpgc bool) {
    s = c.alloc[spc]
    shouldhelpgc = false
    // 从 mcache 的 span 中获取 object 空位的偏移量
    freeIndex := s.nextFreeIndex()
    if freeIndex == s.nelems {
        // The span is full.
        if uintptr(s.allocCount) != s.nelems {
            println("runtime: s.allocCount=", s.allocCount, "s.nelems=", s.nelems)
            throw("s.allocCount != s.nelems && freeIndex == s.nelems")
        }
        // 倘若 mcache 中 span 已经没有空位，则调用 refill 方法从 mcentral 或者 mheap 中获取新的 span
		// shouldhelpgc置为true，表示需要触发gc
        c.refill(spc)
        shouldhelpgc = true
        // 再次从替换后的 span 中获取 object 空位的偏移量
        s = c.alloc[spc]
        freeIndex = s.nextFreeIndex()
    }
    
    if freeIndex >= s.nelems {
        throw("freeIndex is not valid")
    }
    
    v = gclinkptr(freeIndex*s.elemsize + s.base())
    s.allocCount++
    if uintptr(s.allocCount) > s.nelems {
        println("s.allocCount=", s.allocCount, "s.nelems=", s.nelems)
        throw("s.allocCount > s.nelems")
    }
    return
}
~~~

refill函数

将已满的span释放到mcentral中,从mcentral（或者mheap、操作系统中）获取一个新的span放入mcache.alloc中

src/runtime/mcache.go:147

~~~go
func (c *mcache) refill(spc spanClass) {
    // 获取mcache中spc对应的mspan
    s := c.alloc[spc]
    
    if uintptr(s.allocCount) != s.nelems {
        throw("refill of span with free space remaining")
    }
    // 将s释放的到 mcentral
    if s != &emptymspan {
        // Mark this span as no longer cached.
        if s.sweepgen != mheap_.sweepgen+3 {
            throw("bad sweepgen in refill")
        }
        mheap_.central[spc].mcentral.uncacheSpan(s)
        
        // Count up how many slots were used and record it.
        stats := memstats.heapStats.acquire()
        slotsUsed := int64(s.allocCount) - int64(s.allocCountBeforeCache)
        atomic.Xadd64(&stats.smallAllocCount[spc.sizeclass()], slotsUsed)
        
        // Flush tinyAllocs.
        if spc == tinySpanClass {
            atomic.Xadd64(&stats.tinyAllocCount, int64(c.tinyAllocs))
            c.tinyAllocs = 0
        }
        memstats.heapStats.release()
        
        // Count the allocs in inconsistent, internal stats.
        bytesAllocated := slotsUsed * int64(s.elemsize)
        gcController.totalAlloc.Add(bytesAllocated)
        
        // Clear the second allocCount just to be safe.
        s.allocCountBeforeCache = 0
    }
	
    // 从mcentral获取一个新的span
    s = mheap_.central[spc].mcentral.cacheSpan()
    if s == nil {
        throw("out of memory")
    }
    
    if uintptr(s.allocCount) == s.nelems {
        throw("span has no free space")
    }
    
    // Indicate that this span is cached and prevent asynchronous
    // sweeping in the next sweep phase.
    s.sweepgen = mheap_.sweepgen + 3
    
    // Store the current alloc count for accounting later.
    s.allocCountBeforeCache = s.allocCount
    
    usedBytes := uintptr(s.allocCount) * s.elemsize
    gcController.update(int64(s.npages*pageSize)-int64(usedBytes), int64(c.scanAlloc))
    c.scanAlloc = 0
    // 将新获取到的span放入mcache中
    c.alloc[spc] = s
}
~~~

cacheSpan函数

1. 尝试从mcentral获取一个新的span
2. mcentral未能获取到,尝试从mheap中获取新的span

src/runtime/mcentral.go:81

~~~go
func (c *mcentral) cacheSpan() *mspan {
    spanBytes := uintptr(class_to_allocnpages[c.spanclass.sizeclass()]) * _PageSize
    deductSweepCredit(spanBytes, 0)
    
    traceDone := false
    if traceEnabled() {
        traceGCSweepStart()
    }
    
    spanBudget := 100
    
    var s *mspan
    var sl sweepLocker
    
    // Try partial swept spans first.
    sg := mheap_.sweepgen
    if s = c.partialSwept(sg).pop(); s != nil {
        goto havespan
    }
    
    sl = sweep.active.begin()
    if sl.valid {
        // Now try partial unswept spans.
        for ; spanBudget >= 0; spanBudget-- {
            s = c.partialUnswept(sg).pop()
            if s == nil {
                break
            }
            if s, ok := sl.tryAcquire(s); ok {
                // We got ownership of the span, so let's sweep it and use it.
                s.sweep(true)
                sweep.active.end(sl)
                goto havespan
            }
        }
        // 通过 sweepLock，加锁尝试从 mcentral 的非空链表 full 中获取 mspan
        for ; spanBudget >= 0; spanBudget-- {
            s = c.fullUnswept(sg).pop()
            if s == nil {
                break
            }
            if s, ok := sl.tryAcquire(s); ok {
                // We got ownership of the span, so let's sweep it.
                s.sweep(true)
                // Check if there's any free space.
                freeIndex := s.nextFreeIndex()
                if freeIndex != s.nelems {
                    s.freeindex = freeIndex
                    sweep.active.end(sl)
                    goto havespan
                }
                // Add it to the swept list, because sweeping didn't give us any free space.
                c.fullSwept(sg).push(s.mspan)
            }
            // See comment for partial unswept spans.
        }
        sweep.active.end(sl)
    }
    if traceEnabled() {
        traceGCSweepDone()
        traceDone = true
    }
    
    // 我们没能从mcentral得到一个span，所以从mheap得到一个。
    s = c.grow()
    if s == nil {
        return nil
    }

// 执行到此处时，s 已经指向一个存在 object 空位的 mspan 了
havespan:
    if traceEnabled() && !traceDone {
        traceGCSweepDone()
    }
    n := int(s.nelems) - int(s.allocCount)
    if n == 0 || s.freeindex == s.nelems || uintptr(s.allocCount) == s.nelems {
        throw("span has no free objects")
    }
    freeByteBase := s.freeindex &^ (64 - 1)
    whichByte := freeByteBase / 8
    // Init alloc bits cache.
    s.refillAllocCache(whichByte)
    
    // Adjust the allocCache so that s.freeindex corresponds to the low bit in
    // s.allocCache.
    s.allocCache >>= s.freeindex % 64
    
    return s
}
~~~

grow函数

src/runtime/mcentral.go:242

~~~go
func (c *mcentral) grow() *mspan {
    // 需要的pages和size
    npages := uintptr(class_to_allocnpages[c.spanclass.sizeclass()])
    size := uintptr(class_to_size[c.spanclass.sizeclass()])
    // 从mheap_分配一个span
    s := mheap_.alloc(npages, c.spanclass)
    if s == nil {
        return nil
    }
    
    // Use division by multiplication and shifts to quickly compute:
    // n := (npages << _PageShift) / size
    n := s.divideByElemSize(npages << _PageShift)
    s.limit = s.base() + size*n
    s.initHeapBits(false)
    return s
}
~~~

alloc函数

src/runtime/mheap.go:952

~~~go
func (h *mheap) alloc(npages uintptr, spanclass spanClass) *mspan {
    var s *mspan
    // 切换到g0栈
    systemstack(func() {
        if !isSweepDone() {
            h.reclaim(npages)
        }
        // 分配需要的span
        s = h.allocSpan(npages, spanAllocHeap, spanclass)
    })
    return s
}
~~~

allocSpan函数

尝试从mheap中分配span（通过基数树索引快速寻找满足条件的连续空闲页）,若mheap中不足以分配时，尝试从操作系统中申请，若操作系统中也没有足够的内存，则抛出错误

src/runtime/mheap.go:1175

~~~go
    func (h *mheap) allocSpan(npages uintptr, typ spanAllocType, spanclass spanClass) (s *mspan) {
    // g0
    gp := getg()
    base, scav := uintptr(0), uintptr(0)
    growth := uintptr(0)
    
    // 是否需要物理对齐
    needPhysPageAlign := physPageAlignedStacks && typ == spanAllocStack && pageSize < physPageSize
    // g0.m.p
    pp := gp.m.p.ptr()
    // 不需要物理对齐，pp不为空（g0.m绑定了p）,需要的页数小于16时
    // 从每个 P 的页缓存 pageCache 中获取空闲页组装 mspan
    if !needPhysPageAlign && pp != nil && npages < pageCachePages/4 {
        c := &pp.pcache
        
        // pp.pcache为空时，先填充
        if c.empty() {
            lock(&h.lock)
            *c = h.pages.allocToCache()
            unlock(&h.lock)
        }
        
        // 尝试从pp.pcache中获取s
        base, scav = c.alloc(npages)
        if base != 0 {
            s = h.tryAllocMSpan()
            if s != nil {
                goto HaveSpan
            }
        }
    }
    
    lock(&h.lock)
    // 需要物理对齐时
    if needPhysPageAlign {
        extraPages := physPageSize / pageSize
        // 通过页分配器找到满足条件的连续空闲页
        base, _ = h.pages.find(npages + extraPages)
        if base == 0 {
            var ok bool
            // 未找到，向操作系统申请
            growth, ok = h.grow(npages + extraPages)
            if !ok {
                unlock(&h.lock)
                return nil
            }
            base, _ = h.pages.find(npages + extraPages)
            // 操作系统也没有足够的内存
            if base == 0 {
                throw("grew heap, but no adequate free space found")
            }
        }
        base = alignUp(base, physPageSize)
        scav = h.pages.allocRange(base, npages)
    }
    
    if base == 0 {
        // Try to acquire a base address.
        // 通过基数树索引快速寻找满足条件的连续空闲页
        base, scav = h.pages.alloc(npages)
        if base == 0 {
            var ok bool
            // 未找到，向操作系统申请
            growth, ok = h.grow(npages)
            if !ok {
                unlock(&h.lock)
                return nil
            }
            base, scav = h.pages.alloc(npages)
            if base == 0 {
                throw("grew heap, but no adequate free space found")
            }
        }
    }
    if s == nil {
        // 从pp.mspancache获取span
        s = h.allocMSpanLocked()
    }
    unlock(&h.lock)
    
HaveSpan:
    bytesToScavenge := uintptr(0)
    forceScavenge := false
    if limit := gcController.memoryLimit.Load(); !gcCPULimiter.limiting() {
        inuse := gcController.mappedReady.Load()
        if uint64(scav)+inuse > uint64(limit) {
            bytesToScavenge = uintptr(uint64(scav) + inuse - uint64(limit))
            forceScavenge = true
        }
    }
    if goal := scavenge.gcPercentGoal.Load(); goal != ^uint64(0) && growth > 0 {
        if retained := heapRetained(); retained+uint64(growth) > goal {
            todo := growth
            if overage := uintptr(retained + uint64(growth) - goal); todo > overage {
                todo = overage
            }
            if todo > bytesToScavenge {
                bytesToScavenge = todo
            }
        }
    }
    
    var now int64
    if pp != nil && bytesToScavenge > 0 {
        start := nanotime()
        track := pp.limiterEvent.start(limiterEventScavengeAssist, start)
        
        // Scavenge, but back out if the limiter turns on.
        released := h.pages.scavenge(bytesToScavenge, func() bool {
            return gcCPULimiter.limiting()
        }, forceScavenge)
        
        mheap_.pages.scav.releasedEager.Add(released)
        
        // Finish up accounting.
        now = nanotime()
        if track {
            pp.limiterEvent.stop(limiterEventScavengeAssist, now)
        }
        scavenge.assistTime.Add(now - start)
    }
    
    // Initialize the span.
    h.initSpan(s, typ, spanclass, base, npages)
    
    // Commit and account for any scavenged memory that the span now owns.
    nbytes := npages * pageSize
    if scav != 0 {
        sysUsed(unsafe.Pointer(base), nbytes, scav)
        gcController.heapReleased.add(-int64(scav))
    }
    // Update stats.
    gcController.heapFree.add(-int64(nbytes - scav))
    if typ == spanAllocHeap {
        gcController.heapInUse.add(int64(nbytes))
    }
    // Update consistent stats.
    stats := memstats.heapStats.acquire()
    atomic.Xaddint64(&stats.committed, int64(scav))
    atomic.Xaddint64(&stats.released, -int64(scav))
    switch typ {
    case spanAllocHeap:
        atomic.Xaddint64(&stats.inHeap, int64(nbytes))
    case spanAllocStack:
        atomic.Xaddint64(&stats.inStacks, int64(nbytes))
    case spanAllocPtrScalarBits:
        atomic.Xaddint64(&stats.inPtrScalarBits, int64(nbytes))
    case spanAllocWorkBuf:
        atomic.Xaddint64(&stats.inWorkBufs, int64(nbytes))
    }
    memstats.heapStats.release()
    
    pageTraceAlloc(pp, now, base, npages)
    return s
    }
~~~

## 4. 小对象内存分配

所需内存小于等于32kb，或则小于16b但是包含指针时，分配小对象内存

小对象内存分配与为对象分配类似（就不贴相关代码了），无需tiny 内存的相关判断：

1. 尝试从mcache中分配内存
2. mcache不足时，尝试从mcentral中分配
3. mcentral不足时，尝试从mheap中获取内存分配
4. 最后任然不足时，尝试从操作系统中申请内存
5. 若操作系统中内存不足时，表明已经没有可用内存，直接抛出异常

## 5. 大对象内存分配

所需内存大于32kb时，直接从堆（mheap）中分配内存

allocLarge函数

调用mheap_.alloc方法从mheap（或者操作系统）中获取span

usr/runtime/mcache.go:219

~~~go
func (c *mcache) allocLarge(size uintptr, noscan bool) *mspan {
    if size+_PageSize < size {
        throw("out of memory")
    }
    npages := size >> _PageShift
    if size&_PageMask != 0 {
        npages++
    }
    
    deductSweepCredit(npages*_PageSize, npages)
    
    spc := makeSpanClass(0, noscan)
    s := mheap_.alloc(npages, spc)
    if s == nil {
        throw("out of memory")
    }
    
    // Count the alloc in consistent, external stats.
    stats := memstats.heapStats.acquire()
    atomic.Xadd64(&stats.largeAlloc, int64(npages*pageSize))
    atomic.Xadd64(&stats.largeAllocCount, 1)
    memstats.heapStats.release()
    
    // Count the alloc in inconsistent, internal stats.
    gcController.totalAlloc.Add(int64(npages * pageSize))
    
    // Update heapLive.
    gcController.update(int64(s.npages*pageSize), 0)
    
    // Put the large span in the mcentral swept list so that it's
    // visible to the background sweeper.
    mheap_.central[spc].mcentral.fullSwept(mheap_.sweepgen).push(s)
    s.limit = s.base() + size
    s.initHeapBits(false)
    return s
}
~~~