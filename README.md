
# Gopher3D - 3D Rendering Engine in Go

Gopher3D is an experimental 3D rendering engine written in Go, focusing on modern lighting techniques and shader-based rendering. The engine provides OpenGL-based rendering with physically-based materials, advanced lighting models, and efficient instanced rendering for educational and prototyping purposes.

**Current Status**: Active development with functional OpenGL renderer. Vulkan implementation is in early stages and not yet operational.

This project serves as a learning platform for 3D graphics programming concepts and modern rendering techniques, with practical examples demonstrating various rendering and simulation approaches.

## Core Features

### Rendering Pipeline
- **OpenGL 4.1+ Backend**: Modern OpenGL rendering with shader-based pipeline
- **Custom Shader System**: Per-model shader assignment with dynamic uniform passing
- **Instanced Rendering**: Efficient rendering of repeated geometry (particles, voxels)
- **Basic Frustum Culling**: Performance optimization for large scenes

### Lighting & Materials
- **Modern PBR Implementation**: Physically-based rendering with metallic/roughness workflow
- **Advanced BRDF Models**: GGX distribution, Fresnel reflectance, energy conservation
- **Directional & Point Lights**: Configurable lighting with temperature and attenuation
- **Hemisphere Lighting**: Improved lighting model preventing harsh cutoffs and "two halves" artifacts
- **Material System**: Complete PBR material properties with automatic fallbacks for incomplete materials
- **Specialized Materials**: Glass, metal, plastic presets with realistic optical properties
- **HDR Pipeline**: ACES tone mapping with gamma correction and exposure control

### Specialized Systems
- **Voxel Rendering**: GPU-instanced voxel worlds with procedural terrain generation
- **Water Simulation**: Gerstner wave-based water surfaces with realistic shading
- **Particle Systems**: Basic particle simulation with physics integration
- **Procedural Generation**: Perlin noise-based terrain and surface generation

### Development Tools
- **Multiple Examples**: Organized demonstrations of engine capabilities
- **Configurable Rendering**: Runtime adjustment of quality and performance settings
- **Cross-platform Support**: Windows, macOS, and Linux compatibility

## Limitations & Known Issues

### Not Implemented
- Comprehensive physics engine (basic particle physics only)
- Collision detection system
- Scene graph management
- Asset pipeline and model loading
- Audio system integration
- User interface framework

### Current Limitations
- **Vulkan Renderer**: In early development, not functional
- **Performance**: Not optimized for production use
- **Platform Testing**: Primary development on Windows
- **Documentation**: Limited API documentation
- **Stability**: Experimental codebase, expect breaking changes

## Getting Started

### Prerequisites

To run the Gopher3D engine, ensure you have the following dependencies:

- **Go**: The engine is written in Go, so Go must be installed on your machine
- **GCC**: Required for CGO compilation
- **OpenGL**: The engine currently supports OpenGL for rendering
- **GLFW**: Required for managing windowing and input across different platforms

Install the necessary Go modules with:
```bash
go mod tidy
```

### Cloning the Repository

To start using the engine or contribute to it, clone the repository:
```bash
git clone https://github.com/nicolasmd87/Gopher3D.git
cd Gopher3D
```

### Running Examples

Examples demonstrate specific engine features and rendering techniques:

```bash
# Advanced lighting demo (recommended starting point)
cd examples/Basic/Lights
go run lighting_demo.go

# Voxel world with modern lighting
cd examples/Voxel/Cube
go run voxel_world.go

# Water simulation
cd examples/General/water
go run water.go

# Particle simulations
cd examples/General/black_hole
go run black_hole.go

cd examples/General/sand
go run sand.go
```

**Note**: Some examples may require specific OpenGL features or perform better on dedicated graphics hardware.

## Example Descriptions

### Advanced Lighting Demo
**Path**: `examples/Basic/Lights/lighting_demo.go`

Engine's lighting capabilities:
- Physically-based rendering (PBR) materials
- Multiple material types (metals, plastics, glass)
- Advanced BRDF models (clearcoat, sheen, transmission)
- HDR tone mapping and bloom effects
- Screen-space ambient occlusion (SSAO)
- Global illumination approximation
- Inter-object reflections

### Voxel World Rendering
**Path**: `examples/Voxel/Cube/voxel_world.go`

Large-scale voxel rendering with modern lighting:
- GPU-instanced voxel rendering
- Procedural terrain generation using Perlin noise
- Advanced lighting integration for voxel materials
- Performance optimization for large worlds

### Water Simulation
**Path**: `examples/General/water/water.go`

