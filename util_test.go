package turbo

import (
	sjson "github.com/bitly/go-simplejson"
	"github.com/stretchr/testify/assert"
	"os"
	"reflect"
	"testing"
)

func TestMain(m *testing.M) {
	executeCmd("bash", "-c", "go install github.com/vaporz/turbo/turbo")
	executeCmd("bash", "-c", "go install github.com/vaporz/turbo/protoc-gen-buildfields")
	os.Exit(m.Run())
}

func TestIsCamelCase(t *testing.T) {
	assert.Equal(t, true, IsCamelCase("CamelCase"))
	assert.Equal(t, true, IsCamelCase("CAMELCase"))
	assert.Equal(t, true, IsCamelCase("CAMELCASE"))
	assert.Equal(t, true, IsCamelCase("CamelCASE"))
	assert.Equal(t, false, IsCamelCase("camelCase"))
	assert.Equal(t, false, IsCamelCase("camelcase"))
	assert.Equal(t, false, IsCamelCase("camel_case"))
	assert.Equal(t, false, IsCamelCase(""))
	assert.Equal(t, false, IsCamelCase("_"))

	assert.Equal(t, false, IsNotCamelCase("CamelCase"))
}

type testPrimitives struct {
	Int64Value   int64
	Int32Value   int32
	Uint64Value  uint64
	Uint32Value  uint32
	Float32Value float32
	Float64Value float64
	BoolValue    bool
}

func (t *testPrimitives) Reset()         {}
func (t *testPrimitives) String() string { return "" }
func (t *testPrimitives) ProtoMessage()  {}

func TestPrimitives(t *testing.T) {
	m := Marshaler{FilterProtoJson: true, EmitZeroValues: true, Int64AsNumber: true}
	ts := &testPrimitives{Int64Value: 111, Int32Value: 0, Float32Value: 1, BoolValue: true}
	buf, _ := m.JSON(ts)
	assert.Equal(t, "{\"BoolValue\":true,\"Float32Value\":1,\"Float64Value\":0,"+
		"\"Int32Value\":0,\"Int64Value\":111,\"Uint32Value\":0,\"Uint64Value\":0}", string(buf))

	ts = &testPrimitives{Int64Value: 0, Int32Value: 0, Float32Value: 1, BoolValue: true}
	buf, _ = m.JSON(ts)
	assert.Equal(t, "{\"BoolValue\":true,\"Float32Value\":1,\"Float64Value\":0,"+
		"\"Int32Value\":0,\"Int64Value\":0,\"Uint32Value\":0,\"Uint64Value\":0}", string(buf))

	m.Int64AsNumber = false
	ts = &testPrimitives{Int64Value: 0, Int32Value: 0, Float32Value: 1, BoolValue: true}
	buf, _ = m.JSON(ts)
	assert.Equal(t, "{\"BoolValue\":true,\"Float32Value\":1,\"Float64Value\":0,"+
		"\"Int32Value\":0,\"Int64Value\":\"0\",\"Uint32Value\":0,\"Uint64Value\":0}", string(buf))
}

func TestPrimitives_Int64_As_Number_False(t *testing.T) {
	m := Marshaler{FilterProtoJson: true, EmitZeroValues: true}
	ts := &testPrimitives{Int64Value: 111, Float32Value: 1, BoolValue: true}
	buf, _ := m.JSON(ts)
	assert.Equal(t, "{\"BoolValue\":true,\"Float32Value\":1,\"Float64Value\":0,"+
		"\"Int32Value\":0,\"Int64Value\":\"111\",\"Uint32Value\":0,\"Uint64Value\":0}", string(buf))
}

func TestPrimitives_Emit_Zerovalues_False(t *testing.T) {
	m := Marshaler{FilterProtoJson: true, Int64AsNumber: true}
	ts := &testPrimitives{Int64Value: 111, Float32Value: 1, BoolValue: true}
	buf, _ := m.JSON(ts)
	assert.Equal(t, "{\"BoolValue\":true,\"Float32Value\":1,\"Int64Value\":111}", string(buf))
}

