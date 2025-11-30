# Gopher3D Build Script for Windows
param(
    [Parameter(Position=0)]
    [string]$Target = "build"
)

$ErrorActionPreference = "Stop"

function Write-Header($text) {
    Write-Host "`n=== $text ===" -ForegroundColor Cyan
}

function Build {
    Write-Header "Building editor"
    go build -o bin/editor.exe ./editor/cmd/...
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Build complete: bin/editor.exe" -ForegroundColor Green
    }
}

function Test {
    Write-Header "Running tests"
    go test ./internal/behaviour/...
    go test ./internal/loader/...
    go test ./internal/renderer/...
    Write-Host "Tests complete" -ForegroundColor Green
}

function Vet {
    Write-Header "Running go vet"
    go vet ./internal/behaviour/...
    go vet ./internal/loader/...
    go vet ./internal/renderer/...
    go vet ./internal/logger/...
    go vet ./scripts/...
    go vet ./editor/internal/...
    go vet ./editor/cmd/...
    Write-Host "Vet complete" -ForegroundColor Green
}

function Fmt {
    Write-Header "Formatting code"
    go fmt ./...
}

function Tidy {
    Write-Header "Tidying dependencies"
    go mod tidy
}

function Lint {
    Vet
    Write-Header "Running staticcheck"
    $staticcheck = Get-Command staticcheck -ErrorAction SilentlyContinue
    if (-not $staticcheck) {
        Write-Host "Installing staticcheck..." -ForegroundColor Yellow
        go install honnef.co/go/tools/cmd/staticcheck@latest
    }
    staticcheck ./internal/...
    staticcheck ./scripts/...
    Write-Host "Lint complete" -ForegroundColor Green
}

function FieldAlignment {
    Write-Header "Checking struct field alignment"
    $fa = Get-Command fieldalignment -ErrorAction SilentlyContinue
    if (-not $fa) {
        Write-Host "Installing fieldalignment..." -ForegroundColor Yellow
        go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest
    }
    fieldalignment ./internal/...
    Write-Host "Alignment check complete" -ForegroundColor Green
}

function FieldAlignmentFix {
    Write-Header "Fixing struct field alignment"
    $fa = Get-Command fieldalignment -ErrorAction SilentlyContinue
    if (-not $fa) {
        Write-Host "Installing fieldalignment..." -ForegroundColor Yellow
        go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest
    }
    fieldalignment -fix ./internal/...
    Write-Host "Alignment fix complete" -ForegroundColor Green
}

function Tools {
    Write-Header "Installing development tools"
    go install honnef.co/go/tools/cmd/staticcheck@latest
    go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest
    Write-Host "Tools installed" -ForegroundColor Green
}

function Clean {
    Write-Header "Cleaning"
    if (Test-Path bin) { Remove-Item -Recurse -Force bin }
    if (Test-Path editor/cmd/cmd.exe) { Remove-Item editor/cmd/cmd.exe }
}

function Run {
    Build
    Write-Header "Running editor"
    Push-Location bin
    try {
        ./editor.exe
    } finally {
        Pop-Location
    }
}

function CI {
    Fmt
    Tidy
    Vet
    Test
    Build
    Write-Host "`nCI checks passed" -ForegroundColor Green
}

function Help {
    Write-Host @"

Gopher3D Build Script
Usage: .\build.ps1 [target]

Targets:
  build            - Build the editor (default)
  run              - Build and run the editor
  test             - Run all tests
  vet              - Run go vet
  lint             - Run staticcheck and vet
  fieldalignment   - Check struct padding
  fieldalignment-fix - Auto-fix struct padding
  fmt              - Format code
  tidy             - Tidy go.mod
  clean            - Remove build artifacts
  tools            - Install dev tools
  ci               - Full CI checks

"@
}

switch ($Target.ToLower()) {
    "build" { Build }
    "run" { Run }
    "test" { Test }
    "vet" { Vet }
    "lint" { Lint }
    "fieldalignment" { FieldAlignment }
    "fieldalignment-fix" { FieldAlignmentFix }
    "fmt" { Fmt }
    "tidy" { Tidy }
    "clean" { Clean }
    "tools" { Tools }
    "ci" { CI }
    "help" { Help }
    default { 
        Write-Host "Unknown target: $Target" -ForegroundColor Red
        Help 
    }
}