GPU-based water surface simulation:
- Gerstner wave implementation
- Custom water shader with Fresnel effects
- Dynamic wave parameters

### Particle Simulations
**Paths**: `examples/General/black_hole/`, `examples/General/sand/`

Basic particle physics demonstrations:
- **Black Hole**: Orbital mechanics with Verlet integration
- **Sand**: Interactive particle manipulation with mouse input
- Instanced rendering for performance
- Simple physics integration

## Technical Architecture

### Project Structure
```
Gopher3D/
├── internal/
│   ├── engine/          # Core engine and main loop
│   ├── renderer/        # OpenGL rendering pipeline
│   ├── loader/          # Voxel engine and model loading
│   ├── behaviour/       # Component system
│   └── logger/          # Logging utilities
├── examples/            # Demonstration programs
│   ├── Voxel/Cube/      # Voxel world rendering
│   └── General/         # Water and particle simulations
└── resources/           # Textures and models
```

### Key Components

**Rendering Pipeline** (`internal/renderer/`)
- `opengl_renderer.go`: Core OpenGL implementation
- `shaders.go`: Shader management and GLSL programs
- `camera.go`: First-person camera with controls
- `model.go`: 3D model and material representation

**Voxel System** (`internal/loader/`)
- `voxel_core.go`: GPU-instanced voxel rendering
- Procedural terrain generation with Perlin noise
- Chunk-based world management

**Engine Core** (`internal/engine/`)
- Main rendering loop and window management
- Input handling and behavior system integration

## Development Roadmap

### Near-term Goals
- **Vulkan Renderer**: Complete the Vulkan implementation for modern GPU features
- **Asset Pipeline**: Improve model loading and texture management
- **Documentation**: Comprehensive API documentation and tutorials
- **Testing**: Automated testing for rendering correctness

### Future Considerations
- **Physics Integration**: Proper physics engine integration (Bullet, etc.)
- **Scene Management**: Hierarchical scene graph implementation
- **Advanced Shaders**: Deferred rendering, shadow mapping, post-processing
- **Platform Support**: Mobile and web platform exploration

### Research Areas
- **Real-time Ray Tracing**: Explore RT capabilities for reflections and GI
- **Compute Shaders**: GPU-based particle systems and simulations

## Contributing

This project welcomes contributions from developers interested in 3D graphics programming and Go development.

### How to Contribute
1. Fork the repository
2. Create a feature branch (`git checkout -b feature/improvement-name`)
3. Make your changes with appropriate tests
4. Commit with descriptive messages
5. Submit a pull request

### Contribution Areas
- **Bug fixes**: Rendering issues, compilation errors
- **Performance improvements**: Optimization of rendering pipeline
- **New examples**: Demonstrations of graphics techniques
- **Documentation**: Code comments, tutorials, API docs
- **Platform support**: Testing and fixes for different operating systems

### Development Guidelines
- Follow standard Go conventions and formatting
- Test examples on your target platform before submitting
- Include comments for complex graphics algorithms
- Keep examples focused and well-documented

## Technical Requirements

### System Dependencies
- **Go 1.19+**: Programming language runtime
- **OpenGL 4.1+**: Graphics API (most modern GPUs support this)
- **GCC/Clang**: C compiler for CGO compilation
- **Git**: Version control for dependency management

### Go Dependencies (managed by `go mod`)
- `github.com/go-gl/gl/v4.1-core/gl`: OpenGL bindings
- `github.com/go-gl/glfw/v3.3/glfw`: Window and input management
- `github.com/go-gl/mathgl/mgl32`: 3D mathematics library

### Performance Characteristics
- **Target**: Educational and experimental use, not production-ready
- **Voxel Rendering**: Handles large worlds through GPU instancing
- **Water Simulation**: GPU-based vertex shader implementation
- **Particle Systems**: Basic CPU-based physics with GPU rendering
- **Platform**: Primarily tested on Windows, should work on macOS/Linux

### Hardware Recommendations
- **GPU**: Dedicated graphics card recommended for complex examples
- **RAM**: 8GB+ for large voxel worlds
- **CPU**: Multi-core beneficial for procedural generation

## Images

![Water Simulation](https://github.com/user-attachments/assets/water-sim-screenshot.png)
*Advanced water simulation with realistic waves and lighting*

![Black hole instanciated](https://github.com/user-attachments/assets/0f9467b4-e4b5-4ebf-ac66-ed3e8bc87efc)
*Black hole particle simulation with orbital mechanics*

![Mars](https://github.com/nicolasmd87/Gopher3D/assets/8224408/09d2a39b-c1cb-4548-87fb-1a877df24453)
*Basic planetary rendering example*





 

