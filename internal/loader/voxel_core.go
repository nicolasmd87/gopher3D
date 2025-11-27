package loader

import (
	"Gopher3D/internal/renderer"
	"math"
	"runtime"
	"sync"

	"github.com/alitto/pond/v2"
	"github.com/go-gl/mathgl/mgl32"
)

type VoxelID uint16

type VoxelData struct {
	// HOT DATA - Accessed frequently during voxel operations
	Active   bool       // Whether voxel is active (checked most often)
	ID       VoxelID    // Voxel type identifier (uint16)
	Position mgl32.Vec3 // Position in world space
}

type VoxelChunk struct {
	Position    mgl32.Vec3
	Size        int
	Voxels      [][][]VoxelData
	Model       *renderer.Model
	NeedsUpdate bool
}

type VoxelRenderMode int

const (
	InstancedMode VoxelRenderMode = iota
	SurfaceNetsMode
)

type VoxelWorld struct {
	ChunkSize    int
	WorldSizeX   int
	WorldSizeZ   int
	MaxHeight    int
	VoxelSize    float32
	Geometry     *VoxelGeometry
	RenderMode   VoxelRenderMode
	Chunks       [][]*VoxelChunk
	ActiveVoxels int
	SDFData      [][][]float32
}

type VoxelGeometry struct {
	InterleavedData []float32
	Indices         []int32
	Name            string
}

func NewVoxelWorld(chunkSize, worldSizeX, worldSizeZ, maxHeight int, voxelSize float32, geometry *VoxelGeometry, renderMode VoxelRenderMode) *VoxelWorld {
	world := &VoxelWorld{
		ChunkSize:  chunkSize,
		WorldSizeX: worldSizeX,
		WorldSizeZ: worldSizeZ,
		MaxHeight:  maxHeight,
		VoxelSize:  voxelSize,
		Geometry:   geometry,
		RenderMode: renderMode,
		Chunks:     make([][]*VoxelChunk, worldSizeX),
	}

	if renderMode == SurfaceNetsMode {
		totalX := worldSizeX * chunkSize
		totalZ := worldSizeZ * chunkSize
		world.SDFData = make([][][]float32, totalX)
		for x := 0; x < totalX; x++ {
			world.SDFData[x] = make([][]float32, maxHeight)
			for y := 0; y < maxHeight; y++ {
				world.SDFData[x][y] = make([]float32, totalZ)
				for z := 0; z < totalZ; z++ {
					world.SDFData[x][y][z] = 1.0
				}
			}
		}
	}

	for x := 0; x < worldSizeX; x++ {
		world.Chunks[x] = make([]*VoxelChunk, worldSizeZ)
		for z := 0; z < worldSizeZ; z++ {
			chunk := &VoxelChunk{
				Position: mgl32.Vec3{
					float32(x * chunkSize),
					0,
					float32(z * chunkSize),
				},
				Size:        chunkSize,
				Voxels:      make([][][]VoxelData, chunkSize),
				NeedsUpdate: true,
			}

			for i := 0; i < chunkSize; i++ {
				chunk.Voxels[i] = make([][]VoxelData, maxHeight)
				for j := 0; j < maxHeight; j++ {
					chunk.Voxels[i][j] = make([]VoxelData, chunkSize)
				}
			}

			world.Chunks[x][z] = chunk
		}
	}

	return world
}

func (world *VoxelWorld) SetVoxel(x, y, z int, voxelID VoxelID) {
	chunkX := x / world.ChunkSize
	chunkZ := z / world.ChunkSize
	localX := x % world.ChunkSize
	localZ := z % world.ChunkSize

	if chunkX < 0 || chunkX >= world.WorldSizeX || chunkZ < 0 || chunkZ >= world.WorldSizeZ {
		return
	}
	if y < 0 || y >= world.MaxHeight {
		return
	}

	chunk := world.Chunks[chunkX][chunkZ]
	voxel := &chunk.Voxels[localX][y][localZ]

	wasActive := voxel.Active
	voxel.ID = voxelID
	voxel.Active = voxelID != 0
	voxel.Position = mgl32.Vec3{
		float32(x) * world.VoxelSize,
		float32(y) * world.VoxelSize,
		float32(z) * world.VoxelSize,
	}

	if wasActive && !voxel.Active {
		world.ActiveVoxels--
	} else if !wasActive && voxel.Active {
		world.ActiveVoxels++
	}

	chunk.NeedsUpdate = true
}

