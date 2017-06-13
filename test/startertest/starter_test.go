package startertest

import (
	gcomponent "github.com/vaporz/turbo/test/testservice/grpcapi/component"
	gimpl "github.com/vaporz/turbo/test/testservice/grpcservice/impl"
	tcompoent "github.com/vaporz/turbo/test/testservice/thriftapi/component"
	timpl "github.com/vaporz/turbo/test/testservice/thriftservice/impl"
	"github.com/vaporz/turbo/test/testservice/gen"
	"github.com/vaporz/turbo/test"
	"github.com/vaporz/turbo"
	"testing"
	"time"
	"bytes"
	"github.com/stretchr/testify/assert"
	"net/http"
)

func TestGrpcService(t *testing.T) {
	httpPort := "8083"
	turbo.ResetChans()
	test.OverrideServiceYaml(httpPort, "50053", "development")
	go turbo.StartGrpcService(50053, "github.com/vaporz/turbo/test/testservice",
		"service", gimpl.RegisterServer)
	time.Sleep(time.Second * 1)

	go turbo.StartGrpcHTTPServer("github.com/vaporz/turbo/test/testservice", "service",
		gcomponent.GrpcClient, gen.GrpcSwitcher)
	time.Sleep(time.Second)

	testGet(t, "http://localhost:"+httpPort+"/hello/testtest", "{\"message\":\"[grpc server]Hello, testtest\"}")

	turbo.Stop()
	time.Sleep(time.Second)
}

func TestThriftService(t *testing.T) {
	httpPort := "8084"
	turbo.ResetChans()
	test.OverrideServiceYaml(httpPort, "50054", "development")
	go turbo.StartThriftService(50054, "github.com/vaporz/turbo/test/testservice",
		"service", timpl.TProcessor)
	time.Sleep(time.Second * 1)

	go turbo.StartThriftHTTPServer("github.com/vaporz/turbo/test/testservice", "service",
		tcompoent.ThriftClient, gen.ThriftSwitcher)
	time.Sleep(time.Second)

	testGet(t, "http://localhost:"+httpPort+"/hello/testtest", "{\"message\":\"[thrift server]Hello, testtest\"}")

	turbo.Stop()
	time.Sleep(time.Second)
}

func testPost(t *testing.T, url, expected string) {
	resp, err := http.Post(url, "", nil)
	defer resp.Body.Close()
	assert.Nil(t, err)
	assert.Equal(t, expected, readResp(resp))
}

func readResp(resp *http.Response) string {
	var bytes bytes.Buffer
	bytes.ReadFrom(resp.Body)
	return bytes.String()
}

func testGet(t *testing.T, url, expected string) {
	resp, err := http.Get(url)
	defer resp.Body.Close()
	assert.Nil(t, err)
	assert.Equal(t, expected, readResp(resp))
}
