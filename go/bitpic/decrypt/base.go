package decrypt

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"
	filepath2 "path/filepath"
	"strings"
)

func md5Sum(s *string) {
	h := md5.New()
	_, err := io.WriteString(h, *s)
	if err != nil {
		log.Fatal(err.Error())
	}
	sum := fmt.Sprintf("%x", h.Sum(nil))
	fmt.Printf("校验和:\n%s\n", sum)
}

func dstEnPath(filepath string) string {
	path, fileName := filepath2.Split(filepath)
	name := strings.Split(fileName, ".")

	return path + "result-encrypt-" + name[0] + ".png"
}

func dstDePath(filepath string) string {
	path, fileName := filepath2.Split(filepath)
	name := strings.Split(fileName, ".")

	return path + "result-decrypt-" + name[0] + ".txt"
}