type args struct {
}

type testStruct struct {
	TestId   int64
	PtrValue *args
}

type testProtoStruct struct {
	value int64
}

func (t *testProtoStruct) Reset()         {}
func (t *testProtoStruct) String() string { return "" }
func (t *testProtoStruct) ProtoMessage()  {}

func TestJSON(t *testing.T) {
	m := Marshaler{FilterProtoJson: true, EmitZeroValues: true}
	ts := &testStruct{}
	buf, _ := m.JSON(ts)
	assert.Equal(t, "{\"TestId\":0,\"PtrValue\":null}", string(buf))
}

func TestJSON_Proto_OPTION_TRUE(t *testing.T) {
	m := Marshaler{FilterProtoJson: true, EmitZeroValues: true, Int64AsNumber: true}
	ts := &testProtoStruct{}
	buf, _ := m.JSON(ts)
	assert.Equal(t, "{\"value\":0}", string(buf))
}

func TestJSON_Proto_OPTION_FALSE(t *testing.T) {
	m := Marshaler{}
	ts := &testProtoStruct{}
	buf, _ := m.JSON(ts)
	assert.Equal(t, "{}", string(buf))
}

func TestFilterFieldInt64Str(t *testing.T) {
	m := Marshaler{FilterProtoJson: true, EmitZeroValues: true, Int64AsNumber: true}
	s := &testStruct{TestId: 123}
	tp := reflect.TypeOf(s).Elem()
	v := reflect.ValueOf(s).Elem()
	json, _ := sjson.NewJson([]byte("{\"test_id\": \"123\"}"))
	m.filterOf(tp.Field(0).Type.Kind())(json, tp.Field(0), v.Field(0))
	jsonBytes, _ := json.MarshalJSON()

	assert.Equal(t, "{\"test_id\":123}", string(jsonBytes))
}

func TestFilterFieldInt64Number(t *testing.T) {
	m := Marshaler{FilterProtoJson: true, EmitZeroValues: true, Int64AsNumber: true}
	s := &testStruct{TestId: 123}
	json, _ := sjson.NewJson([]byte("{\"test_id\": 123}"))
	m.filterStruct(json, reflect.TypeOf(s).Elem(), reflect.ValueOf(s).Elem())
	jsonBytes, _ := json.MarshalJSON()

	assert.Equal(t, "{\"PtrValue\":null,\"test_id\":123}", string(jsonBytes))
}

func TestFilterFieldNullPointer(t *testing.T) {
	m := Marshaler{FilterProtoJson: true, EmitZeroValues: true}
	s := &testStruct{TestId: 123}
	tp := reflect.TypeOf(s).Elem()
	v := reflect.ValueOf(s).Elem()
	json, _ := sjson.NewJson([]byte("{\"test_id\": \"123\"}"))
	m.filterOf(tp.Field(1).Type.Kind())(json, tp.Field(1), v.Field(1))
	jsonBytes, _ := json.MarshalJSON()

	assert.Equal(t, "{\"PtrValue\":null,\"test_id\":\"123\"}", string(jsonBytes))
}

func TestFilterField_With_Empty_Json(t *testing.T) {
	m := Marshaler{FilterProtoJson: true, EmitZeroValues: true, Int64AsNumber: true}
	s := &testStruct{PtrValue: &args{}}
	tp := reflect.TypeOf(s).Elem()
	v := reflect.ValueOf(s).Elem()
	json, _ := sjson.NewJson([]byte("{}"))
	m.filterStruct(json, tp, v)
	jsonBytes, _ := json.MarshalJSON()

	assert.Equal(t, "{\"PtrValue\":{},\"TestId\":0}", string(jsonBytes))
}

