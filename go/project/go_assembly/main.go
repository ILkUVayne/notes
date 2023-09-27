package main

import (
	"fmt"
	"go_assembly/pkg"
)

func main() {

	println(pkg.Id)
	println(pkg.Name)
	println(pkg.Name2)
	println(pkg.HelloWorld)
	// pkg.HelloWorld1 => [72 101 108 108 111 32 119 111 114 108 100 33]
	fmt.Printf("%v\n", string(pkg.HelloWorld1))
	pkg.Hello()
	println(pkg.Swap(1, 2))
}
