package decrypt

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"os"
	"strconv"
)

// WIDTH 图片宽度
const WIDTH = 256

func Encrypt(filepath string) {
	f, _ := os.Open(filepath)

	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Fatal(err.Error())
		}
	}(f)

	// get byte array
	content, _ := io.ReadAll(f)

	// bit array
	bits := make([]byte, 0)
	// byte array => bit array
	for _, v := range content {
		binary := strconv.FormatInt(int64(v), 2)
		binary = binaryCompletion(binary)

		for _, bv := range binary {
			if string(bv) == "1" {
				bits = append(bits, 1)
				continue
			}
			bits = append(bits, 0)
		}
	}
	// 获取生成图片高度
	height := int(float64(len(bits))/float64(WIDTH)) + 1

	img := image.NewRGBA(image.Rect(0, 0, WIDTH, height))

	// 位图索引
	index := 0
	// 生成位图图片
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			if index >= len(bits) {
				img.Set(x, y, color.White)
				continue
			}
			if bits[index] == 1 {
				img.Set(x, y, color.Black)
			} else {
				img.Set(x, y, color.White)
			}
			index++
		}
	}
	// MD5
	str := string(content)
	md5Sum(&str)
	// 保存生成的像素点图片
	saveImg(img, filepath)
	// 已生成位图图片地址
	fmt.Printf("已生成位图图片地址：\n%s\n", dstEnPath(filepath))
}

func saveImg(img *image.RGBA, filepath string) {
	f, err := os.OpenFile(dstEnPath(filepath), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0600)
	defer func(fileOut *os.File) {
		err := fileOut.Close()
		if err != nil {
			log.Fatal(err.Error())
		}
	}(f)

	err = png.Encode(f, img)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func binaryCompletion(binary string) string {
	switch len(binary) {
	case 1:
		return "0000000" + binary
	case 2:
		return "000000" + binary
	case 3:
		return "00000" + binary
	case 4:
		return "0000" + binary
	case 5:
		return "000" + binary
	case 6:
		return "00" + binary
	case 7:
		return "0" + binary
	default:
		return binary
	}
}
