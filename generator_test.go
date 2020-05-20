package turbo

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
