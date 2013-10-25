package web

import (
	"github.com/ant0ine/go-json-rest"

	"flag"
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
	Links []Link
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
		staticRoot() + "/index.html")
}

/*
 * Provides content for /test
 */
func TestPage(w *rest.ResponseWriter, r *rest.Request) {
	noCache(w)
	http.ServeFile(
		w.ResponseWriter,
		r.Request,
		staticRoot() + "/test/index.html")
}


/*
 * Provides content for /favicon.ico
 */
func FavIcon(w *rest.ResponseWriter, r *rest.Request) {
	http.ServeFile(
		w.ResponseWriter,
		r.Request,
		staticRoot() + "/ico/zenoss-o.png")
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
func hostsLink() []Link {
	return []Link{
		Link{RetrieveLink, "GET", "/hosts"},
		Link{CreateLink, "POST", "/hosts/add"},
		Link{UpdateLink, "PUT", "/hosts/:hostId"},
		Link{DeleteLink, "DELETE", "/hosts/:hostId"},
	}
}

/*
 * Provide a list of pool related API calls
 */
func poolsLink() []Link {
	return []Link{
		Link{RetrieveLink, "GET", "/pools"},
		Link{"RetrieveHosts", "GET", "/pools/:poolId/hosts"},
		Link{CreateLink, "POST", "/pools/add"},
		Link{UpdateLink, "PUT", "/pools/:poolId"},
		Link{DeleteLink, "DELETE", "/pools/:poolId"},
	}
}

/*
 * Provide a list of service related API calls
 */
func servicesLink() []Link {
	return []Link{
		Link{RetrieveLink, "GET", "/services"},
		Link{"ServiceLogs", "GET", "/services/:serviceId/logs"},
		Link{CreateLink, "POST", "/services/add"},
		Link{UpdateLink, "PUT", "/services/:serviceId"},
		Link{DeleteLink, "DELETE", "/services/:serviceId"},
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
		Link{UpdateLink, "PUT", "/templates/:templateId"},
		Link{DeleteLink, "DELETE", "/templates/:templateId"},
	}
}

/*
 * Inform browsers that this call should not be cached. Ever.
 */
func noCache(w *rest.ResponseWriter) {
	headers := w.ResponseWriter.Header()
	headers.Add("Cache-Control","no-cache, no-store, must-revalidate")
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

