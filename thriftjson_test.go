/*
 * Copyright © 2017 Xiao Zhang <zzxx513@gmail.com>.
 * Use of this source code is governed by an MIT-style
 * license that can be found in the LICENSE file.
 */
package turbo

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type thriftArgsStub struct {
	Request  *struct{ StringValue string } `thrift:"request,1" db:"request" json:"request"`
	YourName string                        `thrift:"yourName,2" db:"yourName" json:"yourName"`
	Count    int64                         `thrift:"int64Value,3" db:"int64Value" json:"int64Value"`
}

func TestThriftBodyTakesEveryNameAnArgumentHas(t *testing.T) {
	field := reflect.TypeOf(thriftArgsStub{}).Field(1) // YourName
	assert.Equal(t, []string{"YourName", "yourName", "yourName"}, argumentNames(field))
}

func TestThriftBodyMatchesSpellingVariantsAndReportsWhatItUsed(t *testing.T) {
	for _, body := range []string{
		`{"yourName":"x"}`,
		`{"your_name":"x"}`,
		`{"YOURNAME":"x"}`,
		`{"yourname":"x"}`,
	} {
		parsed, err := decodeThriftBody([]byte(body))
		assert.NoError(t, err, body)

		raw, ok := parsed.take(reflect.TypeOf(thriftArgsStub{}).Field(1))
		assert.True(t, ok, body)
		assert.JSONEq(t, `"x"`, string(raw), body)

		// the key it consumed is not reported as unknown
		assert.Empty(t, parsed.unused(), body)
	}
}

func TestThriftBodyReportsUnknownKeys(t *testing.T) {
	parsed, err := decodeThriftBody([]byte(`{"yourName":"x","typo":1,"other":2}`))
	assert.NoError(t, err)

	_, ok := parsed.take(reflect.TypeOf(thriftArgsStub{}).Field(1))
	assert.True(t, ok)
	assert.Equal(t, []string{"other", "typo"}, parsed.unused())
}

func TestThriftBodyAcceptsAnEmptyBody(t *testing.T) {
	parsed, err := decodeThriftBody(nil)
	assert.NoError(t, err)
	assert.Empty(t, parsed.unused())
	assert.False(t, func() bool {
		_, ok := parsed.take(reflect.TypeOf(thriftArgsStub{}).Field(1))
		return ok
	}())

	// a body that is not an object at all is refused
	_, err = decodeThriftBody([]byte(`[1,2]`))
	assert.Error(t, err)
}
