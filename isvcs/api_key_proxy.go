// Copyright 2018 The Serviced Authors.
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

package isvcs

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/config"
	"github.com/control-center/serviced/utils"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

var apiKeyProxy *IService

// configure transport to ignore key errors - we're using a self-signed key.
var tlsTransport = &http.Transport{
	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
}
var tlsClient = &http.Client{Transport: tlsTransport}

const API_KEY_PROXY_SERVER_HEALTHCHECK_NAME = "api-key-server-running"
const API_KEY_PROXY_SERVER_INTERNAL_API_HEALTHCHECK_NAME = "internal-api-reachable"

func getKeyProxyPort() uint16 {
	return 6443
}

func initApiKeyProxy() {
	logger := log.WithFields(logrus.Fields{"isvc": "APIKeyProxy"})

	startApiKeyProxy := config.GetOptions().StartAPIKeyProxy
	logger.WithField("StartAPIKeyProxy", startApiKeyProxy).Debug("initApiKeyProxy()")

	var err error
	command := `/bin/supervisord -n -c etc/api-key-proxy/supervisord.conf`

	apiKeyPortBinding := portBinding{
		HostIp:         "0.0.0.0",
		HostIpOverride: "",
		HostPort:       getKeyProxyPort(),
	}

	keyServerReachableHealthCheck := healthCheckDefinition{
		healthCheck: SetKeyServerReachableHealthCheck(),
		Interval:    DEFAULT_HEALTHCHECK_INTERVAL,
		Timeout:     DEFAULT_HEALTHCHECK_TIMEOUT,
	}
	keyProxyAnsweringHealthCheck := healthCheckDefinition{
		healthCheck: SetKeyProxyAnsweringHealthCheck(),
		Interval:    DEFAULT_HEALTHCHECK_INTERVAL,
		Timeout:     DEFAULT_HEALTHCHECK_TIMEOUT,
	}

	healthChecks := []map[string]healthCheckDefinition{
		map[string]healthCheckDefinition{
			"API Key Server Reachable": keyServerReachableHealthCheck,
			"API Key Proxy Answering":  keyProxyAnsweringHealthCheck,
		},
	}

	ApiKeyProxy := IServiceDefinition{
		ID:           ApiKeyProxyISVC.ID,
		Name:         "api-key-proxy",
		Repo:         API_KEY_PROXY_REPO,
		Tag:          API_KEY_PROXY_TAG,
		Command:      func() string { return command },
		PortBindings: []portBinding{apiKeyPortBinding},
		Volumes:      map[string]string{},
		StartGroup:   1,
		HealthChecks: healthChecks,
	}

	apiKeyProxy, err = NewIService(ApiKeyProxy)
	if err != nil {
		log.WithError(err).Fatal("Unable to initialize API Key Server internal service container")
	}
}

func SetKeyServerReachableHealthCheck() HealthCheckFunction {
	return func(halt <-chan struct{}) error {
		logger := log.WithFields(logrus.Fields{"HealthCheckName": API_KEY_PROXY_SERVER_HEALTHCHECK_NAME})
		TestURL := config.GetOptions().KeyProxyJsonServer + "RUOK"
		for {
			select {
			case <-halt:
				logger.Debug("Stopped health checks for API Key Proxy Server")
				return nil
			default:
				if err := CheckURL(TestURL); err != nil {
					logger.WithError(err).
						WithFields(logrus.Fields{"TestURL": TestURL}).
						Debug("Bad response from Key server.")
					time.Sleep(1 * time.Second)
					continue
				}
				logger.Debug("API Key Server checked in healthy")
				return nil
			}
		}
	}
}

func getProxyURL() string {
	servicedIP := utils.GetDockerIP()
	port := strings.TrimLeft(config.GetOptions().KeyProxyListenPort, ":")
	proxyURL := fmt.Sprintf("https://%s:%s", servicedIP, port)
	return proxyURL
}

func SetKeyProxyAnsweringHealthCheck() HealthCheckFunction {
	return func(halt <-chan struct{}) error {
		logger := log.WithFields(logrus.Fields{"HealthCheckName": API_KEY_PROXY_SERVER_INTERNAL_API_HEALTHCHECK_NAME})
		TestURL := fmt.Sprintf("%s%s", getProxyURL(), "/apiproxy/RUOK")
		logger.WithFields(logrus.Fields{"TestURL": TestURL}).Debug("Starting Key Proxy Server check")
		tries := 0
		for {
			tries++
			select {
			case <-halt:
				logger.Debug("Stopped health checks for API Key Proxy Server Internal API Check")
				return nil
			default:
				if err := CheckURL(TestURL); err != nil {
					if tries <= 3 {
						logger.WithError(err).
							WithFields(logrus.Fields{
								"TestURL":                       TestURL,
								"SERVICED_KEYPROXY_JSON_SERVER": config.GetOptions().KeyProxyJsonServer,
							}).
							Info("Error connecting to Serviced API server. Verify that SERVICED_KEYPROXY_JSON_SERVER is set properly and that the server is running.")
					}
					time.Sleep(1 * time.Second)
					continue
				}
				logger.Debug("API Key Proxy checked in healthy")
				return nil
			}
		}
	}
}

func CheckURL(TestURL string) error {
	logger := log.WithFields(logrus.Fields{"URL": TestURL})

	// clean up transport connection pool - mostly here for hygiene purposes
	defer tlsTransport.CloseIdleConnections()

	resp, err := tlsClient.Get(TestURL)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		logger.
			WithError(err).
			Debug("GET operation returned error.")
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		logger.
			WithFields(logrus.Fields{
				"StatusCode": resp.StatusCode,
				"Body":       string(body)}).
			Debug("Received response other than 200 (OK)")
		e := errors.New(fmt.Sprintf("Status code %d received from %s.", resp.StatusCode, string(body)))
		return e
	}
	return nil
}
