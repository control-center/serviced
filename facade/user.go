// Copyright 2016 The Serviced Authors.
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

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package facade

import (
	log "github.com/Sirupsen/logrus"
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

// AddUser adds a new user record
func (f *Facade) AddUser(ctx datastore.Context, newUser userdomain.User) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("AddUser"))
	var err error
	logger := plog.WithFields(log.Fields{
		"newUserName": newUser.Name,
	})
	logger.Debug("Started Facade.AddUser")
	defer logger.WithError(err).Debug("Finished Facade.AddUser")

	name := strings.TrimSpace(newUser.Name)
	newUser.Password = hashPassword(newUser.Password)

	_, err = f.GetUser(ctx, name)
	if err != nil && !datastore.IsErrNoSuchEntity(err) {
		return err
	}
	err = f.userStore.Put(ctx, userdomain.Key(name), &newUser)
	return err
}

// UpdateUser updates the user record. NOTE: It is assumed the pasword
// is NOT hashed when updating the user record
func (f *Facade) UpdateUser(ctx datastore.Context, user userdomain.User) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("UpdateUser"))
	var err error
	logger := plog.WithField("userName", user.Name)
	logger.Debug("Started Facade.UpdateUser")
	defer logger.WithError(err).Debug("Finished Facade.UpdateUser")

	id := strings.TrimSpace(user.Name)
	if id == "" {
		err = errors.New("empty User.Name not allowed")
		return err
	}

	user.Name = id
	user.Password = hashPassword(user.Password)
	err = f.userStore.Put(ctx, userdomain.Key(user.Name), &user)
	return err
}

func (f *Facade) GetUser(ctx datastore.Context, userName string) (userdomain.User, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("GetUser"))
	var err error
	logger := plog.WithField("userName", userName)
	logger.Debug("Started Facade.GetUser")
	defer logger.WithError(err).Debug("Finished Facade.GetUser")

	var user userdomain.User
	err = f.userStore.Get(ctx, userdomain.Key(userName), &user)
	return user, err
}

// RemoveUser removes the user specified by the userName string
func (f *Facade) RemoveUser(ctx datastore.Context, userName string) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("RemoveUser"))
	var err error
	logger := plog.WithField("userName", userName)
	logger.Debug("Started Facade.RemoveUser")
	defer logger.WithError(err).Debug("Finished Facade.RemoveUser")

	err = f.userStore.Delete(ctx, userdomain.Key(userName))
	return err
}

// ValidateCredentials takes a user name and password and validates them against a stored user
func (f *Facade) ValidateCredentials(ctx datastore.Context, user userdomain.User) (bool, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("ValidateCredentials"))
	var err error
	logger := plog.WithField("userName", user.Name)
	logger.Debug("Started Facade.ValidateCredentials")
	defer logger.WithError(err).Debug("Finished Facade.ValidateCredentials")

	var storedUser userdomain.User
	storedUser, err = f.GetUser(ctx, user.Name)
	if err != nil {
		return false, err
	}

	// hash the passed in password
	hashedPassword := hashPassword(user.Password)

	// confirm the password
	if storedUser.Password != hashedPassword {
		return false, nil
	}

	// at this point we found the user and confirmed the password
	return true, nil
}

// GetSystemUser returns the system user's credentials.
func (f *Facade) GetSystemUser(ctx datastore.Context) (userdomain.User, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("GetSystemUser"))
	plog.Debug("Started Facade.GetSystemUser")
	defer plog.Debug("Finished Facade.GetSystemUser")

	systemUser := userdomain.User{
		Name:     SYSTEM_USER_NAME,
		Password: INSTANCE_PASSWORD,
	}
	return systemUser, nil
}

// createSystemUser updates the running instance password as well as the user record in elastic
func (f *Facade) CreateSystemUser(ctx datastore.Context) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("CreateSystemUser"))
	user, err := f.GetUser(ctx, SYSTEM_USER_NAME)
	if err != nil {
		plog.WithError(err).Warning("Default user not found; creating one.")

		// create the system user
		user := userdomain.User{}
		user.Name = SYSTEM_USER_NAME

		if err := f.AddUser(ctx, user); err != nil {
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
	return f.UpdateUser(ctx, user)
}
