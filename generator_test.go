package turbo

import (
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"strings"
	"testing"
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
	g := &Generator{c: &Config{configs: make(map[string]string), GOPATH: GOPATH()}}
	var r io.Reader
	r = strings.NewReader("y\n")
	g.c.configs[serviceRootPath] = "a"
	g.validateServiceRootPath(r)

	p := GOPATH() + "/src/" + "github.com/vaporz/turbo/test"
	g.c.configs[serviceRootPath] = p + "/a"
	os.MkdirAll(p+"/a", 0755)
	g.validateServiceRootPath(r)
	_, err := os.Stat(g.c.ServiceRootPath())
	assert.True(t, os.IsNotExist(err))
}
