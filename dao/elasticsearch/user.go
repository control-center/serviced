// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package elasticsearch

import (
	"github.com/zenoss/glog"
	"github.com/control-center/serviced/datastore"
	userdomain "github.com/control-center/serviced/domain/user"
	"github.com/control-center/serviced/utils"

	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"strings"
)

// each time Serviced starts up a new password will be generated. This will be passed into
// the containers so that they can authenticate against the API
var SYSTEM_USER_NAME = "system_user"
var INSTANCE_PASSWORD string

//hashPassword returns the sha-1 of a password
func hashPassword(password string) string {
	h := sha1.New()
	io.WriteString(h, password)
	return fmt.Sprintf("% x", h.Sum(nil))
}

//addUser places a new user record into elastic searchp
func (this *ControlPlaneDao) AddUser(newUser userdomain.User, userName *string) error {
	glog.V(2).Infof("ControlPlane.NewUser: %+v", newUser)
	name := strings.TrimSpace(*userName)
	newUser.Password = hashPassword(newUser.Password)

	// save the user
	var existing userdomain.User
	if err := this.GetUser(name, &existing); err != nil && !datastore.IsErrNoSuchEntity(err) {
		return err
	}
	store := userdomain.NewStore()
	return store.Put(datastore.Get(), userdomain.Key(name), &newUser)
}

//UpdateUser updates the user entry in elastic search. NOTE: It is assumed the
//pasword is NOT hashed when updating the user record
func (this *ControlPlaneDao) UpdateUser(user userdomain.User, unused *int) error {
	glog.V(2).Infof("ControlPlaneDao.UpdateUser: %+v", user)

	id := strings.TrimSpace(user.Name)
	if id == "" {
		return errors.New("empty User.Name not allowed")
	}

	user.Name = id
	user.Password = hashPassword(user.Password)

	store := userdomain.NewStore()
	return store.Put(datastore.Get(), userdomain.Key(user.Name), &user)
}

func (this *ControlPlaneDao) GetUser(userName string, user *userdomain.User) error {
	glog.V(2).Infof("ControlPlaneDao.GetUser: userName=%s", userName)
	store := userdomain.NewStore()
	err := store.Get(datastore.Get(), userdomain.Key(userName), user)
	glog.V(2).Infof("ControlPlaneDao.GetUser: userName=%s, user=%+v, err=%s", userName, user, err)
	return err
}

// RemoveUser removes the user specified by the userName string
func (this *ControlPlaneDao) RemoveUser(userName string, unused *int) error {
	glog.V(2).Infof("ControlPlaneDao.RemoveUser: %s", userName)
	store := userdomain.NewStore()
	return store.Delete(datastore.Get(), userdomain.Key(userName))
}

//ValidateCredentials takes a user name and password and validates them against a stored user
func (this *ControlPlaneDao) ValidateCredentials(user userdomain.User, result *bool) error {
	glog.V(2).Infof("ControlPlaneDao.ValidateCredentials: userName=%s", user.Name)
	storedUser := userdomain.User{}
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
func (this *ControlPlaneDao) GetSystemUser(unused int, user *userdomain.User) error {
	systemUser := userdomain.User{
		Name:     SYSTEM_USER_NAME,
		Password: INSTANCE_PASSWORD,
	}
	*user = systemUser
	return nil
}

//createSystemUser updates the running instance password as well as the user record in elastic
func createSystemUser(s *ControlPlaneDao) error {
	user := userdomain.User{}
	err := s.GetUser(SYSTEM_USER_NAME, &user)
	if err != nil {
		glog.Warningf("%s", err)
		glog.V(0).Info("'default' user not found; creating...")

		// create the system user
		user := userdomain.User{}
		user.Name = SYSTEM_USER_NAME
		userName := SYSTEM_USER_NAME

		if err := s.AddUser(user, &userName); err != nil {
			return err
		}
	}

	// update the instance password
	password, err := utils.NewUUID36()
	if err != nil {
		return err
	}
	user.Name = SYSTEM_USER_NAME
	user.Password = password
	INSTANCE_PASSWORD = password
	unused := 0
	return s.UpdateUser(user, &unused)
}
