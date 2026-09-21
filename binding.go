/*
 * Copyright © 2017 Xiao Zhang <zzxx513@gmail.com>.
 * Use of this source code is governed by an MIT-style
 * license that can be found in the LICENSE file.
 */
package turbo

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
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

// formValue looks fieldName up in req.Form, which holds the query for every
// request and additionally the form body for form requests.
func formValue(fieldName string, req *http.Request) (string, bool) {
	if req == nil || req.Form == nil {
		return "", false
	}
	for _, key := range lookupKeys(fieldName) {
		if v := req.Form[key]; len(v) > 0 {
			return v[0], true
		}
	}
	return "", false
}

// findValue resolves a field of a form request: an injected value wins over
// everything else, which is the whole point -- it is the only source the client
// did not supply.
//
// Everything else keeps the historical order. Path variables are not read here
// on purpose: parseRequestForm has already merged them into req.Form, in front
// of any value sharing their key, so consulting mux.Vars again would give the
// path a weight the rest of the framework does not give it.
func findValue(fieldName string, req *http.Request) (string, bool) {
	if injected, ok := InjectedValue(fieldName, req); ok {
		if form, ok := formValue(fieldName, req); ok && form != injected {
			log.Warnf("binding: %s <- injected %q, overriding the query/form value %q",
				fieldName, injected, form)
		}
		return injected, true
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
	raw map[string]json.RawMessage) {
	walkFields(theType, theValue, raw, func(index int, name string, fieldValue reflect.Value, raw map[string]json.RawMessage) {
		value, ok := formValue(name, req)
		if !ok || rawHas(raw, name) {
			return
		}
		logErrorIf(setValue(theType.Field(index).Type, fieldValue, value))
	})
}

// bindJSONInjected applies injected values last, so that they win over the body
// and over every path parameter. A body that carried the same field is reported:
// that is a caller trying to override something the server verified.
func bindJSONInjected(theType reflect.Type, theValue reflect.Value, req *http.Request,
	raw map[string]json.RawMessage) {
	walkFields(theType, theValue, raw, func(index int, name string, fieldValue reflect.Value, raw map[string]json.RawMessage) {
		value, ok := InjectedValue(name, req)
		if !ok {
			return
		}
		if rawHas(raw, name) {
			log.Warnf("binding: %s <- injected %q, overriding the request body value", name, value)
		}
		logErrorIf(setValue(theType.Field(index).Type, fieldValue, value))
	})
}
