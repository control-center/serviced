// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package docker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/zenoss/glog"
	dockerclient "github.com/zenoss/go-dockerclient"
	"github.com/zenoss/serviced/commons"
)

type DockerRegistry struct {
	host string
	port int64
}

var auth = dockerclient.AuthConfiguration{}

// NewDockerRegistry creates a new DockerRegistry from the given specification.
// The spec must be of the form <host>:<port>.
func NewDockerRegistry(spec string) (*DockerRegistry, error) {
	parts := strings.Split(spec, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("bad registry specification: %s", spec)
	}

	host := parts[0]
	port, err := strconv.ParseUint(parts[1], 10, 16)
	if err != nil {
		return nil, fmt.Errorf("invalid port %s: %v", parts[1], err)
	}

	return &DockerRegistry{host, int64(port)}, nil
}

// String returns the host and port of the registry.
func (dr *DockerRegistry) String() string {
	return strings.Join([]string{dr.host, strconv.Itoa(int(dr.port))}, ":")
}

// ListRemoteRepos returns a list of all repos in the registry.
func (dr *DockerRegistry) ListRemoteRepos() ([]string, error) {
	var response searchResponse
	if err := dr.get("/v1/search", &response); err != nil {
		glog.V(2).Infof("unable to list remote repos: %s", err)
		return []string{}, err
	}
	result := []string{}
	for _, entry := range response.results {
		result = append(result, entry.name)
	}
	return result, nil
}

// ListRemoteRepoTags returns a map from tag to image ID (UUID) for all tags
// in the registry associated with the given docker repo.
func (dr *DockerRegistry) ListRemoteRepoTags(repo string) (map[string]string, error) {
	repoName, _, err := repoNameAndTag(repo)
	if err != nil {
		return map[string]string{}, err
	}

	path := "/v1/repositories/" + repoName + "/tags"

	var response map[string]string
	if err = dr.get(path, &response); err != nil {
		glog.V(2).Infof("cannot get remote repo tags %s: %v", repo, err)
		return map[string]string{}, err
	}

	return response, nil
}

// GetRemoteRepoTag returns the UUID of the given image.
func (dr *DockerRegistry) GetRemoteRepoTag(repoTag string) (string, error) {
	// FIXME: this method is never used out side of this file and so shouldn't be exported.
	repoName, tag, err := repoNameAndTag(repoTag)
	if err != nil {
		return "", err
	}
	if tag == "" {
		tag = "latest"
	}

	path := "/v1/repositories/" + repoName + "/tags/" + tag

	var response string
	if err = dr.get(path, &response); err != nil {
		glog.V(2).Infof("cannot get remote repo tag %s: %v", repoName, err)
		return "", err
	}

	return response, nil
}

// ListRemoteImageTags returns a map from image ID (UUID) to a list of image
// names, where each name is like "namespace/repository:tag".
func (dr *DockerRegistry) ListRemoteImageTags() (map[string][]string, error) {
	result := make(map[string][]string)

	repos, err := dr.ListRemoteRepos()
	if err != nil {
		return result, err
	}

	for _, repo := range repos {
		repoTags, err := dr.ListRemoteRepoTags(repo)
		if err != nil {
			return result, err
		}
		for tag, imageID := range repoTags {
			tags, found := result[imageID]
			if !found {
				tags = []string{}
			}
			result[imageID] = append(tags, fmt.Sprintf("%s:%s", repo, tag))
		}
	}

	return result, nil
}

// TagRemoteImage tags the given image in the registry with the given tag.
// imageID: the UUID of the image
// repoTag: like "namespace/repository:tag" or "repository:tag"
func (dr *DockerRegistry) TagRemoteImage(imageID, repoTag string) error {
	repoName, tag, err := repoNameAndTag(repoTag)
	if err != nil {
		return err
	}
	if tag == "" {
		tag = "latest"
	}

	path := "/v1/repositories/" + repoName + "/tags/" + tag

	if err := dr.put(path, imageID); err != nil {
		glog.V(2).Infof("cannot put remote image tag %s: %v", repoName, err)
		return err
	}

	return nil
}

