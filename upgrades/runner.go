// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package upgrades

import (
//	"fmt"
//	"regexp"
//
//	"github.com/control-center/serviced/commons"
//	"github.com/control-center/serviced/commons/docker"
//	"github.com/control-center/serviced/domain/service"
//	"github.com/docker/docker/pkg/parsers"
)

//type Descriptor struct {
//	Description string
//	Version     string
//	Images      []string
//	Commands    []Command
//}
//
//type Command struct {
//	Type    string
//	Command string
//}
//
//var SERVICE_RUN_TYPE = "service_run"
//
//type Runner interface {
//	Upgrade(serviceID string, descriptor Descriptor) error
//}
//
//type ServiceLookup func(serviceID string) (*service.Service, error)
//type Snapshot func(serviceID string) (string, error)
//
//func NewRunner() Runner {
//	return nil
//}
//
//type runner struct {
//	svcLookup      ServiceLookup
//	snapshot       Snapshot
//	dockerRegistry string
//}
//
//func (r *runner) Upgrade(serviceID string, descriptor Descriptor) error {
//	svc, err := r.svcLookup(serviceID)
//	if err != nil || svc == nil {
//		return fmt.Errorf("could not find service %s for upgrade: %s", serviceID, err)
//	}
//	if svc.ParentID != "" && svc.ParentID != svc.ID {
//		return error.New("must provide parent service for performing upgrade")
//	}
//
//	//TODO: validate descriptor
//
//	//Pull all images
//	imageIDs := make([]commons.ImageID, len(descriptor.Images))
//	for _, imageName := range descriptor.Images {
//		//parse the image
//		imageID, err := commons.ParseImageID(imageName)
//		if err != nil {
//			return err
//		}
//		tag := imageID.Tag
//		if tag == "" {
//			tag = "latest"
//		}
//		image := fmt.Sprintf("%s:%s", imageID.BaseName(), tag)
//		glog.Infof("Pulling image %s, this may take a while...", image)
//		if err := docker.PullImage(image); err != nil {
//			return fmt.Errorf("Unable to pull image %s", image)
//		}
//		imageIDs := append(imageIDs, imageID)
//	}
//
//	images := make(map[string]docker.Image)
//	//verify all images have been pulled
//	for _, imageID := range imageIDs {
//		img, err := docker.FindImage(imageID.String(), false)
//		if err != nil {
//			msg := fmt.Errorf("could not look up image %s: %s. Check your docker login and retry service deployment.", imageID, err)
//			glog.Error(err.Error())
//			return msg
//		}
//		images[imageID.String()] = img
//	}
//
//	//Set up function to roll back to snapshot
//	rollback := true
//	var snapshotID string
//	defer func() {
//		if rollback == true {
//			glog.Infof("Rolling back upgrade")
//		}
//	}()
//	//Take snapshot of service ID
//	snapshotID, err = r.snapshot(serviceID)
//	if err != nil {
//		return err
//	}
//
//	//Tag all images to latest all images
//	//TODO: tag them to something else as well????
//	for _, imageID := range imageIDs {
//		newTag, err = renameImageID(r.dockerRegistry, imageID.String(), serviceID, "latest")
//		if err != nil {
//			return err
//		}
//		if img, found := images[imageID.String()]; !found {
//			return fmt.Errorf("could not find image %s for tagging", imageName)
//		} else {
//			glog.Infof("tagging image %s to %s ", imageName, imageTag)
//			if _, err := img.Tag(newTag.String()); err != nil {
//				glog.Errorf("could not tag image: %s (%v)", imageName, err)
//				return "", err
//			}
//		}
//
//	}
//
//	//	//Iterate through commands to run (shell out to serviced run ... ????)
//	//	for _, runCommands := descriptor.Commands{
//	//		cmd := exec.Command("serviced", "service", "run", commandSvcID, command)
//	//
//	//			// Suppressing docker output (too chatty)
//	//			if err := cmd.Run(); err != nil {
//	//				glog.Errorf("Unable to pull image %s", repotag)
//	//				return fmt.Errorf("image %s not available", repotag)
//	//			}
//	//
//	//	}
//
//	// got to the end without errors, don't rollback
//	rollback = false
//	return nil
//}
//
//func renameImageID(dockerRegistry, tenantId string, imgID string, tag string) (commons.ImageID, error) {
//	repo, _ := parsers.ParseRepositoryTag(imageId)
//	re := regexp.MustCompile("/?([^/]+)\\z")
//	matches := re.FindStringSubmatch(repo)
//	if matches == nil {
//		return "", errors.New("malformed imageid")
//	}
//	name := matches[1]
//	newImageID := fmt.Sprintf("%s/%s/%s:%s", dockerRegistry, tenantId, name, tag)
//	return commons.ParseImageID(newImageID)
//}
