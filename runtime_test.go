/*
 * Copyright © 2017 Xiao Zhang <zzxx513@gmail.com>.
 * Use of this source code is governed by an MIT-style
 * license that can be found in the LICENSE file.
 */
package turbo

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAnUnusableBodyIsNotEchoed pins T21. A body that cannot be parsed is the
// caller's mistake, and the message that says so is written to the service log as
// well as returned: it used to quote the body, so a token, a code or a signature
// in it ended up in the log file. The size and the parse error are what makes the
// failure actionable, and the parser already quotes the character that broke it.
func TestAnUnusableBodyIsNotEchoed(t *testing.T) {
	const secret = "SECRET-CARD-1234"
	body := `{"token":"` + secret + `", "broken":}`
	req := httptest.NewRequest(http.MethodPost, "/hello", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	err := BuildRequest(reloadServable{&Server{}}, &TestPrimitives{}, req)

	assert.Error(t, err)
	assert.NotContains(t, err.Error(), secret, "the request body must not reach the log")
	assert.NotContains(t, err.Error(), `"broken"`)
	assert.Contains(t, err.Error(), fmt.Sprintf("request body: %d bytes", len(body)))
	assert.Contains(t, err.Error(), "invalid character")
}

// TestANumberTheBodyCannotGiveIsNotEchoed pins the rest of T26. A body that
// parses can still carry a value the field cannot hold, and the message
// encoding/json builds for that quotes the number -- which used to reach the log
// with the rest of the parse error. Numbers are not tokens, but they are still
// request content.
func TestANumberTheBodyCannotGiveIsNotEchoed(t *testing.T) {
	const number = "99999999999999999999999"
	body := `{"int64Value":` + number + `}`
	req := httptest.NewRequest(http.MethodPost, "/hello", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	err := BuildRequest(reloadServable{&Server{}}, &TestPrimitives{}, req)

	assert.Error(t, err)
	assert.NotContains(t, err.Error(), number, "the value must not reach the log")
	assert.Contains(t, err.Error(), "number "+redactedValue)
	assert.Contains(t, err.Error(), "int64", "the reason has to survive")
}
