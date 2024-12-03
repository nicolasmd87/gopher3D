
# Gopher3D - Open-Source Render Engine

Gopher3D is an open-source render engine developed in Go. The engine currently supports **OpenGL rendering** with instancing capabilities, allowing efficient rendering of large numbers of objects. While the Vulkan implementation has been started, it is **not yet functional**, and will be developed further in future releases.

This engine is designed for flexibility, ease of use, and experimentation in creating 3D applications. Examples such as particle simulations are included to demonstrate the engine’s features, although a full physics and particle system implementation is a work in progress.

The engine is still in very early stages

## Features

- **OpenGL Rendering with Instancing**: Gopher3D allows efficient rendering of multiple objects (like particles) through instancing, significantly improving performance for scenes with many repeated elements.
- **Camera Controls**: Integrated camera controls with support for mouse and keyboard input.
- **Basic Lighting**: Currently includes basic light setup (static light sources).
- **Examples**: Various examples (particles, black hole, basic rendering) are included to showcase how the engine can be used.

## Not Implemented

- **Proper physics**
- **Collisions**
- **Multiple materials**
- **Scenes**
- **UI**
- **Vulkan (In Progress)**

## Getting Started

### Prerequisites

To run the Gopher3D engine, ensure you have the following dependencies:

- **Go**
- **GCC**

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

### Running examples

You can run examples to see how the engine works. Each example defines a particular scene's behavior and demonstrates features like particle systems, rendering, and camera movement.

To run an example:
```bash
go run ./examples/black_hole.go
```


## Examples

### Black Hole Simulation

This example demonstrates a particle simulation where particles orbit around a black hole. Instanced rendering is used for performance, and particles that enter the event horizon are removed from the scene.

- **File**: `black_hole.go`
- **Description**: Demonstrates particle simulation using instanced rendering and Verlet integration for particle movement.

### Sand Simulation
This example demonstrates a sand particle simulation where you can interact with the sand using your mouse.

- **File**: `black_hole.go`
- **Description**: Simulates sand particles that can be gathered and released using the mouse, showcasing interactive particle dynamics and instanced rendering.

Mouse Interaction in the Sand Example
Gathering Sand: Click and hold the left mouse button to attract nearby sand particles toward the mouse cursor. Particles within a certain radius will move towards the cursor, simulating the gathering of sand.
Releasing Sand: When you release the mouse button, the gathered sand particles will gently explode outward from the release point, spreading out across the scene.
Adjusting Explosion Strength: The explosion force is gentle to mimic a natural dispersal of sand, ensuring the particles spread out smoothly rather than violently.
This interactive example demonstrates how you can manipulate particles in real-time, providing a foundation for more complex simulations and user interactions.

### Basic Example

This is a simple example to showcase basic 3D rendering, camera controls, and light interaction.

- **File**: `basic_example.go`
- **Description**: Basic scene setup with a simple object and light source.

### Particle System Example

This example demonstrates basic particle behaviors, including forces like gravity. Note that this is just an example; a proper particle system that is fully integrated into the engine is a future goal.

- **File**: `particles.go`
- **Description**: A particle simulation with basic movement and interaction, showcasing instancing.

### Voxel Rendering Example (Gocraft)

This example renders voxel-based objects similar to Minecraft. It demonstrates block-like object rendering.

- **File**: `gocraft.go`
- **Description**: Voxel-based scene with basic block rendering.

## Planned Features and Work in Progress

### Physics and Particle Systems

The engine currently includes example implementations of particles, but a fully integrated particle system and a proper physics engine are still in development. The current particle simulations serve as examples, and future updates will introduce robust, engine-level implementations for both.

### Vulkan Renderer

The Vulkan renderer is in progress but currently **not functional**. The OpenGL renderer is fully functional, and Vulkan will be revisited in future development.

## Contributing

As an open-source project, contributions are welcome! Whether you're fixing bugs, improving performance, or adding new features, feel free to submit pull requests and help shape the future of Gopher3D.

To contribute:

1. Fork the repository.
2. Create a feature branch (`git checkout -b feature/my-feature`).
3. Commit your changes (`git commit -m 'Add my feature'`).
4. Push to the branch (`git push origin feature/my-feature`).
5. Create a Pull Request.

## Known Issues

- **Normals**: There are some known issues with how lighting interacts with normals. These will be addressed in future updates.


## Images

![Black hole instanciated](https://github.com/user-attachments/assets/0f9467b4-e4b5-4ebf-ac66-ed3e8bc87efc)

![Mars](https://github.com/nicolasmd87/Gopher3D/assets/8224408/09d2a39b-c1cb-4548-87fb-1a877df24453)