func (world *VoxelWorld) GenerateVoxelsParallel(generatorFunc func(x, y, z int) (VoxelID, bool)) {
	if world.RenderMode != InstancedMode {
		return
	}

	totalX := world.WorldSizeX * world.ChunkSize
	totalZ := world.WorldSizeZ * world.ChunkSize

	numWorkers := runtime.NumCPU()
	pool := pond.NewPool(numWorkers)
	defer pool.StopAndWait()

	var wg sync.WaitGroup

	chunkSize := 32
	xChunks := (totalX + chunkSize - 1) / chunkSize
	zChunks := (totalZ + chunkSize - 1) / chunkSize

	for chunkX := 0; chunkX < xChunks; chunkX++ {
		for chunkZ := 0; chunkZ < zChunks; chunkZ++ {
			wg.Add(1)

			startX := chunkX * chunkSize
			endX := startX + chunkSize
			if endX > totalX {
				endX = totalX
			}

			startZ := chunkZ * chunkSize
			endZ := startZ + chunkSize
			if endZ > totalZ {
				endZ = totalZ
			}

			pool.Submit(func() {
				defer wg.Done()

				for x := startX; x < endX; x++ {
					for y := 0; y < world.MaxHeight; y++ {
						for z := startZ; z < endZ; z++ {
							if voxelID, shouldPlace := generatorFunc(x, y, z); shouldPlace {
								world.SetVoxel(x, y, z, voxelID)
							}
						}
					}
				}
			})
		}
	}

	wg.Wait()
}

func (world *VoxelWorld) GetVoxel(x, y, z int) VoxelID {
	chunkX := x / world.ChunkSize
	chunkZ := z / world.ChunkSize
	localX := x % world.ChunkSize
	localZ := z % world.ChunkSize

	if chunkX < 0 || chunkX >= world.WorldSizeX || chunkZ < 0 || chunkZ >= world.WorldSizeZ {
		return 0
	}
	if y < 0 || y >= world.MaxHeight {
		return 0
	}

	return world.Chunks[chunkX][chunkZ].Voxels[localX][y][localZ].ID
}

func (world *VoxelWorld) ClearChunk(chunkX, chunkZ int) {
	if chunkX < 0 || chunkX >= world.WorldSizeX || chunkZ < 0 || chunkZ >= world.WorldSizeZ {
		return
	}

	chunk := world.Chunks[chunkX][chunkZ]
	for x := 0; x < world.ChunkSize; x++ {
		for y := 0; y < world.MaxHeight; y++ {
			for z := 0; z < world.ChunkSize; z++ {
				if chunk.Voxels[x][y][z].Active {
					world.ActiveVoxels--
				}
				chunk.Voxels[x][y][z] = VoxelData{
					ID:     0,
					Active: false,
				}
			}
		}
	}
	chunk.NeedsUpdate = true
}

func (world *VoxelWorld) SetVoxelSDF(x, y, z int, sdf float32) {
	if world.RenderMode != SurfaceNetsMode {
		return
	}
	if x < 0 || x >= world.WorldSizeX*world.ChunkSize {
		return
	}
	if y < 0 || y >= world.MaxHeight {
		return
	}
	if z < 0 || z >= world.WorldSizeZ*world.ChunkSize {
		return
	}

	world.SDFData[x][y][z] = sdf
}

