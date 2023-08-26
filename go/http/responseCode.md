# golang设置http响应码遇到的问题

> 当前golang版本1.21

## 1.问题描述
调用http.ResponseWriter.WriteHeader()设置httpCode,由于调用时机的不同，会导致意料之外的错误

## 2.代码示例

### 2.1 示例一
~~~go
package main

import (
	"log"
	"net/http"
)
func main() {
	http.HandleFunc("/ping1", handler1)
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}

func handler1(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusFailedDependency)
	_, err := w.Write([]byte("request /ping1"))
	if err != nil {
		return
	}
}
~~~
示例一代码为正常情况，Content-Type header头以及httpCode http.StatusFailedDependency能够正常响应

### 2.2 示例二
~~~go
package main

import (
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/ping1", handler2)
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}

func handler2(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusFailedDependency)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, err := w.Write([]byte("request /ping2"))
	if err != nil {
		return
	}
}
~~~
示例二的结果就出现了异常，httpCode http.StatusFailedDependency能正常响应，但是后面设置的Content-Type并没有生效

### 2.3 示例三
~~~go
package main

import (
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/ping1", handler3)
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}

func handler3(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, err := w.Write([]byte("request /ping3"))
	if err != nil {
		return
	}
	w.WriteHeader(http.StatusFailedDependency)
}
~~~
示例三的httpCode以及Content-Type都能够整个设置并响应，但是会有错误日志**http: superfluous response.WriteHeader call from main.handler3**

## 3. 造成原因
主要在于w.WriteHeader方法的实现方式，代码位于net/http/server.go中
### 3.1 WriteHeader方法
~~~go
func (w *response) WriteHeader(code int) {
    //...
    if w.wroteHeader {
        caller := relevantCaller()
        w.conn.server.logf("http: superfluous response.WriteHeader call from %s (%s:%d)", caller.Function, path.Base(caller.File), caller.Line)
        return
    }
    //...

    w.wroteHeader = true
    w.status = code

    if w.calledHeader && w.cw.header == nil {
        w.cw.header = w.handlerHeader.Clone()
    }

    //...
}
~~~

### 3.2 Write方法
~~~go
func (w *response) write(lenData int, dataB []byte, dataS string) (n int, err error) {
    //...
    if !w.wroteHeader {
        w.WriteHeader(StatusOK)
    }
    //...

}
~~~

### 3.3 Header方法
~~~go
func (w *response) Header() Header {
	if w.cw.header == nil && w.wroteHeader && !w.cw.wroteHeader {
		// Accessing the header between logically writing it
		// and physically writing it means we need to allocate
		// a clone to snapshot the logically written state.
		w.cw.header = w.handlerHeader.Clone()
	}
	w.calledHeader = true
	return w.handlerHeader
}
~~~

## 4 结论
### 4.1 报错问题
从源码中可以看出，write和WriteHeader方法都会这是w.wroteHeader = true（write方法调用WriteHeader设置），如果调用write后再调用WriteHeader就会抛出错误了，需要注意的是例如fmt.Fprintf()、xml.NewEncoder(w).Encode()等方法，也会调用write方法，故要设置httpCode也需要再这些方法之前调用

### 4.2 设置header头失效问题
从WriteHeader方法中可以看出，header实际的响应头存放在w.cw.header，而非w.handlerHeader
#### 4.2.1 先设置header，再设置httpCode
这种情况下，(w.cw.header == nil && w.wroteHeader && !w.cw.wroteHeader)恒等于false，故实际设置的是w.handlerHeader中的值，然会再调用WriteHeader方法时，赋值给w.cw.header，所以能够正常使用
#### 4.2.2 先设置httpCode，再设置header
这种情况下，(w.cw.header == nil && w.wroteHeader && !w.cw.wroteHeader)的值有两种情况，(w.cw.header == nil)既实际响应header未设置时，进行赋值操作，若不等于nil，则直接返回w.handlerHeader，之后的操作并不会影响到w.cw.header，所以调用WriteHeader方法后的设置操作没有生效