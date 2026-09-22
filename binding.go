/*
 * Copyright © 2017 Xiao Zhang <zzxx513@gmail.com>.
 * Use of this source code is governed by an MIT-style
 * license that can be found in the LICENSE file.
 */
package turbo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"

	"github.com/gorilla/mux"
)

// Parameter binding considers every source a request can carry a value in and
// resolves conflicts by precedence:
//
//	injected > path > body > query/form
//
// Injected values come first because they are the only ones the client did not
// supply: whatever an interceptor established after verifying a signature,
// resolving a device code or authenticating a user describes the request more
// reliably than anything the caller sent. The path comes next, because the route
// already matched on it. Query and form values come last, and only fill the
// fields the body did not mention.
//
// Binding used to couple the set of sources to the Content-Type instead: a JSON
// request was bound from its body alone, so an injected value disappeared
// without a trace, and a form request let query/form win over an injected value,
// so a caller could override a value the server had verified.

// injectedParamsKey keys the per-request store of injected values. A struct type
// is used so that no other package can collide with it.
type injectedParamsKey struct{}

type injectedParams struct {
	values map[string]string
}

// InjectParam records a value the server itself established for this request --
// typically an interceptor that verified a signature, resolved a device code or
// authenticated a user.
//
// It is the supported replacement for writing to req.Form directly, which only
// ever worked for form requests: the JSON binding path does not consult req.Form,
// so an injected value silently disappeared. An injected value now wins over the
// body, the path and the query, so a request can no longer override a value the
// server verified.
//
// InjectParam updates req in place, so the caller must keep using the same
// *http.Request afterwards -- exactly as it already does when it calls
// req.Form.Set.
func InjectParam(req *http.Request, key, value string) {
	if req == nil || key == "" {
		return
	}
	injected := injectedParamsFrom(req)
	if injected == nil {
		injected = &injectedParams{values: make(map[string]string)}
		*req = *req.WithContext(context.WithValue(req.Context(), injectedParamsKey{}, injected))
	}
	injected.values[normalizeKey(key)] = value
}

// normalizeKey makes the spelling of a parameter irrelevant: deviceCode,
// device_code, DeviceCode and DEVICE_CODE all name the same parameter. Injected
// values are named by hand by a service author, so they are matched this way
// rather than by the historical three-spelling list.
func normalizeKey(key string) string {
	return strings.ToLower(strings.ReplaceAll(key, "_", ""))
}

func injectedParamsFrom(req *http.Request) *injectedParams {
	if req == nil {
		return nil
	}
	injected, _ := req.Context().Value(injectedParamsKey{}).(*injectedParams)
	return injected
}

// InjectedValue returns the value injected for fieldName, if there is one. Both
// the store filled by InjectParam and plain values placed in the request context
// are consulted, because the latter was the only mechanism available before
// InjectParam existed.
func InjectedValue(fieldName string, req *http.Request) (string, bool) {
	if req == nil {
		return "", false
	}
	if injected := injectedParamsFrom(req); injected != nil {
		for _, key := range lookupKeys(fieldName) {
			if v, ok := injected.values[normalizeKey(key)]; ok && v != "" {
				return v, true
			}
		}
	}
	for _, key := range lookupKeys(fieldName) {
		if v, ok := req.Context().Value(key).(string); ok && v != "" {
			return v, true
		}
	}
	return "", false
}

// lookupKeys lists the spellings a field name is probed with, in order: the Go
// field name, its lower case form and its snake case form. Services have always
// injected and called under all three, so all three keep working.
func lookupKeys(fieldName string) []string {
	return []string{fieldName, strings.ToLower(fieldName), ToSnakeCase(fieldName)}
}

// pathValue looks fieldName up in the variables the route captured. mux.Vars is
// safe to read here: parseRequestForm copies the variables into req.Form but
// leaves the map itself alone.
func pathValue(fieldName string, req *http.Request) (string, bool) {
	if req == nil {
		return "", false
	}
	return findPathParamValue(fieldName, mux.Vars(req))
}

