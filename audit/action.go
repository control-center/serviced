// Copyright 2017 The Serviced Authors.
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

package audit

// Defined actions for audit logging.

const (

	// Add is the string for the add action when logging.
	Add = "add"

	// Remove is the string for the remove action when logging.
	Remove = "remove"

	// Update is the string for the update action when logging.
	Update = "update"

	// Stop is the string for the stop action when logging.
	Stop = "stop"

	// Start is the string for the start action when logging.
	Start = "start"

	// Restart is the string for restart action when logging.
	Restart = "restart"

	//Pause is the string for the pause action when logging.
	Pause= "pause"

	// Migrate is the string for the migrate action when logging.
	Migrate = "migrate"

	// Backup is the string for the backup action when logging.
	Backup = "backup"

	// Restore is the string for the restore action when logging.
	Restore = "restore"

	// Deploy is the string value for the deploy action when logging.
	Deploy = "deploy"
)
