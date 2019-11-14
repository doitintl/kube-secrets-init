package registry

import (
	"sync"

	imagev1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// ImageCache interface
type ImageCache interface {
	Get(image string) *imagev1.ImageConfig
	Put(image string, imageConfig *imagev1.ImageConfig)
}

// InMemoryImageCache Concrete mutex-guarded cache
type InMemoryImageCache struct {
	mutex sync.Mutex
	cache map[string]imagev1.ImageConfig
}

// NewInMemoryImageCache return new mutex guarded cache
func NewInMemoryImageCache() ImageCache {
	return &InMemoryImageCache{cache: map[string]imagev1.ImageConfig{}}
}

// Get image from cache
func (c *InMemoryImageCache) Get(image string) *imagev1.ImageConfig {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if imageConfig, ok := c.cache[image]; ok {
		return &imageConfig
	}
	return nil
}

// Put image into cache
func (c *InMemoryImageCache) Put(image string, imageConfig *imagev1.ImageConfig) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.cache[image] = *imageConfig
}
