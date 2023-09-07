package main

import (
	"bitpic/decrypt"
	"flag"
)

const ENCRYPT = "en"
const DECRYPT = "de"

var mode = flag.String("m", DECRYPT, "mode for file deal en/de, en is encrypt,de is decrypt")
var filePath = flag.String("f", "", "filePath like ./usr/local/example.png")

func main() {
	flag.Parse()
	// 调用方式 go run main.go -m de -f ./hamlet.png
	switch *mode {
	case DECRYPT:
		// 解析位图图片
		decrypt.Decrypt(*filePath)
	case ENCRYPT:
		// 生成位图图片
		decrypt.Encrypt(*filePath)
	}
}
