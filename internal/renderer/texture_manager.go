package renderer

import (
	"Gopher3D/internal/logger"
	"fmt"
	"image"
	"image/draw"
	"os"
	"sync"

	"github.com/go-gl/gl/v4.1-core/gl"
	"go.uber.org/zap"
)

// TextureStats provides debugging and profiling information
type TextureStats struct {
	TotalTextures   int
	CacheHits       int
	CacheMisses     int
	TotalMemoryMB   float64
	ActiveTextures  int
}

// TextureManager manages texture loading, caching, and lifecycle
type TextureManager struct {
	textureCache    map[string]uint32  // path -> OpenGL texture ID
	textureRefCount map[uint32]int     // texture ID -> reference count
	texturePaths    map[uint32]string  // texture ID -> path (for debugging)
	mu              sync.RWMutex       // Thread-safe operations
	stats           TextureStats
}

// NewTextureManager creates a new texture manager instance
func NewTextureManager() *TextureManager {
	return &TextureManager{
		textureCache:    make(map[string]uint32),
		textureRefCount: make(map[uint32]int),
		texturePaths:    make(map[uint32]string),
	}
}

// LoadTexture loads a texture from file or returns cached texture ID
// Automatically increments reference count
func (tm *TextureManager) LoadTexture(filePath string) (uint32, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Check if texture is already cached
	if textureID, exists := tm.textureCache[filePath]; exists {
		// Increment reference count
		tm.textureRefCount[textureID]++
		tm.stats.CacheHits++
		
		logger.Log.Debug("Texture cache hit",
			zap.String("path", filePath),
			zap.Uint32("textureID", textureID),
			zap.Int("refCount", tm.textureRefCount[textureID]))
		
		return textureID, nil
	}

	// Cache miss - load texture from disk
	tm.stats.CacheMisses++
	
	imgFile, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer imgFile.Close()

	img, _, err := image.Decode(imgFile)
	if err != nil {
		return 0, err
	}

	rgba := image.NewRGBA(img.Bounds())
	if rgba.Stride != rgba.Rect.Size().X*4 {
		return 0, fmt.Errorf("unsupported stride")
	}
	draw.Draw(rgba, rgba.Bounds(), img, image.Point{0, 0}, draw.Src)

	var textureID uint32
	gl.GenTextures(1, &textureID)
	gl.BindTexture(gl.TEXTURE_2D, textureID)
	gl.TexImage2D(
		gl.TEXTURE_2D, 0, gl.RGBA,
		int32(rgba.Rect.Size().X), int32(rgba.Rect.Size().Y),
		0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(rgba.Pix))

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)

	// Cache the texture
	tm.textureCache[filePath] = textureID
	tm.textureRefCount[textureID] = 1
	tm.texturePaths[textureID] = filePath
	tm.stats.TotalTextures++
	tm.stats.ActiveTextures++

	logger.Log.Info("Texture loaded and cached",
		zap.String("path", filePath),
		zap.Uint32("textureID", textureID),
		zap.Int("width", rgba.Rect.Size().X),
		zap.Int("height", rgba.Rect.Size().Y))

	return textureID, nil
}

// CreateTextureFromImage creates a texture from an image.Image
// Used for embedded textures like default texture
func (tm *TextureManager) CreateTextureFromImage(img image.Image, name string) (uint32, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Check if already cached by name
	if textureID, exists := tm.textureCache[name]; exists {
		tm.textureRefCount[textureID]++
		tm.stats.CacheHits++
		return textureID, nil
	}

	rgba, ok := img.(*image.RGBA)
	if !ok {
		// Convert to *image.RGBA if necessary
		b := img.Bounds()
		rgba = image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
		draw.Draw(rgba, rgba.Bounds(), img, b.Min, draw.Src)
	}

	var textureID uint32
	gl.GenTextures(1, &textureID)
	gl.BindTexture(gl.TEXTURE_2D, textureID)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(rgba.Rect.Size().X), int32(rgba.Rect.Size().Y), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(rgba.Pix))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)

	// Cache with name
	tm.textureCache[name] = textureID
	tm.textureRefCount[textureID] = 1
	tm.texturePaths[textureID] = name
	tm.stats.TotalTextures++
	tm.stats.ActiveTextures++

	logger.Log.Info("Texture created from image",
		zap.String("name", name),
		zap.Uint32("textureID", textureID))

	return textureID, nil
}

// AddReference increments the reference count for a texture
func (tm *TextureManager) AddReference(textureID uint32) {
	if textureID == 0 {
		return
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.textureRefCount[textureID]++
	
	logger.Log.Debug("Texture reference added",
		zap.Uint32("textureID", textureID),
		zap.Int("refCount", tm.textureRefCount[textureID]))
}

// ReleaseTexture decrements reference count and frees texture if count reaches 0
func (tm *TextureManager) ReleaseTexture(textureID uint32) {
	if textureID == 0 {
		return
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	refCount, exists := tm.textureRefCount[textureID]
	if !exists {
		logger.Log.Warn("Attempted to release unknown texture",
			zap.Uint32("textureID", textureID))
		return
	}

	refCount--
	tm.textureRefCount[textureID] = refCount

	logger.Log.Debug("Texture reference released",
		zap.Uint32("textureID", textureID),
		zap.Int("refCount", refCount))

	if refCount <= 0 {
		// Free the OpenGL texture
		gl.DeleteTextures(1, &textureID)
		
		// Remove from all caches
		path := tm.texturePaths[textureID]
		delete(tm.textureCache, path)
		delete(tm.textureRefCount, textureID)
		delete(tm.texturePaths, textureID)
		tm.stats.ActiveTextures--

		logger.Log.Info("Texture freed",
			zap.Uint32("textureID", textureID),
			zap.String("path", path))
	}
}

// GetStats returns current texture manager statistics
func (tm *TextureManager) GetStats() TextureStats {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	stats := tm.stats
	stats.ActiveTextures = len(tm.textureRefCount)
	return stats
}

// LogStats logs current texture statistics
func (tm *TextureManager) LogStats() {
	stats := tm.GetStats()
	logger.Log.Info("Texture Manager Stats",
		zap.Int("totalTextures", stats.TotalTextures),
		zap.Int("activeTextures", stats.ActiveTextures),
		zap.Int("cacheHits", stats.CacheHits),
		zap.Int("cacheMisses", stats.CacheMisses),
		zap.Float64("hitRate", float64(stats.CacheHits)/float64(stats.CacheHits+stats.CacheMisses)))
}

// Clear releases all textures (for cleanup/testing)
func (tm *TextureManager) Clear() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for textureID := range tm.textureRefCount {
		gl.DeleteTextures(1, &textureID)
	}

	tm.textureCache = make(map[string]uint32)
	tm.textureRefCount = make(map[uint32]int)
	tm.texturePaths = make(map[uint32]string)
	tm.stats.ActiveTextures = 0

	logger.Log.Info("Texture manager cleared")
}

