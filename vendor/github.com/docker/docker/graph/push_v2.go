package graph

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution"
	"github.com/docker/distribution/digest"
	"github.com/docker/distribution/manifest"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/docker/image"
	"github.com/docker/docker/pkg/progressreader"
	"github.com/docker/docker/pkg/streamformatter"
	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/docker/registry"
	"github.com/docker/docker/runconfig"
	"github.com/docker/docker/utils"
	"golang.org/x/net/context"
)

const compressionBufSize = 32768

type v2Pusher struct {
	*TagStore
	endpoint  registry.APIEndpoint
	localRepo repository
	repoInfo  *registry.RepositoryInfo
	config    *ImagePushConfig
	sf        *streamformatter.StreamFormatter
	repo      distribution.Repository

	// layersPushed is the set of layers known to exist on the remote side.
	// This avoids redundant queries when pushing multiple tags that
	// involve the same layers.
	layersPushed map[digest.Digest]bool
}

func (p *v2Pusher) Push() (fallback bool, err error) {
	p.repo, err = newV2Repository(p.repoInfo, p.endpoint, p.config.MetaHeaders, p.config.AuthConfig, "push", "pull")
	if err != nil {
		logrus.Debugf("Error getting v2 registry: %v", err)
		return true, err
	}
	return false, p.pushV2Repository(p.config.Tag)
}

func (p *v2Pusher) getImageTags(askedTag string) ([]string, error) {
	logrus.Debugf("Checking %q against %#v", askedTag, p.localRepo)
	if len(askedTag) > 0 {
		if _, ok := p.localRepo[askedTag]; !ok || utils.DigestReference(askedTag) {
			return nil, fmt.Errorf("Tag does not exist for %s", askedTag)
		}
		return []string{askedTag}, nil
	}
	var tags []string
	for tag := range p.localRepo {
		if !utils.DigestReference(tag) {
			tags = append(tags, tag)
		}
	}
	return tags, nil
}

func (p *v2Pusher) pushV2Repository(tag string) error {
	localName := p.repoInfo.LocalName
	if _, found := p.poolAdd("push", localName); found {
		return fmt.Errorf("push or pull %s is already in progress", localName)
	}
	defer p.poolRemove("push", localName)

	tags, err := p.getImageTags(tag)
	if err != nil {
		return fmt.Errorf("error getting tags for %s: %s", localName, err)
	}
	if len(tags) == 0 {
		return fmt.Errorf("no tags to push for %s", localName)
	}

	for _, tag := range tags {
		if err := p.pushV2Tag(tag); err != nil {
			return err
		}
	}

	return nil
}

func (p *v2Pusher) pushV2Tag(tag string) error {
	logrus.Debugf("Pushing repository: %s:%s", p.repo.Name(), tag)

	layerID, exists := p.localRepo[tag]
	if !exists {
		return fmt.Errorf("tag does not exist: %s", tag)
	}

	layersSeen := make(map[string]bool)

	layer, err := p.graph.Get(layerID)
	if err != nil {
		return err
	}

	m := &schema1.Manifest{
		Versioned: manifest.Versioned{
			SchemaVersion: 1,
		},
		Name:         p.repo.Name(),
		Tag:          tag,
		Architecture: layer.Architecture,
		FSLayers:     []schema1.FSLayer{},
		History:      []schema1.History{},
	}

	var metadata runconfig.Config
	if layer != nil && layer.Config != nil {
		metadata = *layer.Config
	}

	out := p.config.OutStream

	for ; layer != nil; layer, err = p.graph.GetParent(layer) {
		if err != nil {
			return err
		}

		// break early if layer has already been seen in this image,
		// this prevents infinite loops on layers which loopback, this
		// cannot be prevented since layer IDs are not merkle hashes
		// TODO(dmcgowan): throw error if no valid use case is found
		if layersSeen[layer.ID] {
			break
		}

		logrus.Debugf("Pushing layer: %s", layer.ID)

		if layer.Config != nil && metadata.Image != layer.ID {
			if err := runconfig.Merge(&metadata, layer.Config); err != nil {
				return err
			}
		}

		var exists bool
		dgst, err := p.graph.getLayerDigestWithLock(layer.ID)
		switch err {
		case nil:
			if p.layersPushed[dgst] {
				exists = true
				// break out of switch, it is already known that
				// the push is not needed and therefore doing a
				// stat is unnecessary
				break
			}
			_, err := p.repo.Blobs(context.Background()).Stat(context.Background(), dgst)
			switch err {
			case nil:
				exists = true
				out.Write(p.sf.FormatProgress(stringid.TruncateID(layer.ID), "Image already exists", nil))
			case distribution.ErrBlobUnknown:
				// nop
			default:
				out.Write(p.sf.FormatProgress(stringid.TruncateID(layer.ID), "Image push failed", nil))
				return err
			}
		case errDigestNotSet:
			// nop
		case digest.ErrDigestInvalidFormat, digest.ErrDigestUnsupported:
			return fmt.Errorf("error getting image checksum: %v", err)
		}

		// if digest was empty or not saved, or if blob does not exist on the remote repository,
		// then fetch it.
		if !exists {
			var pushDigest digest.Digest
			if pushDigest, err = p.pushV2Image(p.repo.Blobs(context.Background()), layer); err != nil {
				return err
			}
			if dgst == "" {
				// Cache new checksum
				if err := p.graph.setLayerDigestWithLock(layer.ID, pushDigest); err != nil {
					return err
				}
			}
			dgst = pushDigest
		}

		// read v1Compatibility config, generate new if needed
		jsonData, err := p.graph.generateV1CompatibilityChain(layer.ID)
		if err != nil {
			return err
		}

		m.FSLayers = append(m.FSLayers, schema1.FSLayer{BlobSum: dgst})
		m.History = append(m.History, schema1.History{V1Compatibility: string(jsonData)})

		layersSeen[layer.ID] = true
		p.layersPushed[dgst] = true
	}

	logrus.Infof("Signed manifest for %s:%s using daemon's key: %s", p.repo.Name(), tag, p.trustKey.KeyID())
	signed, err := schema1.Sign(m, p.trustKey)
	if err != nil {
		return err
	}

	manifestDigest, manifestSize, err := digestFromManifest(signed, p.repo.Name())
	if err != nil {
		return err
	}
	if manifestDigest != "" {
		out.Write(p.sf.FormatStatus("", "%s: digest: %s size: %d", tag, manifestDigest, manifestSize))
	}

	manSvc, err := p.repo.Manifests(context.Background())
	if err != nil {
		return err
	}
	return manSvc.Put(signed)
}

