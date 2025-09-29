
# Gopher3D - Open-Source Render Engine

Gopher3D is an open-source render engine developed in Go. The engine currently supports **OpenGL rendering** with advanced shader capabilities, including custom per-model shader assignment and instancing for efficient rendering of large numbers of objects. While the Vulkan implementation has been started, it is **not yet functional**, and will be developed further in future releases.

This engine is designed for flexibility, ease of use, and experimentation in creating 3D applications. Examples include advanced water simulation, particle systems, and physics demonstrations to showcase the engine's capabilities.

The engine is still in early stages but includes advanced rendering features.

## Features

- **Advanced Shader System**: Per-model shader assignment with automatic shader switching, supporting custom vertex and fragment shaders
- **Water Simulation**: GPU-based Gerstner wave simulation with realistic lighting, caustics, and Fresnel effects
- **OpenGL Rendering with Instancing**: Efficient rendering of multiple objects through instancing, significantly improving performance for scenes with many repeated elements
- **Voxel Engine**: High-performance voxel rendering system with GPU instancing, supporting millions of voxels with optimized frustum culling
- **Advanced Lighting System**: Directional, point, and ambient lighting with configurable temperature, intensity, and attenuation
- **PBR Materials**: Physically-based rendering materials with metallic/roughness workflow and advanced specular calculations
- **Custom Uniforms**: Dynamic uniform passing for shader customization (floats, integers, vectors, arrays)
- **Camera Controls**: Integrated camera controls with support for mouse and keyboard input
- **Procedural Geometry**: Runtime generation of water surfaces, voxel worlds, and other procedural meshes
- **Performance Optimizations**: Frustum culling, face culling, and parallel world generation using worker pools
- **Examples**: Various physics and rendering examples organized in separate directories

## Not Implemented

- **Proper physics engine**
- **Collision detection**
- **Multiple materials system**
- **Scene management**
- **UI framework**
- **Vulkan renderer (In Progress)**
- **Audio system**

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
git clone https://github.com/your-username/Gopher3D.git
cd Gopher3D
```

### Running Examples

Examples are now organized in separate directories. Each example defines a particular scene's behavior and demonstrates specific engine features.

To run an example:
```bash
# Water simulation
cd examples/General/water
go run water.go

# Voxel rendering (blocky terrain)
cd examples/Voxel/Cube
go run voxel_world.go

# Voxel rendering (smooth terrain)
cd examples/Voxel/SurfaceNets
go run smooth_terrain.go

# Black hole simulation
cd examples/Phyisics/black_hole
go run black_hole.go

# Sand simulation
cd examples/Phyisics/sand
go run sand.go

