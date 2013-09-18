package web

import (
	"github.com/ant0ine/go-json-rest"
	"net/http"
	"path"
	"runtime"
)

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
	Name string
	Url string
}

type Login struct {
	Username string
	Password string
}

type UserData struct {
	Username string
	Name string
	Email string
}

const NextLink = "Next"
const AddLink = "Add"
const RemoveLink = "Remove"

/*******************************************************************************
 *
 * Public Functions
 *
 ******************************************************************************/

func RestUnauthorized(w *rest.ResponseWriter) {
	WriteJson(w, &SimpleResponse{"Not authorized", loginLink()}, http.StatusUnauthorized)
	return
}

func RestServerError(w *rest.ResponseWriter) {
	WriteJson(w, &SimpleResponse{"Internal Server Error", homeLink()}, http.StatusInternalServerError)
	return
}

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
 * Provides content for root /
 */
func WizardPage(w *rest.ResponseWriter, r *rest.Request) {
	noCache(w)
	http.ServeFile(
		w.ResponseWriter,
		r.Request,
		staticRoot() + "/wizard.html")
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

/*
 * Redirect the current request to /login
 */
func RedirectLogin(w *rest.ResponseWriter, r *rest.Request) {
	http.Redirect(
		w.ResponseWriter,
		r.Request,
		"/login",
		http.StatusFound)
}

/*
 * Render HTML login page
 */
func ContentLoginPage(w *rest.ResponseWriter, r *rest.Request) {
	http.ServeFile(
		w.ResponseWriter,
		r.Request,
		staticRoot() + "/login.html")
}

/*******************************************************************************
 *
 * Private helper functions
 *
 ******************************************************************************/

/*
 * Return array of links containing single Next element for login
 */ 
func loginLink() []Link {
	return []Link{Link{NextLink, "/login"}}
}

/*
 * Return array of links containing single Next element for home
 */ 
func homeLink() []Link {
	return []Link{Link{NextLink, "/"}}
}

func managementLink() []Link {
	return []Link{Link{NextLink, "#/management"}}
}

func resourcesLink() []Link {
	return []Link{Link{NextLink, "#/resources"}}
}

func hostsLink() []Link {
	return []Link{
		Link{NextLink, "/hosts"},
		Link{AddLink, "/hosts/add"},
		Link{RemoveLink, "/hosts/:hostId"},
	}
}

func poolsLink() []Link {
	return []Link{
		Link{NextLink, "/pools"},
		Link{AddLink, "/pools/add"},
		Link{RemoveLink, "/pools/:poolId"},
	}
}

func configurationLink() []Link {
	return []Link{Link{NextLink, "#/configuration"}}
}

func noCache(w *rest.ResponseWriter) {
	headers := w.ResponseWriter.Header()
	headers.Add("Cache-Control","no-cache, no-store, must-revalidate")
	headers.Add("Pragma", "no-cache")
	headers.Add("Expires", "0")
}


func staticRoot() string {
	_, filename, _, _ := runtime.Caller(1)
	return path.Join(path.Dir(filename), "static")
}


