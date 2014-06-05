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
	port uint64
}

// NewDockerRegistry creates a new DockerRegistry from the given specification. The
// specification must be of the form: <host>:<port>.
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

	return &DockerRegistry{host, port}, nil
}

// ListRemoteRepos returns a list of all repos in the registry.
func (dr *DockerRegistry) ListRemoteRepos() ([]string, error) {
	response := searchResponse{}

	if err := dr.get("/v1/search", &response); err != nil {
		glog.V(2).Infof("can't list remote repositories: %v", err)
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
		glog.V(2).Infof("can't construct repository name: %v", err)
		return map[string]string{}, err
	}

	path := "/v1/repositories/" + repoName + "/tags"

	response := make(map[string]string)
	if err = dr.get(path, &response); err != nil {
		glog.V(2).Infof("can't list remote repository tags: %v", err)
		return map[string]string{}, err
	}

	return response, nil
}

// GetRemoteRepoTag returns the UUID of the given image.
func (dr *DockerRegistry) GetRemoteRepoTag(repoTag string) (string, error) {
	repoName, tag, err := repoNameAndTag(repoTag)
	if err != nil {
		glog.V(2).Infof("can't construct repository name: %v", err)
		return "", err
	}

	if tag == "" {
		tag = "latest"
	}

	path := "/v1/repositories/" + repoName + "/tags/" + tag
	var response string

	if err = dr.get(path, &response); err != nil {
		glog.V(2).Infof("can't retrieve remote repository tag: %v", err)
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
		glog.V(2).Infof("can't list remote repositories: %v", err)
		return result, err
	}

	for _, repo := range repos {
		repoTags, err := dr.ListRemoteRepoTags(repo)
		if err != nil {
			glog.V(2).Infof("can't list remote repository tags: %v", err)
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
		glog.V(2).Infof("can't construct repository name: %v", err)
		return err
	}

	// Dev short circuit for ZEN-11996
	if noregistry {
		return nil
	}

	if tag == "" {
		tag = "latest"
	}
	path := "/v1/repositories/" + repoName + "/tags/" + tag
	return dr.put(path, imageID)
}

// RemoveRemoteImageTag removes the given tag from the registry.
// It is not considered an error if the tag already does not exist.
func (dr *DockerRegistry) RemoveRemoteImageTag(repoTag string) error {
	repoName, tag, err := repoNameAndTag(repoTag)
	if err != nil {
		glog.V(2).Infof("can't construct repository name: %v", err)
		return err
	}

	if tag == "" {
		return fmt.Errorf("no tag was specified for removal: %s", repoTag)
	}

	path := "/v1/repositories/" + repoName + "/tags/" + tag

	switch err = dr.delete(path); {
	case err == dockerclient.ErrNoSuchImage, err == nil:
		return nil
	default:
		return err
	}
}

// String returns the host and port of the registry.
func (dr *DockerRegistry) String() string {
	return fmt.Sprintf("%s:%d", dr.host, dr.port)
}

func pullImageFromRegistry(registry DockerRegistry, client *dockerclient.Client, name string) (e error) {
	defer func() {
		if e != nil {
			glog.V(2).Infof("commons.pullImageFromRegistry error: %s", e)
		}
	}()

	// Dev short circuit for ZEN-11996
	if noregistry {
		return nil
	}

	repoName, tag, err := repoNameAndTag(name)
	if err != nil {
		return err
	}
	if tag == "" {
		tag = "latest"
	}
	opts := dockerclient.PullImageOptions{
		Repository: fmt.Sprintf("%s:%s", repoName, tag),
		Registry:   registry.String(),
	}
	return client.PullImage(opts, auth)
}

func pushImageToRegistry(registry DockerRegistry, client *dockerclient.Client, name string, force bool) (e error) {
	defer func() {
		if e != nil {
			glog.V(2).Infof("commons.pushImageToRegistry name: %s, force: %t, error: %s", name, force, e)
		} else {
			glog.V(2).Infof("done pushing docker image %s to %s", name, registry)
		}
	}()

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
		hostAndPort += ":" + string(imageID.Port)
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
		glog.V(2).Infof("tagging image %s: %s", image.ID, tagOpts)
		if err = client.TagImage(image.ID, tagOpts); err != nil {
			return err
		}
	}
	opts := dockerclient.PushImageOptions{
		Name:     repoName,
		Tag:      imageID.Tag,
		Registry: registry.String(),
	}
	glog.V(2).Infof("pushing image: %s", opts)
	return client.PushImage(opts, auth)
}

// syncImageFromRegistry gets the local docker image to match the registry.
// If the image (name) is not already in the registry, pushes it in. (Error if
// the image is missing locally too) Otherwise, if the local image is missing,
// or its UUID differs from the registry, pulls from the registry.
func syncImageFromRegistry(registry DockerRegistry, client *dockerclient.Client, name string) (i *dockerclient.Image, e error) {
	defer func() {
		if e != nil {
			glog.V(2).Infof("commons.syncImageFromRegistry error: %s", e)
		}
	}()
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

var auth = dockerclient.AuthConfiguration{}

func (dr *DockerRegistry) url(path string) string {
	result := dr.String()
	if len(result) < 4 || result[:4] != "http" {
		result = "http://" + result
	}
	return strings.TrimRight(result, "/") + "/" + strings.TrimLeft(path, "/")
}

func (dr *DockerRegistry) get(path string, v interface{}) (e error) {
	defer func() {
		if e != nil {
			glog.V(2).Infof("commons.*dockerRegistry.get error: %s", e)
		}
	}()
	var (
		resp *http.Response
		err  error
	)
	req := dr.url(path)
	attempts := 0
	for {
		attempts++
		resp, err = http.Get(req)
		if err == nil {
			break
		}
		if attempts < 3 {
			continue
		}
		return fmt.Errorf("failed to get response. attempts=%d, req=%s, err=%s", attempts, req, err)
	}

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

func (dr *DockerRegistry) put(path string, data interface{}) (e error) {
	defer func() {
		if e != nil {
			glog.V(2).Infof("commons.*dockerRegistry.put error: %s", e)
		}
	}()
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

func (dr *DockerRegistry) delete(path string) (e error) {
	defer func() {
		if e != nil {
			glog.V(2).Infof("commons.*dockerRegistry.delete error: %s", e)
		}
	}()
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
