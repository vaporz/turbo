package turbo

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"io"
	"strings"
	"os"
)

func TestUnknownType(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			assert.Equal(t, "Invalid server type, should be (grpc|thrift)", err)
		} else {
			t.Errorf("The code did not panic")
		}
	}()
	Generate("unknown", "", "", "")
}

func TestValidateServiceRootPath(t *testing.T) {
	var r io.Reader
	r = strings.NewReader("y\n")
	ServiceRootPath = "a"
	validateServiceRootPath(r)

	initPkgPath("github.com/vaporz/turbo/test")
	os.MkdirAll(ServiceRootPath+"/a", 0755)
	ServiceRootPath = ServiceRootPath + "/a"
	validateServiceRootPath(r)
}