// isVoxelSolid checks if a voxel at global coordinates is solid (for face culling)
func (world *VoxelWorld) isVoxelSolid(x, y, z int) bool {
	if x < 0 || x >= world.WorldSizeX*world.ChunkSize {
		return false
	}
	if y < 0 || y >= world.MaxHeight {
		return false
	}
	if z < 0 || z >= world.WorldSizeZ*world.ChunkSize {
		return false
	}
	
	chunkX := x / world.ChunkSize
	chunkZ := z / world.ChunkSize
	localX := x % world.ChunkSize
	localZ := z % world.ChunkSize
	
	return world.Chunks[chunkX][chunkZ].Voxels[localX][y][localZ].Active
}

func (world *VoxelWorld) GetVoxelSDF(x, y, z int) float32 {
	if world.RenderMode != SurfaceNetsMode {
		return 1.0
	}
	if x < 0 || x >= world.WorldSizeX*world.ChunkSize {
		return 1.0
	}
	if y < 0 || y >= world.MaxHeight {
		return 1.0
	}
	if z < 0 || z >= world.WorldSizeZ*world.ChunkSize {
		return 1.0
	}

	return world.SDFData[x][y][z]
}

func (world *VoxelWorld) GenerateSDFParallel(noiseFunc func(x, y, z int) float32) {
	if world.RenderMode != SurfaceNetsMode {
		return
	}

	totalX := world.WorldSizeX * world.ChunkSize
	totalZ := world.WorldSizeZ * world.ChunkSize

	numWorkers := runtime.NumCPU()
	pool := pond.NewPool(numWorkers)
	defer pool.StopAndWait()

	var wg sync.WaitGroup
	mutex := &sync.Mutex{}

	chunkSize := 32
	xChunks := (totalX + chunkSize - 1) / chunkSize
	zChunks := (totalZ + chunkSize - 1) / chunkSize

	for chunkX := 0; chunkX < xChunks; chunkX++ {
		for chunkZ := 0; chunkZ < zChunks; chunkZ++ {
			wg.Add(1)

			startX := chunkX * chunkSize
			endX := startX + chunkSize
			if endX > totalX {
				endX = totalX
			}

			startZ := chunkZ * chunkSize
			endZ := startZ + chunkSize
			if endZ > totalZ {
				endZ = totalZ
			}

			pool.Submit(func() {
				defer wg.Done()

				localData := make([][][]float32, endX-startX)
				for i := range localData {
					localData[i] = make([][]float32, world.MaxHeight)
					for j := range localData[i] {
						localData[i][j] = make([]float32, endZ-startZ)
					}
				}

				for x := startX; x < endX; x++ {
					for y := 0; y < world.MaxHeight; y++ {
						for z := startZ; z < endZ; z++ {
							sdf := noiseFunc(x, y, z)
							localData[x-startX][y][z-startZ] = sdf
						}
					}
				}

				mutex.Lock()
				for x := startX; x < endX; x++ {
					for y := 0; y < world.MaxHeight; y++ {
						for z := startZ; z < endZ; z++ {
							world.SDFData[x][y][z] = localData[x-startX][y][z-startZ]
						}
					}
				}
				mutex.Unlock()
			})
		}
	}

	wg.Wait()
}

