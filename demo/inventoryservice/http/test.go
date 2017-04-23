package main

import (
	"regexp"
	"strings"
	"fmt"
)

type Test struct {
	Abc string
}

func main() {
	var benchmarks = []string{
		"a",
		"snake",
		"A",
		"Snake",
		"SnakeTest",
		"SnakeID",
		"SnakeIDGoogle",
		"LinuxMOTD",
		"OMGWTFBBQ",
		"omg_wtf_bbq",
	}
	for _, v := range benchmarks {
		fmt.Println(ToSnakeCase(v))
	}
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func ToSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}
