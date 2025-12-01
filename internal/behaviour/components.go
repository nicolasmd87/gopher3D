package behaviour

import (
	"github.com/go-gl/mathgl/mgl32"
)

// ComponentType defines the category of a component
type ComponentType string

const (
	ComponentTypeMesh      ComponentType = "Mesh"
	ComponentTypeScript    ComponentType = "Script"
	ComponentTypeBehaviour ComponentType = "Behaviour"
	ComponentTypeRenderer  ComponentType = "Renderer"
	ComponentTypeCollider  ComponentType = "Collider"
	ComponentTypeAudio     ComponentType = "Audio"
	ComponentTypeLight     ComponentType = "Light"
	ComponentTypeCamera    ComponentType = "Camera"
	ComponentTypeWater     ComponentType = "Water"
	ComponentTypeVoxel     ComponentType = "Voxel"
	ComponentTypeCustom    ComponentType = "Custom"
)

// TypedComponent extends Component with type information
type TypedComponent interface {
	Component
	GetComponentType() ComponentType
	GetTypeName() string
}

// MeshComponent holds a reference to a mesh/model
type MeshComponent struct {
	BaseComponent
	MeshPath     string `json:"mesh_path"`     // Path to the mesh file
	MaterialPath string `json:"material_path"` // Optional material/texture path

	// Material properties
	DiffuseColor  [3]float32 `json:"diffuse_color"`
	SpecularColor [3]float32 `json:"specular_color"`
	Metallic      float32    `json:"metallic"`
	Roughness     float32    `json:"roughness"`
	Alpha         float32    `json:"alpha"`

	// Runtime reference
	Model  interface{} `json:"-"` // The actual renderer.Model
	Loaded bool        `json:"-"` // Whether mesh has been loaded
}

func NewMeshComponent() *MeshComponent {
	return &MeshComponent{
		DiffuseColor:  [3]float32{0.8, 0.8, 0.8},
		SpecularColor: [3]float32{1.0, 1.0, 1.0},
		Metallic:      0.0,
		Roughness:     0.5,
		Alpha:         1.0,
		Loaded:        false,
	}
}

func (m *MeshComponent) GetComponentType() ComponentType {
	return ComponentTypeMesh
}

func (m *MeshComponent) GetTypeName() string {
	return "MeshComponent"
}

func (m *MeshComponent) SetMesh(mesh interface{}) {
	m.Model = mesh
	m.Loaded = true
	// Also set on GameObject if available
	if m.GetGameObject() != nil {
		m.GetGameObject().SetModel(mesh)
	}
}

func (m *MeshComponent) GetMesh() interface{} {
	return m.Model
}

// WaterComponent holds water simulation data
type WaterComponent struct {
	BaseComponent
	OceanSize           float32    `json:"ocean_size"`
	BaseAmplitude       float32    `json:"base_amplitude"`
	WaterColor          [3]float32 `json:"water_color"`
	Transparency        float32    `json:"transparency"`
	WaveSpeedMultiplier float32    `json:"wave_speed_multiplier"`
	WaveHeight          float32    `json:"wave_height"`     // Wave amplitude multiplier
	WaveRandomness      float32    `json:"wave_randomness"` // Adds randomness/choppiness to waves
	FoamEnabled         bool       `json:"foam_enabled"`
	FoamIntensity       float32    `json:"foam_intensity"`
	CausticsEnabled     bool       `json:"caustics_enabled"`
	CausticsIntensity   float32    `json:"caustics_intensity"`
	CausticsScale       float32    `json:"caustics_scale"`
	SpecularIntensity   float32    `json:"specular_intensity"`
	NormalStrength      float32    `json:"normal_strength"`
	DistortionStrength  float32    `json:"distortion_strength"`
	ShadowStrength      float32    `json:"shadow_strength"`

	// Runtime references
	Simulation interface{} `json:"-"` // The water simulation behaviour
	Model      interface{} `json:"-"` // The rendered model
	Generated  bool        `json:"-"` // Whether water has been generated
}

func NewWaterComponent() *WaterComponent {
	return &WaterComponent{
		OceanSize:           1000,
		BaseAmplitude:       2.0,
		WaterColor:          [3]float32{0.0, 0.3, 0.5},
		Transparency:        0.7,
		WaveSpeedMultiplier: 1.0,
		WaveHeight:          1.0,
		WaveRandomness:      0.0,
		FoamEnabled:         true,
		FoamIntensity:       0.5,
		CausticsEnabled:     true,
		CausticsIntensity:   0.3,
		CausticsScale:       1.0,
		SpecularIntensity:   1.0,
		NormalStrength:      1.0,
		DistortionStrength:  0.1,
		ShadowStrength:      0.5,
		Generated:           false,
	}
}

func (w *WaterComponent) GetComponentType() ComponentType {
	return ComponentTypeWater
}

func (w *WaterComponent) GetTypeName() string {
	return "WaterComponent"
}

