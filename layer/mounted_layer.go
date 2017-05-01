package layer

import (
	"io"

	"github.com/docker/docker/pkg/archive"
)

type mountedLayer struct {
	name       string
	mountID    string
	initID     string
	parent     *roLayer
	platform   Platform
	path       string
	layerStore *layerStore

	references map[RWLayer]*referencedRWLayer
}

func (ml *mountedLayer) cacheParent() string {
	if ml.initID != "" {
		return ml.initID
	}
	if ml.parent != nil {
		return ml.parent.cacheID
	}
	return ""
}

func (ml *mountedLayer) TarStream() (io.ReadCloser, error) {
	return ml.layerStore.drivers[string(ml.platform)].driver.Diff(ml.mountID, ml.cacheParent())
}

func (ml *mountedLayer) Name() string {
	return ml.name
}

func (ml *mountedLayer) Parent() Layer {
	if ml.parent != nil {
		return ml.parent
	}

	// Return a nil interface instead of an interface wrapping a nil
	// pointer.
	return nil
}

func (ml *mountedLayer) Platform() Platform {
	return ml.platform
}

func (ml *mountedLayer) Size() (int64, error) {
	return ml.layerStore.drivers[string(ml.platform)].driver.DiffSize(ml.mountID, ml.cacheParent())
}

func (ml *mountedLayer) Changes() ([]archive.Change, error) {
	return ml.layerStore.drivers[string(ml.platform)].driver.Changes(ml.mountID, ml.cacheParent())
}

func (ml *mountedLayer) Metadata() (map[string]string, error) {
	return ml.layerStore.drivers[string(ml.platform)].driver.GetMetadata(ml.mountID)
}

func (ml *mountedLayer) getReference() RWLayer {
	ref := &referencedRWLayer{
		mountedLayer: ml,
	}
	ml.references[ref] = ref

	return ref
}

func (ml *mountedLayer) hasReferences() bool {
	return len(ml.references) > 0
}

func (ml *mountedLayer) deleteReference(ref RWLayer) error {
	if _, ok := ml.references[ref]; !ok {
		return ErrLayerNotRetained
	}
	delete(ml.references, ref)
	return nil
}

func (ml *mountedLayer) retakeReference(r RWLayer) {
	if ref, ok := r.(*referencedRWLayer); ok {
		ml.references[ref] = ref
	}
}

type referencedRWLayer struct {
	*mountedLayer
}

func (rl *referencedRWLayer) Mount(mountLabel string) (string, error) {
	return rl.layerStore.drivers[string(rl.platform)].driver.Get(rl.mountedLayer.mountID, mountLabel)
}

// Unmount decrements the activity count and unmounts the underlying layer
// Callers should only call `Unmount` once per call to `Mount`, even on error.
func (rl *referencedRWLayer) Unmount() error {
	return rl.layerStore.drivers[string(rl.platform)].driver.Put(rl.mountedLayer.mountID)
}
