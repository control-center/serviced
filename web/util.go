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

type SimpleResponse struct {
	Detail string
	Links  []Link
}

type Link struct {
	Name   string
	Method string
	Url    string
}

type Login struct {
	Username string
	Password string
}

type UserData struct {
	Username string
	Name     string
	Email    string
}

const CreateLink = "Create"
const UpdateLink = "Update"
const RetrieveLink = "Retrieve"
const DeleteLink = "Delete"

/*******************************************************************************
 *
 * Public Functions
 *
 ******************************************************************************/

/*
 * Inform the user that a login is required
 */
func RestUnauthorized(w *rest.ResponseWriter) {
	WriteJson(w, &SimpleResponse{"Not authorized", loginLink()}, http.StatusUnauthorized)
	return
}

/*
 * Provide a generic response for an oopsie.
 */
func RestServerError(w *rest.ResponseWriter) {
	WriteJson(w, &SimpleResponse{"Internal Server Error", homeLink()}, http.StatusInternalServerError)
	return
}

/*
 * The user sent us junk, or we were incapabale of decoding what they sent.
 */
func RestBadRequest(w *rest.ResponseWriter) {
	WriteJson(w, &SimpleResponse{"Bad Request", homeLink()}, http.StatusBadRequest)
	return
}

/*
 * Write 200 success
 */
func RestSuccess(w *rest.ResponseWriter) {
	w.WriteHeader(200)
	return
}

/*
 * Writes struct as JSON with specified HTTP status code
 */
func WriteJson(w *rest.ResponseWriter, v interface{}, code int) {
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
func MainPage(w *rest.ResponseWriter, r *rest.Request) {
	noCache(w)
	http.ServeFile(
		w.ResponseWriter,
		r.Request,
		staticRoot()+"/index.html")
}

/*
 * Provides content for /test
 */
func TestPage(w *rest.ResponseWriter, r *rest.Request) {
	noCache(w)
	http.ServeFile(
		w.ResponseWriter,
		r.Request,
		staticRoot()+"/test/index.html")
}

/*
 * Provides content for /favicon.ico
 */
func FavIcon(w *rest.ResponseWriter, r *rest.Request) {
	http.ServeFile(
		w.ResponseWriter,
		r.Request,
		staticRoot()+"/ico/zenoss-o.png")
}

/*
 * Serves content from static/
 */
func StaticData(w *rest.ResponseWriter, r *rest.Request) {
	file_to_serve := path.Join(staticRoot(), r.PathParam("resource"))
	http.ServeFile(
		w.ResponseWriter,
		r.Request,
		file_to_serve)
}

/*******************************************************************************
 *
 * Private helper functions
 *
 ******************************************************************************/

/*
 * Provide a list of login related API calls
 */
func loginLink() []Link {
	return []Link{
		Link{CreateLink, "POST", "/login"},
		Link{DeleteLink, "DELETE", "/login"},
	}
}

/*
 * Provide a basic link to the index
 */
func homeLink() []Link {
	return []Link{Link{RetrieveLink, "GET", "/"}}
}

/*
 * Provide a list of host related API calls
 */
func hostsLinks() []Link {
	return []Link{
		Link{RetrieveLink, "GET", "/hosts"},
		Link{CreateLink, "POST", "/hosts/add"},
	}
}

func hostLinks(hostId string) []Link {
	hostUri := fmt.Sprintf("/hosts/%s", hostId)
	return []Link{
		Link{RetrieveLink, "GET", hostUri},
		Link{UpdateLink, "PUT", hostUri},
		Link{DeleteLink, "DELETE", hostUri},
	}
}

/*
 * Provide a list of pool related API calls
 */
func poolsLinks() []Link {
	return []Link{
		Link{RetrieveLink, "GET", "/pools"},
		Link{CreateLink, "POST", "/pools/add"},
	}
}

func poolLinks(poolId string) []Link {
	poolUri := fmt.Sprintf("/pools/%s", poolId)
	return []Link{
		Link{RetrieveLink, "GET", poolUri},
		Link{"RetrieveHosts", "GET", poolUri + "/hosts"},
		Link{UpdateLink, "PUT", poolUri},
		Link{DeleteLink, "DELETE", poolUri},
	}
}

func servicesLinks() []Link {
	return []Link{
		Link{RetrieveLink, "GET", SERVICES_URI},
		Link{CreateLink, "POST", SERVICES_URI + "/add"},
	}
}

/*
 * Provide a list of service related API calls
 */
func serviceLinks(serviceId string) []Link {
	serviceUri := fmt.Sprintf("/services/%s", serviceId)
	return []Link{
		Link{RetrieveLink, "GET", serviceUri},
		Link{"ServiceLogs", "GET", serviceUri + "/logs"},
		Link{UpdateLink, "PUT", serviceUri},
		Link{DeleteLink, "DELETE", serviceUri},
	}
}

/*
 * Provide a list of template related API calls.
 */
func templatesLink() []Link {
	return []Link{
		Link{RetrieveLink, "GET", "/templates"},
		Link{CreateLink, "POST", "/templates/add"},
		Link{"Deploy", "POST", "/templates/deploy"},
	}
}

func templateLinks(templateId string) []Link {
	templateUri := fmt.Sprintf("/templates/%s", templateId)
	return []Link{
		Link{RetrieveLink, "GET", templateUri},
		Link{UpdateLink, "PUT", templateUri},
		Link{DeleteLink, "DELETE", templateUri},
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

const SERVICES_URI = "/services"
const HOSTS_URI = "/hosts"
const TEMPLATES_URI = "/templates"
const POOLS_URI = "/pools"
