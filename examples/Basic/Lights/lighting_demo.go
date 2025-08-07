package main

// Modern Lighting Demo - Showcases PBR, HDR, Gamma Correction, and advanced lighting
import (
	behaviour "Gopher3D/internal/behaviour"
	"Gopher3D/internal/engine"
	loader "Gopher3D/internal/loader"
	"Gopher3D/internal/renderer"
	"fmt"
	"math"
	"time"

	mgl "github.com/go-gl/mathgl/mgl32"
)

type ModernLightingDemo struct {
	engine        *engine.Gopher
	models        []*renderer.Model
	currentDemo   int
	lastSwitch    time.Time
	demoTypes     []string
	materialNames []string
}

func NewModernLightingDemo(engine *engine.Gopher) {
	demo := &ModernLightingDemo{
		engine:        engine,
		lastSwitch:    time.Now(),
		demoTypes:     []string{"materials", "metals", "lighting_types", "exposure_demo", "advanced_materials", "lighting_dance"},
		materialNames: []string{"Plastic", "Rough Metal", "Polished Metal", "Matte", "Glossy"},
	}
	behaviour.GlobalBehaviourManager.Add(demo)
}

func main() {
	fmt.Println("=== 🚀 Gopher3D Modern Lighting Showcase ===")
	fmt.Println("This interactive demo showcases cutting-edge lighting techniques:")
	fmt.Println("🔥 PBR (Physically Based Rendering)")
	fmt.Println("🌈 HDR with ACES tone mapping")
	fmt.Println("⚡ Gamma correction")
	fmt.Println("✨ Fresnel reflectance")
	fmt.Println("🎯 GGX specular distribution")
	fmt.Println("🛡️ Smith geometry masking")
	fmt.Println("🎨 Advanced material presets")
	fmt.Println("🎪 Dynamic lighting scenarios")
	fmt.Println("🌟 Animated material showcase")
	fmt.Println("\n🎬 Demo modes cycle every 10 seconds - sit back and enjoy!")
	fmt.Println("🎮 Use WASD + Mouse to fly around and explore!")

	engine := engine.NewGopher(engine.OPENGL)
	engine.Width = 1200
	engine.Height = 800
	NewModernLightingDemo(engine)
	engine.Render(100, 100) // Window at position 100,100 (lower on screen)
}

func (mld *ModernLightingDemo) Start() {
	// Setup camera for optimal viewing - positioned closer for better sphere visibility
	mld.engine.Camera.InvertMouse = false
	mld.engine.Camera.Position = mgl.Vec3{0, 100, 400} // Closer and lower for better perspective
	mld.engine.Camera.Speed = 500
	mld.engine.SetDebugMode(false)

	// Create high-quality directional lighting (like studio lighting)
	mld.engine.Light = renderer.CreateSunlight(mgl.Vec3{-0.4, -0.7, -0.6})
	mld.engine.Light.Intensity = 1.8 // Higher intensity for PBR
	mld.engine.Light.AmbientStrength = 0.15

	// Create scene with multiple objects to showcase materials
	mld.createScene()
	mld.setDemo(0)
}

