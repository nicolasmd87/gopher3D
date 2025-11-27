package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/sqweek/dialog"
)

type Project struct {
	Name string
	Path string
}

var (
	showProjectManager = true
	currentProject     *Project
	recentProjects     []Project
)

func renderProjectManager() {
	// Center window
	viewport := imgui.MainViewport()
	center := viewport.Center()
	imgui.SetNextWindowPosV(center, imgui.ConditionAppearing, imgui.Vec2{X: 0.5, Y: 0.5})
	imgui.SetNextWindowSizeV(imgui.Vec2{X: 600, Y: 400}, imgui.ConditionAppearing)

	if imgui.BeginV("Gopher3D - Project Manager", nil, imgui.WindowFlagsNoCollapse|imgui.WindowFlagsNoResize) {
		imgui.Text("Welcome to Gopher3D")
		imgui.Separator()
		imgui.Spacing()

		if imgui.Button("Create New Project") {
			directory, err := dialog.Directory().Title("Select Project Directory").Browse()
			if err == nil && directory != "" {
				// Create basic project structure
				createProjectStructure(directory)
				currentProject = &Project{
					Name: filepath.Base(directory),
					Path: directory,
				}
				showProjectManager = false
				currentDirectory = filepath.Join(directory, "resources")
				// Add to recent projects
				addRecentProject(*currentProject)
				saveRecentProjects()
			}
		}
		imgui.SameLine()
		if imgui.Button("Open Project") {
			directory, err := dialog.Directory().Title("Open Project Directory").Browse()
			if err == nil && directory != "" {
				openProject(directory)
			}
		}

		imgui.Separator()
		imgui.Text("Recent Projects:")
		if len(recentProjects) == 0 {
			imgui.Text("(No recent projects)")
		} else {
			for i, proj := range recentProjects {
				if imgui.SelectableV(fmt.Sprintf("%s##recent%d", proj.Name, i), false, 0, imgui.Vec2{}) {
					openProject(proj.Path)
					showProjectManager = false
				}
				imgui.SameLine()
				imgui.Text(fmt.Sprintf("- %s", proj.Path))
			}
		}

		imgui.End()
	}
}

func openProject(directory string) {
	currentProject = &Project{
		Name: filepath.Base(directory),
		Path: directory,
	}
	showProjectManager = false
	// Set file explorer to project root or resources
	resPath := filepath.Join(directory, "resources")
	if _, err := os.Stat(resPath); !os.IsNotExist(err) {
		currentDirectory = resPath
	} else {
		currentDirectory = directory
	}
	// Add to recent projects
	addRecentProject(*currentProject)
	saveRecentProjects()
}

func addRecentProject(proj Project) {
	// Remove duplicate if exists
	for i, p := range recentProjects {
		if p.Path == proj.Path {
			recentProjects = append(recentProjects[:i], recentProjects[i+1:]...)
			break
		}
	}
	
	// Add to front
	recentProjects = append([]Project{proj}, recentProjects...)
	
	// Keep only 10 most recent
	if len(recentProjects) > 10 {
		recentProjects = recentProjects[:10]
	}
}

func saveRecentProjects() {
	// Projects are saved via config system
	saveConfig()
}

func loadRecentProjects() {
	// Projects are loaded via config system
	loadConfig()
}

func createProjectStructure(path string) {
	dirs := []string{
		"assets",
		"resources",
		"resources/models",
		"resources/textures",
		"scenes",
	}

	for _, dir := range dirs {
		fullPath := filepath.Join(path, dir)
		err := os.MkdirAll(fullPath, 0755)
		if err != nil {
			fmt.Printf("Failed to create directory %s: %v\n", fullPath, err)
		}
	}
}


