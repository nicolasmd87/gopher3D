# Gopher3D

A 3D rendering engine and editor written in Go, built for learning and experimentation with real-time graphics.

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Requirements](#requirements)
- [Quick Start](#quick-start)
- [Building](#building)
- [Editor Guide](#editor-guide)
- [Architecture](#architecture)
- [Controls](#controls)
- [Limitations](#limitations)
- [Contributing](#contributing)
- [License](#license)

## Overview

Gopher3D is an OpenGL-based rendering engine with an integrated editor. It provides 3D rendering capabilities, a PBR material system, and tools for scene composition. The project is intended for educational purposes and prototyping.

**Status**: Experimental. Active development.

## Features

### Rendering

- OpenGL 4.1 rendering pipeline
- PBR materials (metallic/roughness workflow)
- Directional and point lights with attenuation
- Instanced rendering for voxels and repeated geometry
- Frustum culling
- Post-processing (MSAA, FXAA, Bloom)
- Configurable render quality presets

### Editor

- Dockable panel layout (Hierarchy, Inspector, Console, Project Browser)
- Transform gizmos for object manipulation
- Orientation gizmo showing camera direction
- Material editing with real-time preview
- Scene save/load (JSON format)
- Standalone game export
- Style customization

### Specialized Systems


### Component System

- Go-based scripting with Start/Update/FixedUpdate lifecycle
- Built-in components: VoxelTerrainComponent, WaterComponent, ScriptComponent
- Scripts attachable via editor UI
- Hot-reload support (requires editor restart)

## Requirements

- Go 1.21+
- OpenGL 4.1 compatible GPU
- GCC (for CGO)

### Platform-Specific Dependencies

**Windows**: MinGW-w64 or TDM-GCC

**Linux**:
```bash
sudo apt install libgl1-mesa-dev xorg-dev
```

**macOS**: Xcode Command Line Tools

## Quick Start

```powershell
# Clone repository
git clone https://github.com/nicholasq/Gopher3D.git
cd Gopher3D

# Install dependencies
go mod tidy

# Build and run editor
.\build.ps1 run
```

## Building

### Build Script (Recommended)

The project includes a PowerShell build script with common tasks:

```powershell
.\build.ps1 build           # Build editor to bin/editor.exe
.\build.ps1 run             # Build and launch editor
.\build.ps1 test            # Run all tests
.\build.ps1 vet             # Run go vet
.\build.ps1 lint            # Run staticcheck + vet
.\build.ps1 fmt             # Format code
.\build.ps1 tidy            # Tidy go.mod
.\build.ps1 ci              # Full CI pipeline
.\build.ps1 clean           # Remove build artifacts
.\build.ps1 help            # Show all commands
```

### Manual Build

```bash
# Build editor
go build -o bin/editor.exe ./editor/cmd

# Run tests
go test ./internal/...
```

### From Releases

Download pre-built binaries from the [Releases page](https://github.com/nicholasmd87/Gopher3D/releases).

<details>
<summary>Windows SmartScreen Notice</summary>

Windows may show a SmartScreen warning for unsigned applications. To run:
1. Right-click the `.exe` → Properties → Check "Unblock" → Apply
2. Or click "More info" → "Run anyway" when SmartScreen appears

This is standard for unsigned open-source software.
</details>

## Editor Guide

See [editor/README.md](editor/README.md) for detailed documentation on:

- Panel layout and functionality
- Adding objects (meshes, lights, water, voxels)
- Scene management
- Exporting standalone games
- Keyboard shortcuts

### Basic Workflow

1. **Create scene**: File → New Scene
2. **Add objects**: Add menu → Mesh/Light/Water/Voxel Terrain
3. **Configure**: Select object, edit in Inspector
4. **Save**: File → Save Scene
5. **Export**: File → Export Game

## Architecture

```
Gopher3D/
├── editor/                    # Editor application
│   ├── cmd/                   # Entry point
│   ├── internal/              # Editor-specific packages
│   ├── platforms/             # GLFW platform layer
│   └── renderers/             # ImGui OpenGL renderer
├── internal/
│   ├── engine/                # Core engine (Gopher struct, game loop)
│   ├── renderer/              # OpenGL renderer, shaders, materials
│   ├── loader/                # Model loading (OBJ), voxel system
│   ├── behaviour/             # Component system, GameObjects
│   ├── water/                 # Water simulation (shared)
│   └── logger/                # Structured logging (zap)
├── scripts/                   # Example user scripts
├── resources/                 # Default assets
└── examples/                  # Standalone demos
```


## Examples

Run standalone demos without the editor:

```bash
# Lighting demo
cd examples/General/lights && go run lighting_demo.go

# Voxel world
cd examples/Voxel/Cube && go run voxel_world.go

# Water simulation
cd examples/General/water && go run water.go
```

## Limitations

- No physics engine
- No collision detection
- No audio system
- OBJ model format only
- Single-threaded rendering
- Vulkan renderer incomplete

## Contributing

1. Fork the repository
2. Create a feature branch
3. Run `.\build.ps1 ci` to verify tests pass
4. Submit a pull request

Please follow existing code style and add tests for new functionality.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

Built with:
- [go-gl](https://github.com/go-gl) - OpenGL bindings
- [GLFW](https://www.glfw.org/) - Window management
- [imgui-go](https://github.com/inkyblackness/imgui-go) - Editor UI
- [zap](https://github.com/uber-go/zap) - Structured logging

## Images

![Black hole](https://github.com/user-attachments/assets/0f9467b4-e4b5-4ebf-ac66-ed3e8bc87efc)
*Black hole particle simulation with orbital mechanics*