# Basic rendering
cd examples/Basic
go run basic_example.go
```

## Examples

### Water Simulation (Advanced Shader Demo)

This example demonstrates advanced water rendering with GPU-based wave simulation, realistic lighting, and custom shader implementation.

- **Directory**: `examples/General/water/`
- **File**: `water.go`
- **Features**:
  - GPU-based Gerstner wave simulation
  - Advanced water shader with Fresnel effects
  - Caustics and subsurface scattering
  - Procedural water surface generation
  - Dynamic lighting and specular reflections
  - Per-model custom shader assignment
  - Configurable wave parameters and ocean size

### Black Hole Simulation (Instanced)

This example demonstrates a particle simulation where particles orbit around a black hole using instanced rendering for performance.

- **Directory**: `examples/Phyisics/black_hole/`
- **File**: `black_hole.go`
- **Features**: 
  - Particle simulation using instanced rendering
  - Verlet integration for particle movement
  - Event horizon particle removal
  - Physics-based orbital mechanics

### Sand Simulation (Interactive)

This example demonstrates an interactive sand particle simulation with mouse interaction.

- **Directory**: `examples/Phyisics/sand/`
- **File**: `sand.go`
- **Features**:
  - Interactive particle manipulation with mouse
  - Sand gathering and releasing mechanics
  - Real-time particle dynamics
  - Instanced rendering for performance

### Particle System Example

This example demonstrates basic particle behaviors including forces like gravity.

- **Directory**: `examples/Phyisics/particles/`
- **File**: `particles.go`
- **Features**:
  - Basic particle simulation with movement
  - Force application (gravity, wind)
  - Instanced rendering demonstration

### Basic Example

This is a simple example showcasing basic 3D rendering, camera controls, and light interaction.

- **Directory**: `examples/Basic/`
- **File**: `basic_example.go`
- **Features**:
  - Basic scene setup with simple objects
  - Default shader usage
  - Camera and lighting basics

### Voxel Rendering Examples

Advanced voxel rendering system demonstrating high-performance instanced rendering and procedural world generation.

- **Directory**: `examples/Voxel/`
- **Files**: 
  - `Cube/voxel_world.go` - Blocky terrain with GPU instancing
  - `SurfaceNets/smooth_terrain.go` - Smooth terrain using Surface Nets algorithm
- **Features**:
  - GPU-instanced voxel rendering (23M+ voxels, single draw call)
  - Procedural terrain generation using improved Perlin noise
  - Optimized frustum culling for large worlds
  - Parallel world generation using worker pools
  - Support for both blocky and smooth voxel rendering modes
  - Configurable chunk sizes and world dimensions

## Architecture

### Shader System

The engine features a flexible shader system allowing:
- **Default Shader**: Basic vertex/fragment shader for standard rendering
- **Custom Shaders**: Per-model shader assignment with automatic compilation
- **Dynamic Uniforms**: Runtime uniform passing for shader customization
- **Automatic Switching**: Efficient shader program switching during rendering

### Renderer Structure

```
internal/renderer/
├── opengl_renderer.go    # Main OpenGL rendering implementation
├── shaders.go           # Shader management and GLSL source code
├── camera.go            # Camera implementation with controls
├── model.go             # 3D model representation and management
├── advanced_rendering_config.go # Advanced rendering configuration system
├── improved_perlin.go   # GPU Gems Chapter 5 improved Perlin noise
└── vulkan_*.go          # Vulkan implementation (in progress)
```

### Voxel Engine Structure

```
internal/loader/
├── voxel_core.go        # Core voxel engine with instancing and Surface Nets
├── voxel_generator.go   # Procedural voxel world generation
└── loader.go            # Model and texture loading utilities
```

## Planned Features and Work in Progress

### Enhanced Shader System
- Shader hot-reloading for development
- Shader parameter UI for real-time tweaking
- Additional built-in shaders (PBR, toon, etc.)

### Advanced Rendering Features
- **Improved Perlin Noise**: GPU Gems Chapter 5 implementation with quintic interpolation and multi-octave turbulence
- **Advanced Rendering Configuration**: Configurable shadows, ambient occlusion, filtering quality, and mesh smoothing
- **PBR Materials**: Metallic/roughness workflow with GGX specular distribution and Schlick Fresnel

### Voxel Engine Features
- **Surface Nets Algorithm**: Smooth terrain generation from Signed Distance Fields
- **Parallel World Generation**: Worker pool-based parallelization for large worlds
- **Optimized Culling**: Frustum culling for instanced models with proper bounding sphere calculations

### Physics and Particle Systems
A fully integrated physics engine and comprehensive particle system are in development. Current particle simulations serve as examples for future engine-level implementations.

### Vulkan Renderer
The Vulkan renderer is in progress but currently **not functional**. The OpenGL renderer is fully functional, and Vulkan development continues.

### Advanced Water Features
- FFT-based ocean simulation
- Underwater rendering
- Water-object interaction
- Multiple water types (ocean, river, lake)

## Contributing

As an open-source project, contributions are welcome! Whether you're fixing bugs, improving performance, adding new features, or enhancing shaders, feel free to submit pull requests.

To contribute:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Commit your changes (`git commit -m 'Add my feature'`)
4. Push to the branch (`git push origin feature/my-feature`)
5. Create a Pull Request

### Development Guidelines
- Follow Go coding standards
- Test examples before submitting
- Document new shader uniforms and features
- Keep examples organized in appropriate directories

## Known Issues

- **Vulkan Implementation**: Currently non-functional, OpenGL is the primary renderer
- **Shader Compilation**: Some edge cases in dynamic shader compilation need refinement
- **Performance**: Large water surfaces may impact performance on lower-end hardware

## Technical Details

### Dependencies
- **OpenGL 4.1+**: Core rendering backend
- **GLFW**: Window management and input handling
- **go-gl**: Go OpenGL bindings
- **mathgl**: Mathematical operations for 3D graphics

### Performance Notes
- Water simulation is GPU-optimized using vertex shaders
- Instanced rendering significantly improves particle and voxel performance
- Voxel rendering supports 23M+ instances with single draw call
- Parallel world generation using worker pools for large scenes
- Configurable water resolution and voxel chunk sizes for performance tuning
- Optimized frustum culling for large instanced scenes

## Images

![Water Simulation](https://github.com/user-attachments/assets/water-sim-screenshot.png)
*Advanced water simulation with realistic waves and lighting*

![Black hole instanciated](https://github.com/user-attachments/assets/0f9467b4-e4b5-4ebf-ac66-ed3e8bc87efc)
*Black hole particle simulation with orbital mechanics*

![Mars](https://github.com/nicolasmd87/Gopher3D/assets/8224408/09d2a39b-c1cb-4548-87fb-1a877df24453)
*Basic planetary rendering example*







