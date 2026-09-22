/*
 * Copyright © 2017 Xiao Zhang <zzxx513@gmail.com>.
 * Use of this source code is governed by an MIT-style
 * license that can be found in the LICENSE file.
 */
package turbo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestJSONFieldNamesSelectsTheKeySpelling pins T11: the JSON names protobuf
// defines (int64Value) become available as an opt in, while the proto field names
// (Int64Value) stay what a Marshaler built by a caller produces.
func TestJSONFieldNamesSelectsTheKeySpelling(t *testing.T) {
	message := &TestPrimitives{Int64Value: 7, BoolValue: true}

	proto := Marshaler{}
	protoNames, err := proto.JSON(message)
	assert.NoError(t, err)
	assert.Contains(t, string(protoNames), `"Int64Value":"7"`)

	jsonNames := Marshaler{UseJSONNames: true}
	withJSONNames, err := jsonNames.JSON(message)
	assert.NoError(t, err)
	assert.Contains(t, string(withJSONNames), `"int64Value":"7"`)
	assert.NotContains(t, string(withJSONNames), `"Int64Value"`)
}

// TestFilterProtoJsonKeepsWorkingWithJSONNames covers the interaction that made
// this option worth checking: the patch looks keys up by several spellings, so it
// has to keep finding the fields when the output uses protobuf's JSON names --
// and the keys it adds for absent fields must use that spelling too, not the
// snake case one it used to write.
func TestFilterProtoJsonKeepsWorkingWithJSONNames(t *testing.T) {
	m := Marshaler{FilterProtoJson: true, EmitZeroValues: true, UseJSONNames: true}

	out, err := m.JSON(&TestPrimitives{Int64Value: 7})
	assert.NoError(t, err)
	assert.Contains(t, string(out), `"int64Value":"7"`)
	assert.Contains(t, string(out), `"boolValue":false`)
	assert.NotContains(t, string(out), "bool_value")
	assert.NotContains(t, string(out), "int64_value")
}

func TestJSONFieldNamesConfig(t *testing.T) {
	config := func(value string) *Config {
		return &Config{configs: map[string]string{jsonFieldNames: value}}
	}

	// unset means the historical behaviour
	assert.Equal(t, jsonFieldNamesProto, config("").JSONFieldNames())
	assert.Equal(t, jsonFieldNamesProto, config("proto").JSONFieldNames())
	assert.Equal(t, jsonFieldNamesCamel, config("camel").JSONFieldNames())
	assert.Equal(t, jsonFieldNamesCamel, config("  Camel ").JSONFieldNames())

	// an unrecognised value is refused when the configuration loads, rather than
	// quietly picking one of them
	assert.NoError(t, config("camel").validate())
	assert.NoError(t, config("").validate())
	assert.Error(t, config("camelCase").validate())
}