func (mld *ModernLightingDemo) createScene() {
	// Create multiple spheres with better spacing for far viewing distance
	positions := []mgl.Vec3{
		{-150, 30, 0}, // Left
		{-75, 30, 0},  // Center-left
		{0, 30, 0},    // Center
		{75, 30, 0},   // Center-right
		{150, 30, 0},  // Right
	}

	for i, pos := range positions {
		sphere, err := loader.LoadObjectWithPath("../../resources/obj/Sphere.obj", true)
		if err != nil {
			fmt.Printf("Warning: Could not load sphere, using cube instead\n")
			sphere, err = loader.LoadObjectWithPath("../../resources/obj/Cube.obj", true)
			if err != nil {
				fmt.Printf("ERROR: Could not load cube either: %v\n", err)
				continue // Skip this sphere if both fail
			}
		}

		sphere.SetPosition(pos.X(), pos.Y(), pos.Z())
		sphere.Scale = mgl.Vec3{20, 20, 20} // Larger spheres for better visibility at far distance

		// Set initial material (will be updated in demo cycles)
		switch i {
		case 0:
			sphere.SetDiffuseColor(0.8, 0.1, 0.1) // Red
		case 1:
			sphere.SetDiffuseColor(0.1, 0.8, 0.1) // Green
		case 2:
			sphere.SetDiffuseColor(0.1, 0.1, 0.8) // Blue
		case 3:
			sphere.SetDiffuseColor(0.8, 0.8, 0.1) // Yellow
		case 4:
			sphere.SetDiffuseColor(0.8, 0.1, 0.8) // Magenta
		}

		mld.engine.AddModel(sphere)
		mld.models = append(mld.models, sphere)
	}

	// No ground plane - clean minimal scene focusing only on materials
}

func (mld *ModernLightingDemo) Update() {
	// Cycle through different demo modes every 10 seconds (longer for better appreciation)
	if time.Since(mld.lastSwitch) > 10*time.Second {
		mld.currentDemo = (mld.currentDemo + 1) % len(mld.demoTypes)
		mld.setDemo(mld.currentDemo)
		mld.lastSwitch = time.Now()
	}

	// Very subtle sphere rotation for better material showcase
	mld.animateSpheres()
}

func (mld *ModernLightingDemo) UpdateFixed() {}

func (mld *ModernLightingDemo) setDemo(index int) {
	demoType := mld.demoTypes[index]

	switch demoType {
	case "materials":
		fmt.Println("\n🎨 MATERIAL SHOWCASE: Different PBR material types")
		mld.setMaterialShowcase()

	case "metals":
		fmt.Println("\n⚡ METALLIC MATERIALS: Different metallic surfaces")
		mld.setMetallicShowcase()

	case "lighting_types":
		fmt.Println("\n💡 LIGHTING COMPARISON: Point vs Directional with PBR")
		mld.setLightingShowcase()

	case "exposure_demo":
		fmt.Println("\n🌈 HDR EXPOSURE DEMO: Dynamic range control")
		mld.setExposureShowcase()

	case "advanced_materials":
		fmt.Println("\n💎 ADVANCED MATERIALS: Real-world material simulation")
		mld.setAdvancedMaterialShowcase()

	case "lighting_dance":
		fmt.Println("\n🎪 LIGHTING DANCE: Dynamic lighting choreography")
		mld.setLightingDance()
	}
}

func (mld *ModernLightingDemo) setMaterialShowcase() {
	colors := [][3]float32{
		{0.8, 0.1, 0.1}, // Red
		{0.1, 0.8, 0.1}, // Green
		{0.1, 0.1, 0.8}, // Blue
		{0.8, 0.8, 0.1}, // Yellow
		{0.8, 0.1, 0.8}, // Magenta
	}

	for i := 0; i < len(mld.models) && i < len(colors); i++ {
		color := colors[i]

		switch i {
		case 0: // Plastic (non-metallic)
			mld.models[i].SetPlasticMaterial(color[0], color[1], color[2], 0.3)
		case 1: // Rough metal
			mld.models[i].SetRoughMetal(color[0], color[1], color[2])
		case 2: // Polished metal
			mld.models[i].SetPolishedMetal(color[0], color[1], color[2])
		case 3: // Matte surface
			mld.models[i].SetMatte(color[0], color[1], color[2])
		case 4: // Glossy surface
			mld.models[i].SetGlossy(color[0], color[1], color[2])
		}

		mld.models[i].SetExposure(1.0) // Standard exposure
		fmt.Printf("   Sphere %d: %s\n", i+1, mld.materialNames[i])
	}
}

