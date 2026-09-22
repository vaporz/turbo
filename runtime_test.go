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