// RemoveRemoteImageTag removes the given tag from the registry.
// It is not considered an error if the tag already does not exist.
func (dr *DockerRegistry) RemoveRemoteImageTag(repoTag string) error {
	repoName, tag, err := repoNameAndTag(repoTag)
	if err != nil {
		return err
	}
	if tag == "" {
		return fmt.Errorf("no tag was specified for removal: %s", repoTag)
	}

	path := "/v1/repositories/" + repoName + "/tags/" + tag

	err = dr.delete(path)
	if err != nil && err != dockerclient.ErrNoSuchImage {
		return err
	}

	return nil
}

func pullImageFromRegistry(registry DockerRegistry, client *dockerclient.Client, name string) error {
	// Dev short circuit for ZEN-11996
	if noregistry {
		return nil
	}

	imageID, err := commons.ParseImageID(name)
	if err != nil {
		return err
	}
	tag := imageID.Tag
	if tag == "" {
		tag = "latest"
	}

	opts := dockerclient.PullImageOptions{
		Repository: imageID.BaseName(),
		Tag:        tag,
		Registry:   registry.String(),
	}
	return client.PullImage(opts, auth)
}

func pushImageToRegistry(registry DockerRegistry, client *dockerclient.Client, name string, force bool) error {
	// Dev short circuit for ZEN-11996
	if noregistry {
		return nil
	}

	glog.V(2).Infof("pushing docker image %s to %s...", name, registry)
	imageID, err := commons.ParseImageID(name)
	if err != nil {
		return fmt.Errorf("invalid image name: %s, error: %s", name, err)
	}

	repoName := registry.String() + "/"
	if imageID.User != "" {
		repoName += imageID.User + "/"
	}

	repoName += imageID.Repo
	fullName := repoName

	if imageID.Tag != "" {
		fullName += ":" + imageID.Tag
	}

	hostAndPort := imageID.Host
	if imageID.Port != 0 {
		hostAndPort += ":" + fmt.Sprintf("%d", imageID.Port)
	}

	if hostAndPort != registry.String() {
		image, err := client.InspectImage(name)
		if err != nil {
			return err
		}

		if force == dontForce {
			// check to make sure that either the full (adjusted) name doesn't exist
			// locally, or if it does exist, that it points to the exact same image.
			foundImage, err := client.InspectImage(fullName)
			if err != nil && err != dockerclient.ErrNoSuchImage {
				return err
			}
			if err == nil && foundImage.ID != image.ID {
				err = fmt.Errorf("refusing to push image %s (%s) to %s because %s is a different image (%s) locally", name, image.ID, registry, repoName, foundImage.ID)
				return err
			}
		}

		// tag the image locally
		tagOpts := dockerclient.TagImageOptions{
			Repo:  repoName,
			Force: true,
			Tag:   imageID.Tag,
		}

		glog.V(2).Infof("tagging image %s: %+v", image.ID, tagOpts)
		if err = client.TagImage(image.ID, tagOpts); err != nil {
			glog.V(2).Infof("tagging image %s: %+v failed: %v", image.ID, tagOpts, err)
			return err
		}
	}

	opts := dockerclient.PushImageOptions{
		Name:     repoName,
		Tag:      imageID.Tag,
		Registry: registry.String(),
	}
	glog.Infof("pushing image: %+v", opts)
	if err := client.PushImage(opts, auth); err != nil {
		glog.V(2).Infof("pushing image: %+v failed: %v", opts, err)
	}

	return nil
}

