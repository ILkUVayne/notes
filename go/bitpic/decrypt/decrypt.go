package decrypt

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"math"
	"os"
	"unicode/utf8"
)

func Decrypt(filepath string) {

	//读取bit pic
	f, err := os.Open(filepath)

	if err != nil {
		log.Fatal(err.Error())
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Fatal(err.Error())
		}
	}(f)

	g, _, err := image.Decode(bufio.NewReader(f))
	if err != nil {
		log.Fatal(err.Error())
	}

	rect := g.Bounds()
	size := rect.Size()

	picWidth := size.X
	picHeight := size.Y

	// get bit array
	bin := make([]byte, 0)
	for y := 0; y < picHeight; y++ {
		for x := 0; x < picWidth; x++ {
			pixelItem := g.At(x, y)
			r0, _, _, _ := pixelItem.RGBA()
			if r0 == 65535 {
				bin = append(bin, 0)
			} else if r0 == 0 {
				bin = append(bin, 1)
			}
		}
	}

	// get byte array
	str := ""
	byteArr := make([]byte, 0)
	for i := 0; i < len(bin); i = i + 8 {
		// 如果后面连续4个字节都为0，则退出循环
		if bytes.Equal(bin[i:i+32], make([]byte, 32, 32)) {
			break
		}
		sum := 0
		// 计算每个字节的大小（值）
		for j, sub := i, 0; j < i+8; j, sub = j+1, sub+1 {
			if bin[j] == 0 {
				continue
			}
			sum += int(math.Pow(2, float64(7-sub)))
		}
		byteArr = append(byteArr, byte(sum))
	}

	// 获取utf8字符串
	for i := 0; i < len(byteArr); i++ {
		r, n := utf8.DecodeRune(byteArr[i:])
		i += n - 1
		str += string(r)
	}
	// 输出（cli）解析的utf8字符串
	fmt.Printf("解析出的utf8字符串为：\n%s\n", str)
	// 保存结果到本地txt中
	saveToFile(str, filepath)
	// MD5
	md5Sum(&str)
	// 已生解析文件地址
	fmt.Printf("已生成位图图片地址：\n%s\n", dstDePath(filepath))
}

func saveToFile(s, filepath string) {
	f, err := os.OpenFile(dstDePath(filepath), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0600)
	if err != nil {
		return
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {

		}
	}(f)
	_, err = f.WriteString(s)
}
