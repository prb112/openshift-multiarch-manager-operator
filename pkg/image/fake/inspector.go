package fake

import (
	"context"
	"errors"

	"k8s.io/apimachinery/pkg/util/sets"
)

type registryInspector struct {
	globalPullSecret []byte
}

const (
	ArchitectureAmd64   = "amd64"
	ArchitectureArm64   = "arm64"
	ArchitecturePpc64le = "ppc64le"
	ArchitectureS390x   = "s390x"
)

const (
	SingleArchAmd64Image = "my-registry.io/library/single-arch-amd64-image:latest"
	SingleArchArm64Image = "my-registry.io/library/single-arch-arm64-image:latest"
	MultiArchImage       = "my-registry.io/library/multi-arch-image:latest"
	MultiArchImage2      = "my-registry.io/library/multi-arch-image2:latest"
)

// MockImagesArchitectureMap returns a map of image references to their supported architectures
// We use a function instead of a global variable to force immutability
func MockImagesArchitectureMap() map[string]sets.Set[string] {
	return map[string]sets.Set[string]{
		SingleArchAmd64Image: sets.New[string](ArchitectureAmd64),
		SingleArchArm64Image: sets.New[string](ArchitectureArm64),
		MultiArchImage:       sets.New[string](ArchitectureAmd64, ArchitectureArm64),
		MultiArchImage2: sets.New[string](ArchitectureAmd64, ArchitectureArm64,
			ArchitecturePpc64le, ArchitectureS390x),
	}
}

func (i *registryInspector) GetCompatibleArchitecturesSet(ctx context.Context, imageReference string,
	secrets [][]byte) (supportedArchitectures sets.Set[string], err error) {
	// we expect the imageReference to start with `//`. Let's remove it
	imageReference = imageReference[2:]
	if archSet, ok := MockImagesArchitectureMap()[imageReference]; ok {
		return archSet, nil
	}
	// The image is not in the mock map, return an empty set (emulating an image not found or any other error)
	return nil, errors.New("image not found")
}

func newRegistryInspector() *registryInspector {
	return &registryInspector{}
}
