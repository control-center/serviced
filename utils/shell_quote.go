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

// Quote a single argument
func ShellQuoteArg(arg string) string {
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
