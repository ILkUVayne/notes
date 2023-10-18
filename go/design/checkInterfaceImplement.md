# 编译期检查某个类型是否实现了特定接口

## 1. 准备一个接口

~~~go
type Person interface {
    getName() string
}
~~~

## 2. Student结构体

~~~go
type Student struct {
    name string
    age  int
}
~~~

## 3. 检查Student结构体是否实现Person

因为在go语言中，并不需要显式地声明实现了哪一个接口，只需要直接实现该接口对应的所有方法即可。所以并不必能直接看出来是否实现了该接口

~~~go
// 将空值 nil 转换为 *Student 类型，再转换为 Person 接口，
// 如果转换失败，说明 Student 并没有实现 Person 接口的所有方法。
var _ Person = (*Student)(nil)
~~~