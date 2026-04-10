package imageutil

import (
	"fmt"
	"sync"
	"unsafe"
)

// Sprite registry caches loaded sprites so that the same asset data is only
// decoded once. Cache keys are derived from the backing-array pointer of the
// embedded []byte slice plus the load parameters, so identical calls return
// the previously loaded result.

var (
	registry = make(map[string]any)
	mu       sync.Mutex
)

func cacheKey(data []byte, extra string) string {
	if len(data) == 0 {
		return extra
	}
	ptr := unsafe.Pointer(unsafe.SliceData(data))
	return fmt.Sprintf("%p:%s", ptr, extra)
}

func registryGet(key string) (any, bool) {
	mu.Lock()
	defer mu.Unlock()
	v, ok := registry[key]
	return v, ok
}

func registrySet(key string, value any) {
	mu.Lock()
	defer mu.Unlock()
	registry[key] = value
}

// ResetRegistry clears all cached sprites. Useful for testing.
func ResetRegistry() {
	mu.Lock()
	defer mu.Unlock()
	registry = make(map[string]any)
}

// RegistryLen returns the number of cached entries.
func RegistryLen() int {
	mu.Lock()
	defer mu.Unlock()
	return len(registry)
}
