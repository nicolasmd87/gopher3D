package renderer

import "github.com/go-gl/gl/v4.1-core/gl"

// UniformCache caches uniform locations to avoid repeated gl.GetUniformLocation calls
type UniformCache struct {
	locations map[string]int32
	program   uint32
}

// NewUniformCache creates a new uniform cache for a shader program
func NewUniformCache(program uint32) *UniformCache {
	return &UniformCache{
		locations: make(map[string]int32),
		program:   program,
	}
}

// GetLocation returns the cached uniform location or fetches and caches it
func (uc *UniformCache) GetLocation(name string) int32 {
	if loc, exists := uc.locations[name]; exists {
		return loc
	}

	// Fetch and cache the location
	loc := gl.GetUniformLocation(uc.program, gl.Str(name+"\x00"))
	uc.locations[name] = loc
	return loc
}

// SetFloat sets a float uniform using cached location
func (uc *UniformCache) SetFloat(name string, value float32) {
	loc := uc.GetLocation(name)
	if loc != -1 {
		gl.Uniform1f(loc, value)
	}
}

// SetVec3 sets a vec3 uniform using cached location
func (uc *UniformCache) SetVec3(name string, x, y, z float32) {
	loc := uc.GetLocation(name)
	if loc != -1 {
		gl.Uniform3f(loc, x, y, z)
	}
}

// SetInt sets an int uniform using cached location
func (uc *UniformCache) SetInt(name string, value int32) {
	loc := uc.GetLocation(name)
	if loc != -1 {
		gl.Uniform1i(loc, value)
	}
}

// Clear clears the cache (call when shader program changes)
func (uc *UniformCache) Clear() {
	uc.locations = make(map[string]int32)
}
