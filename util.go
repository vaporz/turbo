package turbo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	sjson "github.com/bitly/go-simplejson"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/mux"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var matchCamelCase = regexp.MustCompile("^([A-Z]+[a-z]*)+$")

func IsCamelCase(name string) bool {
	return matchCamelCase.Match([]byte(name))
}

func IsNotCamelCase(name string) bool {
	return !IsCamelCase(name)
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

// ToSnakeCase convert a camelCase string into a snake_case string
func ToSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

// ParseRequestForm prepares param values before further use,
// 1, run http.Request.ParseForm()
// 2, find keys with upper case characters, and append their values to a lower case key
// 3, merge route variables, route variables will come at the first place
func ParseRequestForm(req *http.Request) {
	req.ParseForm()
	mergeUpperCaseKeysToLowerCase(req)
	mergeMuxVars(req)
}

func mergeUpperCaseKeysToLowerCase(req *http.Request) {
	for k, vArr := range req.Form {
		if k == strings.ToLower(k) {
			continue
		}
		list := []string{}
		if lowerArr, ok := req.Form[strings.ToLower(k)]; ok {
			list = lowerArr
		}
		req.Form[strings.ToLower(k)] = append(list, vArr...)
		delete(req.Form, k)
	}
}

func mergeMuxVars(req *http.Request) {
	muxVars := mux.Vars(req)
	if muxVars == nil {
		return
	}
	for key, valueArr := range req.Form {
		if v, ok := muxVars[key]; ok {
			// route params comes first
			req.Form[key] = append([]string{v}, valueArr...)
			delete(muxVars, key)
		}
	}
	for key, value := range muxVars {
		req.Form[key] = []string{value}
	}
}

// JSON returns the json encoding of v,
// if v implements 'proto.Message', then FilterJsonWithStruct() is called, see comments for FilterJsonWithStruct(),
// otherwise, call encoding/json.Marshal()
func JSON(v interface{}) ([]byte, error) {
	if _, ok := v.(proto.Message); ok {
		var buf bytes.Buffer
		m := &jsonpb.Marshaler{}
		if err := m.Marshal(&buf, v.(proto.Message)); err != nil {
			return []byte{}, err
		}

		if Config.FilterProtoJson() {
			return FilterJsonWithStruct(buf.Bytes(), v)
		}
		return buf.Bytes(), nil
	}
	return json.Marshal(v)
}

// FilterJsonWithStruct walks through a struct, comparing each struct field with the key which have
// same name('fieldName'=='field_name') in json, and change the json by:
// 1, if struct field type is 'int64', then change the value in Json into a number
// 2, if field type is 'Ptr', and field value is 'nil', then set "[key_name]":null in Json
// 3, if any key in json is missing, set zero value to that key
// The reason why we do this is
// (a) protobuf parse int64 as string,
// (b) a Key with a nil Ptr value is missing in the json marshaled by github.com/golang/protobuf/jsonpb.Marshaler
// So, this func is a 'patch' to protobuf
func FilterJsonWithStruct(jsonBytes []byte, structObj interface{}) (bytes []byte, e error) {
	defer func() {
		if err := recover(); err != nil {
			bytes = jsonBytes
			e = errors.New("panic in FilterJsonWithStruct()! Error:" + err.(error).Error())
		}
	}()
	json, err := sjson.NewJson(jsonBytes)
	if err != nil {
		panic(err)
	}
	if reflect.TypeOf(structObj).Kind() == reflect.Ptr {
		filterStruct(json, reflect.TypeOf(structObj).Elem(), reflect.ValueOf(structObj).Elem())
	} else {
		filterStruct(json, reflect.TypeOf(structObj), reflect.ValueOf(structObj))
	}
	result, err := json.MarshalJSON()
	if err != nil {
		panic(err)
	}
	return result, nil
}

func filterStruct(structJson *sjson.Json, t reflect.Type, v reflect.Value) {
	numField := t.NumField()
	for i := 0; i < numField; i++ {
		filterOf(t.Field(i).Type.Kind())(structJson, t.Field(i), v.Field(i))
	}
}

type fieldFilterFunc func(*sjson.Json, reflect.StructField, reflect.Value)

func filterOf(kind reflect.Kind) fieldFilterFunc {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return intFieldFilter
	case reflect.Int64:
		return int64FieldFilter
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return uintFieldFilter
	case reflect.Float32, reflect.Float64:
		return floatFieldFilter
	case reflect.Ptr:
		return ptrFieldFilter
	case reflect.Slice:
		return sliceFieldFilter
	case reflect.Bool:
		return boolFieldFilter
	case reflect.String:
		return stringFieldFilter
	default:
		return emptyFilter
	}
}

func emptyFilter(*sjson.Json, reflect.StructField, reflect.Value) {
	// do nothing
}

func boolFieldFilter(structJson *sjson.Json, field reflect.StructField, v reflect.Value) {
	jsonFieldName, ok := jsonFieldName(structJson, field)
	if ok || Config.FilterProtoJsonEmitZeroValues() {
		structJson.Set(jsonFieldName, v.Bool())
	}
}

func stringFieldFilter(structJson *sjson.Json, field reflect.StructField, v reflect.Value) {
	jsonFieldName, ok := jsonFieldName(structJson, field)
	if ok || Config.FilterProtoJsonEmitZeroValues() {
		structJson.Set(jsonFieldName, v.String())
	}
}

