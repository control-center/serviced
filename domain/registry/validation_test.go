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

package registry

import (
	"strings"
	"testing"
)

func TestRegistryImage_EmptyLibrary(t *testing.T) {
	image := &Image{
		Library: "",
		Repo:    "repo",
		Tag:     "tag",
		UUID:    "1234",
	}
	if err := image.ValidEntity(); err == nil {
		t.Errorf("Expected error, got nil")
	} else if !strings.Contains(err.Error(), "Image.Library") {
		t.Errorf("Validation on the wrong field: %s", err)
	}
}

func TestRegistryImage_InvalidCharsLibrary(t *testing.T) {
	image := &Image{
		Library: " library ",
		Repo:    "repo",
		Tag:     "tag",
		UUID:    "1234",
	}
	if err := image.ValidEntity(); err == nil {
		t.Errorf("Expected error, got nil")
	} else if !strings.Contains(err.Error(), "Image.Library") {
		t.Errorf("Validation on the wrong field: %s", err)
	}
}

func TestRegistryImage_EmptyRepo(t *testing.T) {
	image := &Image{
		Library: "library",
		Repo:    "",
		Tag:     "tag",
		UUID:    "1234",
	}
	if err := image.ValidEntity(); err == nil {
		t.Errorf("Expected error, got nil")
	} else if !strings.Contains(err.Error(), "Image.Repo") {
		t.Errorf("Validation on the wrong field: %s", err)
	}
}

func TestRegistryImage_InvalidCharsRepo(t *testing.T) {
	image := &Image{
		Library: "library",
		Repo:    " repo ",
		Tag:     "tag",
		UUID:    "1234",
	}
	if err := image.ValidEntity(); err == nil {
		t.Errorf("Expected error, got nil")
	} else if !strings.Contains(err.Error(), "Image.Repo") {
		t.Errorf("Validation on the wrong field: %s", err)
	}
}

func TestRegistryImage_EmptyTag(t *testing.T) {
	image := &Image{
		Library: "library",
		Repo:    "repo",
		Tag:     "",
		UUID:    "1234",
	}
	if err := image.ValidEntity(); err == nil {
		t.Errorf("Expected error, got nil")
	} else if !strings.Contains(err.Error(), "Image.Tag") {
		t.Errorf("Validation on the wrong field: %s", err)
	}
}

func TestRegistryImage_InvalidCharsTag(t *testing.T) {
	image := &Image{
		Library: "library",
		Repo:    "repo",
		Tag:     " tag ",
		UUID:    "1234",
	}
	if err := image.ValidEntity(); err == nil {
		t.Errorf("Expected error, got nil")
	} else if !strings.Contains(err.Error(), "Image.Tag") {
		t.Errorf("Validation on the wrong field: %s", err)
	}
}

func TestRegistryImage_EmptyUUID(t *testing.T) {
	image := &Image{
		Library: "library",
		Repo:    "repo",
		Tag:     "tag",
		UUID:    "",
	}
	if err := image.ValidEntity(); err == nil {
		t.Errorf("Expected error, got nil")
	} else if !strings.Contains(err.Error(), "Image.UUID") {
		t.Errorf("Validation on the wrong field: %s", err)
	}
}

func TestRegistryImage_InvalidCharsUUID(t *testing.T) {
	image := &Image{
		Library: "library",
		Repo:    "repo",
		Tag:     "tag",
		UUID:    " 1234 ",
	}
	if err := image.ValidEntity(); err == nil {
		t.Errorf("Expected error, got nil")
	} else if !strings.Contains(err.Error(), "Image.UUID") {
		t.Errorf("Validation on the wrong field: %s", err)
	}
}

func TestRegistryImage_Success(t *testing.T) {
	image := &Image{
		Library: "library",
		Repo:    "repo",
		Tag:     "tag",
		UUID:    "1234",
	}
	if err := image.ValidEntity(); err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
}
