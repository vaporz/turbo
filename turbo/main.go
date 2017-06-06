package main

import (
	"github.com/vaporz/turbo/turbo/cmd"
	"github.com/vaporz/turbo"
	"os"
)

func init() {
	turbo.SetOutput(os.Stdout)
}

func main() {
	cmd.Execute()
}
