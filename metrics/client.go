// This file was adapted from
// https://github.com/fsouza/go-dockerclient/blob/f1f91d5ba55810454f1d75e61d61d8b4c45e6e9b/client.go

package metrics

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const userAgent = "serviced"

var (
	// ErrInvalidEndpoint is returned when the endpoint is not a valid HTTP URL.
	ErrInvalidEndpoint = errors.New("invalid endpoint")

	// ErrConnectionRefused is returned when the client cannot connect to the given endpoint
	ErrConnectionRefused = errors.New("cannot connect to the metric query service")
)

// Client defines the interface of the metrics client.
type Client interface {
	GetAvailableStorage(window time.Duration, aggregator string, tenants ...string) (*StorageMetrics, error)
	GetHostMemoryStats(startDate time.Time, hostID string) (*MemoryUsageStats, error)
	GetInstanceMemoryStats(startDate time.Time, instances ...ServiceInstance) ([]MemoryUsageStats, error)
	GetServiceMemoryStats(startDate time.Time, serviceID string) (*MemoryUsageStats, error)
}

// clientImpl is data about a connection to the metrics server
type clientImpl struct {
	httpClient  *http.Client
	endpoint    string
	endpointURL *url.URL
}

// Error contains error data
type Error struct {
	Status  int
	Message string
}

// NewClient returns a Client instance ready for communication with the given server endpoint.
func NewClient(endpoint string) (Client, error) {
	u, err := parseEndpoint(endpoint)
	if err != nil {
		return nil, err
	}

	return &clientImpl{
		httpClient:  &http.Client{Timeout: time.Duration(5 * time.Second)},
		endpoint:    endpoint,
		endpointURL: u,
	}, nil
}

func (c *clientImpl) do(method, path string, data interface{}) ([]byte, int, error) {
	var params io.Reader
	if data != nil {
		buf, err := json.Marshal(data)
		if err != nil {
			return nil, -1, err
		}
		params = bytes.NewBuffer(buf)
	}
	req, err := http.NewRequest(method, c.getURL(path), params)
	if err != nil {
		return nil, -1, err
	}
	req.Header.Set("User-Agent", userAgent)
	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	} else if method == "POST" {
		req.Header.Set("Content-Type", "plain/text")
	}
	var resp *http.Response
	protocol := c.endpointURL.Scheme
	address := c.endpointURL.Path
	if protocol == "unix" {
		dial, err := net.Dial(protocol, address)
		if err != nil {
			return nil, -1, err
		}
		defer dial.Close()
		clientconn := httputil.NewClientConn(dial, nil)
		resp, err = clientconn.Do(req)
		if err != nil {
			return nil, -1, err
		}
		defer clientconn.Close()
	} else {
		resp, err = c.httpClient.Do(req)
	}
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			return nil, -1, ErrConnectionRefused
		}
		return nil, -1, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, -1, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return nil, resp.StatusCode, newError(resp.StatusCode, body)
	}
	return body, resp.StatusCode, nil
}

func (c *clientImpl) getURL(path string) string {
	urlStr := strings.TrimRight(c.endpointURL.String(), "/")
	if c.endpointURL.Scheme == "unix" {
		urlStr = ""
	}
	return fmt.Sprintf("%s%s", urlStr, path)
}

func newError(status int, body []byte) *Error {
	return &Error{Status: status, Message: string(body)}
}

func (e *Error) Error() string {
	return fmt.Sprintf("API error (%d): %s", e.Status, e.Message)
}

func parseEndpoint(endpoint string) (*url.URL, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, ErrInvalidEndpoint
	}
	if u.Scheme == "tcp" {
		u.Scheme = "http"
	}
	if u.Scheme != "unix" {
		_, port, err := net.SplitHostPort(u.Host)
		if err != nil {
			if e, ok := err.(*net.AddrError); ok {
				if e.Err == "missing port in address" {
					return u, nil
				}
			}
			return nil, ErrInvalidEndpoint
		}
		number, err := strconv.ParseInt(port, 10, 64)
		if err == nil && number > 0 && number < 65536 {
			return u, nil
		}
	} else {
		return u, nil // we don't need port when using a unix socket
	}
	return nil, ErrInvalidEndpoint
}