func (p *v2Pusher) pushV2Image(bs distribution.BlobService, img *image.Image) (digest.Digest, error) {
	out := p.config.OutStream

	out.Write(p.sf.FormatProgress(stringid.TruncateID(img.ID), "Preparing", nil))

	image, err := p.graph.Get(img.ID)
	if err != nil {
		return "", err
	}
	arch, err := p.graph.tarLayer(image)
	if err != nil {
		return "", err
	}
	defer arch.Close()

	// Send the layer
	layerUpload, err := bs.Create(context.Background())
	if err != nil {
		return "", err
	}
	defer layerUpload.Close()

	reader := progressreader.New(progressreader.Config{
		In:        ioutil.NopCloser(arch), // we'll take care of close here.
		Out:       out,
		Formatter: p.sf,

		// TODO(stevvooe): This may cause a size reporting error. Try to get
		// this from tar-split or elsewhere. The main issue here is that we
		// don't want to buffer to disk *just* to calculate the size.
		Size: img.Size,

		NewLines: false,
		ID:       stringid.TruncateID(img.ID),
		Action:   "Pushing",
	})

	digester := digest.Canonical.New()
	// HACK: The MultiWriter doesn't write directly to layerUpload because
	// we must make sure the ReadFrom is used, not Write. Using Write would
	// send a PATCH request for every Write call.
	pipeReader, pipeWriter := io.Pipe()
	// Use a bufio.Writer to avoid excessive chunking in HTTP request.
	bufWriter := bufio.NewWriterSize(io.MultiWriter(pipeWriter, digester.Hash()), compressionBufSize)
	compressor := gzip.NewWriter(bufWriter)

	go func() {
		_, err := io.Copy(compressor, reader)
		if err == nil {
			err = compressor.Close()
		}
		if err == nil {
			err = bufWriter.Flush()
		}
		if err != nil {
			pipeWriter.CloseWithError(err)
		} else {
			pipeWriter.Close()
		}
	}()

	out.Write(p.sf.FormatProgress(stringid.TruncateID(img.ID), "Pushing", nil))
	nn, err := layerUpload.ReadFrom(pipeReader)
	pipeReader.Close()
	if err != nil {
		return "", err
	}

	dgst := digester.Digest()
	if _, err := layerUpload.Commit(context.Background(), distribution.Descriptor{Digest: dgst}); err != nil {
		return "", err
	}

	logrus.Debugf("uploaded layer %s (%s), %d bytes", img.ID, dgst, nn)
	out.Write(p.sf.FormatProgress(stringid.TruncateID(img.ID), "Pushed", nil))

	return dgst, nil
}