// formValue looks fieldName up in req.Form, which holds the query for every
// request and additionally the form body for form requests. The spellings a
// field has always been probed with are tried first, so the value a caller gets
// cannot change merely because two keys normalise to the same name; only then
// does a spelling insensitive pass run, so that yourName, your_name and
// YOURNAME all reach the same field.
func formValue(fieldName string, req *http.Request) (string, bool) {
	if req == nil || req.Form == nil {
		return "", false
	}
	for _, key := range lookupKeys(fieldName) {
		if v := req.Form[key]; len(v) > 0 {
			return v[0], true
		}
	}
	return normalisedFormValue(fieldName, req)
}

// normalisedFormValue compares keys with their spelling removed. Keys are visited
// in sorted order so that the result does not depend on map iteration order.
func normalisedFormValue(fieldName string, req *http.Request) (string, bool) {
	wanted := normalizeKey(fieldName)
	keys := make([]string, 0, len(req.Form))
	for key := range req.Form {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if normalizeKey(key) != wanted {
			continue
		}
		if v := req.Form[key]; len(v) > 0 {
			return v[0], true
		}
	}
	return "", false
}

// findValue resolves a field: an injected value wins over everything else,
// because it is the only source the client did not supply, and the path wins
// over the query, because the route already matched on it. The path is read from
// the route variables rather than from req.Form, so which source wins no longer
// depends on the spelling the caller happened to use -- a query parameter was
// merged into req.Form under one particular spelling, and if the caller chose
// another one the route variable used to lose.
func findValue(fieldName string, req *http.Request) (string, bool) {
	if injected, ok := InjectedValue(fieldName, req); ok {
		if form, ok := formValue(fieldName, req); ok && form != injected {
			log.Warnf("binding: %s <- injected %q, overriding the query/form value %q",
				fieldName, injected, form)
		}
		return injected, true
	}
	if value, ok := pathValue(fieldName, req); ok {
		return value, true
	}
	return formValue(fieldName, req)
}

// jsonObjectKeys parses body as a JSON object and returns its members keyed by
// lower case name. It is how the binding path knows which fields the body
// actually mentioned, so that query/form values only fill the gaps and an
// injected value can report what it overrides.
func jsonObjectKeys(body string) map[string]json.RawMessage {
	if strings.TrimSpace(body) == "" {
		return nil
	}
	var members map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &members); err != nil {
		return nil
	}
	return lowerKeys(members)
}

// lowerKeys rekeys a JSON object by lower case name, so that lookups do not
// depend on whether the client sent camelCase or snake_case.
func lowerKeys(members map[string]json.RawMessage) map[string]json.RawMessage {
	if members == nil {
		return nil
	}
	keys := make(map[string]json.RawMessage, len(members))
	for k, v := range members {
		keys[strings.ToLower(k)] = v
	}
	return keys
}

// rawHas reports whether the JSON object mentioned fieldName, under any of the
// spellings a field name is probed with -- jsonpb emits camelCase by default
// while the proto field name may be snake_case.
func rawHas(raw map[string]json.RawMessage, fieldName string) bool {
	if raw == nil {
		return false
	}
	for _, key := range lookupKeys(fieldName) {
		if _, ok := raw[strings.ToLower(key)]; ok {
			return true
		}
	}
	return false
}

// rawSub returns the JSON object nested under fieldName, so that the walk can
// descend into a struct the body carried.
func rawSub(raw map[string]json.RawMessage, fieldName string) map[string]json.RawMessage {
	if raw == nil {
		return nil
	}
	for _, key := range lookupKeys(fieldName) {
		v, ok := raw[strings.ToLower(key)]
		if !ok {
			continue
		}
		var sub map[string]json.RawMessage
		if err := json.Unmarshal(v, &sub); err != nil {
			return nil
		}
		return lowerKeys(sub)
	}
	return nil
}