func (world *VoxelWorld) CreateInstancedModel() (*renderer.Model, error) {
	if world.RenderMode == InstancedMode {
		// First pass: count visible voxels (those with at least one exposed face)
		visibleCount := 0
		for chunkX := 0; chunkX < world.WorldSizeX; chunkX++ {
			for chunkZ := 0; chunkZ < world.WorldSizeZ; chunkZ++ {
				chunk := world.Chunks[chunkX][chunkZ]
				for x := 0; x < world.ChunkSize; x++ {
					for y := 0; y < world.MaxHeight; y++ {
						for z := 0; z < world.ChunkSize; z++ {
							voxel := &chunk.Voxels[x][y][z]
							if voxel.Active {
								// Check if voxel has at least one exposed face
								globalX := chunkX*world.ChunkSize + x
								globalY := y
								globalZ := chunkZ*world.ChunkSize + z
								
								hasExposedFace := false
								// Check 6 neighbors
								if !world.isVoxelSolid(globalX-1, globalY, globalZ) ||
								   !world.isVoxelSolid(globalX+1, globalY, globalZ) ||
								   !world.isVoxelSolid(globalX, globalY-1, globalZ) ||
								   !world.isVoxelSolid(globalX, globalY+1, globalZ) ||
								   !world.isVoxelSolid(globalX, globalY, globalZ-1) ||
								   !world.isVoxelSolid(globalX, globalY, globalZ+1) {
									hasExposedFace = true
								}
								
								if hasExposedFace {
									visibleCount++
								}
							}
						}
					}
				}
			}
		}
		
		// Create model with only visible voxels
		model, err := CreateInstancedVoxelModel(world.Geometry, visibleCount)
		if err != nil {
			return nil, err
		}

		// Second pass: populate instance matrices for visible voxels only
		// Process in Y-layers for better cache coherence (Y-major order)
		instanceIndex := 0
		for y := 0; y < world.MaxHeight; y++ {
			for chunkX := 0; chunkX < world.WorldSizeX; chunkX++ {
				for chunkZ := 0; chunkZ < world.WorldSizeZ; chunkZ++ {
					chunk := world.Chunks[chunkX][chunkZ]
					for x := 0; x < world.ChunkSize; x++ {
						for z := 0; z < world.ChunkSize; z++ {
							voxel := &chunk.Voxels[x][y][z]
							if voxel.Active {
								globalX := chunkX*world.ChunkSize + x
								globalY := y
								globalZ := chunkZ*world.ChunkSize + z
								
								hasExposedFace := false
								if !world.isVoxelSolid(globalX-1, globalY, globalZ) ||
								   !world.isVoxelSolid(globalX+1, globalY, globalZ) ||
								   !world.isVoxelSolid(globalX, globalY-1, globalZ) ||
								   !world.isVoxelSolid(globalX, globalY+1, globalZ) ||
								   !world.isVoxelSolid(globalX, globalY, globalZ-1) ||
								   !world.isVoxelSolid(globalX, globalY, globalZ+1) {
									hasExposedFace = true
								}
								
								if hasExposedFace && instanceIndex < len(model.InstanceModelMatrices) {
									model.InstanceModelMatrices[instanceIndex] = mgl32.Translate3D(
										voxel.Position.X(),
										voxel.Position.Y(),
										voxel.Position.Z(),
									)
									
									// Assign color based on VoxelID
									model.InstanceColors[instanceIndex] = GetVoxelColor(voxel.ID)
									
									instanceIndex++
								}
							}
						}
					}
				}
			}
		}

		model.InstanceMatricesUpdated = true
		return model, nil
	} else {
		return world.CreateSurfaceNetsModel()
	}
}

