// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package utils

import (
	"encoding/json"
	"fmt"
	docker "github.com/zenoss/go-dockerclient"
	"io/ioutil"
	"net/http"
	"strings"
)

type DockerRegistry interface {
	PushImageToRemote(client *docker.Client, imageName string) error
	PullRemoteImage(client *docker.Client, repoTag string) error
	ListRemoteRepos() ([]string, error)
	ListRemoteRepoTags(repo string) ([]string, error)
	ListRemoteImageTags() (map[string][]string, error)
	TagRemoteImage(id, repoTag string) error
	RemoveRemoteImageTag(id, repoTag string) error
}

type dockerRegistry struct {
	hostAndPort string
}

func NewDockerRegistry(hostAndPort string) (DockerRegistry, error) {
	result := dockerRegistry{
		hostAndPort: hostAndPort,
	}
	return &result, nil
}

func (r *dockerRegistry) String() string {
	return r.hostAndPort
}

func (r *dockerRegistry) PushImageToRemote(client *docker.Client, imageName string) error {
	opts := docker.PushImageOptions{
		Name:     imageName,
		Registry: r.hostAndPort,
	}
	return client.PushImage(opts, auth)
}

func (r *dockerRegistry) PullRemoteImage(client *docker.Client, repoTag string) error {
	opts := docker.PullImageOptions{
		Repository: repoTag,
		Registry:   r.hostAndPort,
	}
	return client.PullImage(opts, auth)
}

func (r *dockerRegistry) ListRemoteRepos() ([]string, error) {
	var response map[string]interface{}
	if err := r.get("/v1/search?q=%", &response); err != nil {
		return []string{}, err
	}
	// results, ok := response["results"]
	// if !ok || len(results) == 0 {
	// 	return []string{}, fmt.Errorf("no results")
	// }
	return []string{}, nil
}

func (r *dockerRegistry) ListRemoteRepoTags(repo string) ([]string, error) {
	result := []string{}
	return result, nil
}

func (r *dockerRegistry) ListRemoteImageTags() (map[string][]string, error) {
	result := make(map[string][]string, 0)
	return result, nil
}

func (r *dockerRegistry) TagRemoteImage(id, tag string) error {
	return nil
}

func (r *dockerRegistry) RemoveRemoteImageTag(id, tag string) error {
	return nil
}

var auth = docker.AuthConfiguration{
	Username: "",
	Password: "",
	Email:    "",
}

func (r *dockerRegistry) url(path string) string {
	result := r.hostAndPort
	if len(result) < 4 && result[:4] != "http" {
		result = "http://" + result
	}
	return strings.TrimRight(result, "/") + "/" + strings.TrimLeft(path, "/")
}

func (r *dockerRegistry) get(path string, v interface{}) error {
	req := r.url(path)
	resp, err := http.Get(req)
	if err != nil {
		return fmt.Errorf("failed to get response. req=%s, err=%s", req, err)
	}

	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)

	if resp.StatusCode > 304 {
		return fmt.Errorf("got error response. req=%s, code=%d, body=%s", req, resp.StatusCode, bytes)
	}

	if err = json.Unmarshal(bytes, v); err != nil {
		return fmt.Errorf("could not parse response. req=%s, body=%s", req, bytes)
	}

	return nil
}
