package turbo

import (
	sjson "github.com/bitly/go-simplejson"
	"testing"
	"reflect"
	"github.com/stretchr/testify/assert"
)

type args struct {
}

type testStruct struct {
	testId   int64
	ptrValue *args
}

func TestFilterFieldInt64Str(t *testing.T) {
	s := &testStruct{testId: 123}
	tp := reflect.TypeOf(s).Elem()
	json, _ := sjson.NewJson([]byte("{\"test_id\": \"123\"}"))
	structFieldFilter(tp.Field(0))(json, tp.Field(0), reflect.ValueOf(s))
	jsonBytes, _ := json.MarshalJSON()

	assert.Equal(t, "{\"test_id\":123}", string(jsonBytes))
}

func TestFilterFieldInt64Number(t *testing.T) {
	s := &testStruct{testId: 123}
	json, _ := sjson.NewJson([]byte("{\"test_id\": 123}"))
	filterStruct(json, reflect.TypeOf(s).Elem(), reflect.ValueOf(s).Elem())
	jsonBytes, _ := json.MarshalJSON()

	assert.Equal(t, "{\"ptr_value\":null,\"test_id\":123}", string(jsonBytes))
}

func TestFilterFieldNullPointer(t *testing.T) {
	s := &testStruct{testId: 123}
	tp := reflect.TypeOf(s).Elem()
	json, _ := sjson.NewJson([]byte("{\"test_id\": \"123\"}"))
	structFieldFilter(tp.Field(1))(json, tp.Field(1), reflect.ValueOf(s).Elem().Field(1))
	jsonBytes, _ := json.MarshalJSON()

	assert.Equal(t, "{\"ptr_value\":null,\"test_id\":\"123\"}", string(jsonBytes))
}

func TestFilterStruct(t *testing.T) {
	s := &testStruct{testId: 123}
	json, _ := sjson.NewJson([]byte("{\"test_id\": \"123\"}"))
	filterStruct(json, reflect.TypeOf(s).Elem(), reflect.ValueOf(s).Elem())
	jsonBytes, _ := json.MarshalJSON()

	assert.Equal(t, "{\"ptr_value\":null,\"test_id\":123}", string(jsonBytes))
}

type nestedValue struct {
	ptrValue *args
}

type nestedStruct struct {
	testId      int64
	nestedValue *nestedValue
}

func TestFilterNestedStruct_Nil_field(t *testing.T) {
	s := &nestedStruct{testId: 123}
	json, _ := sjson.NewJson([]byte("{\"test_id\": \"123\"}"))
	filterStruct(json, reflect.TypeOf(s).Elem(), reflect.ValueOf(s).Elem())
	jsonBytes, _ := json.MarshalJSON()
	assert.Equal(t, "{\"nested_value\":null,\"test_id\":123}", string(jsonBytes))
}

func TestFilterNestedStructField_Empty_Field(t *testing.T) {
	s := &nestedStruct{testId: 123, nestedValue: &nestedValue{}}
	json, _ := sjson.NewJson([]byte("{\"test_id\": \"123\", \"nested_value\":{}}"))
	structField := reflect.TypeOf(s).Elem().Field(1)
	structFieldFilter(structField)(json, structField, reflect.ValueOf(s).Elem().Field(1))
	jsonBytes, _ := json.MarshalJSON()
	assert.Equal(t, "{\"nested_value\":{\"ptr_value\":null},\"test_id\":\"123\"}", string(jsonBytes))
}

func TestFilterNestedStruct_Empty_Field(t *testing.T) {
	s := &nestedStruct{testId: 123, nestedValue: &nestedValue{}}
	json, _ := sjson.NewJson([]byte("{\"test_id\": \"123\", \"nested_value\":{}}"))
	filterStruct(json, reflect.TypeOf(s).Elem(), reflect.ValueOf(s).Elem())
	jsonBytes, _ := json.MarshalJSON()
	assert.Equal(t, "{\"nested_value\":{\"ptr_value\":null},\"test_id\":123}", string(jsonBytes))
}

