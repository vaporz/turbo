// this is a learning test
package turbo

import (
	sjson "github.com/bitly/go-simplejson"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLen(t *testing.T) {
	json, _ := sjson.NewJson([]byte("{\"number\": \"123\"}"))
	jmap, _ := json.Map()
	assert.Equal(t, 1, len(jmap))
}

func TestGet(t *testing.T) {
	json, _ := sjson.NewJson([]byte("{\"number\": \"123\"}"))
	s, _ := json.Get("number").String()
	assert.Equal(t, "123", s)
	j, ok := json.CheckGet("number")
	jStr, _ := j.String()
	assert.Equal(t, true, ok)
	assert.Equal(t, "123", jStr)
}

func TestGetUnknown(t *testing.T) {
	json, _ := sjson.NewJson([]byte("{\"number\": \"123\"}"))
	_, ok := json.CheckGet("a")
	assert.Equal(t, false, ok)
}

func TestDel(t *testing.T) {
	json, _ := sjson.NewJson([]byte("{\"number\": \"123\"}"))
	jmap, _ := json.Map()
	assert.Equal(t, 1, len(jmap))
	json.Del("number")
	assert.Equal(t, 0, len(jmap))
}

func TestSet(t *testing.T) {
	json, _ := sjson.NewJson([]byte("{\"number\": \"123\"}"))
	json.Set("number", int64(123))
	n, _ := json.Get("number").Int64()
	assert.Equal(t, int64(123), n)
	s, _ := json.MarshalJSON()
	assert.Equal(t, "{\"number\":123}", string(s))
}

func TestSetNull(t *testing.T) {
	json, _ := sjson.NewJson([]byte("{\"number\": \"123\"}"))
	json.Set("nullptr", nil)
	s, _ := json.MarshalJSON()
	assert.Equal(t, "{\"nullptr\":null,\"number\":\"123\"}", string(s))
}

func TestArray(t *testing.T) {
	json, _ := sjson.NewJson([]byte("[\"123\",\"456\"]"))
	arr, _ := json.Array()
	assert.Equal(t, 2, len(arr))
	assert.Equal(t, "123", arr[0].(string))
	assert.Equal(t, "456", arr[1].(string))
}

func TestChangeArray(t *testing.T) {
	json, _ := sjson.NewJson([]byte("[\"123\",\"456\"]"))
	s, _ := json.MarshalJSON()
	assert.Equal(t, "[\"123\",\"456\"]", string(s))

	arr, _ := json.Array()
	arr[0] = int64(111)
	arr[1] = int64(222)
	s, _ = json.MarshalJSON()
	assert.Equal(t, "[111,222]", string(s))
}

func TestStructArray(t *testing.T) {
	json, _ := sjson.NewJson([]byte("[{\"name\":\"apple\"}]"))
	jo := json.GetIndex(0)
	s, _ := jo.MarshalJSON()
	assert.Equal(t, "{\"name\":\"apple\"}", string(s))
}
