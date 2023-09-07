# go options模式

> go version: 1.21

## 实现方式

### 1. 定义option方法和对于结构体

~~~
type option func(s *server)

type server struct {
	protocol string
	host     string
	port     int
}
~~~

### 2. 实现option方法

~~~
func withProtocol(protocol string) option {
	return func(s *server) {
		s.protocol = protocol
	}
}

//...

~~~

### 3. 实现New方法

~~~
func newServer(opts ...option) *server {
	s := new(server)
	for _, v := range opts {
		v(s)
	}
	return s
}
~~~

通过上述方式，以实现了option模式

## 调用方式

~~~
func main() {
	s := newServer(
		withProtocol("tcp"),
		withHost("127.0.0.1"),
		withPort(8181),
	)
	fmt.Printf("%s\n", s)
	s2 := newServer(
		withProtocol("tcp"),
		withHost("127.0.1.1"),
	)
	fmt.Printf("%s\n", s2)
}
~~~

通过调用代码，能发现，option模式能够更加灵活的new一个新的结构体（或者说对象），只需要with需要设置的参数即可，而无需每次newServer时都必须传入所有参数。