/*
 * Copyright © 2017 Xiao Zhang <zzxx513@gmail.com>.
 * Use of this source code is governed by an MIT-style
 * license that can be found in the LICENSE file.
 */
package turbo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestValidateIncludePaths pins T7. -I takes the directory that holds the .proto
// or .thrift files; passing a file used to build a path like
// service.proto/*.proto, and protoc then complained about a file nobody had
// mentioned.
func TestValidateIncludePaths(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "service.proto")
	assert.NoError(t, os.WriteFile(file, []byte("syntax = \"proto3\";\n"), 0644))

	assert.NoError(t, ValidateIncludePaths([]string{dir}))
	assert.NoError(t, ValidateIncludePaths(nil))

	err := ValidateIncludePaths([]string{file})
	assert.Error(t, err)
	// the message has to say what to write instead
	assert.Contains(t, err.Error(), "is a file")
	assert.Contains(t, err.Error(), dir)

	assert.Error(t, ValidateIncludePaths([]string{filepath.Join(dir, "missing")}))
}

// TestLegacyProtocGenGo pins T8①'s discriminator: a protoc-gen-go that still
// supports --go_out=plugins=grpc answers --version with "this program should be
// run by protoc", while a modern one prints its version.
func TestLegacyProtocGenGo(t *testing.T) {
	legacy := `protoc-gen-go: unknown argument "--version" (this program should be run by protoc, not directly)`
	assert.True(t, legacyProtocGenGo(legacy))

	assert.False(t, legacyProtocGenGo("protoc-gen-go v1.28.1\n"))
	assert.False(t, legacyProtocGenGo("protoc-gen-go v1.3.5\n"))
}

func TestCheckToolchainRefusesAModernProtocGenGo(t *testing.T) {
	// protoc-gen-go v1.4 removed --go_out=plugins=grpc, which turbo generates
	// with; the point of the check is to say that, instead of letting protoc
	// report an unknown flag somewhere else
	dir := t.TempDir()
	fake := filepath.Join(dir, "protoc-gen-go")
	assert.NoError(t, os.WriteFile(fake, []byte("#!/bin/sh\necho 'protoc-gen-go v1.28.1'\n"), 0755))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	g := &Generator{RpcType: "grpc", FilePaths: []string{t.TempDir()}}
	assert.Panics(t, func() { g.checkToolchain() })
}