// walkFields visits every exported scalar field of theType, descending into
// nested struct pointers, and hands the callback the JSON object of the level it
// is currently at.
func walkFields(theType reflect.Type, theValue reflect.Value, raw map[string]json.RawMessage,
	visit func(index int, name string, fieldValue reflect.Value, raw map[string]json.RawMessage)) {
	fieldNum := theType.NumField()
	for i := 0; i < fieldNum; i++ {
		name := theType.Field(i).Name
		if name == "" || name[0] < 'A' || name[0] > 'Z' {
			continue
		}
		fieldValue := theValue.FieldByName(name)
		if fieldValue.Kind() == reflect.Ptr && fieldValue.Type().Elem().Kind() == reflect.Struct {
			if !fieldValue.IsNil() {
				walkFields(fieldValue.Type().Elem(), fieldValue.Elem(), rawSub(raw, name), visit)
			}
			continue
		}
		visit(i, name, fieldValue, raw)
	}
}

// bindJSONGaps fills the fields a JSON body did not mention from query/form
// values. A JSON request now considers the URL as well, so a parameter the
// caller put in the query is no longer silently dropped; the body still wins for
// the fields it carries.
func bindJSONGaps(theType reflect.Type, theValue reflect.Value, req *http.Request,
	raw map[string]json.RawMessage) error {
	var failure error
	walkFields(theType, theValue, raw, func(index int, name string, fieldValue reflect.Value, raw map[string]json.RawMessage) {
		if failure != nil {
			return
		}
		value, ok := formValue(name, req)
		if !ok || rawHas(raw, name) {
			return
		}
		if err := setValue(theType.Field(index).Type, fieldValue, value); err != nil {
			failure = bindingErrorFor(name, value, req, err)
		}
	})
	return failure
}

// bindJSONInjected applies injected values last, so that they win over the body
// and over every path parameter. A body that carried the same field is reported:
// that is a caller trying to override something the server verified.
func bindJSONInjected(theType reflect.Type, theValue reflect.Value, req *http.Request,
	raw map[string]json.RawMessage) error {
	var failure error
	walkFields(theType, theValue, raw, func(index int, name string, fieldValue reflect.Value, raw map[string]json.RawMessage) {
		if failure != nil {
			return
		}
		value, ok := InjectedValue(name, req)
		if !ok {
			return
		}
		if rawHas(raw, name) {
			log.Warnf("binding: %s <- injected %q, overriding the request body value", name, value)
		}
		if err := setValue(theType.Field(index).Type, fieldValue, value); err != nil {
			failure = bindingErrorFor(name, value, req, err)
		}
	})
	return failure
}

// bindingErrorFor describes a value the request carries but the field cannot
// take. The status says whose mistake it is: a value the server injected is the
// server's, everything else came from the caller.
func bindingErrorFor(fieldName, value string, req *http.Request, cause error) error {
	status := http.StatusBadRequest
	source := "query/form parameter"
	if _, ok := InjectedValue(fieldName, req); ok {
		status = http.StatusInternalServerError
		source = "injected value"
	} else if _, ok := pathValue(fieldName, req); ok {
		source = "path parameter"
	}
	return WithStatus(fmt.Errorf("turbo: cannot bind %s from %s %q: %w", fieldName, source, value, cause), status)
}

// A thrift method takes a list of arguments rather than one request message, so
// a JSON body names them:
//
//	{"values": {...}, "yourName": "a name", "int64Value": 7}
//
// Those are the names the generated Args struct already carries in its json and
// thrift tags, and the same ones the form branch binds to. Building every
// argument from the body is what the generated switcher expects; the old code
// unmarshalled the whole body into the first argument and returned one value, so
// any method with more than one argument panicked with an index out of range.
type thriftBody struct {
	raw   map[string]json.RawMessage
	byKey map[string]string // normalised name -> the key the caller wrote
	used  map[string]bool
}

func decodeThriftBody(body []byte) (*thriftBody, error) {
	b := &thriftBody{
		raw:   map[string]json.RawMessage{},
		byKey: map[string]string{},
		used:  map[string]bool{},
	}
	if len(bytes.TrimSpace(body)) == 0 {
		// an empty body names no argument at all
		return b, nil
	}
	if err := json.Unmarshal(body, &b.raw); err != nil {
		return nil, err
	}
	for key := range b.raw {
		b.byKey[normalizeKey(key)] = key
	}
	return b, nil
}

