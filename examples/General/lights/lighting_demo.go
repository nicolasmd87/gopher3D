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
	configs       []renderer.AdvancedRenderingConfig
}

func NewModernLightingDemo(engine *engine.Gopher) {
	demo := &ModernLightingDemo{
		engine:        engine,
		lastSwitch:    time.Now(),
		demoTypes:     []string{"materials", "metals", "lighting_types", "exposure_demo", "advanced_materials", "lighting_dance", "modern_pbr", "volumetric_lighting", "clearcoat_demo", "transmission_demo"},
		materialNames: []string{"Plastic", "Rough Metal", "Polished Metal", "Matte", "Glossy"},
		configs: []renderer.AdvancedRenderingConfig{
			renderer.DefaultAdvancedRenderingConfig(),
			renderer.HighQualityRenderingConfig(),
			renderer.PerformanceRenderingConfig(),
		},
	}
	behaviour.GlobalBehaviourManager.Add(demo)
}

func main() {
	fmt.Println("=== Gopher3D Advanced Lighting Showcase 2024 ===")
	fmt.Println("This interactive demo showcases cutting-edge lighting techniques:")
	fmt.Println("- Enhanced PBR (Physically Based Rendering)")
	fmt.Println("- HDR with ACES tone mapping")
	fmt.Println("- Gamma correction")
	fmt.Println("- Fresnel reflectance")
	fmt.Println("- GGX specular distribution")
	fmt.Println("- Smith geometry masking")
	fmt.Println("- Advanced material presets")
	fmt.Println("- Dynamic lighting scenarios")
	fmt.Println("- Animated material showcase")
	fmt.Println("- Clearcoat materials (automotive paint)")
	fmt.Println("- Sheen materials (fabric, velvet)")
	fmt.Println("- Transmission materials (glass)")
	fmt.Println("- Volumetric lighting effects")
	fmt.Println("- Screen Space Ambient Occlusion (SSAO)")
	fmt.Println("- Global Illumination approximation")
	fmt.Println("- Bloom effects")
	fmt.Println("- Multiple scattering compensation")
	fmt.Println("- Energy conservation")
	fmt.Println("\nDemo modes cycle every 10 seconds - sit back and enjoy!")
	fmt.Println("Use WASD + Mouse to fly around and explore!")

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

	// Create consistent directional lighting (sun-like) that stays fixed
	// Light coming from above and slightly to the side (like afternoon sun)
	// Note: Direction vector points WHERE the light comes FROM
	mld.engine.Light = renderer.CreateSunlight(mgl.Vec3{0.3, 0.8, 0.5})
	mld.engine.Light.Intensity = 1.5        // Reduced intensity to prevent washing out
	mld.engine.Light.AmbientStrength = 0.15 // Reduced ambient for better contrast

	fmt.Printf("FIXED DIRECTIONAL LIGHT: Direction=(0.3, 0.8, 0.5), Mode=%s\n", mld.engine.Light.Mode)

	// Add skybox for better lighting and reflections
	mld.addSkybox()

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
		sphere, err := loader.LoadObjectWithPath("../../resources/obj/Sphere.obj", false)
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
		fmt.Println("\nMATERIAL SHOWCASE: Different PBR material types")
		mld.setMaterialShowcase()

	case "metals":
		fmt.Println("\nMETALLIC MATERIALS: Different metallic surfaces")
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

	case "modern_pbr":
		fmt.Println("\n💎 MODERN PBR: Enhanced physically based rendering")
		mld.setModernPBRDemo()

	case "volumetric_lighting":
		fmt.Println("\n🌫️ VOLUMETRIC LIGHTING: Light shafts and atmospheric effects")
		mld.setVolumetricDemo()

	case "clearcoat_demo":
		fmt.Println("\n🚗 CLEARCOAT DEMO: Automotive paint and lacquered surfaces")
		mld.setClearcoatDemo()

	case "transmission_demo":
		fmt.Println("\n🔍 TRANSMISSION DEMO: Glass and translucent materials")
		mld.setTransmissionDemo()
	}
}

