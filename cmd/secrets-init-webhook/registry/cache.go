package registry

import (
	"sync"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

// ImageCache interface
type ImageCache interface {
	Get(image string) *v1.Config
	Put(image string, imageConfig *v1.Config)
}

// InMemoryImageCache Concrete mutex-guarded cache
type InMemoryImageCache struct {
	mutex sync.Mutex
	cache map[string]v1.Config
}

// NewInMemoryImageCache return new mutex guarded cache
func NewInMemoryImageCache() ImageCache {
	return &InMemoryImageCache{cache: map[string]v1.Config{}}
}

// Get image from cache
func (c *InMemoryImageCache) Get(image string) *v1.Config {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if imageConfig, ok := c.cache[image]; ok {
		return &imageConfig
	}
	return nil
}

// Put image into cache
func (c *InMemoryImageCache) Put(image string, imageConfig *v1.Config) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.cache[image] = *imageConfig
}
