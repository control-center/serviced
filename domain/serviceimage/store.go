// Copyright 2015 The Serviced Authors.
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

package serviceimage

import (
    "github.com/control-center/serviced/datastore"
    "github.com/zenoss/elastigo/search"
    "github.com/zenoss/glog"

    "errors"
    "fmt"
    "strconv"
    "strings"
)

// NewStore creates a ServiceImageStore
func NewStore() *ServiceImageStore {
    return &ServiceImageStore{}
}

// UserStore type for interacting with User persistent storage
type ServiceImageStore struct {
    datastore.DataStore
}

// ImageKey creates a Key suitable for getting, putting and deleting Images
func ImageKey(imageID string) datastore.Key {
    keyID := strings.TrimSpace(imageID)
    return datastore.NewKey(kind, keyID)
}

var kind = "serviceimage"

// GetImagesByImageID returns all ServiceImage for an image ID
func (hs *ServiceImageStore) GetImagesByImageID(ctx datastore.Context, imageID string) ([]ServiceImage, error) {
    if imageID = strings.TrimSpace(imageID); imageID == "" {
        return nil, errors.New("empty imageID not allowed")
    }

    query := search.Query().Term("ImageID", imageID)
    search := search.Search("controlplane").Type(kind).Query(query)
    results, err := datastore.NewQuery(ctx).Execute(search)
    if err != nil {
        return nil, err
    }

    if results.Len() == 0 {
        return nil, nil
    }

    return convert(results)
}

// GetImagesByStatus returns all ServiceImage for a status
func (hs *ServiceImageStore) GetImagesByStatus(ctx datastore.Context, status ImageStatus) ([]ServiceImage, error) {
    if status.String() == "unknown" {
        return nil, fmt.Errorf("status %d is invalid", int(status))
    }

    query := search.Query().Term("Status", strconv.Itoa(int(status)))
    search := search.Search("controlplane").Type(kind).Query(query)
    results, err := datastore.NewQuery(ctx).Execute(search)
    if err != nil {
        return nil, err
    }

    if results.Len() == 0 {
        return nil, nil
    }

    return convert(results)
}

func convert(results datastore.Results) ([]ServiceImage, error) {
    glog.V(4).Infof("Results are %v", results)
    images := make([]ServiceImage, results.Len())
    for idx := range images {
        var image ServiceImage
        err := results.Get(idx, &image)
        if err != nil {
            return nil, err
        }
        glog.V(4).Infof("Adding %v to images", image)
        images[idx] = image
    }
    return images, nil
}