// syncImageFromRegistry gets the local docker image to match the registry.
// If the image (name) is not already in the registry, pushes it in. (Error if
// the image is missing locally too) Otherwise, if the local image is missing,
// or its UUID differs from the registry, pulls from the registry.
func syncImageFromRegistry(registry DockerRegistry, client *dockerclient.Client, name string) (*dockerclient.Image, error) {
	image, err := client.InspectImage(name)
	if err != nil && err != dockerclient.ErrNoSuchImage {
		return nil, err
	}

	// Dev short circuit for ZEN-11996
	if noregistry {
		return image, err
	}

	uuid, err := registry.GetRemoteRepoTag(name)
	if err != nil {
		if err != dockerclient.ErrNoSuchImage {
			return nil, err
		}
		// Well, let's push then...
		if err = pushImageToRegistry(registry, client, name, dontForce); err != nil {
			return nil, err
		}
		uuid, err = registry.GetRemoteRepoTag(name)
		if err != nil {
			return nil, err
		}
	}

	if image == nil || image.ID != uuid {
		if err = pullImageFromRegistry(registry, client, name); err != nil {
			return nil, fmt.Errorf("failed to pull image \"%s\" from registry (%s): %s", name, registry, err)
		}
		// Assert that it actually got pulled...
		image, err = client.InspectImage(name)
		if err != nil {
			return nil, err
		}
		if image == nil || image.ID != uuid {
			return nil, fmt.Errorf("pulled image \"%s\" from registry (%s), but image id (%s) does not match", name, registry, uuid)
		}
	}

	return image, err
}

func (dr *DockerRegistry) url(path string) string {
	result := dr.String()

	if len(result) < 4 || result[:4] != "http" {
		result = "http://" + result
	}

	return strings.TrimRight(result, "/") + "/" + strings.TrimLeft(path, "/")
}

func (dr *DockerRegistry) get(path string, v interface{}) error {
	var (
		resp *http.Response
		err  error
	)

	req := dr.url(path)

	for attempts := 0; attempts < 3; attempts++ {
		if resp, err = http.Get(req); err != nil {
			continue
		}
		goto success // yes, it's a goto and here it happens to make sense
	}
	return fmt.Errorf("failed to get response")

success:
	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)

	if resp.StatusCode == 404 {
		return dockerclient.ErrNoSuchImage
	}
	if resp.StatusCode > 304 {
		return fmt.Errorf("got error response. req=%s, code=%d, body=%s", req, resp.StatusCode, bytes)
	}
	if err = json.Unmarshal(bytes, v); err != nil {
		return fmt.Errorf("could not parse response. req=%s, body=%s", req, bytes)
	}

	return nil
}

func (dr *DockerRegistry) put(path string, data interface{}) error {
	body, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", dr.url(path), bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if res.StatusCode == 404 {
		return dockerclient.ErrNoSuchImage
	}

	bodyBytes, err := ioutil.ReadAll(res.Body)
	if res.StatusCode > 304 {
		return fmt.Errorf("status: %d, error: %s", res.StatusCode, bodyBytes)
	}

	return nil
}

func (dr *DockerRegistry) delete(path string) error {
	req, err := http.NewRequest("DELETE", dr.url(path), nil)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if res.StatusCode == 404 {
		return dockerclient.ErrNoSuchImage
	}

	bodyBytes, err := ioutil.ReadAll(res.Body)
	if res.StatusCode > 304 {
		return fmt.Errorf("status: %d, error: %s", res.StatusCode, bodyBytes)
	}

	return nil
}

// repoNameAndTag parses the image name, removing any host, port,
// and tag, and returns a string like "(user|'library')/repo" for the
// repo name, and another string containing just the tag (or "" if none).
func repoNameAndTag(imageName string) (string, string, error) {
	s := make([]string, 0, 5)
	imageID, err := commons.ParseImageID(imageName)
	if err != nil {
		return "", "", fmt.Errorf("invalid image name: %s, error: %s", imageName, err)
	}
	if imageID.User == "" {
		s = append(s, "library/")
	} else {
		s = append(s, imageID.User, "/")
	}
	s = append(s, imageID.Repo)
	return strings.Join(s, ""), imageID.Tag, nil
}

type searchResponse struct {
	results []searchResponseEntry
}

type searchResponseEntry struct {
	description string
	name        string
}

func removeString(a []string, s string) []string {
	result := make([]string, 0, len(a))
	for _, e := range a {
		if e != s {
			result = append(result, e)
		}
	}
	return result
}
