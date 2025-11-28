# Gopher3D

A 3D rendering engine and editor written in Go, built for learning and experimentation with real-time graphics.

## Overview

Gopher3D is an OpenGL-based rendering engine with an integrated editor. It provides basic 3D rendering capabilities, a material system, and tools for scene composition. The project is intended for educational purposes and small-scale prototyping.

**Status**: Experimental. Expect bugs and incomplete features.

## Features

### Rendering
- OpenGL 4.1 rendering pipeline
- PBR materials (metallic/roughness workflow)
- Directional and point lights
- Basic shadow support
- Instanced rendering for repeated geometry
- Frustum culling
- Post-processing (FXAA, Bloom)

### Editor
- Scene hierarchy panel
- Object inspector with transform and material editing
- File browser for asset management
- Console for logging
- Gizmos for object manipulation
- Scene save/load (JSON format)
- Style customization
- Game export functionality

### Specialized Systems
- **Voxel terrain**: Procedural generation with Perlin noise, chunk-based rendering, customizable colors
- **Water simulation**: Gerstner wave vertex shader with reflections
- **Skybox**: Image-based or solid color backgrounds

### Scripting
- Go-based component system
- Start/Update/FixedUpdate lifecycle
- Scripts attachable to objects via editor

## Requirements

- Go 1.21+
- OpenGL 4.1 compatible GPU
- GCC (for CGO)
- GLFW dependencies (platform-specific)

## Installation

### From Source

```bash
git clone https://github.com/nicholasq/Gopher3D.git
cd Gopher3D
go mod tidy
cd editor
go build -o gopher3d-editor .
```

### From Releases

Download the latest release from the [Releases page](https://github.com/nicholasq/Gopher3D/releases).

#### Windows SmartScreen Notice

Windows may show a SmartScreen warning for unsigned applications. This is normal for open-source software without a paid code signing certificate (~$200-500/year).

**To run the editor:**
1. Right-click the downloaded `.exe`
2. Select "Properties"
3. Check "Unblock" at the bottom
4. Click "Apply" then "OK"

Alternatively, when SmartScreen appears:
1. Click "More info"
2. Click "Run anyway"

> **Note**: This is standard for all unsigned applications. Major engines like Unity and Unreal are signed because they have funding for certificates. Open-source projects like Godot faced the same issue before receiving donations.

#### Linux

```bash
tar -xzf gopher3d-editor-linux-amd64.tar.gz
chmod +x gopher3d-editor-linux-amd64
./gopher3d-editor-linux-amd64
```

#### macOS

Extract and run. You may need to allow the app in System Preferences → Security & Privacy.

## Examples using the lib direclty without the editor (old implementation and tests)

```bash
# Lighting demo
cd examples/General/lights
go run lighting_demo.go

# Voxel world
cd examples/Voxel/Cube
go run voxel_world.go

# Water simulation
cd examples/General/water
go run water.go
```

## Development

### Build Script (Windows PowerShell)

```powershell
.\build.ps1 build    # Build editor
.\build.ps1 test     # Run tests
.\build.ps1 vet      # Run go vet
.\build.ps1 ci       # Full CI (format, tidy, vet, test, build)
.\build.ps1 help     # Show all commands
```

### Project Structure

```
Gopher3D/
├── editor/           # Editor application
├── internal/
│   ├── engine/       # Core engine loop
│   ├── renderer/     # OpenGL rendering
│   ├── loader/       # Model and voxel loading
│   ├── behaviour/    # Component system
│   └── logger/       # Logging
├── examples/         # Demo programs
├── scripts/          # User scripts (Rotate, Orbit, Bounce)
├── .github/workflows # CI/CD pipelines
└── resources/        # Assets
```

## Controls

### Camera
- **WASD**: Move
- **Mouse**: Look around (hold right click)
- **Shift**: Speed boost

### Editor
- **Left click**: Select object
- **Double click**: Focus camera on object

## Limitations

- No physics engine
- No collision detection
- No audio
- OBJ model format only
- Vulkan renderer incomplete

## Contributing

Contributions welcome. Please:
1. Fork the repository
2. Create a feature branch
3. Run `.\build.ps1 ci` to verify tests pass
4. Submit a pull request

## License

MIT License

## Acknowledgments

Built with:
- [go-gl](https://github.com/go-gl) - OpenGL bindings
- [GLFW](https://www.glfw.org/) - Window management
- [imgui-go](https://github.com/inkyblackness/imgui-go) - Editor UI

## Images

![Black hole](https://github.com/user-attachments/assets/0f9467b4-e4b5-4ebf-ac66-ed3e8bc87efc)
*Black hole particle simulation with orbital mechanics*

