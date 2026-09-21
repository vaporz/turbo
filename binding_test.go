/*
 * Copyright © 2017 Xiao Zhang <zzxx513@gmail.com>.
 * Use of this source code is governed by an MIT-style
 * license that can be found in the LICENSE file.
 */
package turbo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

// bindingRequest builds a request whose Form holds the query, optionally with
// path variables captured by the route.
func bindingRequest(target string, pathVars map[string]string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.Form = req.URL.Query()
	if pathVars != nil {
		req = mux.SetURLVars(req, pathVars)
	}
	return req
}

func TestInjectParamIsVisibleUnderEveryFieldSpelling(t *testing.T) {
	req := bindingRequest("/hello", nil)
	InjectParam(req, "device_code", "A3")

	for _, fieldName := range []string{"DeviceCode", "device_code", "deviceCode", "DEVICE_CODE"} {
		value, ok := InjectedValue(fieldName, req)
		assert.True(t, ok, fieldName)
		assert.Equal(t, "A3", value, fieldName)
	}
	_, ok := InjectedValue("SomethingElse", req)
	assert.False(t, ok)
}

func TestInjectParamAccumulatesAcrossCalls(t *testing.T) {
	req := bindingRequest("/hello", nil)
	InjectParam(req, "first", "1")
	InjectParam(req, "second", "2")

	first, ok := InjectedValue("First", req)
	assert.True(t, ok)
	assert.Equal(t, "1", first)
	second, ok := InjectedValue("Second", req)
	assert.True(t, ok)
	assert.Equal(t, "2", second)
}

func TestInjectedValueAlsoReadsPlainContextValues(t *testing.T) {
	req := bindingRequest("/hello", nil)
	*req = *req.WithContext(context.WithValue(req.Context(), "device_code", "from-context"))

	value, ok := InjectedValue("DeviceCode", req)
	assert.True(t, ok)
	assert.Equal(t, "from-context", value)
}

func TestFindValuePrefersInjectedValues(t *testing.T) {
	req := bindingRequest("/hello?your_name=from-query", nil)
	// the injected spelling deliberately differs from the query spelling
	InjectParam(req, "your_Name", "from-server")

	// an injected value is the only one the client did not supply
	value, ok := findValue("YourName", req)
	assert.True(t, ok)
	assert.Equal(t, "from-server", value)

	// without an injection the query is used
	queryOnly := bindingRequest("/hello?your_name=from-query", nil)
	value, ok = findValue("YourName", queryOnly)
	assert.True(t, ok)
	assert.Equal(t, "from-query", value)

	_, ok = findValue("Missing", queryOnly)
	assert.False(t, ok)
}

func TestJSONObjectKeysDescribeWhatTheBodyCarried(t *testing.T) {
	raw := jsonObjectKeys(`{"yourName":"a name","nested":{"int64Value":7}}`)
	assert.True(t, rawHas(raw, "YourName"))
	assert.True(t, rawHas(raw, "YOURNAME"))
	assert.False(t, rawHas(raw, "SomeOtherField"))

	nested := rawSub(raw, "Nested")
	assert.NotNil(t, nested)
	assert.True(t, rawHas(nested, "Int64Value"))
	assert.False(t, rawHas(nested, "Missing"))

	// bodies that are not a JSON object mention no field at all
	assert.Nil(t, jsonObjectKeys(""))
	assert.Nil(t, jsonObjectKeys("   "))
	assert.Nil(t, jsonObjectKeys("{oops"))
	assert.Nil(t, jsonObjectKeys(`[1,2,3]`))
}

func TestFormValueIsSpellingInsensitive(t *testing.T) {
	// Deliberate decision (2026-09-21): spellings of a parameter name that differ
	// only in case or in underscores name the same parameter. Treating yourname
	// and your_name as two parameters was rejected -- turbo already fed both into
	// the same field, and a client library that rewrites one spelling into the
	// other would otherwise silently change the value a service reads. The cost is
	// that two parameters differing only by underscores cannot coexist; see
	// Turbo-优化候选清单.md §14.6.
	for _, query := range []string{
		"yourName=x", "yourname=x", "your_name=x", "YOURNAME=x", "YoUrNaMe=x", "your_n_ame=x",
	} {
		req := bindingRequest("/hello?"+query, nil)
		value, ok := formValue("YourName", req)
		assert.True(t, ok, query)
		assert.Equal(t, "x", value, query)
	}

	// when two keys name the same parameter, the spelling a field has always been
	// probed with first wins, and the order they appear in does not matter
	for _, query := range []string{"your_name=turbo&yourname=xxx", "yourname=xxx&your_name=turbo"} {
		req := bindingRequest("/hello?"+query, nil)
		value, ok := formValue("YourName", req)
		assert.True(t, ok, query)
		assert.Equal(t, "xxx", value, query)
	}
}

func TestFindValuePrefersThePathWhateverTheSpelling(t *testing.T) {
	// the route declares {your_Name}; the caller may spell the query any way,
	// and it must not decide which source wins
	for _, query := range []string{"your_name=from-query", "yourName=from-query", "YOURNAME=from-query"} {
		req := bindingRequest("/hello/vaporz?"+query, map[string]string{"your_Name": "from-path"})
		value, ok := findValue("YourName", req)
		assert.True(t, ok, query)
		assert.Equal(t, "from-path", value, query)
	}

	// without a route variable the query is used
	req := bindingRequest("/hello?yourName=from-query", nil)
	value, ok := findValue("YourName", req)
	assert.True(t, ok)
	assert.Equal(t, "from-query", value)
}
