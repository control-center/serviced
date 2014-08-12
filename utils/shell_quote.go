/*****************************************************************************
 *
 * Copyright (C) Zenoss, Inc. 2014, all rights reserved.
 *
 * This content is made available according to terms specified in
 * License.zenoss under the directory where your Zenoss product is installed.
 *
 ****************************************************************************/


package utils

import "strings"
import "regexp"

var hasUnsafeChar *regexp.Regexp
func init() {
	hasUnsafeChar, _ =regexp.Compile("[^\\w@%+=:,./-]")
}

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