func (world *VoxelWorld) CreateSurfaceNetsModel() (*renderer.Model, error) {
	var vertices []float32
	var indices []int32

	totalX := world.WorldSizeX * world.ChunkSize
	totalZ := world.WorldSizeZ * world.ChunkSize

	for x := 0; x < totalX; x++ {
		for z := 0; z < totalZ; z++ {
			height := float32(10)
			for y := world.MaxHeight - 1; y >= 0; y-- {
				if world.GetVoxelSDF(x, y, z) < 0 {
					height = float32(y)
					break
				}
			}

			u := float32(x) / float32(totalX-1)
			v := float32(z) / float32(totalZ-1)

			vertices = append(vertices,
				float32(x)*world.VoxelSize, height*world.VoxelSize, float32(z)*world.VoxelSize,
				u, v,
				0.0, 1.0, 0.0,
			)
		}
	}

	for x := 0; x < totalX-1; x++ {
		for z := 0; z < totalZ-1; z++ {
			i0 := int32(x*totalZ + z)
			i1 := int32((x+1)*totalZ + z)
			i2 := int32(x*totalZ + (z + 1))
			i3 := int32((x+1)*totalZ + (z + 1))

			indices = append(indices, i0, i2, i1)
			indices = append(indices, i1, i2, i3)
		}
	}

	if len(vertices) == 0 {
		return nil, nil
	}

	vertexCount := len(vertices) / 8
	positions := make([]float32, vertexCount*3)

	for i := 0; i < vertexCount; i++ {
		positions[i*3] = vertices[i*8]
		positions[i*3+1] = vertices[i*8+1]
		positions[i*3+2] = vertices[i*8+2]
	}

	model := &renderer.Model{
		InterleavedData: vertices,
		Vertices:        positions,
		Faces:           indices,
		Position:        [3]float32{0, 0, 0},
		Rotation:        mgl32.Quat{W: 1, V: mgl32.Vec3{0, 0, 0}},
		Scale:           [3]float32{1, 1, 1},
		IsInstanced:     false,
		Material:        renderer.DefaultMaterial,
	}

	uniqueMaterial := *model.Material
	model.Material = &uniqueMaterial
	model.CalculateBoundingSphere()

	return model, nil
}

func (world *VoxelWorld) sampleHeightAt(x, z float32) float32 {
	if world.RenderMode != SurfaceNetsMode {
		return 0
	}

	ix := int(x)
	iz := int(z)

	totalX := world.WorldSizeX * world.ChunkSize
	totalZ := world.WorldSizeZ * world.ChunkSize

	if ix < 0 || ix >= totalX || iz < 0 || iz >= totalZ {
		return 0
	}

	for y := world.MaxHeight - 1; y >= 0; y-- {
		if world.GetVoxelSDF(ix, y, iz) < 0 {
			return float32(y)
		}
	}

	return 0
}

func (world *VoxelWorld) hasSurfaceVertex(x, y, z int) bool {
	corners := [8]float32{
		world.GetVoxelSDF(x, y, z),
		world.GetVoxelSDF(x+1, y, z),
		world.GetVoxelSDF(x, y+1, z),
		world.GetVoxelSDF(x+1, y+1, z),
		world.GetVoxelSDF(x, y, z+1),
		world.GetVoxelSDF(x+1, y, z+1),
		world.GetVoxelSDF(x, y+1, z+1),
		world.GetVoxelSDF(x+1, y+1, z+1),
	}

	positive := 0
	negative := 0
	for _, sdf := range corners {
		if sdf < 0 {
			negative++
		} else {
			positive++
		}
	}

	return positive > 0 && negative > 0
}

