package component

import (
	"log"
	"strings"
	"bufio"
	"io"
	"os"
)

var UrlServiceMap [][3]string

func LoadServiceConfig() {
	//currentDir, err := filepath.Abs(filepath.Dir("."))
	//if err != nil {
	//	log.Fatal("load config fail")
	//}
	//log.Println(currentDir)
	//TODO get filepath
	f, err := os.Open("/Users/xiaozhang/goworkspace/src/zx/demo/inventoryservice/http/service.config")
	if err != nil {
		log.Println(err)
	}
	buf := bufio.NewReader(f)
	for {
		line, err := buf.ReadString('\n')
		line = strings.TrimSpace(line)
		appendUrlServiceMap(line)
		if err != nil {
			if err == io.EOF {
				return
			}
			log.Println(err)
			break
		}
	}
}

func appendUrlServiceMap(line string) {
	pair := strings.Split(line, "=")
	urlPair := strings.Split(strings.TrimSpace(pair[0]), " ")
	methodName := strings.TrimSpace(pair[1])
	UrlServiceMap = append(UrlServiceMap, [3]string{urlPair[0], urlPair[1], methodName})
}
