package turbo

import (
	"github.com/gorilla/mux"
	"net/http"
	"regexp"
	"strings"
	"reflect"
	"fmt"
	sjson "github.com/bitly/go-simplejson"
	"strconv"
	"errors"
	"github.com/golang/protobuf/proto"
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

// FilterProtoJson walks through a Json(marshaled by ), comparing each key with the field which have same name in protoMessage,
// 1, if field type is 'int64', then change the value in Json into a number
// 2, if field type is 'Ptr', and field value is 'nil', then append a "[key_name]":null in Json
func FilterProtoJson(jsonBytes []byte, protoMessage interface{}) ([]byte, error) {
	json, err := sjson.NewJson(jsonBytes)
	if err != nil {
		return jsonBytes, err
	}
	err = filterStruct(json, reflect.TypeOf(protoMessage).Elem(), reflect.ValueOf(protoMessage).Elem())
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
		err := structFieldFilter(t.Field(i))(structJson, t.Field(i), v.Field(i))
		if err != nil {
			return err
		}
	}
	return nil
}

type fieldFilterFunc func(*sjson.Json, reflect.StructField, reflect.Value) error

func structFieldFilter(field reflect.StructField) fieldFilterFunc {
	// TODO make this configurable
	switch field.Type.Kind() {
	case reflect.Int32, reflect.Int64, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		return int64FieldFilter
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
	jsonFieldName, err := jsonFieldName(structJson, field)
	if err != nil {
		structJson.Set(jsonFieldName, false)
	}
	return nil
}

func stringFieldFilter(structJson *sjson.Json, field reflect.StructField, v reflect.Value) error {
	jsonFieldName, err := jsonFieldName(structJson, field)
	if err != nil {
		structJson.Set(jsonFieldName, "")
	}
	return nil
}

func int64FieldFilter(structJson *sjson.Json, field reflect.StructField, v reflect.Value) error {
	jsonFieldName, _ := jsonFieldName(structJson, field)
	asInt64, err := int64Value(structJson, jsonFieldName)
	if err != nil {
		return err
	}
	structJson.Set(jsonFieldName, asInt64)
	return nil
}

func int64Value(structJson *sjson.Json, jsonFieldName string) (int64, error) {
	fieldJson, ok := structJson.CheckGet(jsonFieldName)
	if !ok {
		return 0, nil
	}
	int64Number, err := fieldJson.Int64()
	if err == nil {
		return int64Number, nil
	}
	int64Str, err := fieldJson.String()
	if err != nil {
		return 0, err
	}
	int64Value, err := strconv.ParseInt(int64Str, 10, 64)
	if err != nil {
		return 0, err
	}
	return int64Value, nil
}

func ptrFieldFilter(structJson *sjson.Json, field reflect.StructField, v reflect.Value) error {
	jsonFieldName, _ := jsonFieldName(structJson, field)
	if v.Elem().Kind() == reflect.Invalid {
		structJson.Set(jsonFieldName, nil)
	} else {
		return filterStruct(structJson.Get(jsonFieldName), field.Type.Elem(), v.Elem())
	}
	return nil
}

func sliceFieldFilter(structJson *sjson.Json, field reflect.StructField, v reflect.Value) error {
	jsonFieldName, _ := jsonFieldName(structJson, field)
	if v.Len() == 0 {
		structJson.Set(jsonFieldName, make([]int64, 0))
		return nil
	}
	sliceInnerKind := field.Type.Elem().Kind()
	sliceJson := structJson.Get(jsonFieldName)
	arr, err := sliceJson.Array()
	if err != nil {
		return err
	}
	for i, item := range arr {
		if sliceInnerKind == reflect.Int64 {
			intValue, err := strconv.ParseInt(item.(string), 10, 64)
			if err != nil {
				return err
			}
			arr[i] = intValue
		}
		if sliceInnerKind == reflect.Ptr && field.Type.Elem().Elem().Kind() == reflect.Struct {
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
	defaultName := nameToSnake
	protoTag := field.Tag.Get("protobuf")
	if len(protoTag) > 0 {
		var prop proto.Properties
		prop.Parse(protoTag)
		if len(prop.OrigName) > 0 {
			defaultName = prop.OrigName
		}
	}
	return defaultName, errors.New(fmt.Sprintf("fieldName [%s] not exist in json", fieldName))
}