func (world *VoxelWorld) calculateSurfaceVertex(x, y, z int) *mgl32.Vec3 {
	corners := [8]float32{
		world.GetVoxelSDF(x, y, z),
		world.GetVoxelSDF(x+1, y, z),
		world.GetVoxelSDF(x, y+1, z),
		world.GetVoxelSDF(x+1, y+1, z),
		world.GetVoxelSDF(x, y, z+1),
		world.GetVoxelSDF(x+1, y, z+1),
		world.GetVoxelSDF(x, y+1, z+1),
		world.GetVoxelSDF(x+1, y+1, z+1),
	}

	signChanges := 0
	for i := 0; i < 8; i++ {
		if corners[i] < 0 {
			signChanges++
		}
	}

	if signChanges == 0 || signChanges == 8 {
		return nil
	}

	edgePoints := []mgl32.Vec3{}
	edges := [][2][3]int{
		{{0, 0, 0}, {1, 0, 0}}, {{1, 0, 0}, {1, 1, 0}}, {{1, 1, 0}, {0, 1, 0}}, {{0, 1, 0}, {0, 0, 0}},
		{{0, 0, 1}, {1, 0, 1}}, {{1, 0, 1}, {1, 1, 1}}, {{1, 1, 1}, {0, 1, 1}}, {{0, 1, 1}, {0, 0, 1}},
		{{0, 0, 0}, {0, 0, 1}}, {{1, 0, 0}, {1, 0, 1}}, {{1, 1, 0}, {1, 1, 1}}, {{0, 1, 0}, {0, 1, 1}},
	}

	for _, edge := range edges {
		p1 := edge[0]
		p2 := edge[1]

		sdf1 := corners[p1[2]*4+p1[1]*2+p1[0]]
		sdf2 := corners[p2[2]*4+p2[1]*2+p2[0]]

		if (sdf1 < 0) != (sdf2 < 0) && sdf1 != sdf2 {
			t := sdf1 / (sdf1 - sdf2)
			if t < 0 {
				t = 0
			}
			if t > 1 {
				t = 1
			}

			point := mgl32.Vec3{
				float32(x+p1[0]) + t*float32(p2[0]-p1[0]),
				float32(y+p1[1]) + t*float32(p2[1]-p1[1]),
				float32(z+p1[2]) + t*float32(p2[2]-p1[2]),
			}
			edgePoints = append(edgePoints, point)
		}
	}

	if len(edgePoints) == 0 {
		return nil
	}

	avg := mgl32.Vec3{0, 0, 0}
	for _, point := range edgePoints {
		avg = avg.Add(point)
	}
	avg = avg.Mul(1.0 / float32(len(edgePoints)))

	worldPos := mgl32.Vec3{
		avg.X() * world.VoxelSize,
		avg.Y() * world.VoxelSize,
		avg.Z() * world.VoxelSize,
	}

	return &worldPos
}

func (world *VoxelWorld) calculateSurfaceNormal(x, y, z int) (float32, float32, float32) {
	dx := world.GetVoxelSDF(x+1, y, z) - world.GetVoxelSDF(x-1, y, z)
	dy := world.GetVoxelSDF(x, y+1, z) - world.GetVoxelSDF(x, y-1, z)
	dz := world.GetVoxelSDF(x, y, z+1) - world.GetVoxelSDF(x, y, z-1)

	length := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
	if length > 0 {
		return dx / length, dy / length, dz / length
	}
	return 0, 1, 0
}

func (world *VoxelWorld) generateSurfaceQuads(x, y, z int, vertexMap map[[3]int]int, indices *[]int32) {
	if idx, exists := vertexMap[[3]int{x, y, z}]; exists {
		neighbors := [][3]int{
			{x + 1, y, z},
			{x, y + 1, z},
			{x, y, z + 1},
		}

		for _, neighbor := range neighbors {
			if nIdx, nExists := vertexMap[neighbor]; nExists {
				diagonal := [3]int{neighbor[0], neighbor[1], neighbor[2]}
				if neighbor[0] == x+1 && neighbor[1] == y {
					diagonal = [3]int{x + 1, y + 1, z}
				} else if neighbor[1] == y+1 && neighbor[2] == z {
					diagonal = [3]int{x, y + 1, z + 1}
				} else if neighbor[2] == z+1 && neighbor[0] == x {
					diagonal = [3]int{x + 1, y, z + 1}
				}

				if dIdx, dExists := vertexMap[diagonal]; dExists {
					*indices = append(*indices, int32(idx), int32(nIdx), int32(dIdx))
				}
			}
		}
	}
}

