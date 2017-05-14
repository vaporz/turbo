package turbo

import (
	"github.com/gorilla/mux"
	"net/http"
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
