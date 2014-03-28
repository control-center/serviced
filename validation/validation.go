// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package validation

import (
	"fmt"
)

type ValidationError struct {
	Errors []error
}

func NewValidationError() *ValidationError {
	return &ValidationError{make([]error, 0)}
}

func (v *ValidationError) AddViolation(violationMsg string) {
	v.Add(NewViolation(violationMsg))
}

func (v *ValidationError) Add(err error) {
	if err != nil {
		v.Errors = append(v.Errors, err)
	}
}

func (v *ValidationError) Error() string {
	errString := "ValidationError: "
	for idx, err := range v.Errors {
		errString = fmt.Sprintf("%v\n   %v -  %v", errString, idx, err)
	}
	return errString
}

type Violation interface {
	Error() string
}

func NewViolation(msg string) Violation {
	return &violation{msg}
}

type violation struct {
	msg string
}

func (v *violation) Error() string {
	return v.msg
}