// VoxelTerrainComponent holds voxel terrain data
type VoxelTerrainComponent struct {
	BaseComponent

	// Terrain generation settings
	Scale       float32 `json:"scale"`
	Amplitude   float32 `json:"amplitude"`
	Seed        int32   `json:"seed"`
	Threshold   float32 `json:"threshold"`
	Octaves     int32   `json:"octaves"`
	ChunkSize   int32   `json:"chunk_size"`
	WorldSize   int32   `json:"world_size"` // in chunks
	Biome       int32   `json:"biome"`
	TreeDensity float32 `json:"tree_density"`

	// Color configuration
	GrassColor  [3]float32 `json:"grass_color"`
	DirtColor   [3]float32 `json:"dirt_color"`
	StoneColor  [3]float32 `json:"stone_color"`
	SandColor   [3]float32 `json:"sand_color"`
	WoodColor   [3]float32 `json:"wood_color"`
	LeavesColor [3]float32 `json:"leaves_color"`

	// Runtime references
	VoxelWorld interface{} `json:"-"` // The actual voxel world
	Model      interface{} `json:"-"` // The rendered model
	Generated  bool        `json:"-"` // Whether terrain has been generated
}

func NewVoxelTerrainComponent() *VoxelTerrainComponent {
	return &VoxelTerrainComponent{
		Scale:       0.05,
		Amplitude:   15.0,
		Seed:        0,
		Threshold:   0.2,
		Octaves:     4,
		ChunkSize:   32,
		WorldSize:   2,
		Biome:       0, // Plains
		TreeDensity: 0.02,
		GrassColor:  [3]float32{0.3, 0.7, 0.2},
		DirtColor:   [3]float32{0.6, 0.4, 0.2},
		StoneColor:  [3]float32{0.5, 0.5, 0.5},
		SandColor:   [3]float32{0.9, 0.8, 0.5},
		WoodColor:   [3]float32{0.4, 0.25, 0.1},
		LeavesColor: [3]float32{0.2, 0.6, 0.2},
		Generated:   false,
	}
}

func (v *VoxelTerrainComponent) GetComponentType() ComponentType {
	return ComponentTypeVoxel
}

func (v *VoxelTerrainComponent) GetTypeName() string {
	return "VoxelTerrainComponent"
}

// LightComponent holds light data
type LightComponent struct {
	BaseComponent
	LightMode       string // "directional", "point", "spot"
	Color           [3]float32
	Intensity       float32
	Range           float32
	AmbientStrength float32
	Direction       mgl32.Vec3

	// Runtime reference
	LightData interface{}
}

func NewLightComponent() *LightComponent {
	return &LightComponent{
		LightMode:       "point",
		Color:           [3]float32{1.0, 1.0, 1.0},
		Intensity:       1.0,
		Range:           100.0,
		AmbientStrength: 0.1,
		Direction:       mgl32.Vec3{0, -1, 0},
	}
}

func (l *LightComponent) GetComponentType() ComponentType {
	return ComponentTypeLight
}

func (l *LightComponent) GetTypeName() string {
	return "LightComponent"
}

// CameraComponent holds camera data
type CameraComponent struct {
	BaseComponent
	FOV    float32
	Near   float32
	Far    float32
	IsMain bool // Is this the main game camera?

	// Runtime reference
	CameraData interface{}
}

func NewCameraComponent() *CameraComponent {
	return &CameraComponent{
		FOV:    45.0,
		Near:   0.1,
		Far:    10000.0,
		IsMain: false,
	}
}

func (c *CameraComponent) GetComponentType() ComponentType {
	return ComponentTypeCamera
}

func (c *CameraComponent) GetTypeName() string {
	return "CameraComponent"
}

// ScriptComponent is a wrapper for user scripts to identify them as scripts
type ScriptComponent struct {
	BaseComponent
	ScriptName string
	Script     Component // The actual script implementation
}

func NewScriptComponent(scriptName string, script Component) *ScriptComponent {
	return &ScriptComponent{
		ScriptName: scriptName,
		Script:     script,
	}
}

func (s *ScriptComponent) GetComponentType() ComponentType {
	return ComponentTypeScript
}

func (s *ScriptComponent) GetTypeName() string {
	return s.ScriptName
}

func (s *ScriptComponent) Awake() {
	if s.Script != nil {
		s.Script.SetGameObject(s.GetGameObject())
		s.Script.Awake()
	}
}

func (s *ScriptComponent) Start() {
	if s.Script != nil {
		s.Script.Start()
	}
}

func (s *ScriptComponent) Update() {
	if s.Script != nil && s.GetEnabled() {
		s.Script.Update()
	}
}

func (s *ScriptComponent) FixedUpdate() {
	if s.Script != nil && s.GetEnabled() {
		s.Script.FixedUpdate()
	}
}

func (s *ScriptComponent) OnDestroy() {
	if s.Script != nil {
		s.Script.OnDestroy()
	}
}

// Helper function to get component type name
func GetComponentTypeName(comp Component) string {
	if typed, ok := comp.(TypedComponent); ok {
		return typed.GetTypeName()
	}
	return "Unknown"
}

// Helper function to get component category
func GetComponentCategory(comp Component) ComponentType {
	if typed, ok := comp.(TypedComponent); ok {
		return typed.GetComponentType()
	}
	return ComponentTypeCustom
}

// BuiltInComponents returns a list of built-in component types that can be added
func BuiltInComponents() []string {
	return []string{
		"MeshComponent",
		"WaterComponent",
		"VoxelTerrainComponent",
		"LightComponent",
		"CameraComponent",
	}
}

// CreateBuiltInComponent creates a built-in component by name
func CreateBuiltInComponent(name string) Component {
	switch name {
	case "MeshComponent":
		return NewMeshComponent()
	case "WaterComponent":
		return NewWaterComponent()
	case "VoxelTerrainComponent":
		return NewVoxelTerrainComponent()
	case "LightComponent":
		return NewLightComponent()
	case "CameraComponent":
		return NewCameraComponent()
	default:
		return nil
	}
}
