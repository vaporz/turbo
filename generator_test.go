package turbo

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
