package distribution

import (
	"context"
	"fmt"
	"mime"

	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// Manifest represents a registry object specifying a set of
// references and an optional target
type Manifest interface {
	// References returns a list of objects which make up this manifest.
	// A reference is anything which can be represented by a
	// Descriptor. These can consist of layers, resources or other
	// manifests.
	//
	// While no particular order is required, implementations should return
	// them from highest to lowest priority. For example, one might want to
	// return the base layer before the top layer.
	References() []v1.Descriptor

	// Payload provides the serialized format of the manifest, in addition to
	// the media type.
	Payload() (mediaType string, payload []byte, err error)
}

// ManifestService describes operations on manifests.
type ManifestService interface {
	// Exists returns true if the manifest exists.
	Exists(ctx context.Context, dgst digest.Digest) (bool, error)

	// Get retrieves the manifest specified by the given digest
	Get(ctx context.Context, dgst digest.Digest, options ...ManifestServiceOption) (Manifest, error)

	// Put creates or updates the given manifest returning the manifest digest
	Put(ctx context.Context, manifest Manifest, options ...ManifestServiceOption) (digest.Digest, error)

	// Delete removes the manifest specified by the given digest. Deleting
	// a manifest that doesn't exist will return ErrManifestNotFound
	Delete(ctx context.Context, dgst digest.Digest) error
}

// ManifestEnumerator enables iterating over manifests
type ManifestEnumerator interface {
	// Enumerate calls ingester for each manifest.
	Enumerate(ctx context.Context, ingester func(digest.Digest) error) error
}

// Describable is an interface for descriptors.
//
// Implementations of Describable are generally objects which can be
// described, not simply descriptors.
type Describable interface {
	// Descriptor returns the descriptor.
	Descriptor() v1.Descriptor
}

// ManifestMediaTypes returns the supported media types for manifests.
func ManifestMediaTypes() (mediaTypes []string) {
	for t := range mappings {
		if t != "" {
			mediaTypes = append(mediaTypes, t)
		}
	}
	return
}

// UnmarshalFunc implements manifest unmarshalling a given MediaType
type UnmarshalFunc func([]byte) (Manifest, v1.Descriptor, error)

var mappings = make(map[string]UnmarshalFunc)

// UnmarshalManifest looks up manifest unmarshal functions based on
// MediaType
func UnmarshalManifest(ctHeader string, p []byte) (Manifest, v1.Descriptor, error) {
	// Need to look up by the actual media type, not the raw contents of
	// the header. Strip semicolons and anything following them.
	var mediaType string
	if ctHeader != "" {
		var err error
		mediaType, _, err = mime.ParseMediaType(ctHeader)
		if err != nil {
			return nil, v1.Descriptor{}, err
		}
	}

	unmarshalFunc, ok := mappings[mediaType]
	if !ok {
		unmarshalFunc, ok = mappings[""]
		if !ok {
			return nil, v1.Descriptor{}, fmt.Errorf("unsupported manifest media type and no default available: %s", mediaType)
		}
	}

	return unmarshalFunc(p)
}

// RegisterManifestSchema registers an UnmarshalFunc for a given schema type.  This
// should be called from specific
func RegisterManifestSchema(mediaType string, u UnmarshalFunc) error {
	if _, ok := mappings[mediaType]; ok {
		return fmt.Errorf("manifest media type registration would overwrite existing: %s", mediaType)
	}
	mappings[mediaType] = u
	return nil
}
