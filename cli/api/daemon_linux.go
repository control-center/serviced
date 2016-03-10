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

package api

import (
// Need to do btrfs driver initializations
	_ "github.com/control-center/serviced/volume/btrfs"
// Need to do rsync driver initializations
	_ "github.com/control-center/serviced/volume/rsync"
// Need to do devicemapper driver initializations
	_ "github.com/control-center/serviced/volume/devicemapper"
// Need to do nfs driver initializations
	_ "github.com/control-center/serviced/volume/nfs"
)
