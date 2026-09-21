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
)

// statusError is an error that also says which HTTP status the response should
// carry.
type statusError struct {
	status int
	err    error
}

func (e *statusError) Error() string { return e.err.Error() }

func (e *statusError) Unwrap() error { return e.err }

// Errorf returns an error that asks the error handler to answer with status
// instead of the default 500.
//
// The HTTP status is what everything in front of a service reads: a load
// balancer, a gateway, an alert rule, a dashboard. Answering every failure with
// 500 -- which is what turbo used to do -- makes "the client sent a bad request"
// indistinguishable from "the service is broken", so a real outage drowns in
// business errors. A service that knows what went wrong can now say so:
//
//	return turbo.Errorf(http.StatusForbidden, "user %d may not do that", id)
func Errorf(status int, format string, args ...interface{}) error {
	return &statusError{status: status, err: fmt.Errorf(format, args...)}
}

// WithStatus attaches status to err. An error that already carries a status keeps
// it, so wrapping an error cannot silently overwrite the decision its author
// made.
func WithStatus(err error, status int) error {
	if err == nil {
		return nil
	}
	if StatusOf(err) != 0 {
		return err
	}
	return &statusError{status: status, err: err}
}

// StatusOf returns the HTTP status err asks for, or 0 when it asks for nothing
// in particular. It looks through wrapping, so a status set deep inside a chain
// of errors is still found.
func StatusOf(err error) int {
	var status *statusError
	if errors.As(err, &status) {
		return status.status
	}
	return 0
}

// defaultErrorHandler answers with the status the error carries, and with 500
// when it carries none.
func defaultErrorHandler(resp http.ResponseWriter, req *http.Request, err error) {
	log.Error(err.Error())
	status := StatusOf(err)
	if status == 0 {
		status = http.StatusInternalServerError
	}
	http.Error(resp, err.Error(), status)
}
