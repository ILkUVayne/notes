# 报错 “# command-line-arguments” 问题

## 报错信息
main包中有两个或者两个以上的.go文件,点击运行main.go时，就会报如下错误.
~~~
# command-line-arguments
./main.go:14:26: undefined: base
./main.go:16:32: undefined: v1
./main.go:24:32: undefined: v2
~~~

## 造成原因
main包中的文件不能相互调用，其他包可以，所以除main.go外，其他文件并未被一起编译.

## 解决方案

### 1.手动选中需要编译的文件，一起编译即可

### 2.把需要调用的方法移动到main.go中，或者放到其他非main包中调用