type someArgs struct {
}

type childValue struct {
	testId      int64
	stringValue string
	intArray    []int64
	args        *someArgs
}

type complexNestedValue struct {
	testId        int64
	stringValue   string
	intArray      []int64
	childValueArr []*childValue
	childValue1   *childValue
}

type complexNestedStruct struct {
	testId              int64
	stringValue         string
	intArray            []int64  `protobuf:"varint,1,opt,name=new_name" json:"id,omitempty"`
	complexNestedValue  *complexNestedValue
	complexNestedValue1 *complexNestedValue
	complexNestedValue2 *complexNestedValue
}

func TestFilterComplexNestedStruct(t *testing.T) {
	cv := &childValue{testId: 123, stringValue: "a string"}
	cv1 := &childValue{testId: 456, args: &someArgs{}}
	cv2 := &childValue{testId: 789, intArray: []int64{44, 55, 66}}

	cnv := &complexNestedValue{testId: 456, intArray: []int64{11, 22, 33}, childValueArr: []*childValue{cv1, cv2}, childValue1: cv}
	s := &complexNestedStruct{stringValue: "struct string", complexNestedValue: cnv}
	bytes := []byte("{\"string_value\":\"struct string\", \"complex_nested_value\":{\"test_id\":\"456\"" +
		", \"int_array\":[\"11\",\"22\",\"33\"], \"child_value_arr\":[{\"test_id\":\"456\",\"args\":{}}," +
		"{\"test_id\":\"789\",\"int_array\":[\"44\",\"55\",\"66\"]}]" +
		", \"child_value1\":{\"test_id\":\"123\",\"string_value\":\"a string\"}}}")
	bytes, _ = FilterProtoJson(bytes, s)
	assert.Equal(t, "{\"complex_nested_value\":{\"child_value1\":{\"args\":null,\"int_array\":[],\"string_value\":"+
		"\"a string\",\"test_id\":123},\"child_value_arr\":[{\"args\":{},\"int_array\":[],\"string_value\":\"\",\"test_id\":456},"+
		"{\"args\":null,\"int_array\":[44,55,66],\"string_value\":\"\",\"test_id\":789}],\"int_array\":[11,22,33],\"string_value\":\"\""+
		",\"test_id\":456},\"complex_nested_value1\":null,\"complex_nested_value2\":null,\"new_name\":[],\"string_value\":\"struct string\","+
		"\"test_id\":0}", string(bytes))
	/*
Before filter:
{
    "test_id": "0",
    "string_value": "struct string",
    "complex_nested_value": {
        "test_id": "456",
        "string_value": "",
        "int_array": [
            "11",
            "22",
            "33"
        ],
        "child_value_arr": [
            {
                "test_id": "456",
                "string_value": "",
                "args": {}
            },
            {
                "test_id": "789",
                "string_value": "",
                "int_array": [
                    "44",
                    "55",
                    "66"
                ]
            }
        ],
        "child_value1": {
            "test_id": "123",
            "string_value": "a string"
        }
    }
}

After filter:
{
    "complex_nested_value": {
        "child_value1": {
            "args": null,
            "int_array": [],
            "string_value": "a string",
            "test_id": 123
        },
        "child_value_arr": [
            {
                "args": {},
                "int_array": [],
                "string_value": "",
                "test_id": 456
            },
            {
                "args": null,
                "int_array": [
                    44,
                    55,
                    66
                ],
                "string_value": "",
                "test_id": 789
            }
        ],
        "int_array": [
            11,
            22,
            33
        ],
        "string_value": "",
        "test_id": 456
    },
    "complex_nested_value1": null,
    "complex_nested_value2": null,
    "int_array": [],
    "string_value": "struct string",
    "test_id": 0
}
	 */
}
