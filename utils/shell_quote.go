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

package utils

import "strings"
import "regexp"

var hasUnsafeChar = regexp.MustCompile("[^\\w@%+=:,./-]")

// Quote a single argument
func ShellQuoteArg(arg string) string {
	if arg == "" {
		return "''"
	}

	if !hasUnsafeChar.MatchString(arg) {
		return arg
	}

	// wrap with single quotes; put existing single quotes into double quotes
	return "'" + strings.Replace(arg, "'", "'\"'\"'", -1) + "'"
}

// Quote a list of arguments and join them with spaces
func ShellQuoteArgs(args []string) string {
	quotedArgs := []string{}
	for _, arg := range args {
		quotedArgs = append(quotedArgs, ShellQuoteArg(arg))
	}
	return strings.Join(quotedArgs, " ")
}