func CreateCubeGeometry(size float32) *VoxelGeometry {
	halfSize := size * 0.5

	interleavedData := []float32{
		-halfSize, -halfSize, halfSize, 0.0, 0.0, 0.0, 0.0, 1.0,
		halfSize, -halfSize, halfSize, 1.0, 0.0, 0.0, 0.0, 1.0,
		halfSize, halfSize, halfSize, 1.0, 1.0, 0.0, 0.0, 1.0,
		-halfSize, halfSize, halfSize, 0.0, 1.0, 0.0, 0.0, 1.0,

		-halfSize, -halfSize, -halfSize, 1.0, 0.0, 0.0, 0.0, -1.0,
		-halfSize, halfSize, -halfSize, 1.0, 1.0, 0.0, 0.0, -1.0,
		halfSize, halfSize, -halfSize, 0.0, 1.0, 0.0, 0.0, -1.0,
		halfSize, -halfSize, -halfSize, 0.0, 0.0, 0.0, 0.0, -1.0,

		-halfSize, -halfSize, -halfSize, 0.0, 0.0, -1.0, 0.0, 0.0,
		-halfSize, -halfSize, halfSize, 1.0, 0.0, -1.0, 0.0, 0.0,
		-halfSize, halfSize, halfSize, 1.0, 1.0, -1.0, 0.0, 0.0,
		-halfSize, halfSize, -halfSize, 0.0, 1.0, -1.0, 0.0, 0.0,

		halfSize, -halfSize, -halfSize, 1.0, 0.0, 1.0, 0.0, 0.0,
		halfSize, halfSize, -halfSize, 1.0, 1.0, 1.0, 0.0, 0.0,
		halfSize, halfSize, halfSize, 0.0, 1.0, 1.0, 0.0, 0.0,
		halfSize, -halfSize, halfSize, 0.0, 0.0, 1.0, 0.0, 0.0,

		-halfSize, halfSize, -halfSize, 0.0, 1.0, 0.0, 1.0, 0.0,
		-halfSize, halfSize, halfSize, 0.0, 0.0, 0.0, 1.0, 0.0,
		halfSize, halfSize, halfSize, 1.0, 0.0, 0.0, 1.0, 0.0,
		halfSize, halfSize, -halfSize, 1.0, 1.0, 0.0, 1.0, 0.0,

		-halfSize, -halfSize, -halfSize, 1.0, 1.0, 0.0, -1.0, 0.0,
		halfSize, -halfSize, -halfSize, 0.0, 1.0, 0.0, -1.0, 0.0,
		halfSize, -halfSize, halfSize, 0.0, 0.0, 0.0, -1.0, 0.0,
		-halfSize, -halfSize, halfSize, 1.0, 0.0, 0.0, -1.0, 0.0,
	}

	indices := []int32{
		0, 1, 2, 2, 3, 0,
		4, 5, 6, 6, 7, 4,
		8, 9, 10, 10, 11, 8,
		12, 13, 14, 14, 15, 12,
		16, 17, 18, 18, 19, 16,
		20, 21, 22, 22, 23, 20,
	}

	return &VoxelGeometry{
		InterleavedData: interleavedData,
		Indices:         indices,
		Name:            "Cube",
	}
}

func CreateSphereGeometry(radius float32, segments int) *VoxelGeometry {
	var interleavedData []float32
	var indices []int32

	for i := 0; i <= segments; i++ {
		lat := float32(i) * 3.14159 / float32(segments)
		for j := 0; j <= segments; j++ {
			lon := float32(j) * 2.0 * 3.14159 / float32(segments)

			x := radius * float32(math.Sin(float64(lat))) * float32(math.Cos(float64(lon)))
			y := radius * float32(math.Cos(float64(lat)))
			z := radius * float32(math.Sin(float64(lat))) * float32(math.Sin(float64(lon)))

			u := float32(j) / float32(segments)
			v := float32(i) / float32(segments)

			nx := x / radius
			ny := y / radius
			nz := z / radius

			interleavedData = append(interleavedData, x, y, z, u, v, nx, ny, nz)
		}
	}

	for i := 0; i < segments; i++ {
		for j := 0; j < segments; j++ {
			first := int32(i*(segments+1) + j)
			second := first + int32(segments+1)

			indices = append(indices,
				first, second, first+1,
				second, second+1, first+1,
			)
		}
	}

	return &VoxelGeometry{
		InterleavedData: interleavedData,
		Indices:         indices,
		Name:            "Sphere",
	}
}

