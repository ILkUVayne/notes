# go汇编问题汇总

## 1.unexpected EOF

### 报错信息

~~~bash
# src_test/pkg
pkg/pkg_amd64.s:10: unexpected EOF
asm: assembly of pkg/pkg_amd64.s failed
~~~

### 解决方法

go 汇编代码中最后一行缺少换行，加上换行即可

## 2.missing Go type information for global symbol xxx

~~~bash
# command-line-arguments
runtime.gcdata: missing Go type information for global symbol src_test/pkg.Id: size 8
~~~

### 文件目录结构

~~~bash
$ tree
.
├── go.mod
├── go_assembly.md
├── main.go
├── pkg
│   ├── pkg.go
│   └── pkg_amd64.s
~~~

错误提示汇编中定义的Id符号没有类型信息。其实Go汇编语言中定义的数据并没有所谓的类型，每个符号只不过是对应一块内存而已，因此Id符号也是没有类型的。但是Go语言是再带垃圾回收器的语言，而Go汇编语言是工作在自动垃圾回收体系框架内的。当Go语言的垃圾回收器在扫描到Id变量的时候，无法知晓该变量内部是否包含指针，因此就出现了这种错误。错误的根本原因并不是Id没有类型，而是Id变量没有标注是否会含有指针信息。

通过给Id变量增加一个NOPTR标志，表示其中不会包含指针数据可以修复该错误：

~~~cgo
#include "textflag.h"

DATA ·Id+0(SB)/1,$0x37
DATA ·Id+1(SB)/1,$0x25
DATA ·Id+2(SB)/1,$0x00
DATA ·Id+3(SB)/1,$0x00
DATA ·Id+4(SB)/1,$0x00
DATA ·Id+5(SB)/1,$0x00
DATA ·Id+6(SB)/1,$0x00
DATA ·Id+7(SB)/1,$0x00
GLOBL ·Id(SB),NOPTR,$8
~~~

## 3.relocation target runtime.printstring not defined for ABI0 (but is defined for ABIInternal)

报错信息

~~~bash
# command-line-arguments
go_assembly/pkg.Hello: relocation target runtime.printstring not defined for ABI0 (but is defined for ABIInternal)
go_assembly/pkg.Hello: relocation target runtime.printnl not defined for ABI0 (but is defined for ABIInternal)
~~~

解决方法

使用go:linkname注释分别引入 runtime.printstring 和 runtime.printnl 方法

> //go:linkname localname [importpath.name]
> 
> go:linkname 注释作用：引入别的包下没有导出的方法或者变量，需要import _ "unsafe"

~~~go
package pkg

import _ "unsafe"

//go:linkname printnl runtime.printnl
func printnl()

//go:linkname printstring runtime.printstring
func printstring(s string)
~~~