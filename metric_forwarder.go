package serviced

import (
	"github.com/zenoss/glog"
	rest "github.com/zenoss/go-json-rest"

	"io"
	"net/http"
)

// MetricForwarder contains all configuration parameters required to provide a
// forward metrics inside a docker container.
type MetricForwarder struct {
	Port               string
	MetricsRedirectUrl string
}

// Create a new metric forwarder at port, all metrics are forwarded to metricsRedirectUrl
func NewMetricForwarder(port, metricsRedirectUrl string) (config MetricForwarder, err error) {
	config.Port = port
	config.MetricsRedirectUrl = metricsRedirectUrl
	return config, err
}

// Serve configures all http method handlers for the container controller.
// Then starts the server.  This method blocks.
func (forwarder *MetricForwarder) Serve() {
	routes := []rest.Route{
		rest.Route{
			HttpMethod: "POST",
			PathExp:    "/api/metrics/store",
			Func:       post_api_metrics_store(forwarder.MetricsRedirectUrl),
		},
	}

	handler := rest.ResourceHandler{}
	handler.SetRoutes(routes...)

	http.ListenAndServe(forwarder.Port, &handler)
}

// post_api_metrics_store redirects the post request to the configured address
// Any additional parameters should be encoded in the redirect url.  For
// example, encode the containers tenant and service id.
func post_api_metrics_store(redirect_url string) func(*rest.ResponseWriter, *rest.Request) {
	return func(w *rest.ResponseWriter, request *rest.Request) {
		client := &http.Client{}

		proxy_request, _ := http.NewRequest(request.Method, redirect_url, request.Body)
		for k, v := range request.Header {
			proxy_request.Header[k] = v
		}
		proxy_response, err := client.Do(proxy_request)
		if err == nil {
			defer proxy_response.Body.Close()
			w.WriteHeader(proxy_response.StatusCode)
			io.Copy(w, proxy_response.Body)
		} else {
			glog.Errorf("Failed to proxy request: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}
