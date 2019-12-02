// Copyright 2019 The Serviced Authors.
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

package userinfo

import (
	"errors"
	"fmt"
	osuser "os/user"
	"strconv"
	"strings"
)

// Info has user info
type Info struct {
	UID      uint32
	GID      uint32
	HomeDir  string
	Username string
}

func isMemeberOf(u *osuser.User, group *osuser.Group) (bool, error) {

	groupids, err := u.GroupIds()
	if err != nil {
		return false, err
	}
	for _, gid := range groupids {
		if gid == group.Gid {
			return true, nil
		}
	}
	return false, nil
}

// New sets up user
func New(u string) (*Info, error) {
	var username, usergroup string
	var gid int

	input := strings.Split(u, ":")

	switch len(input) {
	case 1:
		username = strings.Trim(input[0], " ")
	case 2:
		username = strings.Trim(input[0], " ")
		usergroup = strings.Trim(input[1], " ")
	default:
		return nil, errors.New("Incorrect user or user:group data")
	}

	user, err := osuser.Lookup(username)
	if err != nil {
		return nil, err
	}

	uid, err := strconv.Atoi(user.Uid)
	if err != nil {
		return nil, err
	}

	if usergroup != "" {
		group, err := osuser.LookupGroup(usergroup)
		if err != nil {
			return nil, err
		}
		isMember, err := isMemeberOf(user, group)
		if err != nil {
			return nil, err
		}
		if isMember {
			gid, err = strconv.Atoi(group.Gid)
			if err != nil {
				return nil, err
			}
		} else {
			errorMessage := fmt.Sprintf("user %s is not a member of %s group\n", username, usergroup)
			return nil, errors.New(errorMessage)
		}
	} else {
		groupids, err := user.GroupIds()
		if err != nil {
			return nil, err
		}

		gid, err = strconv.Atoi(groupids[0])
		if err != nil {
			return nil, err
		}
	}

	return &Info{
		UID:      uint32(uid),
		GID:      uint32(gid),
		HomeDir:  user.HomeDir,
		Username: user.Username,
	}, nil
}