func TestFilterStruct(t *testing.T) {
	m := Marshaler{FilterProtoJson: true, EmitZeroValues: true, Int64AsNumber: true}
	s := &testStruct{TestId: 123}
	json, _ := sjson.NewJson([]byte("{\"test_id\": \"123\"}"))
	m.filterStruct(json, reflect.TypeOf(s).Elem(), reflect.ValueOf(s).Elem())
	jsonBytes, _ := json.MarshalJSON()

	assert.Equal(t, "{\"PtrValue\":null,\"test_id\":123}", string(jsonBytes))
}

func TestFilterStruct_Missing_Key(t *testing.T) {
	m := Marshaler{FilterProtoJson: true, EmitZeroValues: true, Int64AsNumber: true}
	s := &testStruct{TestId: 123}
	json, _ := sjson.NewJson([]byte("{}"))
	m.filterStruct(json, reflect.TypeOf(s).Elem(), reflect.ValueOf(s).Elem())
	jsonBytes, _ := json.MarshalJSON()

	assert.Equal(t, "{\"PtrValue\":null,\"TestId\":123}", string(jsonBytes))
}

type testSlice struct {
	Values []int64
}

func TestFilterSlice_Missing_Key(t *testing.T) {
	m := Marshaler{FilterProtoJson: true, EmitZeroValues: true, Int64AsNumber: true}
	s := &testSlice{Values: []int64{1, 2, 3}}
	json, _ := sjson.NewJson([]byte("{\"values\":[1]}"))
	m.filterStruct(json, reflect.TypeOf(s).Elem(), reflect.ValueOf(s).Elem())
	jsonBytes, _ := json.MarshalJSON()

	assert.Equal(t, "{\"values\":[1,2,3]}", string(jsonBytes))
}

func TestFilterSlice_Empty(t *testing.T) {
	m := Marshaler{FilterProtoJson: true, EmitZeroValues: true, Int64AsNumber: true}
	s := &testSlice{Values: []int64{1, 2, 3}}
	json, _ := sjson.NewJson([]byte("{}"))
	m.filterStruct(json, reflect.TypeOf(s).Elem(), reflect.ValueOf(s).Elem())
	jsonBytes, _ := json.MarshalJSON()

	assert.Equal(t, "{\"Values\":[1,2,3]}", string(jsonBytes))
}

type child struct {
	Num int
}

type testStructSlice struct {
	Values []*child
}

func TestFilterSlice_Missing_Struct_Member(t *testing.T) {
	m := Marshaler{FilterProtoJson: true, EmitZeroValues: true}
	c := &child{}
	c1 := &child{Num: 123}
	s := &testStructSlice{Values: []*child{c, c1}}
	json, _ := sjson.NewJson([]byte("{\"values\":[{\"num\":111}]}"))
	m.filterStruct(json, reflect.TypeOf(s).Elem(), reflect.ValueOf(s).Elem())
	jsonBytes, _ := json.MarshalJSON()

	assert.Equal(t, "{\"values\":[{\"num\":0},{\"Num\":123}]}", string(jsonBytes))
}

func TestFilterSlice_Empty_Struct_Member(t *testing.T) {
	m := Marshaler{FilterProtoJson: true, EmitZeroValues: true}
	c := &child{}
	c1 := &child{Num: 123}
	s := &testStructSlice{Values: []*child{c, c1}}
	json, _ := sjson.NewJson([]byte("{}"))
	m.filterStruct(json, reflect.TypeOf(s).Elem(), reflect.ValueOf(s).Elem())
	jsonBytes, _ := json.MarshalJSON()

	assert.Equal(t, "{\"Values\":[{\"Num\":0},{\"Num\":123}]}", string(jsonBytes))
}

type nestedValue struct {
	PtrValue *args
}

type nestedStruct struct {
	TestId      int64
	NestedValue *nestedValue
}