// argumentNames lists the names one argument may be called by, including the
// spelling variants normaliseKey makes equivalent.
func argumentNames(field reflect.StructField) []string {
	names := []string{field.Name}
	for _, tag := range []string{"json", "thrift"} {
		if value := field.Tag.Get(tag); value != "" {
			names = append(names, strings.Split(value, ",")[0])
		}
	}
	return names
}

// take returns the body member naming this argument and marks it as consumed.
func (b *thriftBody) take(field reflect.StructField) (json.RawMessage, bool) {
	for _, name := range argumentNames(field) {
		key, ok := b.byKey[normalizeKey(name)]
		if !ok {
			continue
		}
		b.used[key] = true
		return b.raw[key], true
	}
	return nil, false
}

// unused lists the keys that name no argument of the method.
func (b *thriftBody) unused() []string {
	var left []string
	for key := range b.raw {
		if !b.used[key] {
			left = append(left, key)
		}
	}
	sort.Strings(left)
	return left
}

func argumentNameList(theType reflect.Type) []string {
	names := make([]string, 0, theType.NumField())
	for i := 0; i < theType.NumField(); i++ {
		names = append(names, theType.Field(i).Name)
	}
	sort.Strings(names)
	return names
}

// bindThriftArgsFromJSON fills a thrift Args struct from a JSON body, then from
// the sources that are not the body, in the order the grpc path uses: query/form
// fills what the body did not name, the path overrides it, and an injected value
// overrides everything.
//
// A thrift method takes a list of arguments rather than one request message, so
// a body with more than one argument to fill names them:
//
//	{"values": {...}, "yourName": "a name", "int64Value": 7}
//
// A method with a single argument has no other sensible reading: there the body
// is that argument, which is what it has always been for a single argument
// method. The rule depends only on the method signature, never on the content of
// the body, so there is nothing to guess.
//
// An argument the body does not name keeps its zero value. A key that names no
// argument, or no field of the single argument, is refused rather than ignored:
// a misspelling would otherwise look exactly like a caller who sent nothing,
// which is the failure mode T3 removed from the other binding path.
func bindThriftArgsFromJSON(args reflect.Value, body []byte, req *http.Request) error {
	theType := args.Type()
	if theType.NumField() == 1 {
		return bindSingleThriftArg(args.Field(0), theType.Field(0), body, req)
	}

	parsed, err := decodeThriftBody(body)
	if err != nil {
		return WithStatus(fmt.Errorf("turbo: the request body is not a JSON object naming the arguments of "+
			"this method (%v): %w", argumentNameList(theType), err), http.StatusBadRequest)
	}
	fromBody := make([]bool, theType.NumField())
	left, err := bindStructFieldsFromBody(args, parsed, fromBody)
	if err != nil {
		return err
	}
	if len(left) > 0 {
		return WithStatus(fmt.Errorf("turbo: the request body names %v, which are not arguments of this method (%v)",
			left, argumentNameList(theType)), http.StatusBadRequest)
	}

	for i := 0; i < theType.NumField(); i++ {
		name := theType.Field(i).Name
		field := args.Field(i)
		if isNestedArgument(field) {
			// the argument is a message of its own: bind inside it, skipping a
			// pointer the body did not provide rather than inventing one
			if field.IsNil() {
				continue
			}
			if err := bindThriftArgSources(field.Type().Elem(), field.Elem(), req,
				lowerKeys(mustObject(bodyKeyFor(parsed, theType.Field(i)))), nil); err != nil {
				return err
			}
			continue
		}
		if err := bindScalarArg(field, name, fromBody[i], req); err != nil {
			return err
		}
	}
	return nil
}

