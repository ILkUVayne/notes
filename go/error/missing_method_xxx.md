# panic: interface conversion: main.Dog is not main.Eater: missing method eat

> 具体复现代码可以查看missing_method_xxx.go

## 问题问题分析

~~~go
type Eater interface {
    eat(s string)
}

type Dog struct{}

func (d *Dog) eat(s string) {
    fmt.Printf("eat %s", s)
}

var animalMaps = map[string]interface{}{
    "dog": Dog{},
}

func doEat(animal interface{}, s string) {
    animal.(Eater).eat(s)
}
~~~

Dog 类（结构体）实现了接口Eater，这个方法是指针接受者，而animalMaps中存的是值类型（不是指针），所以在doEat进行断言的时候报错了

## 解决方法

### 方法1

将animalMaps中的值改为指针即可

~~~go
var animalMaps = map[string]interface{}{
    "dog": &Dog{},
}
~~~

### 方法2

若是不介意Dog类的eat方法是值接收还是指针接受，则修改eat方法为值接受，此时animalMaps中存指针还是值都能够正常断言并执行eat方法

~~~go
func (d Dog) eat(s string) {
    fmt.Printf("eat %s", s)
}
~~~