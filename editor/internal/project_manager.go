package editor

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/sqweek/dialog"
)

type Project struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

var (
	ShowProjectManager = true
	CurrentProject     *Project
	recentProjects     []Project
)

func RenderProjectManager() {
	viewport := imgui.MainViewport()
	center := viewport.Center()

	cardWidth := float32(480)
	cardHeight := float32(400)

	imgui.SetNextWindowPosV(center, imgui.ConditionAlways, imgui.Vec2{X: 0.5, Y: 0.5})
	imgui.SetNextWindowSizeV(imgui.Vec2{X: cardWidth, Y: cardHeight}, imgui.ConditionAlways)

	flags := imgui.WindowFlagsNoResize | imgui.WindowFlagsNoCollapse

	imgui.PushStyleVarFloat(imgui.StyleVarWindowRounding, 8)
	imgui.PushStyleVarVec2(imgui.StyleVarWindowPadding, imgui.Vec2{X: 24, Y: 20})
	imgui.PushStyleColor(imgui.StyleColorWindowBg, imgui.Vec4{X: 0.1, Y: 0.1, Z: 0.12, W: 1.0})
	imgui.PushStyleColor(imgui.StyleColorTitleBg, imgui.Vec4{X: 0.0, Y: 0.5, Z: 0.65, W: 1.0})
	imgui.PushStyleColor(imgui.StyleColorTitleBgActive, imgui.Vec4{X: 0.0, Y: 0.678, Z: 0.847, W: 1.0})

	if imgui.BeginV("Gopher3D", nil, flags) {
		imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: 0.5, Y: 0.5, Z: 0.55, W: 1.0})
		imgui.Text("Select or create a project")
		imgui.PopStyleColor()

		imgui.Spacing()
		imgui.Spacing()

		btnWidth := (cardWidth - 48 - 12) / 2
		btnHeight := float32(36)

		imgui.PushStyleVarVec2(imgui.StyleVarItemSpacing, imgui.Vec2{X: 12, Y: 0})
		imgui.PushStyleVarFloat(imgui.StyleVarFrameRounding, 4)

		imgui.PushStyleColor(imgui.StyleColorButton, imgui.Vec4{X: 0.0, Y: 0.678, Z: 0.847, W: 1.0})
		imgui.PushStyleColor(imgui.StyleColorButtonHovered, imgui.Vec4{X: 0.0, Y: 0.75, Z: 0.9, W: 1.0})
		imgui.PushStyleColor(imgui.StyleColorButtonActive, imgui.Vec4{X: 0.0, Y: 0.6, Z: 0.75, W: 1.0})
		imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: 0.0, Y: 0.0, Z: 0.0, W: 1.0})

		if imgui.ButtonV("New Project", imgui.Vec2{X: btnWidth, Y: btnHeight}) {
			directory, err := dialog.Directory().Title("Select Project Directory").Browse()
			if err == nil && directory != "" {
				createProjectStructure(directory)
				CurrentProject = &Project{Name: filepath.Base(directory), Path: directory}
				ShowProjectManager = false
				currentDirectory = filepath.Join(directory, "resources")
				addRecentProject(*CurrentProject)
				saveRecentProjects()
			}
		}

		imgui.PopStyleColorV(4)

		imgui.SameLine()

		imgui.PushStyleColor(imgui.StyleColorButton, imgui.Vec4{X: 0.2, Y: 0.2, Z: 0.22, W: 1.0})
		imgui.PushStyleColor(imgui.StyleColorButtonHovered, imgui.Vec4{X: 0.28, Y: 0.28, Z: 0.3, W: 1.0})
		imgui.PushStyleColor(imgui.StyleColorButtonActive, imgui.Vec4{X: 0.18, Y: 0.18, Z: 0.2, W: 1.0})
		imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: 0.9, Y: 0.9, Z: 0.9, W: 1.0})

		if imgui.ButtonV("Open Project", imgui.Vec2{X: btnWidth, Y: btnHeight}) {
			directory, err := dialog.Directory().Title("Open Project Directory").Browse()
			if err == nil && directory != "" {
				openProject(directory)
			}
		}

		imgui.PopStyleColorV(4)
		imgui.PopStyleVarV(2)

		imgui.Spacing()
		imgui.Spacing()

		imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: 0.6, Y: 0.6, Z: 0.65, W: 1.0})
		imgui.Text("Recent Projects")
		imgui.PopStyleColor()

		imgui.Separator()
		imgui.Spacing()

		listHeight := cardHeight - 170
		imgui.PushStyleColor(imgui.StyleColorChildBg, imgui.Vec4{X: 0.08, Y: 0.08, Z: 0.1, W: 1.0})
		imgui.PushStyleVarFloat(imgui.StyleVarChildRounding, 4)

		if imgui.BeginChildV("##RecentList", imgui.Vec2{X: cardWidth - 48, Y: listHeight}, true, 0) {
			if len(recentProjects) == 0 {
				imgui.Spacing()
				imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: 0.4, Y: 0.4, Z: 0.45, W: 1.0})
				imgui.Text("  No recent projects")
				imgui.PopStyleColor()
			} else {
				for i, proj := range recentProjects {
					imgui.PushIDInt(i)

					imgui.PushStyleColor(imgui.StyleColorHeader, imgui.Vec4{X: 0.0, Y: 0.678, Z: 0.847, W: 0.15})
					imgui.PushStyleColor(imgui.StyleColorHeaderHovered, imgui.Vec4{X: 0.0, Y: 0.678, Z: 0.847, W: 0.25})
					imgui.PushStyleColor(imgui.StyleColorHeaderActive, imgui.Vec4{X: 0.0, Y: 0.678, Z: 0.847, W: 0.35})

					if imgui.SelectableV("##item", false, 0, imgui.Vec2{X: 0, Y: 44}) {
						openProject(proj.Path)
						ShowProjectManager = false
					}

					imgui.PopStyleColorV(3)

					imgui.SameLine()
					imgui.BeginGroup()
					imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: 0.9, Y: 0.9, Z: 0.95, W: 1.0})
					imgui.Text(proj.Name)
					imgui.PopStyleColor()
					imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: 0.45, Y: 0.45, Z: 0.5, W: 1.0})
					imgui.Text(proj.Path)
					imgui.PopStyleColor()
					imgui.EndGroup()

					imgui.PopID()
				}
			}
		}
		imgui.EndChild()
		imgui.PopStyleVar()
		imgui.PopStyleColor()

		imgui.End()
	}

	imgui.PopStyleColorV(3)
	imgui.PopStyleVarV(2)
}

func openProject(directory string) {
	CurrentProject = &Project{
		Name: filepath.Base(directory),
		Path: directory,
	}
	ShowProjectManager = false
	// Set file explorer to project root or resources
	resPath := filepath.Join(directory, "resources")
	if _, err := os.Stat(resPath); !os.IsNotExist(err) {
		currentDirectory = resPath
	} else {
		currentDirectory = directory
	}
	// Add to recent projects
	addRecentProject(*CurrentProject)
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
	SaveConfig()
}

func loadRecentProjects() {
	LoadConfig()
}

func createProjectStructure(path string) {
	dirs := []string{
		"assets",
		"resources",
		"resources/models",
		"resources/textures",
		"resources/scripts",
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
