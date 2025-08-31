package loader

import (
	"Gopher3D/internal/renderer"
	"math"

	"github.com/go-gl/mathgl/mgl32"
)

type VoxelID uint16

type VoxelData struct {
	ID       VoxelID
	Position mgl32.Vec3
	Active   bool
}

type VoxelChunk struct {
	Position    mgl32.Vec3
	Size        int
	Voxels      [][][]VoxelData
	Model       *renderer.Model
	NeedsUpdate bool
}

type VoxelWorld struct {
	ChunkSize    int
	WorldSizeX   int
	WorldSizeZ   int
	MaxHeight    int
	VoxelSize    float32
	Geometry     *VoxelGeometry
	Chunks       [][]*VoxelChunk
	ActiveVoxels int
}

type VoxelGeometry struct {
	InterleavedData []float32
	Indices         []int32
	Name            string
}

func NewVoxelWorld(chunkSize, worldSizeX, worldSizeZ, maxHeight int, voxelSize float32, geometry *VoxelGeometry) *VoxelWorld {
	world := &VoxelWorld{
		ChunkSize:  chunkSize,
		WorldSizeX: worldSizeX,
		WorldSizeZ: worldSizeZ,
		MaxHeight:  maxHeight,
		VoxelSize:  voxelSize,
		Geometry:   geometry,
		Chunks:     make([][]*VoxelChunk, worldSizeX),
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

func (world *VoxelWorld) CreateInstancedModel() (*renderer.Model, error) {
	return CreateInstancedVoxelModel(world.Geometry, world.ActiveVoxels)
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
	for i := 0; i < instanceCount; i++ {
		model.InstanceModelMatrices[i] = mgl32.Ident4()
	}

	model.CalculateBoundingSphere()
	return model, nil
}
