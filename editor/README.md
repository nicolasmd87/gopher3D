# Gopher3D Editor

The integrated development environment for Gopher3D engine. Create scenes, add objects, configure materials, and export standalone games.

## Table of Contents

- [Building](#building)
- [Running](#running)
- [Interface Overview](#interface-overview)
- [Panels](#panels)
- [Adding Objects](#adding-objects)
- [Scene Management](#scene-management)
- [Exporting Games](#exporting-games)
- [Controls](#controls)
- [Keyboard Shortcuts](#keyboard-shortcuts)

## Building

### Using Build Script (Recommended)

From the project root:

```powershell
# Build the editor
.\build.ps1 build

# Build and run
.\build.ps1 run

# Full CI check (format, test, build)
.\build.ps1 ci
```

### Manual Build

```bash
go build -o bin/editor.exe ./editor/cmd
```

## Running

```powershell
# From project root after building
cd bin
.\editor.exe

# Or use the build script
.\build.ps1 run
```

## Interface Overview

The editor uses a dockable panel layout with the following default arrangement:

```
┌─────────────────────────────────────────────────────────────┐
│  Menu Bar (File | Add | View | Experimental)                │
├──────────┬──────────────────────────────────┬───────────────┤
│          │                                  │               │
│ Hierarchy│         Scene Viewport           │   Inspector   │
│          │                                  │               │
│          │                                  ├───────────────┤
│          │                                  │ Scene Settings│
├──────────┴──────────────────────────────────┴───────────────┤
│  Project Browser              │           Console           │
└─────────────────────────────────────────────────────────────┘
```

## Panels

### Hierarchy

Displays all objects in the current scene organized by type:

- **GameObjects** - Models, voxel terrain, water surfaces
- **Lights** - Directional and point lights
- **Cameras** - Scene cameras

**Actions:**
- Click to select an object
- Double-click to focus camera on object
- Right-click for context menu (delete)

### Inspector

Shows properties for the selected object:

- **Transform** - Position, rotation, scale
- **Material** - Diffuse color, metallic, roughness, alpha
- **Components** - Attached scripts and behaviors
- **Light Properties** - Color, intensity, attenuation (for lights)

### Scene Settings

Configure scene-wide settings:

- **Skybox** - Solid color or image background
- **Window** - Resolution and display options

### Advanced Rendering

Fine-tune rendering quality:

- **Basic Rendering** - Wireframe, frustum culling, face culling, depth testing
- **PBR Materials** - Clearcoat, sheen, transmission
- **Lighting Effects** - Shadows, ambient occlusion, global illumination
- **Post Processing** - Bloom, FXAA anti-aliasing
- **Performance** - Quality presets (Performance, Balanced, High Quality, Voxel)

### Project Browser

Navigate project files:

- Browse models, textures, and scripts
- Drag files into the scene (planned)
- Create new scripts

### Console

View engine output:

- Info, warning, and error messages
- Scene load/save status
- Export progress

## Adding Objects

### Menu: Add

| Option | Description |
|--------|-------------|
| **Mesh** | Import OBJ model files |
| **Light** | Add directional or point lights |
| **Water** | Create animated water surface with Gerstner waves |
| **Voxel Terrain** | Generate procedural terrain with customizable biomes |
| **Camera** | Add additional scene cameras |

### Voxel Terrain Options

- **Biomes**: Plains, Mountains, Desert, Islands, Caves
- **Parameters**: Scale, amplitude, seed, threshold, octaves
- **Colors**: Customizable grass, dirt, stone, sand, wood, leaves
- **Size**: Chunk size and world dimensions

### Water Surface Options

- **Appearance**: Color, transparency, wave speed
- **Effects**: Foam, caustics, specular reflections
- **Mesh**: Ocean size, base amplitude (requires regeneration)

## Scene Management

### File Menu

| Action | Description |
|--------|-------------|
| **New Scene** | Clear current scene and start fresh |
| **Save Scene** | Save to JSON format |
| **Load Scene** | Open existing scene file |
| **Export Game** | Build standalone executable |

### Scene Format

Scenes are saved as JSON files containing:
- Models with transforms and materials
- Lights with all properties
- Water and voxel configurations
- Camera positions
- Skybox settings
- Rendering configuration

## Exporting Games

Export your scene as a standalone executable:

1. **File → Export Game** or use the Export dialog
2. Enter game name
3. Select output directory
4. Choose scenes to include
5. Click **Export**

### Export Output

```
output_directory/
├── windows/
│   ├── MyGame.exe
│   └── assets/
│       ├── scene.json
│       └── meshes/
```

### What Gets Exported

- Scene data (models, lights, water, voxels)
- Serialized mesh data (.gmesh format)
- Textures and materials
- Script components
- Water shader and simulation
- Full runtime with game loop

## Controls

### Camera Navigation

| Input | Action |
|-------|--------|
| **W/A/S/D** | Move forward/left/back/right |
| **Right Mouse + Drag** | Look around |
| **Shift** | Speed boost |
| **Mouse Wheel** | Zoom (planned) |

### Object Manipulation

| Input | Action |
|-------|--------|
| **Left Click** | Select object |
| **Double Click** | Focus camera on object |
| **Gizmo Drag** | Move object along axis |

### Gizmos

The 3D gizmo appears when an object is selected:
- **Red arrow (X)** - Move along X axis
- **Green arrow (Y)** - Move along Y axis  
- **Blue arrow (Z)** - Move along Z axis

The orientation gizmo in the top-right corner shows current camera orientation.

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| **Ctrl+N** | New scene |
| **Ctrl+S** | Save scene |
| **Ctrl+O** | Load scene |
| **Delete** | Remove selected object |

## Tips

- Save frequently - there's no auto-save yet
- Water and voxel terrain are GameObjects with dedicated components
