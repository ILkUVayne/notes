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