func intFieldFilter(structJson *sjson.Json, field reflect.StructField, v reflect.Value) {
	jsonFieldName, ok := jsonFieldName(structJson, field)
	if ok || Config.FilterProtoJsonEmitZeroValues() {
		structJson.Set(jsonFieldName, v.Int())
	}
}

func int64FieldFilter(structJson *sjson.Json, field reflect.StructField, v reflect.Value) {
	jsonFieldName, _ := jsonFieldName(structJson, field)
	if Config.FilterProtoJsonInt64AsNumber() && Config.FilterProtoJsonEmitZeroValues() {
		structJson.Set(jsonFieldName, v.Int())
	} else {
		if v.Int() == 0 && Config.FilterProtoJsonEmitZeroValues() {
			if Config.FilterProtoJsonInt64AsNumber() {
				structJson.Set(jsonFieldName, 0)
			} else {
				structJson.Set(jsonFieldName, "0")
			}
		} else if v.Int() != 0 {
			if Config.FilterProtoJsonInt64AsNumber() {
				structJson.Set(jsonFieldName, v.Int())
			} else {
				structJson.Set(jsonFieldName, strconv.FormatInt(v.Int(), 10))
			}
		}
	}
}

func uintFieldFilter(structJson *sjson.Json, field reflect.StructField, v reflect.Value) {
	jsonFieldName, ok := jsonFieldName(structJson, field)
	if ok || Config.FilterProtoJsonEmitZeroValues() {
		structJson.Set(jsonFieldName, v.Uint())
	}
}

func floatFieldFilter(structJson *sjson.Json, field reflect.StructField, v reflect.Value) {
	jsonFieldName, ok := jsonFieldName(structJson, field)
	if ok || Config.FilterProtoJsonEmitZeroValues() {
		structJson.Set(jsonFieldName, v.Float())
	}
}

func ptrFieldFilter(structJson *sjson.Json, field reflect.StructField, v reflect.Value) {
	jsonFieldName, ok := jsonFieldName(structJson, field)
	if v.Elem().Kind() == reflect.Invalid && Config.FilterProtoJsonEmitZeroValues() {
		structJson.Set(jsonFieldName, nil)
	} else {
		if !ok {
			structJson.Set(jsonFieldName, make(map[string]interface{}))
		}
		filterStruct(structJson.Get(jsonFieldName), field.Type.Elem(), v.Elem())
	}
}

func sliceFieldFilter(structJson *sjson.Json, field reflect.StructField, v reflect.Value) {
	jsonFieldName, ok := jsonFieldName(structJson, field)
	if !ok && Config.FilterProtoJsonEmitZeroValues() {
		structJson.Set(jsonFieldName, make([]interface{}, 0))
	}
	if v.Len() == 0 {
		return
	}
	sliceJson := structJson.Get(jsonFieldName)
	arr, err := sliceJson.Array()
	if err != nil {
		panic(err)
	}
	sliceInnerKind := field.Type.Elem().Kind()
	l := v.Len()
	arrLength := len(arr)
	if arrLength < l {
		newArr := make([]interface{}, l)
		copy(newArr, arr)
		structJson.Set(jsonFieldName, newArr)
		arr = newArr
		sliceJson = structJson.Get(jsonFieldName)
	}

	if sliceInnerKind == reflect.Int64 && Config.FilterProtoJsonInt64AsNumber() {
		for i := 0; i < l; i++ {
			arr[i] = v.Index(i).Int()
		}
	}
	if sliceInnerKind == reflect.Ptr && field.Type.Elem().Elem().Kind() == reflect.Struct {
		for i := 0; i < l; i++ {
			if i >= arrLength {
				arr[i] = make(map[string]interface{})
			}
			filterStruct(sliceJson.GetIndex(i), v.Index(i).Type().Elem(), v.Index(i).Elem())
		}
	}
}

func jsonFieldName(structJson *sjson.Json, field reflect.StructField) (string, bool) {
	fieldName := field.Name
	_, ok := structJson.CheckGet(fieldName)
	if ok {
		return fieldName, true
	}
	nameToSnake := ToSnakeCase(fieldName)
	_, ok = structJson.CheckGet(nameToSnake)
	if ok {
		return nameToSnake, true
	}
	nameInTag, ok := lookupNameInTag(field)
	if ok {
		_, ok = structJson.CheckGet(nameInTag)
		return nameInTag, ok
	}
	return fieldName, false
}

func lookupNameInTag(field reflect.StructField) (string, bool) {
	name, ok := lookupNameInProtoTag(field)
	if ok {
		return name, true
	}
	name, ok = lookupNameInJsonTag(field)
	if ok {
		return name, true
	}
	fmt.Print("no name in tag")
	return "", false
}

func lookupNameInProtoTag(field reflect.StructField) (string, bool) {
	protoTag := strings.TrimSpace(field.Tag.Get("protobuf"))
	if len(protoTag) > 0 {
		var prop proto.Properties
		prop.Parse(protoTag)
		if len(prop.OrigName) > 0 {
			return prop.OrigName, true
		}
	}
	fmt.Print("no such tag: protobuf")
	return "", false
}

func lookupNameInJsonTag(field reflect.StructField) (string, bool) {
	jsonTag := strings.TrimSpace(field.Tag.Get("json"))
	if len(jsonTag) > 0 {
		if jsonTag == "-" {
			return "", false
		}
		tagItems := strings.Split(jsonTag, ",")
		if len(tagItems[0]) > 0 {
			return tagItems[0], true
		}
	}
	fmt.Print("no such tag: json")
	return "", false
}
