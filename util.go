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
	"strings"
)

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

		filterProtoJson, ok := configs[filterProtoJson]
		if ok && filterProtoJson == "true" {
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
func FilterJsonWithStruct(jsonBytes []byte, structObj interface{}) ([]byte, error) {
	json, err := sjson.NewJson(jsonBytes)
	if err != nil {
		return jsonBytes, err
	}
	if reflect.TypeOf(structObj).Kind() == reflect.Ptr {
		err = filterStruct(json, reflect.TypeOf(structObj).Elem(), reflect.ValueOf(structObj).Elem())
	} else {
		err = filterStruct(json, reflect.TypeOf(structObj), reflect.ValueOf(structObj))
	}
	if err != nil {
		return jsonBytes, err
	}
	result, err := json.MarshalJSON()
	if err != nil {
		return jsonBytes, err
	}
	return result, nil
}

func filterStruct(structJson *sjson.Json, t reflect.Type, v reflect.Value) error {
	numField := t.NumField()
	for i := 0; i < numField; i++ {
		err := filterOf(t.Field(i).Type.Kind())(structJson, t.Field(i), v.Field(i))
		if err != nil {
			return err
		}
	}
	return nil
}

type fieldFilterFunc func(*sjson.Json, reflect.StructField, reflect.Value) error

func filterOf(kind reflect.Kind) fieldFilterFunc {
	// TODO make this configurable
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return intFieldFilter
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

func emptyFilter(*sjson.Json, reflect.StructField, reflect.Value) error {
	// do nothing
	return nil
}

func boolFieldFilter(structJson *sjson.Json, field reflect.StructField, v reflect.Value) error {
	jsonFieldName, _ := jsonFieldName(structJson, field)
	structJson.Set(jsonFieldName, v.Bool())
	return nil
}

func stringFieldFilter(structJson *sjson.Json, field reflect.StructField, v reflect.Value) error {
	jsonFieldName, _ := jsonFieldName(structJson, field)
	structJson.Set(jsonFieldName, v.String())
	return nil
}

func intFieldFilter(structJson *sjson.Json, field reflect.StructField, v reflect.Value) error {
	jsonFieldName, _ := jsonFieldName(structJson, field)
	structJson.Set(jsonFieldName, v.Int())
	return nil
}

func uintFieldFilter(structJson *sjson.Json, field reflect.StructField, v reflect.Value) error {
	jsonFieldName, _ := jsonFieldName(structJson, field)
	structJson.Set(jsonFieldName, v.Uint())
	return nil
}

func floatFieldFilter(structJson *sjson.Json, field reflect.StructField, v reflect.Value) error {
	jsonFieldName, _ := jsonFieldName(structJson, field)
	structJson.Set(jsonFieldName, v.Float())
	return nil
}

func ptrFieldFilter(structJson *sjson.Json, field reflect.StructField, v reflect.Value) error {
	jsonFieldName, err := jsonFieldName(structJson, field)
	if v.Elem().Kind() == reflect.Invalid {
		structJson.Set(jsonFieldName, nil)
	} else {
		if err != nil {
			structJson.Set(jsonFieldName, make(map[string]interface{}))
		}
		return filterStruct(structJson.Get(jsonFieldName), field.Type.Elem(), v.Elem())
	}
	return nil
}

func sliceFieldFilter(structJson *sjson.Json, field reflect.StructField, v reflect.Value) error {
	jsonFieldName, err := jsonFieldName(structJson, field)
	if err != nil {
		structJson.Set(jsonFieldName, make([]interface{}, 0))
	}
	if v.Len() == 0 {
		return nil
	}
	sliceJson := structJson.Get(jsonFieldName)
	arr, err := sliceJson.Array()
	if err != nil {
		fmt.Println(err.Error())
		return err
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

	if sliceInnerKind == reflect.Int64 {
		for i := 0; i < l; i++ {
			arr[i] = v.Index(i).Int()
		}
	}
	if sliceInnerKind == reflect.Ptr && field.Type.Elem().Elem().Kind() == reflect.Struct {
		for i := 0; i < l; i++ {
			if i >= arrLength {
				arr[i] = make(map[string]interface{})
			}
			err = filterStruct(sliceJson.GetIndex(i), v.Index(i).Type().Elem(), v.Index(i).Elem())
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func jsonFieldName(structJson *sjson.Json, field reflect.StructField) (string, error) {
	fieldName := field.Name
	_, ok := structJson.CheckGet(fieldName)
	if ok {
		return fieldName, nil
	}
	nameToSnake := ToSnakeCase(fieldName)
	_, ok = structJson.CheckGet(nameToSnake)
	if ok {
		return nameToSnake, nil
	}
	defaultName := fieldName
	nameInTag, err := lookupNameInTag(field)
	if err == nil {
		_, ok = structJson.CheckGet(nameInTag)
		if ok {
			return nameInTag, nil
		}
		defaultName = nameInTag
	}
	return defaultName, fmt.Errorf("fieldName [%s] not exist in json", fieldName)
}

func lookupNameInTag(field reflect.StructField) (string, error) {
	name, err := lookupNameInProtoTag(field)
	if err == nil {
		return name, nil
	}
	name, err = lookupNameInJsonTag(field)
	if err == nil {
		return name, nil
	}
	return "", errors.New("no name in tag")
}

func lookupNameInProtoTag(field reflect.StructField) (string, error) {
	protoTag := strings.TrimSpace(field.Tag.Get("protobuf"))
	if len(protoTag) > 0 {
		var prop proto.Properties
		prop.Parse(protoTag)
		if len(prop.OrigName) > 0 {
			return prop.OrigName, nil
		}
	}
	return "", errors.New("no such tag: protobuf")
}

func lookupNameInJsonTag(field reflect.StructField) (string, error) {
	jsonTag := strings.TrimSpace(field.Tag.Get("json"))
	if len(jsonTag) > 0 {
		if jsonTag == "-" {
			return "", errors.New("no name in json tag")
		}
		tagItems := strings.Split(jsonTag, ",")
		if len(tagItems[0]) > 0 {
			return tagItems[0], nil
		}
	}
	return "", errors.New("no such tag: json")
}
