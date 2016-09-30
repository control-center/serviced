// Copyright 2016 The Serviced Authors.
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

package container

/*
 *  Http proxy that injects the auth token in the header when
 *  sending requests to CC REST API
 *  It listens on http port and proxies to https
 */

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strconv"

	"github.com/control-center/serviced/auth"
	"github.com/control-center/serviced/node"
)

type servicedApiProxy struct {
	proxyPort int
	ccApiPort int
	proxy     *httputil.ReverseProxy
}

func newServicedApiProxy() *servicedApiProxy {
	proxyPort := node.SERVICED_UI_ENDPOINT_PROXY
	ccApiPort := node.SERVICED_UI_ENDPOINT
	director := func(req *http.Request) {
		req.URL.Scheme = "https"
		req.URL.Host = "localhost:" + strconv.Itoa(ccApiPort)
		token := auth.BuildRestToken()
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	proxy := &httputil.ReverseProxy{Director: director, Transport: transport}

	return &servicedApiProxy{proxyPort, ccApiPort, proxy}
}

func (p *servicedApiProxy) run() {
	port := ":" + strconv.Itoa(p.proxyPort)
	http.ListenAndServe(port, p.proxy)
}
