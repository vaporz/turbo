/*
 * Copyright © 2017 Xiao Zhang <zzxx513@gmail.com>.
 * Use of this source code is governed by an MIT-style
 * license that can be found in the LICENSE file.
 */
package turbo

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorfCarriesAStatus(t *testing.T) {
	err := Errorf(http.StatusForbidden, "user %d may not do that", 7)
	assert.Equal(t, "user 7 may not do that", err.Error())
	assert.Equal(t, http.StatusForbidden, StatusOf(err))
}

func TestWithStatus(t *testing.T) {
	// a plain error takes the status it is given
	err := WithStatus(errors.New("boom"), http.StatusBadRequest)
	assert.Equal(t, "boom", err.Error())
	assert.Equal(t, http.StatusBadRequest, StatusOf(err))

	// an error that already carries one keeps it, so wrapping cannot overwrite
	// the decision its author made
	assert.Equal(t, http.StatusNotFound, StatusOf(WithStatus(Errorf(http.StatusNotFound, "gone"), http.StatusBadRequest)))

	// nil stays nil
	assert.Nil(t, WithStatus(nil, http.StatusBadRequest))
}

func TestStatusOfLooksThroughWrapping(t *testing.T) {
	wrapped := fmt.Errorf("error in preprocessor: %w", Errorf(http.StatusTeapot, "teapot"))
	assert.Equal(t, http.StatusTeapot, StatusOf(wrapped))

	assert.Equal(t, 0, StatusOf(errors.New("nothing in particular")))
	assert.Equal(t, 0, StatusOf(nil))
}
