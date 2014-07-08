package layer

import (
	"archive/tar"
)

type byName []*tar.Header

func (a byName) Len() int           { return len(a) }
func (a byName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byName) Less(i, j int) bool { return a[i].Name < a[j].Name }

type layersT []string

func (layers layersT) contains(layer string) bool {
	for i := range layers {
		if layers[i] == layer {
			return true
		}
	}
	return false
}

// getLayers returns a list of layer IDs starting from name down to the base layer
func getLayers(client DockerClient, name string) (layers layersT, err error) {

	layers = make(layersT, 0)
	for {
		image, err := client.InspectImage(name)
		if err != nil {
			return layers, err
		}
		layers = append(layers, image.ID)
		if image.Parent == "" {
			break
		}
		name = image.Parent
	}
	return layers, err
}
