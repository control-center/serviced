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

package layer

import (
	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/commons/circular"
	"github.com/fsouza/go-dockerclient"
	"github.com/zenoss/glog"

	"archive/tar"
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
	"sort"
	"strings"
)

// FilesToIgnore are the files that Squash will ignore when computing diffs.
var FilesToIgnore = map[string]struct{}{
	".dockerenv":      struct{}{},
	".dockerinit":     struct{}{},
	"./":              struct{}{},
	"dev/":            struct{}{},
	"dev/console":     struct{}{},
	"dev/shm/":        struct{}{},
	"etc/":            struct{}{},
	"etc/hostname":    struct{}{},
	"etc/hosts":       struct{}{},
	"etc/mtab":        struct{}{},
	"etc/resolv.conf": struct{}{},
}

// DockerClient is the minimal docker client interface this package needs.
type DockerClient interface {
	CreateContainer(docker.CreateContainerOptions) (*docker.Container, error)
	RemoveContainer(docker.RemoveContainerOptions) error
	ExportContainer(docker.ExportContainerOptions) error
	InspectImage(name string) (*docker.Image, error)
	ImportImage(docker.ImportImageOptions) error
}

func export(client DockerClient, image string) (f *os.File, err error) {

	glog.V(1).Infof("creating container for export of %s", image)
	container, err := client.CreateContainer(docker.CreateContainerOptions{Config: &docker.Config{Cmd: []string{"/bin/true"}, Image: image}})
	if err != nil {
		return f, err
	}
	glog.V(1).Infof("create container %s for image %s", container.ID, image)
	defer client.RemoveContainer(docker.RemoveContainerOptions{ID: container.ID})

	glog.V(1).Infof("exporting %s", image)
	file, err := ioutil.TempFile("", fmt.Sprintf("docker_squash_%s_", image))
	if err == nil {
		err = client.ExportContainer(docker.ExportContainerOptions{container.ID, file})
		if err == nil {
			_, err = file.Seek(0, 0)
		}
	}
	return file, err
}

func getTarHeaders(tr *tar.Reader) ([]*tar.Header, error) {
	topHeaders := make([]*tar.Header, 0)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		topHeaders = append(topHeaders, hdr)
	}
	sort.Sort(byName(topHeaders))
	return topHeaders, nil
}

func pushImage(client DockerClient, imageName string, image *os.File) (imageID string, err error) {
	parsedImage, err := commons.ParseImageID(imageName)
	if err != nil {
		return "", err
	}

	buffer := circular.NewBuffer(1000)
	opts := docker.ImportImageOptions{
		Repository:   parsedImage.BaseName(),
		Tag:          parsedImage.Tag,
		Source:       "-",
		InputStream:  image,
		OutputStream: buffer,
	}
	err = client.ImportImage(opts)
	if err != nil {
		return "", err
	}
	idbuffer := make([]byte, 100)
	n, err := buffer.Read(idbuffer)
	if err != nil {
		return "", err
	}
	s := fmt.Sprintf("%s", strings.TrimSpace(string(idbuffer[0:n])))
	return s, nil
}

