package renderer

import (
	"testing"
)

func TestNewUniformCache(t *testing.T) {
	cache := NewUniformCache(0)

	if cache == nil {
		t.Fatal("NewUniformCache returned nil")
	}

	if cache.locations == nil {
		t.Error("locations map should be initialized")
	}
}

func TestUniformCacheClear(t *testing.T) {
	cache := NewUniformCache(0)
	cache.locations["test"] = 5

	cache.Clear()

	if len(cache.locations) != 0 {
		t.Error("Clear should empty the cache")
	}
}

func TestUniformCacheLocationsMap(t *testing.T) {
	cache := NewUniformCache(0)

	cache.locations["exists"] = 1

	if _, ok := cache.locations["exists"]; !ok {
		t.Error("Should be able to store in locations map")
	}

	if _, ok := cache.locations["nonexistent"]; ok {
		t.Error("Non-existent key should not be in map")
	}
}
