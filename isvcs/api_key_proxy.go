// Copyright 2014 The Serviced Authors.
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

var apiKeyProxy *IService

//const (
//	apikeyproxyport = 6443
//)
func getKeyProxyPort() uint16 {
	return 6443
}

func initApiKeyProxy() {

	var err error
	command := `/bin/supervisord -n -c etc/api-key-proxy/supervisord.conf`

	apiKeyPortBinding := portBinding{
		HostIp:         "0.0.0.0",
		HostIpOverride: "", // TODO: verify that api key proxy should always be open
		HostPort:       getKeyProxyPort(),
	}

	apiKeyProxy, err = NewIService(
		IServiceDefinition{
			ID:           ApiKeyProxyISVC.ID,
			Name:         "api-key-proxy",
			Repo:         API_KEY_PROXY_REPO,
			Tag:          API_KEY_PROXY_TAG,
			Command:      func() string { return command },
			PortBindings: []portBinding{apiKeyPortBinding},
			Volumes:      map[string]string{},
			StartGroup:   1,
		})
	if err != nil {
		log.WithError(err).Fatal("Unable to initialize API Key Server internal service container")
	}
}
