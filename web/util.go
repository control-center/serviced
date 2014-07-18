// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package web

import (
	"flag"
	"fmt"
	"github.com/zenoss/go-json-rest"
	"net/http"
	"os"
	"path"
	"runtime"
)

var webroot string

func init() {
	webrootDefault := ""
	servicedHome := os.Getenv("SERVICED_HOME")
	if len(servicedHome) > 0 {
		webrootDefault = servicedHome + "/share/web/static"
	}
	flag.StringVar(&webroot, "webroot", webrootDefault, "static director for web content, defaults to GO runtime path of src")
}

/*******************************************************************************
 *
 * Data Structures
 *
 ******************************************************************************/

type simpleResponse struct {
	Detail string
	Links  []link
}

type link struct {
	Name   string
	Method string
	Url    string
}

type login struct {
	Username string
	Password string
}

const createlink = "Create"
const updatelink = "Update"
const retrievelink = "Retrieve"
const deletelink = "Delete"

/*******************************************************************************
 *
 * Public Functions
 *
 ******************************************************************************/

/*
 * Inform the user that a login is required
 */
func restUnauthorized(w *rest.ResponseWriter) {
	writeJSON(w, &simpleResponse{"Not authorized", loginLink()}, http.StatusUnauthorized)
	return
}

/*
 * Provide a generic response for an oopsie.
 */
func restServerError(w *rest.ResponseWriter) {
	writeJSON(w, &simpleResponse{"Internal Server Error", homeLink()}, http.StatusInternalServerError)
	return
}

/*
 * The user sent us junk, or we were incapabale of decoding what they sent.
 */
func restBadRequest(w *rest.ResponseWriter) {
	writeJSON(w, &simpleResponse{"Bad Request", homeLink()}, http.StatusBadRequest)
	return
}

/*
 * Write 200 success
 */
func restSuccess(w *rest.ResponseWriter) {
	w.WriteHeader(200)
	return
}

// WriteJSON struct as JSON with specified HTTP status code
func writeJSON(w *rest.ResponseWriter, v interface{}, code int) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(code)
	err := w.WriteJson(v)
	if err != nil {
		panic(err)
	}
}

/*
 * Provides content for root /
 */
func mainPage(w *rest.ResponseWriter, r *rest.Request) {
	noCache(w)
	http.ServeFile(
		w.ResponseWriter,
		r.Request,
		staticRoot()+"/index.html")
}

/*
 * Provides content for /test
 */
func testPage(w *rest.ResponseWriter, r *rest.Request) {
	noCache(w)
	http.ServeFile(
		w.ResponseWriter,
		r.Request,
		staticRoot()+"/test/index.html")
}

/*
 * Provides content for /favicon.ico
 */
func favIcon(w *rest.ResponseWriter, r *rest.Request) {
	http.ServeFile(
		w.ResponseWriter,
		r.Request,
		staticRoot()+"/ico/zenoss-o.png")
}

/*
 * Serves content from static/
 */
func staticData(w *rest.ResponseWriter, r *rest.Request) {
	fileToServe := path.Join(staticRoot(), r.PathParam("resource"))
	http.ServeFile(
		w.ResponseWriter,
		r.Request,
		fileToServe)
}

/*******************************************************************************
 *
 * Private helper functions
 *
 ******************************************************************************/

/*
 * Provide a list of login related API calls
 */
func loginLink() []link {
	return []link{
		link{createlink, "POST", "/login"},
		link{deletelink, "DELETE", "/login"},
	}
}

/*
 * Provide a basic link to the index
 */
func homeLink() []link {
	return []link{link{retrievelink, "GET", "/"}}
}

/*
 * Provide a list of host related API calls
 */
func hostsLinks() []link {
	return []link{
		link{retrievelink, "GET", "/hosts"},
		link{createlink, "POST", "/hosts/add"},
	}
}

func hostLinks(hostID string) []link {
	hostURI := fmt.Sprintf("/hosts/%s", hostID)
	return []link{
		link{retrievelink, "GET", hostURI},
		link{updatelink, "PUT", hostURI},
		link{deletelink, "DELETE", hostURI},
	}
}

func eventsLinks(eventID string) []link {
	eventURI := fmt.Sprintf("/events/%s", eventID)
	return []link{
		link{retrievelink, "GET", eventURI},
		link{createlink, "POST", "/events/add"},
		link{deletelink, "DELETE", eventURI},
	}
}

/*
 * Provide a list of pool related API calls
 */
func poolsLinks() []link {
	return []link{
		link{retrievelink, "GET", "/pools"},
		link{createlink, "POST", "/pools/add"},
	}
}

func poolLinks(poolID string) []link {
	poolURI := fmt.Sprintf("/pools/%s", poolID)
	return []link{
		link{retrievelink, "GET", poolURI},
		link{"RetrieveHosts", "GET", poolURI + "/hosts"},
		link{updatelink, "PUT", poolURI},
		link{deletelink, "DELETE", poolURI},
	}
}

func servicesLinks() []link {
	return []link{
		link{retrievelink, "GET", servicesURI},
		link{createlink, "POST", servicesURI + "/add"},
	}
}

/*
 * Provide a list of service related API calls
 */
func serviceLinks(serviceID string) []link {
	serviceURI := fmt.Sprintf("/services/%s", serviceID)
	return []link{
		link{retrievelink, "GET", serviceURI},
		link{"ServiceLogs", "GET", serviceURI + "/logs"},
		link{updatelink, "PUT", serviceURI},
		link{deletelink, "DELETE", serviceURI},
	}
}

/*
 * Provide a list of template related API calls.
 */
func templatesLink() []link {
	return []link{
		link{retrievelink, "GET", "/templates"},
		link{createlink, "POST", "/templates/add"},
		link{"Deploy", "POST", "/templates/deploy"},
	}
}

func templateLinks(templateID string) []link {
	templateURI := fmt.Sprintf("/templates/%s", templateID)
	return []link{
		link{retrievelink, "GET", templateURI},
		link{updatelink, "PUT", templateURI},
		link{deletelink, "DELETE", templateURI},
	}
}

/*
 * Inform browsers that this call should not be cached. Ever.
 */
func noCache(w *rest.ResponseWriter) {
	headers := w.ResponseWriter.Header()
	headers.Add("Cache-Control", "no-cache, no-store, must-revalidate")
	headers.Add("Pragma", "no-cache")
	headers.Add("Expires", "0")
}

/*
 * Hack to get us the location on the filesystem of our static files.
 */
func staticRoot() string {
	if len(webroot) == 0 {
		_, filename, _, _ := runtime.Caller(1)
		return path.Join(path.Dir(filename), "static")
	}
	return webroot
}

const servicesURI = "/services"
const hostsURI = "/hosts"
const templatesURI = "/templates"
const poolsURI = "/pools"
