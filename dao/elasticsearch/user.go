// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package elasticsearch

import (
	"github.com/mattbaird/elastigo/api"
	"github.com/mattbaird/elastigo/core"
	"github.com/mattbaird/elastigo/indices"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"

	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"strings"
)

var (
	//enable pretty printed responses
	Pretty bool = false

	//model index functions
	newUser func(string, interface{}) (api.BaseResponse, error) = create(&Pretty, "controlplane", "user")

	//model index functions
	indexUser func(string, interface{}) (api.BaseResponse, error) = index(&Pretty, "controlplane", "user")

	//model delete functions
	deleteUser func(string) (api.BaseResponse, error) = _delete(&Pretty, "controlplane", "user")

	//model get functions
	getUser func(string, interface{}) error = getSource("controlplane", "user")
)

// each time Serviced starts up a new password will be generated. This will be passed into
// the containers so that they can authenticate against the API
var SYSTEM_USER_NAME = "system_user"
var INSTANCE_PASSWORD string

// closure for geting a model
func getSource(index string, _type string) func(string, interface{}) error {
	return func(id string, source interface{}) error {
		return core.GetSource(index, _type, id, &source)
	}
}

// closure for searching a model
func searchUri(index string, _type string) func(string) (core.SearchResult, error) {
	return func(query string) (core.SearchResult, error) {
		return core.SearchUri(index, _type, query, "", 0)
	}
}

// closure for testing model existence
func exists(pretty *bool, index string, _type string) func(string) (bool, error) {
	return func(id string) (bool, error) {
		return core.Exists(*pretty, index, _type, id)
	}
}

// closure for indexing a model
func create(pretty *bool, index string, _type string) func(string, interface{}) (api.BaseResponse, error) {
	var (
		parentId  string = ""
		version   int    = 0
		op_type   string = "create"
		routing   string = ""
		timestamp string = ""
		ttl       int    = 0
		percolate string = ""
		timeout   string = ""
		refresh   bool   = true
	)
	return func(id string, data interface{}) (api.BaseResponse, error) {
		return core.IndexWithParameters(
			*pretty, index, _type, id, parentId, version, op_type, routing, timestamp, ttl, percolate, timeout, refresh, data)
	}
}

// closure for indexing a model
func index(pretty *bool, index string, _type string) func(string, interface{}) (api.BaseResponse, error) {
	var (
		parentId  string = ""
		version   int    = 0
		op_type   string = ""
		routing   string = ""
		timestamp string = ""
		ttl       int    = 0
		percolate string = ""
		timeout   string = ""
		refresh   bool   = true
	)
	return func(id string, data interface{}) (api.BaseResponse, error) {
		return core.IndexWithParameters(
			*pretty, index, _type, id, parentId, version, op_type, routing, timestamp, ttl, percolate, timeout, refresh, data)
	}
}

// closure for deleting a model
func _delete(pretty *bool, index string, _type string) func(string) (api.BaseResponse, error) {
	return func(id string) (api.BaseResponse, error) {
		r, err := core.Delete(*pretty, index, _type, id, -1, "")
		if err != nil {
			return r, err
		}
		indices.Refresh(index)
		return r, err
	}
}

//hashPassword returns the sha-1 of a password
func hashPassword(password string) string {
	h := sha1.New()
	io.WriteString(h, password)
	return fmt.Sprintf("% x", h.Sum(nil))
}

//addUser places a new user record into elastic searchp
func (this *ControlPlaneDao) AddUser(user dao.User, userName *string) error {
	glog.V(2).Infof("ControlPlane.NewUser: %+v", user)
	name := strings.TrimSpace(*userName)
	user.Password = hashPassword(user.Password)

	// save the user
	response, err := newUser(name, user)
	if response.Ok {
		*userName = name
		return nil
	}
	return err
}

//UpdateUser updates the user entry in elastic search. NOTE: It is assumed the
//pasword is NOT hashed when updating the user record
func (this *ControlPlaneDao) UpdateUser(user dao.User, unused *int) error {
	glog.V(2).Infof("ControlPlaneDao.UpdateUser: %+v", user)

	id := strings.TrimSpace(user.Name)
	if id == "" {
		return errors.New("empty User.Name not allowed")
	}

	user.Name = id
	user.Password = hashPassword(user.Password)
	response, err := indexUser(id, user)
	glog.V(2).Infof("ControlPlaneDao.UpdateUser response: %+v", response)
	if response.Ok {
		return nil
	}
	return err
}

func (this *ControlPlaneDao) GetUser(userName string, user *dao.User) error {
	glog.V(2).Infof("ControlPlaneDao.GetUser: userName=%s", userName)
	request := dao.User{}
	err := getUser(userName, &request)
	glog.V(2).Infof("ControlPlaneDao.GetUser: userName=%s, user=%+v, err=%s", userName, request, err)
	*user = request
	return err
}

// RemoveUser removes the user specified by the userName string
func (this *ControlPlaneDao) RemoveUser(userName string, unused *int) error {
	glog.V(2).Infof("ControlPlaneDao.RemoveUser: %s", userName)
	response, err := deleteUser(userName)
	glog.V(2).Infof("ControlPlaneDao.RemoveUser response: %+v", response)
	return err
}

//ValidateCredentials takes a user name and password and validates them against a stored user
func (this *ControlPlaneDao) ValidateCredentials(user dao.User, result *bool) error {
	glog.V(2).Infof("ControlPlaneDao.ValidateCredentials: userName=%s", user.Name)
	storedUser := dao.User{}
	err := this.GetUser(user.Name, &storedUser)
	if err != nil {
		*result = false
		return err
	}

	// hash the passed in password
	hashedPassword := hashPassword(user.Password)

	// confirm the password
	if storedUser.Password != hashedPassword {
		*result = false
		return nil
	}

	// at this point we found the user and confirmed the password
	*result = true
	return nil
}

//GetSystemUser returns the system user's credentials. The "unused int" is required by the RPC interface.
func (this *ControlPlaneDao) GetSystemUser(unused int, user *dao.User) error {
	systemUser := dao.User{
		Name:     SYSTEM_USER_NAME,
		Password: INSTANCE_PASSWORD,
	}
	*user = systemUser
	return nil
}

//createSystemUser updates the running instance password as well as the user record in elastic
func createSystemUser(s *ControlPlaneDao) error {
	user := dao.User{}
	err := s.GetUser(SYSTEM_USER_NAME, &user)
	if err != nil {
		glog.Warningf("%s", err)
		glog.V(0).Info("'default' user not found; creating...")

		// create the system user
		user := dao.User{}
		user.Name = SYSTEM_USER_NAME
		userName := SYSTEM_USER_NAME

		if err := s.AddUser(user, &userName); err != nil {
			return err
		}
	}

	// update the instance password
	password, err := dao.NewUuid()
	if err != nil {
		return err
	}
	user.Name = SYSTEM_USER_NAME
	user.Password = password
	INSTANCE_PASSWORD = password
	unused := 0
	return s.UpdateUser(user, &unused)
}
