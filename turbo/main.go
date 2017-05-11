package main

import (
	"turbo/turbo/cmd"
)

func main() {
	// TODO support UNIX-style flags, and improve usage
	// TODO generate interceptors
	cmd.Execute()

	//turbo.LoadServiceConfigWith("turbo/example/bookservice")
	//options := "-I /Users/xiaozhang/goworkspace/src/turbo/example/bookservice " +
	//	"-I /Users/xiaozhang/goworkspace/src/turbo/example/common " +
	//	"/Users/xiaozhang/goworkspace/src/turbo/example/common/shared.proto " +
	//	"/Users/xiaozhang/goworkspace/src/turbo/example/bookservice/*.proto"
	//
	//turbo.GenerateProtobufStub(options)
	//turbo.GenerateGrpcSwitcher()
}
