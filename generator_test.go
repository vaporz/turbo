package turbo

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestFailingCommandExplainsItself pins what a reader gets when a generation tool
// fails. protoc and the thrift compiler explain themselves on their own terms --
// for instance "plugins are not supported; use 'protoc --go-grpc_out=...'" -- and
// a bare "exit status 1" used to throw that away, sending the reader looking for
// it somewhere above in the terminal.
func TestFailingCommandExplainsItself(t *testing.T) {
	var recovered interface{}
	func() {
		defer func() { recovered = recover() }()
		executeCmd("bash", "-c", "echo the-reason >&2; exit 3")
	}()

	err, ok := recovered.(error)
	assert.True(t, ok, "a failing command must panic with an error, got %v", recovered)
	assert.Contains(t, err.Error(), "bash -c echo the-reason")
	assert.Contains(t, err.Error(), "the-reason")
	assert.Contains(t, err.Error(), "exit status 3")
}

// TestSuccessfulCommandDoesNotPanic is the other half: the helper sits on the
// normal generation path, so it must stay quiet when the tool succeeds.
func TestSuccessfulCommandDoesNotPanic(t *testing.T) {
	executeCmd("bash", "-c", "echo fine")
}

func TestCommandErrorWithoutOutput(t *testing.T) {
	err := commandError("thrift -r --gen go x.thrift", errors.New("exit status 1"), "  \n ")
	assert.Equal(t, "turbo: thrift -r --gen go x.thrift failed: exit status 1", err.Error())
}

// TestOutputTailKeepsTheEnd keeps the tail bounded: a long generation must not
// grow the panic message with output nobody reads.
func TestOutputTailKeepsTheEnd(t *testing.T) {
	tail := &outputTail{}
	tail.Write([]byte("FIRST-MARKER" + strings.Repeat("x", outputTailBytes) + "THE-END"))

	assert.Len(t, tail.String(), outputTailBytes)
	assert.True(t, strings.HasSuffix(tail.String(), "THE-END"))
	assert.NotContains(t, tail.String(), "FIRST-MARKER")
}

// TestMethodNamesAreSorted pins the order of the method list a generated switcher
// is written from. The names are collected in a map, so without an explicit sort
// the same urlmapping produces a different switch order on every run -- which
// makes regeneration look like a change and hides the real ones.
func TestMethodNamesAreSorted(t *testing.T) {
	mappings := [][4]string{
		{"POST", "/a", "Svc", "Zulu"},
		{"POST", "/b", "Svc", "Alpha"},
		{"POST", "/c", "Svc", "Mike"},
		{"POST", "/d", "Svc", "Bravo"},
		{"POST", "/e", "Other", "Yankee"},
		{"POST", "/f", "Other", "Xray"},
	}
	// the map inside has no order, so a single matching run would be luck
	for i := 0; i < 200; i++ {
		names := methodNames(mappings)
		assert.Equal(t, []string{"Alpha", "Bravo", "Mike", "Zulu"}, names["Svc"])
		assert.Equal(t, []string{"Xray", "Yankee"}, names["Other"])
	}
}

func TestUnknownType(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			assert.Equal(t, "Invalid server type, should be (grpc|thrift)", err)
		} else {
			t.Errorf("The code did not panic")
		}
	}()
	g := &Generator{}
	g.Generate()
}

func TestValidateServiceRootPath(t *testing.T) {
	g := &Creator{PkgPath: "a"}
	var r io.Reader
	r = strings.NewReader("y\n")
	g.validateServiceRootPath(r)

	rp, _ := filepath.Abs("../../../")
	g = &Creator{FileRootPath: rp, PkgPath: "github.com/vaporz/turbo/test/a"}
	p := rp + "/github.com/vaporz/turbo/test/a"
	os.MkdirAll(p, 0755)
	g.validateServiceRootPath(r)
	_, err := os.Stat(p)
	assert.True(t, os.IsNotExist(err))
}

func TestInvalidPkgPath(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			assert.Equal(t, "pkgPath is blank", err)
		} else {
			t.Errorf("The code did not panic")
		}
	}()
	g := &Creator{}
	g.validateServiceRootPath(nil)
}