// Squash flattens the image down to the downToLayer and optionally retag the generated
// layer with newName. The resulting layer is returned in resultImageID.
func Squash(client DockerClient, imageName, downToLayer, newName, tempDir string) (resultImageID string, err error) {

	// get list of layers for the image
	layers, err := getLayers(client, imageName)
	if err != nil {
		return "", err
	}
	topName := layers[0] // top layer is the first layer

	// if the downToLayer is specified, make sure it's in our list of layers
	if downToLayer != "" {
		if !layers.contains(downToLayer) {
			return "", fmt.Errorf("invalid downto layer id: %s is not in %+v", downToLayer, layers)
		}
	}

	// export the top image
	topTar, err := export(client, topName)
	if err != nil {
		return "", fmt.Errorf("error exporting %s: %s", topName, err)
	}
	defer topTar.Close()
	defer os.Remove(topTar.Name())

	// if no downToLayer was specified, lets squash the whole thing by uploaded the exported image
	if downToLayer == "" {
		return pushImage(client, imageName, topTar)
	}

	// lets extract the headers of the top image and sort them
	topHeaders, err := getTarHeaders(tar.NewReader(topTar))
	if err != nil {
		return "", fmt.Errorf("error getting headers %s: %s", topName, err)
	}
	if len(topHeaders) == 0 {
		return "", fmt.Errorf("empty top layer")
	}

	// let's extract the headers of the base image and sort them
	glog.V(1).Infof("exporting base layer %s", downToLayer)
	baseTar, err := export(client, downToLayer)
	if err != nil {
		return "", fmt.Errorf("error exporting base image %s: %s", downToLayer, err)
	}
	defer baseTar.Close()
	defer os.Remove(baseTar.Name())
	baseHeaders, err := getTarHeaders(tar.NewReader(baseTar))
	if err != nil {
		return "", fmt.Errorf("error getting headers %s: %s", downToLayer, err)
	}

	// the headers for the top and base images are sorted
	// we can merge them and determine which files were added/deleted
	var i, j int

	baseHdr, topHdr := baseHeaders[0], topHeaders[0]
	removals := make([]string, 0)
	additions := make(map[string]struct{})
	for {

		if i > len(baseHeaders)-1 {
			//get all of j
			for n := j; n < len(topHeaders); n++ {
				additions[topHeaders[n].Name] = struct{}{}
			}
			break
		}
		if j > len(topHeaders)-1 {
			//get all of i
			for n := i; n < len(baseHeaders); n++ {
				removals = append(removals, baseHeaders[n].Name)
			}
			break
		}
		baseHdr = baseHeaders[i]
		topHdr = topHeaders[j]

		switch {
		case baseHdr.Name == topHdr.Name:
			if !reflect.DeepEqual(baseHdr, topHdr) {
				if _, ignore := FilesToIgnore[topHdr.Name]; !ignore {
					additions[topHdr.Name] = struct{}{}
				}
			}
			i++
			j++
		case baseHdr.Name > topHdr.Name:
			if _, ignore := FilesToIgnore[topHdr.Name]; !ignore {
				additions[topHdr.Name] = struct{}{}
			}
			j++
		case baseHdr.Name < topHdr.Name:
			if _, ignore := FilesToIgnore[baseHdr.Name]; !ignore {
				removals = append(removals, baseHdr.Name)
			}
			i++
		}
	}
	// reset top layer
	if _, err = topTar.Seek(0, 0); err != nil {
		return "", err
	}
	tarReader := tar.NewReader(topTar)

	// create tar directory
	tdirName, err := ioutil.TempDir(tempDir, "docker_build_")
	if err != nil {
		return "", err
	}
	outputFilename := fmt.Sprintf("%s/%s.tar", tdirName, topName)
	output, err := os.Create(outputFilename)
	if err != nil {
		return "", err
	}
	// defer os.Remove(output.Name)
	tarWriter := tar.NewWriter(output)
	buffer := make([]byte, 1024*1024)
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if _, found := additions[hdr.Name]; found {
			if err := tarWriter.WriteHeader(hdr); err != nil {
				return "", err
			}
			for {
				n, err := tarReader.Read(buffer)
				if err == io.EOF {
					break
				}
				if err != nil {
					return "", err
				}
				wn, err := tarWriter.Write(buffer[0:n])
				if err != nil {
					return "", err
				}
				if wn != n {
					return "", fmt.Errorf("got %d, wrote %d", n, wn)
				}
			}
		}
	}
	dfile, err := os.Create(tdirName + "/Dockerfile")
	if err != nil {
		return "", err
	}
	dockerfile := bufio.NewWriter(dfile)
	if _, err := dockerfile.WriteString(fmt.Sprintf("FROM %s\n ADD %s.tar /\n", downToLayer, topName)); err != nil {
		return "", err
	}
	if len(removals) > 0 {
		removalStr := "#/bin/sh\nrm -Rf " + strings.Join(removals, "\nrm -Rf ") + "\nrm /removal.sh\n"
		if err := tarWriter.WriteHeader(&tar.Header{
			Name:  "removal.sh",
			Mode:  0775,
			Uid:   0,
			Gid:   0,
			Size:  int64(len(removalStr)),
			Uname: "root",
			Gname: "root",
		}); err != nil {
			return "", err
		}
		if _, err := tarWriter.Write([]byte(removalStr)); err != nil {
			return "", err
		}
		if _, err := dockerfile.WriteString("RUN /removal.sh\n"); err != nil {
			return "", err
		}
	}
	if err := tarWriter.Flush(); err != nil {
		return "", err
	}
	if err := dockerfile.Flush(); err != nil {
		return "", err
	}
	dfile.Close()

	// TODO: replace this with a streaming tar built on the fly
	// so that we don't rely on calling the CLI
	buildArgs := []string{"build"}
	tagName := ""
	if newName != "" {
		tagName = newName
		buildArgs = append(buildArgs, "--tag", newName)
	} else {
		tagName = imageName
		buildArgs = append(buildArgs, "--tag", imageName)
	}
	buildArgs = append(buildArgs, tdirName)
	build := exec.Command("docker", buildArgs...)
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		return "", fmt.Errorf("error building image")
	}

	newImage, err := client.InspectImage(tagName)
	if err != nil {
		return "", err
	}

	return newImage.ID, nil
}