// bindSingleThriftArg fills the one argument of a single argument method. The
// body is that argument: an object when the argument is a message, a bare JSON
// value when it is not.
func bindSingleThriftArg(field reflect.Value, structField reflect.StructField, body []byte, req *http.Request) error {
	if !isNestedArgument(field) {
		if err := json.Unmarshal(body, field.Addr().Interface()); err != nil {
			return WithStatus(fmt.Errorf("turbo: cannot read thrift argument %s from the request body: %w",
				structField.Name, err), http.StatusBadRequest)
		}
		return bindScalarArg(field, structField.Name, true, req)
	}

	parsed, err := decodeThriftBody(body)
	if err != nil {
		return WithStatus(fmt.Errorf("turbo: the request body is not a JSON object holding the fields of %s: %w",
			structField.Name, err), http.StatusBadRequest)
	}
	argument := reflect.New(field.Type().Elem())
	left, err := bindStructFieldsFromBody(argument.Elem(), parsed, nil)
	if err != nil {
		return err
	}
	if len(left) > 0 {
		return WithStatus(fmt.Errorf("turbo: the request body names %v, which are not fields of the "+
			"argument %s (%v)", left, structField.Name, fieldNameList(field.Type().Elem())), http.StatusBadRequest)
	}
	field.Set(argument)

	return bindThriftArgSources(field.Type().Elem(), field.Elem(), req, lowerKeys(mustObject(body)), nil)
}

// bindThriftArgSources applies the sources that are not the body to a message
// argument: query/form fills what the body did not name, then the path, then an
// injected value.
func bindThriftArgSources(theType reflect.Type, theValue reflect.Value, req *http.Request,
	raw map[string]json.RawMessage, _ []bool) error {
	if err := bindJSONGaps(theType, theValue, req, raw); err != nil {
		return err
	}
	if err := setPathParams(theType, theValue, req); err != nil {
		return err
	}
	return bindJSONInjected(theType, theValue, req, raw)
}

// bindScalarArg fills an argument that is a value rather than a message: query
// and form values name the argument itself, then the path overrides it, then an
// injected value overrides everything.
func bindScalarArg(field reflect.Value, name string, fromBody bool, req *http.Request) error {
	if !fromBody {
		if value, ok := formValue(name, req); ok {
			if err := setValue(field.Type(), field, value); err != nil {
				return bindingErrorFor(name, value, req, err)
			}
		}
	}
	if value, ok := pathValue(name, req); ok {
		if err := setValue(field.Type(), field, value); err != nil {
			return bindingErrorFor(name, value, req, err)
		}
	}
	if value, ok := InjectedValue(name, req); ok {
		if err := setValue(field.Type(), field, value); err != nil {
			return bindingErrorFor(name, value, req, err)
		}
	}
	return nil
}

// bindStructFieldsFromBody fills the fields of a struct from a decoded body,
// recording which of them the body named and returning the keys that name no
// field at all.
func bindStructFieldsFromBody(structValue reflect.Value, parsed *thriftBody, named []bool) ([]string, error) {
	theType := structValue.Type()
	for i := 0; i < theType.NumField(); i++ {
		raw, ok := parsed.take(theType.Field(i))
		if !ok {
			continue
		}
		if err := json.Unmarshal(raw, structValue.Field(i).Addr().Interface()); err != nil {
			return nil, WithStatus(fmt.Errorf("turbo: cannot read %s from the request body: %w",
				theType.Field(i).Name, err), http.StatusBadRequest)
		}
		if named != nil {
			named[i] = true
		}
	}
	return parsed.unused(), nil
}

// bodyKeyFor returns the body member an argument was read from.
func bodyKeyFor(parsed *thriftBody, field reflect.StructField) json.RawMessage {
	for _, name := range argumentNames(field) {
		if key, ok := parsed.byKey[normalizeKey(name)]; ok {
			return parsed.raw[key]
		}
	}
	return nil
}

func fieldNameList(theType reflect.Type) []string {
	names := make([]string, 0, theType.NumField())
	for i := 0; i < theType.NumField(); i++ {
		names = append(names, theType.Field(i).Name)
	}
	sort.Strings(names)
	return names
}

// isNestedArgument reports whether an argument is a message of its own rather
// than a value the request can spell out directly.
func isNestedArgument(field reflect.Value) bool {
	return field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.Struct
}

// mustObject decodes a raw JSON object, returning nil when it is absent or is
// not an object.
func mustObject(raw json.RawMessage) map[string]json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	var members map[string]json.RawMessage
	if err := json.Unmarshal(raw, &members); err != nil {
		return nil
	}
	return members
}