func TestFilterNestedStruct_Nil_field(t *testing.T) {
	m := Marshaler{FilterProtoJson: true, EmitZeroValues: true, Int64AsNumber: true}
	s := &nestedStruct{TestId: 123}
	json, _ := sjson.NewJson([]byte("{\"test_id\": \"123\"}"))
	m.filterStruct(json, reflect.TypeOf(s).Elem(), reflect.ValueOf(s).Elem())
	jsonBytes, _ := json.MarshalJSON()
	assert.Equal(t, "{\"NestedValue\":null,\"test_id\":123}", string(jsonBytes))
}

func TestFilterNestedStructField_Empty_Field(t *testing.T) {
	m := Marshaler{FilterProtoJson: true, EmitZeroValues: true}
	s := &nestedStruct{TestId: 123, NestedValue: &nestedValue{}}
	json, _ := sjson.NewJson([]byte("{\"test_id\": \"123\", \"nested_value\":{}}"))
	structField := reflect.TypeOf(s).Elem().Field(1)
	m.filterOf(structField.Type.Kind())(json, structField, reflect.ValueOf(s).Elem().Field(1))
	jsonBytes, _ := json.MarshalJSON()
	assert.Equal(t, "{\"nested_value\":{\"PtrValue\":null},\"test_id\":\"123\"}", string(jsonBytes))
}

func TestFilterNestedStruct_Empty_Field(t *testing.T) {
	m := Marshaler{FilterProtoJson: true, EmitZeroValues: true, Int64AsNumber: true}
	s := &nestedStruct{TestId: 123, NestedValue: &nestedValue{}}
	json, _ := sjson.NewJson([]byte("{\"test_id\": \"123\", \"nested_value\":{}}"))
	m.filterStruct(json, reflect.TypeOf(s).Elem(), reflect.ValueOf(s).Elem())
	jsonBytes, _ := json.MarshalJSON()
	assert.Equal(t, "{\"nested_value\":{\"PtrValue\":null},\"test_id\":123}", string(jsonBytes))
}

type testTag struct {
	Value          int `protobuf:"varint,1,opt,name=test_name_proto,json=json_proto" json:"id,omitempty"`
	Value1         int `protobuf:"varint,1,opt" json:"-"`
	CamelCaseValue int `protobuf:"varint,1,opt,name=CamelCaseValue" json:"camel_case_value,omitempty"`
}

func TestLookupOrigNameInProtoTag(t *testing.T) {
	m := Marshaler{FilterProtoJson: true, EmitZeroValues: true}
	var v testTag
	sf := reflect.TypeOf(v).Field(0)
	name, _ := m.lookupOrigNameInProtoTag(sf)
	assert.Equal(t, "test_name_proto", name)
}

func TestLookupJSONNameInProtoTag(t *testing.T) {
	m := Marshaler{FilterProtoJson: true, EmitZeroValues: true}
	var v testTag
	sf := reflect.TypeOf(v).Field(0)
	name, _ := m.lookupJSONNameInProtoTag(sf)
	assert.Equal(t, "json_proto", name)
}

func TestLookupNameInJsonTag(t *testing.T) {
	m := Marshaler{FilterProtoJson: true, EmitZeroValues: true}
	var v testTag
	sf := reflect.TypeOf(v).Field(2)
	name, _ := m.lookupNameInJsonTag(sf)
	assert.Equal(t, "camel_case_value", name)
}

func TestTag(t *testing.T) {
	m := Marshaler{FilterProtoJson: true, EmitZeroValues: true}
	v := &testTag{Value: 1}
	json, _ := sjson.NewJson([]byte("{\"json_proto\": 1}"))
	m.filterStruct(json, reflect.TypeOf(v).Elem(), reflect.ValueOf(v).Elem())
	jsonBytes, _ := json.MarshalJSON()
	assert.Equal(t, "{\"CamelCaseValue\":0,\"Value1\":0,\"json_proto\":1}", string(jsonBytes))

	v = &testTag{Value: 1}
	json, _ = sjson.NewJson([]byte("{\"id\": 1}"))
	m.filterStruct(json, reflect.TypeOf(v).Elem(), reflect.ValueOf(v).Elem())
	jsonBytes, _ = json.MarshalJSON()
	assert.Equal(t, "{\"CamelCaseValue\":0,\"Value1\":0,\"id\":1}", string(jsonBytes))

	v = &testTag{Value1: 1}
	json, _ = sjson.NewJson([]byte("{}"))
	m.filterStruct(json, reflect.TypeOf(v).Elem(), reflect.ValueOf(v).Elem())
	jsonBytes, _ = json.MarshalJSON()
	assert.Equal(t, "{\"CamelCaseValue\":0,\"Value1\":1,\"test_name_proto\":0}", string(jsonBytes))
}