func (mld *ModernLightingDemo) setMaterialShowcase() {
	// Keep the same directional light - don't change it
	// mld.engine.Light stays as the fixed directional light

	colors := [][3]float32{
		{0.8, 0.1, 0.1}, // Red
		{0.1, 0.8, 0.1}, // Green
		{0.1, 0.1, 0.8}, // Blue
		{0.8, 0.8, 0.1}, // Yellow
		{0.8, 0.1, 0.8}, // Magenta
	}

	materialTypes := []string{"Plastic (Non-metallic)", "Rough Metal", "Polished Metal (Reflective)", "Matte Surface", "Glossy Surface"}

	for i := 0; i < len(mld.models) && i < len(colors); i++ {
		color := colors[i]

		switch i {
		case 0: // Plastic (non-metallic)
			mld.models[i].SetPlasticMaterial(color[0], color[1], color[2], 0.3)
			// Ensure it's non-metallic
			mld.models[i].SetMaterialPBR(0.0, 0.3)
		case 1: // Rough metal
			mld.models[i].SetRoughMetal(color[0], color[1], color[2])
			// Ensure it's metallic and rough
			mld.models[i].SetMaterialPBR(1.0, 0.8)
		case 2: // Polished metal (highly reflective)
			mld.models[i].SetPolishedMetal(color[0], color[1], color[2])
			// Ensure it's metallic and smooth (reflective)
			mld.models[i].SetMaterialPBR(1.0, 0.05)
		case 3: // Matte surface
			mld.models[i].SetMatte(color[0], color[1], color[2])
			// Ensure it's non-metallic and rough
			mld.models[i].SetMaterialPBR(0.0, 0.9)
		case 4: // Glossy surface
			mld.models[i].SetGlossy(color[0], color[1], color[2])
			// Ensure it's non-metallic and smooth
			mld.models[i].SetMaterialPBR(0.0, 0.2)
		}

		mld.models[i].SetExposure(1.0) // Standard exposure
		mld.models[i].SetAlpha(1.0)    // Ensure fully opaque

		// Safely access material properties
		metallic := float32(0.0)
		roughness := float32(0.5)
		if mld.models[i].Material != nil {
			metallic = mld.models[i].Material.Metallic
			roughness = mld.models[i].Material.Roughness
		}
		fmt.Printf("   Sphere %d: %s (Metallic=%.1f, Roughness=%.1f)\n", i+1, materialTypes[i], metallic, roughness)
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
	// Keep the same directional light - demonstrate consistent lighting behavior
	// Don't change the light type to avoid confusion
	fmt.Println("   Current: Fixed Directional Light (Consistent Behavior)")
	fmt.Printf("   Light Direction: (%.1f, %.1f, %.1f)\n", mld.engine.Light.Direction[0], mld.engine.Light.Direction[1], mld.engine.Light.Direction[2])
	fmt.Printf("   Light Mode: %s\n", mld.engine.Light.Mode)

	// Set all materials to different reflectivity levels to show lighting consistency
	reflectivityLevels := []struct {
		name      string
		metallic  float32
		roughness float32
	}{
		{"Matte Plastic", 0.0, 0.9},
		{"Glossy Plastic", 0.0, 0.2},
		{"Rough Metal", 1.0, 0.8},
		{"Polished Metal", 1.0, 0.1},
		{"Mirror Metal", 1.0, 0.05},
	}

	colors := [][3]float32{
		{0.8, 0.1, 0.1}, {0.1, 0.8, 0.1}, {0.1, 0.1, 0.8},
		{0.8, 0.8, 0.1}, {0.8, 0.1, 0.8},
	}

	for i := 0; i < len(mld.models) && i < len(colors) && i < len(reflectivityLevels); i++ {
		mld.models[i].SetDiffuseColor(colors[i][0], colors[i][1], colors[i][2])
		mld.models[i].SetMaterialPBR(reflectivityLevels[i].metallic, reflectivityLevels[i].roughness)
		mld.models[i].SetExposure(1.0)
		fmt.Printf("   Sphere %d: %s (M=%.1f, R=%.1f)\n", i+1, reflectivityLevels[i].name, reflectivityLevels[i].metallic, reflectivityLevels[i].roughness)
	}
}

func (mld *ModernLightingDemo) setExposureShowcase() {
	// Keep the same directional light - just increase intensity for HDR demo
	mld.engine.Light.Intensity = 3.0 // Very bright for HDR effect
	mld.engine.Light.AmbientStrength = 0.2

	// Demonstrate HDR exposure control
	exposureValues := []float32{0.5, 0.8, 1.0, 1.5, 2.5}

	for i := 0; i < len(mld.models) && i < len(exposureValues); i++ {
		// Set bright materials to show exposure effect
		mld.models[i].SetGlossy(0.9, 0.9, 0.9) // Bright white
		mld.models[i].SetMaterialPBR(0.0, 0.2) // Non-metallic, glossy
		mld.models[i].SetExposure(exposureValues[i])
		fmt.Printf("   Sphere %d: Exposure %.1f (HDR Effect)\n", i+1, exposureValues[i])
	}
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
	// Keep the same directional light - reset intensity if it was changed
	mld.engine.Light.Intensity = 2.0
	mld.engine.Light.AmbientStrength = 0.15

	realWorldMaterials := []struct {
		name      string
		setter    func(*renderer.Model)
		exposure  float32
		metallic  float32
		roughness float32
	}{
		{"Chrome (Mirror)", func(m *renderer.Model) { m.SetPolishedMetal(0.95, 0.95, 0.95) }, 1.3, 1.0, 0.05},
		{"Gold (Reflective)", func(m *renderer.Model) { m.SetPolishedMetal(1.0, 0.8, 0.2) }, 1.4, 1.0, 0.1},
		{"Copper (Metal)", func(m *renderer.Model) { m.SetRoughMetal(0.9, 0.4, 0.3) }, 1.2, 1.0, 0.6},
		{"Ceramic (Glossy)", func(m *renderer.Model) { m.SetGlossy(0.9, 0.9, 0.95) }, 1.0, 0.0, 0.2},
		{"Rubber (Matte)", func(m *renderer.Model) { m.SetMatte(0.2, 0.2, 0.2) }, 0.8, 0.0, 0.9},
	}

	for i := 0; i < len(mld.models) && i < len(realWorldMaterials); i++ {
		material := realWorldMaterials[i]
		material.setter(mld.models[i])
		// Ensure proper PBR values
		mld.models[i].SetMaterialPBR(material.metallic, material.roughness)
		mld.models[i].SetExposure(material.exposure)
		fmt.Printf("   Sphere %d: %s (M=%.1f, R=%.1f)\n", i+1, material.name, material.metallic, material.roughness)
	}
}

// Dynamic lighting choreography - animate materials instead of light position
func (mld *ModernLightingDemo) setLightingDance() {
	// Keep the same directional light - don't move it around
	mld.engine.Light.Intensity = 2.5
	mld.engine.Light.AmbientStrength = 0.1

	currentTime := float32(time.Since(mld.lastSwitch).Seconds())

	// Instead of moving the light, animate the material properties
	colors := [][3]float32{
		{0.9, 0.1, 0.1}, {0.1, 0.9, 0.1}, {0.1, 0.1, 0.9},
		{0.9, 0.9, 0.1}, {0.9, 0.1, 0.9},
	}

	for i := 0; i < len(mld.models) && i < len(colors); i++ {
		color := colors[i]

		// Animate roughness for dynamic material behavior
		animatedRoughness := 0.05 + 0.3*float32(math.Sin(float64(currentTime*0.5+float32(i)*0.5)))

		// Animate exposure for dynamic brightness
		animatedExposure := 1.0 + 0.3*float32(math.Sin(float64(currentTime*0.3+float32(i)*0.3)))

		mld.models[i].SetPolishedMetal(color[0], color[1], color[2])
		mld.models[i].SetMaterialPBR(1.0, animatedRoughness) // Metallic with animated roughness
		mld.models[i].SetExposure(animatedExposure)

		fmt.Printf("   🎪 Sphere %d: Animated roughness=%.2f, exposure=%.2f\n", i+1, animatedRoughness, animatedExposure)
	}

	fmt.Printf("   🌞 Fixed Directional Light: Direction=(%.1f, %.1f, %.1f), Intensity=%.1f\n",
		mld.engine.Light.Direction[0], mld.engine.Light.Direction[1], mld.engine.Light.Direction[2], mld.engine.Light.Intensity)
}

// Modern PBR demo with advanced features
func (mld *ModernLightingDemo) setModernPBRDemo() {
	// Keep the same directional light - just increase intensity for bloom
	mld.engine.Light.Intensity = 2.5
	mld.engine.Light.AmbientStrength = 0.15

	// Apply high-quality rendering config to all models
	config := renderer.HighQualityRenderingConfig()
	config.EnableBloom = true
	config.EnableSSAO = true
	config.EnableMultipleScattering = true
	config.EnableEnergyConservation = true
	config.EnableGlobalIllumination = true
	config.GIIntensity = 0.3
	config.GIBounces = 2

	materialTypes := []struct {
		name      string
		setter    func(*renderer.Model)
		metallic  float32
		roughness float32
		exposure  float32
	}{
		{"Enhanced Plastic", func(m *renderer.Model) { m.SetPlasticMaterial(0.9, 0.1, 0.1, 0.2) }, 0.0, 0.2, 1.5},
		{"Rough Metal", func(m *renderer.Model) { m.SetRoughMetal(0.1, 0.9, 0.1) }, 1.0, 0.8, 1.2},
		{"Polished Metal", func(m *renderer.Model) { m.SetPolishedMetal(0.1, 0.1, 0.9) }, 1.0, 0.1, 1.3},
		{"Glossy Surface", func(m *renderer.Model) { m.SetGlossy(0.9, 0.9, 0.1) }, 0.0, 0.2, 1.1},
		{"Matte Surface", func(m *renderer.Model) { m.SetMatte(0.9, 0.1, 0.9) }, 0.0, 0.9, 1.0},
	}

	for i := 0; i < len(mld.models) && i < len(materialTypes); i++ {
		renderer.ApplyAdvancedRenderingConfig(mld.models[i], config)

		material := materialTypes[i]
		material.setter(mld.models[i])
		mld.models[i].SetMaterialPBR(material.metallic, material.roughness)
		mld.models[i].SetExposure(material.exposure)

		fmt.Printf("   Sphere %d: %s (M=%.1f, R=%.1f, E=%.1f)\n", i+1, material.name, material.metallic, material.roughness, material.exposure)
	}
}

// Volumetric lighting demo
func (mld *ModernLightingDemo) setVolumetricDemo() {
	// Keep the same directional light - volumetric effects work with directional lights too
	mld.engine.Light.Intensity = 3.0
	mld.engine.Light.AmbientStrength = 0.05

	// Enable volumetric lighting
	config := renderer.DefaultAdvancedRenderingConfig()
	config.EnableVolumetricLighting = true
	config.VolumetricIntensity = 0.8
	config.VolumetricSteps = 16
	config.VolumetricScattering = 0.2

	colors := [][3]float32{
		{0.8, 0.2, 0.2}, {0.2, 0.8, 0.2}, {0.2, 0.2, 0.8},
		{0.8, 0.8, 0.2}, {0.8, 0.2, 0.8},
	}

	for i := 0; i < len(mld.models) && i < len(colors); i++ {
		renderer.ApplyAdvancedRenderingConfig(mld.models[i], config)

		color := colors[i]
		mld.models[i].SetGlossy(color[0], color[1], color[2])
		mld.models[i].SetMaterialPBR(0.0, 0.2) // Non-metallic, glossy
		mld.models[i].SetExposure(1.2)
		fmt.Printf("   Sphere %d: Volumetric lighting enabled (Directional)\n", i+1)
	}
}

// Clearcoat demo (automotive paint)
func (mld *ModernLightingDemo) setClearcoatDemo() {
	// Keep the same directional light - perfect for automotive showcase
	mld.engine.Light.Intensity = 2.2
	mld.engine.Light.AmbientStrength = 0.12

	// Enable clearcoat for all models
	config := renderer.DefaultAdvancedRenderingConfig()
	config.EnableClearcoat = true
	config.ClearcoatIntensity = 0.8
	config.ClearcoatRoughness = 0.1
	config.EnableEnergyConservation = true

	clearcoatColors := [][3]float32{
		{0.8, 0.1, 0.1}, // Red car paint
		{0.1, 0.8, 0.1}, // Green car paint
		{0.1, 0.1, 0.8}, // Blue car paint
		{0.9, 0.9, 0.9}, // White car paint
		{0.1, 0.1, 0.1}, // Black car paint
	}

	for i := 0; i < len(mld.models) && i < len(clearcoatColors); i++ {
		renderer.ApplyAdvancedRenderingConfig(mld.models[i], config)

		color := clearcoatColors[i]
		mld.models[i].SetPolishedMetal(color[0], color[1], color[2])
		mld.models[i].SetMaterialPBR(1.0, 0.1) // Metallic base with clearcoat
		mld.models[i].SetExposure(1.3)         // Higher exposure to show clearcoat

		fmt.Printf("   Sphere %d: Clearcoat automotive paint (M=1.0, R=0.1)\n", i+1)
	}
}

// Transmission demo (glass materials)
func (mld *ModernLightingDemo) setTransmissionDemo() {
	// Enable transmission for glass-like materials
	config := renderer.DefaultAdvancedRenderingConfig()
	config.EnableTransmission = true
	config.TransmissionFactor = 0.7
	config.EnableEnergyConservation = true
	config.EnableBloom = true // Glass can create bright highlights

	transmissionFactors := []float32{0.9, 0.7, 0.5, 0.3, 0.1}

	for i := 0; i < len(mld.models) && i < len(transmissionFactors); i++ {
		// Customize transmission factor per model
		modelConfig := config
		modelConfig.TransmissionFactor = transmissionFactors[i]

		renderer.ApplyAdvancedRenderingConfig(mld.models[i], modelConfig)

		// Set proper glass materials with transparency and tint
		alpha := 0.2 + transmissionFactors[i]*0.3 // More transparent

		// Different glass tints for variety
		glassColors := [][3]float32{
			{0.9, 0.95, 1.0},  // Clear blue tint
			{0.95, 1.0, 0.9},  // Clear green tint
			{1.0, 0.95, 0.9},  // Clear amber tint
			{0.95, 0.95, 1.0}, // Clear purple tint
			{1.0, 0.9, 0.9},   // Clear pink tint
		}

		if i < len(glassColors) {
			color := glassColors[i]
			mld.models[i].SetGlass(color[0], color[1], color[2], alpha)
		} else {
			mld.models[i].SetGlass(0.95, 0.95, 0.98, alpha)
		}

		fmt.Printf("   Sphere %d: Glass transmission %.1f, alpha %.1f (M=0.0, R=0.05)\n", i+1, transmissionFactors[i], alpha)
	}

	// Keep the same directional light - transmission works with directional lights
	mld.engine.Light.Intensity = 4.0 // Brighter for transmission effect
	mld.engine.Light.AmbientStrength = 0.08
}

// Add skybox for better environment lighting
func (mld *ModernLightingDemo) addSkybox() {
	// Set skybox color and create solid color skybox
	renderer.SetSkyboxColor(0.5, 0.7, 1.0)  // Light blue sky
	err := mld.engine.SetSkybox("dark_sky") // Use special path for solid color
	if err != nil {
		fmt.Printf("⚠️ SKYBOX: Failed to set skybox: %v\n", err)
		return
	}
	fmt.Println("🌌 SKYBOX: Added gradient sky for environment lighting")
}