func CreateTetrahedronGeometry(size float32) *VoxelGeometry {
	h := size * 0.816496

	interleavedData := []float32{
		0.0, h / 2, 0.0, 0.5, 1.0, 0.0, 1.0, 0.0,
		-size / 2, -h / 2, size / 2, 0.0, 0.0, -0.577, -0.577, 0.577,
		size / 2, -h / 2, size / 2, 1.0, 0.0, 0.577, -0.577, 0.577,

		0.0, h / 2, 0.0, 0.5, 1.0, 0.0, 1.0, 0.0,
		size / 2, -h / 2, size / 2, 0.0, 0.0, 0.577, -0.577, 0.577,
		0.0, -h / 2, -size, 1.0, 0.0, 0.0, -0.577, -0.816,

		0.0, h / 2, 0.0, 0.5, 1.0, 0.0, 1.0, 0.0,
		0.0, -h / 2, -size, 0.0, 0.0, 0.0, -0.577, -0.816,
		-size / 2, -h / 2, size / 2, 1.0, 0.0, -0.577, -0.577, 0.577,

		-size / 2, -h / 2, size / 2, 0.0, 0.0, 0.0, -1.0, 0.0,
		0.0, -h / 2, -size, 0.5, 1.0, 0.0, -1.0, 0.0,
		size / 2, -h / 2, size / 2, 1.0, 0.0, 0.0, -1.0, 0.0,
	}

	indices := []int32{
		0, 1, 2,
		3, 4, 5,
		6, 7, 8,
		9, 10, 11,
	}

	return &VoxelGeometry{
		InterleavedData: interleavedData,
		Indices:         indices,
		Name:            "Tetrahedron",
	}
}

// GetVoxelColor returns the color for a given voxel type
func GetVoxelColor(voxelID VoxelID) mgl32.Vec3 {
	switch voxelID {
	case 0: // Air
		return mgl32.Vec3{0, 0, 0}
	case 1: // Grass
		return mgl32.Vec3{0.3, 0.7, 0.2}
	case 2: // Dirt
		return mgl32.Vec3{0.6, 0.4, 0.2}
	case 3: // Stone
		return mgl32.Vec3{0.5, 0.5, 0.5}
	case 4: // Sand
		return mgl32.Vec3{0.9, 0.8, 0.5}
	case 5: // Wood (trunk)
		return mgl32.Vec3{0.4, 0.25, 0.1} // Brown
	case 6: // Leaves
		return mgl32.Vec3{0.2, 0.6, 0.2} // Green
	default:
		return mgl32.Vec3{0.8, 0.8, 0.8}
	}
}

func CreateInstancedVoxelModel(geometry *VoxelGeometry, instanceCount int) (*renderer.Model, error) {
	vertexCount := len(geometry.InterleavedData) / 8
	vertices := make([]float32, vertexCount*3)

	for i := 0; i < vertexCount; i++ {
		vertices[i*3] = geometry.InterleavedData[i*8]
		vertices[i*3+1] = geometry.InterleavedData[i*8+1]
		vertices[i*3+2] = geometry.InterleavedData[i*8+2]
	}

	model := &renderer.Model{
		InterleavedData: geometry.InterleavedData,
		Vertices:        vertices,
		Faces:           geometry.Indices,
		Position:        [3]float32{0, 0, 0},
		Rotation:        mgl32.Quat{W: 1, V: mgl32.Vec3{0, 0, 0}},
		Scale:           [3]float32{1, 1, 1},
		IsInstanced:     true,
		InstanceCount:   instanceCount,
		Material:        renderer.DefaultMaterial,
	}

	uniqueMaterial := *model.Material
	model.Material = &uniqueMaterial

	model.InstanceModelMatrices = make([]mgl32.Mat4, instanceCount)
	model.InstanceColors = make([]mgl32.Vec3, instanceCount) // Per-instance colors

	model.CalculateBoundingSphere()
	return model, nil
}
