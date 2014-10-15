// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package validation

import (
	"fmt"
)

//ValidationError is an error that contains other errors
type ValidationError struct {
	Errors []error
}

//NewValidationError creates a ValidationError with an empty slice of errors
func NewValidationError() *ValidationError {
	return &ValidationError{make([]error, 0)}
}

//AddViolation adds Violation error
func (v *ValidationError) AddViolation(violationMsg string) {
	v.Add(NewViolation(violationMsg))
}

//Add adds a an error
func (v *ValidationError) Add(err error) {
	if err != nil {
		v.Errors = append(v.Errors, err)
	}
}

//Error returns the error string
func (v *ValidationError) Error() string {
	errString := "ValidationError: "
	for idx, err := range v.Errors {
		errString = fmt.Sprintf("%v\n   %v -  %v", errString, idx, err)
	}
	return errString
}

//HasError test to see if length of  Errors slice is greater than 0
func (v *ValidationError) HasError() bool {

	return len(v.Errors) > 0
}

//NewViolation creates a violation error
func NewViolation(msg string) *Violation {
	return &Violation{msg}
}

//Violation is an error type for validation violations
type Violation struct {
	msg string
}

//Error returns the error string
func (v *Violation) Error() string {
	return v.msg
}