type someArgs struct {
}

type childValue struct {
	TestId      int64
	StringValue string
	IntArray    []int64
	Args        *someArgs
}

type complexNestedValue struct {
	TestId        int64
	StringValue   string
	IntArray      []int64
	ChildValueArr []*childValue
	ChildValue1   *childValue
}

type complexNestedStruct struct {
	TestId              int64
	StringValue         string  `protobuf:"varint,1,opt,name=s_value" json:"json_s_value,omitempty"`
	IntArray            []int64 `protobuf:"varint,1,opt,name=new_name" json:"json_new_name,omitempty"`
	ComplexNestedValue  *complexNestedValue
	ComplexNestedValue1 *complexNestedValue `protobuf:"varint,1,opt,name=c_n_v1" json:"c_n_v111,omitempty"`
	ComplexNestedValue2 *complexNestedValue `protobuf:"varint,1,opt" json:"c_n_v2,omitempty"`
}

func TestFilterComplexNestedStructWithTags(t *testing.T) {
	m := Marshaler{FilterProtoJson: true, EmitZeroValues: true, Int64AsNumber: true}
	cv := &childValue{TestId: 123, StringValue: "a string"}
	cv1 := &childValue{TestId: 456, Args: &someArgs{}}
	cv2 := &childValue{TestId: 789, IntArray: []int64{44, 55, 66}}
	cnv := &complexNestedValue{TestId: 456, IntArray: []int64{11, 22, 33}, ChildValueArr: []*childValue{cv1, cv2}, ChildValue1: cv}
	s := &complexNestedStruct{StringValue: "struct string", ComplexNestedValue: cnv}

	bytes := []byte("{\"s_value\":\"struct string\", \"complex_nested_value\":{\"test_id\":\"456\"" +
		", \"int_array\":[\"11\",\"22\",\"33\"], \"child_value_arr\":[{\"test_id\":\"456\",\"args\":{}}," +
		"{\"test_id\":\"789\",\"int_array\":[\"44\",\"55\",\"66\"]}]" +
		", \"child_value1\":{\"test_id\":\"123\",\"string_value\":\"a string\"}}}")
	bytes, _ = m.FilterJsonWithStruct(bytes, s)
	assert.Equal(t, "{\"TestId\":0,\"c_n_v1\":null,\"c_n_v2\":null,\"complex_nested_value\":"+
		"{\"StringValue\":\"\",\"child_value1\":{\"Args\":null,\"IntArray\":[],\"string_value\":"+
		"\"a string\",\"test_id\":123},\"child_value_arr\":[{\"IntArray\":[],\"StringValue\":\"\","+
		"\"args\":{},\"test_id\":456},{\"Args\":null,\"StringValue\":\"\",\"int_array\":[44,55,66],"+
		"\"test_id\":789}],\"int_array\":[11,22,33],\"test_id\":456},\"new_name\":[],\"s_value\":"+
		"\"struct string\"}", string(bytes))
	/*
		Before filter:
		{
		    "string_value": "struct string",
		    "complex_nested_value": {
		        "test_id": "456",
		        "int_array": [
		            "11",
		            "22",
		            "33"
		        ],
		        "child_value_arr": [
		            {
		                "test_id": "456",
		                "args": {}
		            },
		            {
		                "test_id": "789",
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
		    "TestId": 0,
		    "c_n_v1": null,
		    "c_n_v2": null,
		    "complex_nested_value": {
			"StringValue": "",
			"child_value1": {
			    "Args": null,
			    "IntArray": [],
			    "string_value": "a string",
			    "test_id": 123
			},
			"child_value_arr": [
			    {
				"IntArray": [],
				"StringValue": "",
				"args": {},
				"test_id": 456
			    },
			    {
				"Args": null,
				"StringValue": "",
				"int_array": [
				    44,
				    55,
				    66
				],
				"test_id": 789
			    }
			],
			"int_array": [
			    11,
			    22,
			    33
			],
			"test_id": 456
		    },
		    "new_name": [],
		    "s_value": "struct string"
		}
	*/
}

