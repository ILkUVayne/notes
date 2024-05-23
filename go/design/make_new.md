# make 和 new 的区别

> go 1.21

Go语言中new和make都是用来内存分配的原语

## 区别一：new可以为所有类型分配内存，make只能用于slice，map，和channel

**new**

~~~go
func main() {
    i := new(int)
    *i = 1
    println(*i) // 1
    println(i)  // 0xc000042708
    channel := new(chan string)
    println(channel) // 0xc000042710
    m := new(map[string]int)
    println(m) // 0xc000042708
    s := new([]int)
    println(s) // 0xc000042718
}
~~~

**make**

~~~go
// slice，map，和channel
func main() {
    channel := make(chan string)
    println(channel) // 0xc000070060
    m := make(map[string]int)
    println(m) // 0xc000042630
    s := make([]int, 10)
    println(s) // [10/10]0xc0000425d8
}

// 其他类型
func main() {
    i := make(int) // invalid argument: cannot make int; type must be slice, map, or channel
    *i = 1
    println(*i)
    println(i)
}
~~~

## 区别二：new返回类型的指针，make返回类型

**new**

~~~go
func main() {
    channel := new(chan string)
    fmt.Printf("%T\n", channel) // *chan string
    m := new(map[string]int)
    fmt.Printf("%T\n", m) // *map[string]int
    s := new([]int)
    fmt.Printf("%T\n", s) // *[]int
    i := new(int)
    fmt.Printf("%T\n", i) // *int
    var mm map[string]int
    var ii int
    fmt.Printf("%T\n", ii) // int
    fmt.Printf("%T\n", mm) // map[string]int
}
~~~

**make**

~~~go
func main() {
    channel := make(chan string)
    fmt.Printf("%T\n", channel) // chan string
    m := make(map[string]int)
    fmt.Printf("%T\n", m) // map[string]int
    s := make([]int, 10)
    fmt.Printf("%T\n", s) // []int
}
~~~

## 区别三：make会分配内存并初始化，new只分配内存

**new**

~~~go
// int
i := new(int)
fmt.Printf("i: %p %v\n", &i, *i) // i: 0xc000046020 0
*i = 1
fmt.Printf("i: %p %v\n", &i, *i) // i: 0xc000046020 1

// [5]int
i := new([5]int)
fmt.Printf("i: %p %v\n", &i, *i) // i: 0xc000046020 [0 0 0 0 0]
(*i)[1] = 1
fmt.Printf("i: %p %v\n", &i, *i) // i: 0xc000046020 [0 1 0 0 0]

// []int
i := new([]int)
fmt.Printf("i: %p %v\n", &i, *i) // i: 0xc000046020 []
(*i)[1] = 1 // panic: runtime error: index out of range [1] with length 0
fmt.Printf("i: %p %v\n", &i, *i)

// map[int]string
i := new(map[int]string)
fmt.Printf("i: %p %v\n", &i, *i) // i: 0xc00009c018 map[]
(*i)[1] = "hello world!" // panic: assignment to entry in nil map
fmt.Printf("i: %p %v\n", &i, *i)

// chan int
i := new(chan int)
fmt.Printf("i: %p %v\n", &i, *i) // i: 0xc00009c018 <nil>
//i <- 1 // invalid operation: cannot send to non-channel i (variable of type *chan int)
~~~

**make**

~~~go
// []int
i := make([]int, 10)
fmt.Printf("i: %p %v\n", &i, i) // i: 0xc000010018 [0 0 0 0 0 0 0 0 0 0]
i[1] = 1                        
fmt.Printf("i: %p %v\n", &i, i) // i: 0xc000010018 [0 1 0 0 0 0 0 0 0 0]

// map[]
i := make(map[int]string)
fmt.Printf("i: %p %v\n", &i, i) // i: 0xc000046020 map[]
i[1] = "hello world!"
fmt.Printf("i: %p %v\n", &i, i) // i: 0xc000046020 map[1:hello world!]

// chan int
i := make(chan int)
fmt.Printf("i: %p %v\n", &i, i) // i: 0xc00009c018 0xc000094060
go func() {
    i <- 1
}()
fmt.Printf("i: %p %v\n", &i, <-i) // i: 0xc00009c018 1
~~~

## 总结

1. new和make都可以分配内存，但是new可以为所有类型分配内存，但是make只能用于chan、map、slice
2. new分配内存后返回变量的指针，make返回类型本身
3. make会分配内存并初始化。new只分配内存，将内存清零，并没有初始化内存。当new的类型为非引用类型（chan、map、slice）时，可以直接对变量进行赋值操作，当new的类型为引用类型时，则还需要调用make进行初始化