func (mld *ModernLightingDemo) setMetallicShowcase() {
	// Show different metallic roughness values
	roughnessValues := []float32{0.0, 0.3, 0.6, 0.8, 1.0}
	baseColor := [3]float32{0.7, 0.7, 0.8} // Metallic silver-like color

	for i := 0; i < len(mld.models) && i < len(roughnessValues); i++ {
		mld.models[i].SetMetallicMaterial(baseColor[0], baseColor[1], baseColor[2], roughnessValues[i])
		mld.models[i].SetExposure(1.2) // Slightly higher exposure for metals
		fmt.Printf("   Sphere %d: Metallic, Roughness %.1f\n", i+1, roughnessValues[i])
	}
}

func (mld *ModernLightingDemo) setLightingShowcase() {
	// Switch between point and directional lighting
	static := time.Now().Unix()%2 == 0

	if static {
		// Directional lighting (sun-like)
		mld.engine.Light = renderer.CreateSunlight(mgl.Vec3{-0.4, -0.7, -0.6})
		mld.engine.Light.Intensity = 2.0
		mld.engine.Light.AmbientStrength = 0.1
		fmt.Println("   Current: Directional Light (Sun-like)")
	} else {
		// Point lighting (studio-like) - positioned for perfect reflection alignment
		mld.engine.Light = renderer.CreatePointLight(
			mgl.Vec3{0, 120, 100}, // Higher and further for better reflection visibility
			mgl.Vec3{1.0, 0.95, 0.8},
			3.5, 250.0, // Slightly higher intensity and range
		)
		mld.engine.Light.AmbientStrength = 0.06 // Lower ambient to see reflections better
		fmt.Println("   Current: Point Light (Perfect Reflection Mode)")
	}

	// Set all materials to glossy for good reflection showcase
	for i := 0; i < len(mld.models); i++ {
		colors := [][3]float32{
			{0.8, 0.1, 0.1}, {0.1, 0.8, 0.1}, {0.1, 0.1, 0.8},
			{0.8, 0.8, 0.1}, {0.8, 0.1, 0.8},
		}
		if i < len(colors) {
			mld.models[i].SetGlossy(colors[i][0], colors[i][1], colors[i][2])
			mld.models[i].SetExposure(1.0)
		}
	}
}

func (mld *ModernLightingDemo) setExposureShowcase() {
	// Demonstrate HDR exposure control
	exposureValues := []float32{0.5, 0.8, 1.0, 1.5, 2.5}

	for i := 0; i < len(mld.models) && i < len(exposureValues); i++ {
		// Set bright materials to show exposure effect
		mld.models[i].SetGlossy(0.9, 0.9, 0.9) // Bright white
		mld.models[i].SetExposure(exposureValues[i])
		fmt.Printf("   Sphere %d: Exposure %.1f\n", i+1, exposureValues[i])
	}

	// Use bright lighting for dramatic effect
	mld.engine.Light = renderer.CreateSunlight(mgl.Vec3{-0.3, -0.8, -0.5})
	mld.engine.Light.Intensity = 3.0 // Very bright
	mld.engine.Light.AmbientStrength = 0.2
}

// Removed background elements to avoid artifacts and visual clutter
// Clean scene focuses attention on the materials being showcased

// Animate spheres with subtle rotation for better material visibility
func (mld *ModernLightingDemo) animateSpheres() {
	currentTime := float32(time.Since(mld.lastSwitch).Seconds())

	// Only rotate spheres during non-critical demos for perfect reflection alignment
	if mld.currentDemo != 2 { // Don't rotate during "Lighting Comparison" for perfect reflections
		for i := 0; i < 5 && i < len(mld.models); i++ {
			rotationSpeed := 0.3 + float32(i)*0.05                      // Even slower, minimal variation
			mld.models[i].Rotate(0, rotationSpeed*currentTime*0.003, 0) // Ultra-slow Y-axis rotation
		}
	}
}