type TestTags struct {
	Data *TestTagsData `protobuf:"bytes,1,opt,name=data" json:"data,omitempty"`
}

func (t *TestTags) Reset()         {}
func (t *TestTags) String() string { return "" }
func (t *TestTags) ProtoMessage()  {}

type TestTagsData struct {
	UploadFile       string  `protobuf:"bytes,1,opt,name=upload_file,json=uploadFile" json:"upload_file,omitempty"`
	UploadUrl        string  `protobuf:"bytes,2,opt,name=upload_url,json=uploadUrl" json:"upload_url,omitempty"`
	MetadataOnly     string  `protobuf:"bytes,3,opt,name=metadata_only,json=metadataOnly" json:"metadata_only,omitempty"`
	ContentTypeId    int64   `protobuf:"varint,4,opt,name=content_type_id,json=contentTypeId" json:"content_type_id,omitempty"`
	CreativeApiId    int64   `protobuf:"varint,5,opt,name=creative_api_id,json=creativeApiId" json:"creative_api_id,omitempty"`
	Duration         int32   `protobuf:"varint,6,opt,name=duration" json:"duration,omitempty"`
	PhysicalDuration float32 `protobuf:"fixed32,7,opt,name=physical_duration,json=physicalDuration" json:"physical_duration,omitempty"`
	Bitrate          int32   `protobuf:"varint,8,opt,name=bitrate" json:"bitrate,omitempty"`
	Height           int32   `protobuf:"varint,9,opt,name=height" json:"height,omitempty"`
	Width            int32   `protobuf:"varint,10,opt,name=width" json:"width,omitempty"`
	Fps              float32 `protobuf:"fixed32,11,opt,name=fps" json:"fps,omitempty"`
	Id3Tag           string  `protobuf:"bytes,12,opt,name=id3tag" json:"id3tag,omitempty"`
}

func (t *TestTagsData) Reset()         {}
func (t *TestTagsData) String() string { return "" }
func (t *TestTagsData) ProtoMessage()  {}

func TestFilterStructWithTag(t *testing.T) {
	m := Marshaler{FilterProtoJson: true, EmitZeroValues: true, Int64AsNumber: true}
	ts := &TestTags{Data: &TestTagsData{
		UploadUrl:        "http://testlink.dev.fwmrm.net/testlink/ui_asset/111_1311662179.mp4",
		ContentTypeId:    42,
		CreativeApiId:    100,
		Duration:         15,
		PhysicalDuration: 15.044,
		Bitrate:          1322,
		Height:           360,
		Width:            640,
		Fps:              23}}
	bytes, _ := m.JSON(ts)
	assert.Equal(t, "{\"data\":{\"bitrate\":1322,\"content_type_id\":42,\"creative_api_id\":100,"+
		"\"duration\":15,\"fps\":23,\"height\":360,\"id3tag\":\"\",\"metadata_only\":\"\","+
		"\"physical_duration\":15.043999671936035,\"upload_file\":\"\",\"upload_url\":\"http://testlink.dev."+
		"fwmrm.net/testlink/ui_asset/111_1311662179.mp4\",\"width\":640}}", string(bytes))
}
