// Copyright 2015 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
)

type TestSynchronizer struct {
	mock.Mock
}

func (o *TestSynchronizer) IDs() ([]string, error) {
	args := o.Called()
	return args.Get(0).([]string), args.Error(1)
}

func (o *TestSynchronizer) Create(obj interface{}) error {
	args := o.Called(obj)
	return args.Error(0)
}

func (o *TestSynchronizer) Update(obj interface{}) error {
	args := o.Called(obj)
	return args.Error(0)
}

func (o *TestSynchronizer) Delete(id string) error {
	args := o.Called(id)
	return args.Error(0)
}

func TestSync_IDError(t *testing.T) {
	sync := new(TestSynchronizer)
	sync.On("IDs").Return([]string{}, errors.New("error"))
	if err := Sync(sync, make(map[string]interface{})); err == nil {
		t.Errorf("Expected error!")
	}
	sync.AssertExpectations(t)
}

func TestSync_Create(t *testing.T) {
	data := map[string]interface{}{
		"test1": "hello",
	}
	sync := new(TestSynchronizer)
	sync.On("IDs").Return([]string{}, nil)
	sync.On("Create", "hello").Return(nil)
	if err := Sync(sync, data); err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	sync.AssertExpectations(t)
}

func TestSync_CreateError(t *testing.T) {
	data := map[string]interface{}{
		"test1": "hello",
	}
	sync := new(TestSynchronizer)
	sync.On("IDs").Return([]string{}, nil)
	sync.On("Create", "hello").Return(errors.New("error"))
	if err := Sync(sync, data); err == nil {
		t.Errorf("Expected error!")
	}
	sync.AssertExpectations(t)

	data = map[string]interface{}{
		"test2": nil,
	}
	sync.On("Create", nil).Return(errors.New("error"))
	if err := Sync(sync, data); err != ErrBadSync {
		t.Errorf("Expected %s; Got %s", ErrBadSync, err)
	}
	sync.AssertNotCalled(t, "Create", nil)
}

func TestSync_Update(t *testing.T) {
	data := map[string]interface{}{
		"test1": "hello",
		"test2": nil,
	}
	sync := new(TestSynchronizer)
	sync.On("IDs").Return([]string{"test1", "test2"}, nil)
	sync.On("Update", "hello").Return(nil)
	sync.On("Update", nil).Return(errors.New("error"))
	if err := Sync(sync, data); err != nil {
		t.Errorf("Unexpected Error: %s", err)
	}
	sync.AssertCalled(t, "IDs")
	sync.AssertCalled(t, "Update", "hello")
	sync.AssertNotCalled(t, "Update", nil)
}

func TestSync_UpdateError(t *testing.T) {
	data := map[string]interface{}{
		"test1": "hello",
	}
	sync := new(TestSynchronizer)
	sync.On("IDs").Return([]string{"test1"}, nil)
	sync.On("Update", "hello").Return(errors.New("error"))
	if err := Sync(sync, data); err == nil {
		t.Errorf("Expected Error!")
	}
	sync.AssertExpectations(t)
}

func TestSync_Delete(t *testing.T) {
	data := map[string]interface{}{}
	sync := new(TestSynchronizer)
	sync.On("IDs").Return([]string{"test1"}, nil)
	sync.On("Delete", "test1").Return(nil)
	if err := Sync(sync, data); err != nil {
		t.Errorf("Unexpected Error: %s", err)
	}
	sync.AssertExpectations(t)
}

func TestSync_DeleteError(t *testing.T) {
	data := map[string]interface{}{}
	sync := new(TestSynchronizer)
	sync.On("IDs").Return([]string{"test1"}, nil)
	sync.On("Delete", "test1").Return(errors.New("error"))
	if err := Sync(sync, data); err == nil {
		t.Errorf("Expected Error!")
	}
	sync.AssertExpectations(t)
}