// Advanced material showcase with real-world materials
func (mld *ModernLightingDemo) setAdvancedMaterialShowcase() {
	realWorldMaterials := []struct {
		name     string
		setter   func(*renderer.Model)
		exposure float32
	}{
		{"Chrome", func(m *renderer.Model) { m.SetPolishedMetal(0.95, 0.95, 0.95) }, 1.3},
		{"Gold", func(m *renderer.Model) { m.SetPolishedMetal(1.0, 0.8, 0.2) }, 1.4},
		{"Copper", func(m *renderer.Model) { m.SetRoughMetal(0.9, 0.4, 0.3) }, 1.2},
		{"Ceramic", func(m *renderer.Model) { m.SetGlossy(0.9, 0.9, 0.95) }, 1.0},
		{"Rubber", func(m *renderer.Model) { m.SetMatte(0.2, 0.2, 0.2) }, 0.8},
	}

	materialNames := []string{"Chrome", "Gold", "Copper", "Ceramic", "Rubber"}

	for i := 0; i < len(mld.models) && i < len(realWorldMaterials); i++ { // All models are spheres
		realWorldMaterials[i].setter(mld.models[i])
		mld.models[i].SetExposure(realWorldMaterials[i].exposure)
		fmt.Printf("   Sphere %d: %s\n", i+1, materialNames[i])
	}

	// Use dramatic lighting for material showcase
	mld.engine.Light = renderer.CreatePointLight(
		mgl.Vec3{-80, 120, 80},
		mgl.Vec3{1.0, 0.9, 0.7}, // Warm studio light
		4.0, 300.0,
	)
	mld.engine.Light.AmbientStrength = 0.05 // Low ambient for dramatic shadows
}

// Dynamic lighting choreography - lights move and change
func (mld *ModernLightingDemo) setLightingDance() {
	currentTime := float32(time.Since(mld.lastSwitch).Seconds())

	// Create moving light based on time
	lightAngle := currentTime * 0.3 // Slow rotation
	lightHeight := 100.0 + 30.0*float32(math.Sin(float64(currentTime*0.5)))
	lightRadius := float32(120.0)

	lightX := lightRadius * float32(math.Cos(float64(lightAngle)))
	lightZ := lightRadius * float32(math.Sin(float64(lightAngle)))

	// Create dynamic point light that moves in a circle
	mld.engine.Light = renderer.CreatePointLight(
		mgl.Vec3{lightX, lightHeight, lightZ},
		mgl.Vec3{
			0.8 + 0.2*float32(math.Sin(float64(currentTime*0.7))), // Red oscillation
			0.6 + 0.4*float32(math.Sin(float64(currentTime*0.9))), // Green oscillation
			0.7 + 0.3*float32(math.Sin(float64(currentTime*1.1))), // Blue oscillation
		},
		2.5+1.5*float32(math.Sin(float64(currentTime*0.6))), // Intensity oscillation
		250.0,
	)
	mld.engine.Light.AmbientStrength = 0.1

	// Set all spheres to highly reflective materials for best light dance effect
	colors := [][3]float32{
		{0.9, 0.1, 0.1}, {0.1, 0.9, 0.1}, {0.1, 0.1, 0.9},
		{0.9, 0.9, 0.1}, {0.9, 0.1, 0.9},
	}

	for i := 0; i < len(mld.models) && i < len(colors); i++ { // All models are spheres
		color := colors[i]
		mld.models[i].SetPolishedMetal(color[0], color[1], color[2]) // All polished for reflections
		mld.models[i].SetExposure(1.1)
	}

	fmt.Printf("   🎪 Light dancing at position (%.1f, %.1f, %.1f)\n", lightX, lightHeight, lightZ)
	fmt.Printf("   🌈 Dynamic color: (%.2f, %.2f, %.2f)\n",
		mld.engine.Light.Color[0], mld.engine.Light.Color[1], mld.engine.Light.Color[2])
}
