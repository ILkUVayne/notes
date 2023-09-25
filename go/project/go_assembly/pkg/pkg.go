package pkg

import _ "unsafe"

//go:linkname printnl runtime.printnl
func printnl()

//go:linkname printstring runtime.printstring
func printstring(s string)

var Id int
var Name string
var Name2 string

// helloWorld pkg_amd64.s ·helloWorld
var helloWorld = "你好, 世界"

func Hello()
