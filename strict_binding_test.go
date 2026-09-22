/*
 * Copyright © 2017 Xiao Zhang <zzxx513@gmail.com>.
 * Use of this source code is governed by an MIT-style
 * license that can be found in the LICENSE file.
 */
package turbo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

type strictBindingPayload struct {
	YourName   string
	Int64Value int64
	Flag       bool
}

// strictBindingRequest builds a request that carries the components the binding
// code looks up from the context, the way the request handler installs them.
func strictBindingRequest(target string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.Form = req.URL.Query()
	return req.WithContext(context.WithValue(req.Context(), componentsKey, &Components{}))
}

func TestBuildStructErrRefusesAValueItCannotBind(t *testing.T) {
	req := strictBindingRequest("/hello?your_name=ok&int64_value=abc")
	var payload strictBindingPayload

	err := BuildStructErr(nil, reflect.TypeOf(payload), reflect.ValueOf(&payload).Elem(), req)
	assert.Error(t, err)
	// the message has to name the field and the offending value, because that is
	// all a caller needs to fix the request
	assert.Contains(t, err.Error(), "Int64Value")
	assert.Contains(t, err.Error(), `"abc"`)
	assert.Equal(t, http.StatusBadRequest, StatusOf(err))
}

func TestBuildStructErrBindsUsableValues(t *testing.T) {
	req := strictBindingRequest("/hello?your_name=ok&int64_value=42&flag=true")
	var payload strictBindingPayload

	assert.NoError(t, BuildStructErr(nil, reflect.TypeOf(payload), reflect.ValueOf(&payload).Elem(), req))
	assert.Equal(t, "ok", payload.YourName)
	assert.Equal(t, int64(42), payload.Int64Value)
	assert.True(t, payload.Flag)
}

func TestBuildStructErrLeavesAnAbsentParameterAlone(t *testing.T) {
	req := strictBindingRequest("/hello?your_name=ok")
	var payload strictBindingPayload

	// absence is not an error: the field may be optional, and the caller sent
	// nothing to be wrong about
	assert.NoError(t, BuildStructErr(nil, reflect.TypeOf(payload), reflect.ValueOf(&payload).Elem(), req))
	assert.Equal(t, "ok", payload.YourName)
	assert.Equal(t, int64(0), payload.Int64Value)
	assert.False(t, payload.Flag)
}

func TestBindingErrorSaysWhoseMistakeItIs(t *testing.T) {
	cause := errors.New("cannot parse")

	query := strictBindingRequest("/hello?your_name=abc")
	err := bindingErrorFor("YourName", "abc", query, cause)
	assert.Equal(t, http.StatusBadRequest, StatusOf(err))
	assert.Contains(t, err.Error(), "query/form parameter")

	path := httptest.NewRequest(http.MethodGet, "/hello/vaporz", nil)
	path.Form = path.URL.Query()
	path = mux.SetURLVars(path, map[string]string{"your_Name": "vaporz"})
	path = path.WithContext(context.WithValue(path.Context(), componentsKey, &Components{}))
	err = bindingErrorFor("YourName", "vaporz", path, cause)
	assert.Equal(t, http.StatusBadRequest, StatusOf(err))
	assert.Contains(t, err.Error(), "path parameter")

	// a value the server injected is the server's mistake, not the caller's
	injected := strictBindingRequest("/hello")
	InjectParam(injected, "your_Name", "from-server")
	err = bindingErrorFor("YourName", "from-server", injected, cause)
	assert.Equal(t, http.StatusInternalServerError, StatusOf(err))
	assert.Contains(t, err.Error(), "injected value")
}

func TestReflectValueLeavesAnAbsentParameterAtItsZero(t *testing.T) {
	payload := struct {
		Flag  bool
		Count int16
		Name  string
	}{}
	fields := reflect.ValueOf(&payload).Elem()

	// an argument a request does not carry must not be reported as a conversion
	// failure. The thrift path used to log one for every absent argument, and
	// strict binding would otherwise have turned that into a 400.
	for i := 0; i < fields.NumField(); i++ {
		field := fields.Field(i)
		got, err := reflectValue(field.Type(), field, "")
		assert.NoError(t, err, field.Type().Name())
		assert.Equal(t, field.Type(), got.Type(), field.Type().Name())
		assert.True(t, got.IsZero(), field.Type().Name())
	}
}
