package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// binaryPath holds the path to the compiled cosm binary
var binaryPath string

func TestMain(m *testing.M) {
	tempDir := os.TempDir()
	binaryPath = filepath.Join(tempDir, "cosm")

	cmd := exec.Command("go", "build", "-o", binaryPath, "main.go")
	if err := cmd.Run(); err != nil {
		println("Failed to build cosm binary:", err.Error())
		os.Exit(1)
	}

	exitCode := m.Run()
	// os.Remove(binaryPath) // Uncomment to clean up
	os.Exit(exitCode)
}

// runCommand runs the cosm binary with given args in a directory and returns output and error
func runCommand(t *testing.T, dir string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = dir
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	err = cmd.Run()
	return out.String(), errOut.String(), err
}

// checkOutput verifies the command output and exit code
func checkOutput(t *testing.T, stdout, stderr, expectedOutput string, err error, expectError bool, expectedExitCode int) {
	t.Helper()
	if expectError {
		if err == nil {
			t.Fatalf("Expected an error, got none (stdout: %q, stderr: %q)", stdout, stderr)
		}
		if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != expectedExitCode {
			t.Errorf("Expected exit code %d, got %v", expectedExitCode, err)
		}
	} else {
		if err != nil {
			t.Fatalf("Expected no error, got %v (stderr: %q)", err, stderr)
		}
	}
	if stdout != expectedOutput {
		t.Errorf("Expected output %q, got %q", expectedOutput, stdout)
	}
}

// setupRegistriesFile creates a registries.json with given registries
func setupRegistriesFile(t *testing.T, dir string, registries []struct {
	Name        string              `json:"name"`
	GitURL      string              `json:"giturl"`
	Packages    map[string][]string `json:"packages,omitempty"`
	LastUpdated time.Time           `json:"last_updated,omitempty"`
}) string {
	t.Helper()
	cosmDir := filepath.Join(dir, ".cosm")
	if err := os.MkdirAll(cosmDir, 0755); err != nil {
		t.Fatalf("Failed to create .cosm directory: %v", err)
	}
	registriesFile := filepath.Join(cosmDir, "registries.json")
	data, err := json.MarshalIndent(registries, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal registries: %v", err)
	}
	if err := os.WriteFile(registriesFile, data, 0644); err != nil {
		t.Fatalf("Failed to write registries.json: %v", err)
	}
	return registriesFile
}

// checkRegistriesFile verifies the contents of registries.json
func checkRegistriesFile(t *testing.T, file string, expected []struct {
	Name        string              `json:"name"`
	GitURL      string              `json:"giturl"`
	Packages    map[string][]string `json:"packages,omitempty"`
	LastUpdated time.Time           `json:"last_updated,omitempty"`
}) {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("Failed to read registries.json: %v", err)
	}
	var registries []struct {
		Name        string              `json:"name"`
		GitURL      string              `json:"giturl"`
		Packages    map[string][]string `json:"packages,omitempty"`
		LastUpdated time.Time           `json:"last_updated,omitempty"`
	}
	if err := json.Unmarshal(data, &registries); err != nil {
		t.Fatalf("Failed to parse registries.json: %v", err)
	}
	if len(registries) != len(expected) {
		t.Errorf("Expected %d registries, got %d", len(expected), len(registries))
	}
	for i, exp := range expected {
		if i >= len(registries) {
			break
		}
		got := registries[i]
		if got.Name != exp.Name || got.GitURL != exp.GitURL {
			t.Errorf("Expected registry %d: {Name: %q, GitURL: %q}, got {Name: %q, GitURL: %q}",
				i, exp.Name, exp.GitURL, got.Name, got.GitURL)
		}
		if len(got.Packages) != len(exp.Packages) {
			t.Errorf("Expected %d packages for %s, got %d", len(exp.Packages), exp.Name, len(got.Packages))
		}
		for pkgName, expVersions := range exp.Packages {
			gotVersions, exists := got.Packages[pkgName]
			if !exists {
				t.Errorf("Package %s not found in registry %s", pkgName, exp.Name)
				continue
			}
			if len(gotVersions) != len(expVersions) {
				t.Errorf("Expected %d versions for %s in %s, got %d", len(expVersions), pkgName, exp.Name, len(gotVersions))
			}
			for j, v := range expVersions {
				if j >= len(gotVersions) || gotVersions[j] != v {
					t.Errorf("Expected version %d for %s in %s: %q, got %q", j, pkgName, exp.Name, v, gotVersions[j])
				}
			}
		}
		if !exp.LastUpdated.IsZero() && got.LastUpdated.IsZero() {
			t.Errorf("Expected LastUpdated to be set for %s, got zero", exp.Name)
		}
	}
}

// checkProjectFile verifies the contents of Project.json
func checkProjectFile(t *testing.T, file string, expected struct {
	Name         string       `json:"name"`
	Version      string       `json:"version"`
	Dependencies []Dependency `json:"dependencies,omitempty"`
}) {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("Failed to read Project.json: %v", err)
	}
	var project struct {
		Name         string       `json:"name"`
		Version      string       `json:"version"`
		Dependencies []Dependency `json:"dependencies,omitempty"`
	}
	if err := json.Unmarshal(data, &project); err != nil {
		t.Fatalf("Failed to parse Project.json: %v", err)
	}
	if project.Name != expected.Name || project.Version != expected.Version {
		t.Errorf("Expected Project {Name: %q, Version: %q}, got {Name: %q, Version: %q}",
			expected.Name, expected.Version, project.Name, project.Version)
	}
	if len(project.Dependencies) != len(expected.Dependencies) {
		t.Errorf("Expected %d dependencies, got %d", len(expected.Dependencies), len(project.Dependencies))
	}
	for i, expDep := range expected.Dependencies {
		if i >= len(project.Dependencies) {
			break
		}
		gotDep := project.Dependencies[i]
		if gotDep.Name != expDep.Name || gotDep.Version != expDep.Version {
			t.Errorf("Expected dependency %d: {Name: %q, Version: %q}, got {Name: %q, Version: %q}",
				i, expDep.Name, expDep.Version, gotDep.Name, gotDep.Version)
		}
	}
}

func TestVersion(t *testing.T) {
	tempDir := t.TempDir()
	stdout, _, err := runCommand(t, tempDir, "--version")
	checkOutput(t, stdout, "", "cosm version 0.1.0\n", err, false, 0)
}

func TestStatus(t *testing.T) {
	tempDir := t.TempDir()
	stdout, _, err := runCommand(t, tempDir, "status")
	checkOutput(t, stdout, "", "Cosmic Status:\n  Orbit: Stable\n  Systems: All green\n  Pending updates: None\nRun 'cosm status' in a project directory for more details.\n", err, false, 0)
}

func TestActivateSuccess(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "cosm.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create mock cosm.json: %v", err)
	}
	stdout, _, err := runCommand(t, tempDir, "activate")
	checkOutput(t, stdout, "", "Activated current project\n", err, false, 0)
}

func TestActivateFailure(t *testing.T) {
	tempDir := t.TempDir()
	stdout, _, err := runCommand(t, tempDir, "activate")
	checkOutput(t, stdout, "", "Error: No project found in current directory (missing cosm.json)\n", err, true, 1)
}

func TestInit(t *testing.T) {
	tempDir := t.TempDir()
	packageName := "myproject"

	stdout, _, err := runCommand(t, tempDir, "init", packageName)
	checkOutput(t, stdout, "", fmt.Sprintf("Initialized project '%s' with version v0.1.0\n", packageName), err, false, 0)

	checkProjectFile(t, filepath.Join(tempDir, "Project.json"), struct {
		Name         string       `json:"name"`
		Version      string       `json:"version"`
		Dependencies []Dependency `json:"dependencies,omitempty"`
	}{Name: packageName, Version: "v0.1.0"})
}

func TestInitDuplicate(t *testing.T) {
	tempDir := t.TempDir()
	packageName := "myproject"

	projectFile := filepath.Join(tempDir, "Project.json")
	initialProject := struct {
		Name         string       `json:"name"`
		Version      string       `json:"version"`
		Dependencies []Dependency `json:"dependencies,omitempty"`
	}{Name: "existing", Version: "v0.1.0"}
	data, _ := json.MarshalIndent(initialProject, "", "  ")
	if err := os.WriteFile(projectFile, data, 0644); err != nil {
		t.Fatalf("Failed to create initial Project.json: %v", err)
	}
	dataBefore, _ := os.ReadFile(projectFile)

	stdout, _, err := runCommand(t, tempDir, "init", packageName)
	checkOutput(t, stdout, "", "Error: Project.json already exists in this directory\n", err, true, 1)

	dataAfter, _ := os.ReadFile(projectFile)
	if !bytes.Equal(dataBefore, dataAfter) {
		t.Errorf("Project.json changed unexpectedly")
	}
}

func TestAdd(t *testing.T) {
	tempDir := t.TempDir()
	projectName := "myproject"
	depName := "mypkg"
	depVersion := "v1.2.3"

	projectFile := filepath.Join(tempDir, "Project.json")
	initialProject := struct {
		Name         string       `json:"name"`
		Version      string       `json:"version"`
		Dependencies []Dependency `json:"dependencies,omitempty"`
	}{Name: projectName, Version: "v0.1.0"}
	data, _ := json.MarshalIndent(initialProject, "", "  ")
	if err := os.WriteFile(projectFile, data, 0644); err != nil {
		t.Fatalf("Failed to create initial Project.json: %v", err)
	}

	stdout, _, err := runCommand(t, tempDir, "add", depName, depVersion)
	checkOutput(t, stdout, "", fmt.Sprintf("Added dependency '%s' v%s to project\n", depName, depVersion), err, false, 0)

	checkProjectFile(t, projectFile, struct {
		Name         string       `json:"name"`
		Version      string       `json:"version"`
		Dependencies []Dependency `json:"dependencies,omitempty"`
	}{Name: projectName, Version: "v0.1.0", Dependencies: []Dependency{{Name: depName, Version: depVersion}}})
}

func TestAddNoProject(t *testing.T) {
	tempDir := t.TempDir()
	depName := "mypkg"
	depVersion := "v1.2.3"

	stdout, _, err := runCommand(t, tempDir, "add", depName, depVersion)
	checkOutput(t, stdout, "", "Error: No Project.json found in current directory\n", err, true, 1)
}

func TestAddInvalidVersion(t *testing.T) {
	tempDir := t.TempDir()
	projectName := "myproject"
	depName := "mypkg"
	depVersion := "1.2.3"

	projectFile := filepath.Join(tempDir, "Project.json")
	initialProject := struct {
		Name         string       `json:"name"`
		Version      string       `json:"version"`
		Dependencies []Dependency `json:"dependencies,omitempty"`
	}{Name: projectName, Version: "v0.1.0"}
	data, _ := json.MarshalIndent(initialProject, "", "  ")
	if err := os.WriteFile(projectFile, data, 0644); err != nil {
		t.Fatalf("Failed to create initial Project.json: %v", err)
	}
	dataBefore, _ := os.ReadFile(projectFile)

	stdout, _, err := runCommand(t, tempDir, "add", depName, depVersion)
	checkOutput(t, stdout, "", fmt.Sprintf("Error: Version '%s' must start with 'v'\n", depVersion), err, true, 1)

	dataAfter, _ := os.ReadFile(projectFile)
	if !bytes.Equal(dataBefore, dataAfter) {
		t.Errorf("Project.json changed unexpectedly")
	}
}

func TestRm(t *testing.T) {
	tempDir := t.TempDir()
	projectName := "myproject"
	depName := "mypkg"

	projectFile := filepath.Join(tempDir, "Project.json")
	initialProject := struct {
		Name         string       `json:"name"`
		Version      string       `json:"version"`
		Dependencies []Dependency `json:"dependencies,omitempty"`
	}{Name: projectName, Version: "v0.1.0", Dependencies: []Dependency{{Name: depName, Version: "v1.2.3"}, {Name: "otherpkg", Version: "v2.0.0"}}}
	data, _ := json.MarshalIndent(initialProject, "", "  ")
	if err := os.WriteFile(projectFile, data, 0644); err != nil {
		t.Fatalf("Failed to create initial Project.json: %v", err)
	}

	stdout, _, err := runCommand(t, tempDir, "rm", depName)
	checkOutput(t, stdout, "", fmt.Sprintf("Removed dependency '%s' from project\n", depName), err, false, 0)

	checkProjectFile(t, projectFile, struct {
		Name         string       `json:"name"`
		Version      string       `json:"version"`
		Dependencies []Dependency `json:"dependencies,omitempty"`
	}{Name: projectName, Version: "v0.1.0", Dependencies: []Dependency{{Name: "otherpkg", Version: "v2.0.0"}}})
}

func TestRmNoProject(t *testing.T) {
	tempDir := t.TempDir()
	depName := "mypkg"

	stdout, _, err := runCommand(t, tempDir, "rm", depName)
	checkOutput(t, stdout, "", "Error: No Project.json found in current directory\n", err, true, 1)
}

func TestRmDependencyNotFound(t *testing.T) {
	tempDir := t.TempDir()
	projectName := "myproject"
	depName := "mypkg"

	projectFile := filepath.Join(tempDir, "Project.json")
	initialProject := struct {
		Name         string       `json:"name"`
		Version      string       `json:"version"`
		Dependencies []Dependency `json:"dependencies,omitempty"`
	}{Name: projectName, Version: "v0.1.0", Dependencies: []Dependency{{Name: "otherpkg", Version: "v2.0.0"}}}
	data, _ := json.MarshalIndent(initialProject, "", "  ")
	if err := os.WriteFile(projectFile, data, 0644); err != nil {
		t.Fatalf("Failed to create initial Project.json: %v", err)
	}
	dataBefore, _ := os.ReadFile(projectFile)

	stdout, _, err := runCommand(t, tempDir, "rm", depName)
	checkOutput(t, stdout, "", fmt.Sprintf("Error: Dependency '%s' not found in project\n", depName), err, true, 1)

	dataAfter, _ := os.ReadFile(projectFile)
	if !bytes.Equal(dataBefore, dataAfter) {
		t.Errorf("Project.json changed unexpectedly")
	}
}

func TestReleaseExplicit(t *testing.T) {
	tempDir := t.TempDir()
	projectName := "myproject"
	newVersion := "v0.2.0"

	projectFile := filepath.Join(tempDir, "Project.json")
	initialProject := struct {
		Name         string       `json:"name"`
		Version      string       `json:"version"`
		Dependencies []Dependency `json:"dependencies,omitempty"`
	}{Name: projectName, Version: "v0.1.0"}
	data, _ := json.MarshalIndent(initialProject, "", "  ")
	if err := os.WriteFile(projectFile, data, 0644); err != nil {
		t.Fatalf("Failed to create initial Project.json: %v", err)
	}

	stdout, _, err := runCommand(t, tempDir, "release", newVersion)
	checkOutput(t, stdout, "", fmt.Sprintf("Released '%s' v%s\n", projectName, newVersion), err, false, 0)

	checkProjectFile(t, projectFile, struct {
		Name         string       `json:"name"`
		Version      string       `json:"version"`
		Dependencies []Dependency `json:"dependencies,omitempty"`
	}{Name: projectName, Version: newVersion})
}

func TestReleasePatch(t *testing.T) {
	tempDir := t.TempDir()
	projectName := "myproject"
	newVersion := "v0.1.1"

	projectFile := filepath.Join(tempDir, "Project.json")
	initialProject := struct {
		Name         string       `json:"name"`
		Version      string       `json:"version"`
		Dependencies []Dependency `json:"dependencies,omitempty"`
	}{Name: projectName, Version: "v0.1.0"}
	data, _ := json.MarshalIndent(initialProject, "", "  ")
	if err := os.WriteFile(projectFile, data, 0644); err != nil {
		t.Fatalf("Failed to create initial Project.json: %v", err)
	}

	stdout, _, err := runCommand(t, tempDir, "release", "--patch")
	checkOutput(t, stdout, "", fmt.Sprintf("Released '%s' v%s\n", projectName, newVersion), err, false, 0)

	checkProjectFile(t, projectFile, struct {
		Name         string       `json:"name"`
		Version      string       `json:"version"`
		Dependencies []Dependency `json:"dependencies,omitempty"`
	}{Name: projectName, Version: newVersion})
}

func TestReleaseMinor(t *testing.T) {
	tempDir := t.TempDir()
	projectName := "myproject"
	newVersion := "v0.2.0"

	projectFile := filepath.Join(tempDir, "Project.json")
	initialProject := struct {
		Name         string       `json:"name"`
		Version      string       `json:"version"`
		Dependencies []Dependency `json:"dependencies,omitempty"`
	}{Name: projectName, Version: "v0.1.0"}
	data, _ := json.MarshalIndent(initialProject, "", "  ")
	if err := os.WriteFile(projectFile, data, 0644); err != nil {
		t.Fatalf("Failed to create initial Project.json: %v", err)
	}

	stdout, _, err := runCommand(t, tempDir, "release", "--minor")
	checkOutput(t, stdout, "", fmt.Sprintf("Released '%s' v%s\n", projectName, newVersion), err, false, 0)

	checkProjectFile(t, projectFile, struct {
		Name         string       `json:"name"`
		Version      string       `json:"version"`
		Dependencies []Dependency `json:"dependencies,omitempty"`
	}{Name: projectName, Version: newVersion})
}

func TestReleaseMajor(t *testing.T) {
	tempDir := t.TempDir()
	projectName := "myproject"
	newVersion := "v1.0.0"

	projectFile := filepath.Join(tempDir, "Project.json")
	initialProject := struct {
		Name         string       `json:"name"`
		Version      string       `json:"version"`
		Dependencies []Dependency `json:"dependencies,omitempty"`
	}{Name: projectName, Version: "v0.1.0"}
	data, _ := json.MarshalIndent(initialProject, "", "  ")
	if err := os.WriteFile(projectFile, data, 0644); err != nil {
		t.Fatalf("Failed to create initial Project.json: %v", err)
	}

	stdout, _, err := runCommand(t, tempDir, "release", "--major")
	checkOutput(t, stdout, "", fmt.Sprintf("Released '%s' v%s\n", projectName, newVersion), err, false, 0)

	checkProjectFile(t, projectFile, struct {
		Name         string       `json:"name"`
		Version      string       `json:"version"`
		Dependencies []Dependency `json:"dependencies,omitempty"`
	}{Name: projectName, Version: newVersion})
}

func TestReleaseNoProject(t *testing.T) {
	tempDir := t.TempDir()
	newVersion := "v0.2.0"

	stdout, _, err := runCommand(t, tempDir, "release", newVersion)
	checkOutput(t, stdout, "", "Error: No Project.json found in current directory\n", err, true, 1)
}

func TestReleaseInvalidVersion(t *testing.T) {
	tempDir := t.TempDir()
	projectName := "myproject"
	newVersion := "v0.2.x" // Invalid SemVer

	projectFile := filepath.Join(tempDir, "Project.json")
	initialProject := struct {
		Name         string       `json:"name"`
		Version      string       `json:"version"`
		Dependencies []Dependency `json:"dependencies,omitempty"`
	}{Name: projectName, Version: "v0.1.0"}
	data, _ := json.MarshalIndent(initialProject, "", "  ")
	if err := os.WriteFile(projectFile, data, 0644); err != nil {
		t.Fatalf("Failed to create initial Project.json: %v", err)
	}
	dataBefore, _ := os.ReadFile(projectFile)

	stdout, _, err := runCommand(t, tempDir, "release", newVersion)
	checkOutput(t, stdout, "", fmt.Sprintf("Error parsing new version '%s': Invalid Semantic Version\n", newVersion), err, true, 1)

	dataAfter, _ := os.ReadFile(projectFile)
	if !bytes.Equal(dataBefore, dataAfter) {
		t.Errorf("Project.json changed unexpectedly")
	}
}

func TestReleaseNotGreater(t *testing.T) {
	tempDir := t.TempDir()
	projectName := "myproject"
	newVersion := "v0.0.1"

	projectFile := filepath.Join(tempDir, "Project.json")
	initialProject := struct {
		Name         string       `json:"name"`
		Version      string       `json:"version"`
		Dependencies []Dependency `json:"dependencies,omitempty"`
	}{Name: projectName, Version: "v0.1.0"}
	data, _ := json.MarshalIndent(initialProject, "", "  ")
	if err := os.WriteFile(projectFile, data, 0644); err != nil {
		t.Fatalf("Failed to create initial Project.json: %v", err)
	}
	dataBefore, _ := os.ReadFile(projectFile)

	stdout, _, err := runCommand(t, tempDir, "release", newVersion)
	checkOutput(t, stdout, "", fmt.Sprintf("Error: New version '%s' must be greater than current version '%s'\n", newVersion, "v0.1.0"), err, true, 1)

	dataAfter, _ := os.ReadFile(projectFile)
	if !bytes.Equal(dataBefore, dataAfter) {
		t.Errorf("Project.json changed unexpectedly")
	}
}

func TestReleaseNoArgs(t *testing.T) {
	tempDir := t.TempDir()
	projectName := "myproject"

	projectFile := filepath.Join(tempDir, "Project.json")
	initialProject := struct {
		Name         string       `json:"name"`
		Version      string       `json:"version"`
		Dependencies []Dependency `json:"dependencies,omitempty"`
	}{Name: projectName, Version: "v0.1.0"}
	data, _ := json.MarshalIndent(initialProject, "", "  ")
	if err := os.WriteFile(projectFile, data, 0644); err != nil {
		t.Fatalf("Failed to create initial Project.json: %v", err)
	}
	dataBefore, _ := os.ReadFile(projectFile)

	stdout, _, err := runCommand(t, tempDir, "release")
	checkOutput(t, stdout, "", "Error: Must specify either a version (v<version>) or one of --patch, --minor, or --major\n", err, true, 1)

	dataAfter, _ := os.ReadFile(projectFile)
	if !bytes.Equal(dataBefore, dataAfter) {
		t.Errorf("Project.json changed unexpectedly")
	}
}

func TestDevelopExistingDependency(t *testing.T) {
	tempDir := t.TempDir()
	projectName := "myproject"
	depName := "mypkg"
	depVersion := "v1.2.3"

	projectFile := filepath.Join(tempDir, "Project.json")
	initialProject := struct {
		Name         string       `json:"name"`
		Version      string       `json:"version"`
		Dependencies []Dependency `json:"dependencies,omitempty"`
	}{Name: projectName, Version: "v0.1.0", Dependencies: []Dependency{{Name: depName, Version: depVersion, Develop: false}}}
	data, _ := json.MarshalIndent(initialProject, "", "  ")
	if err := os.WriteFile(projectFile, data, 0644); err != nil {
		t.Fatalf("Failed to create initial Project.json: %v", err)
	}

	stdout, _, err := runCommand(t, tempDir, "develop", depName)
	checkOutput(t, stdout, "", fmt.Sprintf("Switched '%s' v%s to development mode\n", depName, depVersion), err, false, 0)

	checkProjectFile(t, projectFile, struct {
		Name         string       `json:"name"`
		Version      string       `json:"version"`
		Dependencies []Dependency `json:"dependencies,omitempty"`
	}{Name: projectName, Version: "v0.1.0", Dependencies: []Dependency{{Name: depName, Version: depVersion, Develop: true}}})
}

func TestDevelopNonExistingDependency(t *testing.T) {
	tempDir := t.TempDir()
	projectName := "myproject"
	depName := "mypkg"

	projectFile := filepath.Join(tempDir, "Project.json")
	initialProject := struct {
		Name         string       `json:"name"`
		Version      string       `json:"version"`
		Dependencies []Dependency `json:"dependencies,omitempty"`
	}{Name: projectName, Version: "v0.1.0", Dependencies: []Dependency{{Name: "otherpkg", Version: "v2.0.0", Develop: false}}}
	data, _ := json.MarshalIndent(initialProject, "", "  ")
	if err := os.WriteFile(projectFile, data, 0644); err != nil {
		t.Fatalf("Failed to create initial Project.json: %v", err)
	}
	dataBefore, _ := os.ReadFile(projectFile)

	stdout, _, err := runCommand(t, tempDir, "develop", depName)
	checkOutput(t, stdout, "", fmt.Sprintf("Error: Dependency '%s' not found in project. Use 'cosm add' to add it first.\n", depName), err, true, 1)

	dataAfter, _ := os.ReadFile(projectFile)
	if !bytes.Equal(dataBefore, dataAfter) {
		t.Errorf("Project.json changed unexpectedly")
	}
}

func TestDevelopNoProject(t *testing.T) {
	tempDir := t.TempDir()
	depName := "mypkg"

	stdout, _, err := runCommand(t, tempDir, "develop", depName)
	checkOutput(t, stdout, "", "Error: No Project.json found in current directory\n", err, true, 1)
}

func TestFreeExistingDevDependency(t *testing.T) {
	tempDir := t.TempDir()
	projectName := "myproject"
	depName := "mypkg"
	depVersion := "v1.2.3"

	projectFile := filepath.Join(tempDir, "Project.json")
	initialProject := struct {
		Name         string       `json:"name"`
		Version      string       `json:"version"`
		Dependencies []Dependency `json:"dependencies,omitempty"`
	}{Name: projectName, Version: "v0.1.0", Dependencies: []Dependency{{Name: depName, Version: depVersion, Develop: true}}}
	data, _ := json.MarshalIndent(initialProject, "", "  ")
	if err := os.WriteFile(projectFile, data, 0644); err != nil {
		t.Fatalf("Failed to create initial Project.json: %v", err)
	}

	stdout, _, err := runCommand(t, tempDir, "free", depName)
	checkOutput(t, stdout, "", fmt.Sprintf("Closed development mode for '%s' v%s\n", depName, depVersion), err, false, 0)

	checkProjectFile(t, projectFile, struct {
		Name         string       `json:"name"`
		Version      string       `json:"version"`
		Dependencies []Dependency `json:"dependencies,omitempty"`
	}{Name: projectName, Version: "v0.1.0", Dependencies: []Dependency{{Name: depName, Version: depVersion, Develop: false}}})
}

func TestFreeNonDevDependency(t *testing.T) {
	tempDir := t.TempDir()
	projectName := "myproject"
	depName := "mypkg"
	depVersion := "v1.2.3"

	projectFile := filepath.Join(tempDir, "Project.json")
	initialProject := struct {
		Name         string       `json:"name"`
		Version      string       `json:"version"`
		Dependencies []Dependency `json:"dependencies,omitempty"`
	}{Name: projectName, Version: "v0.1.0", Dependencies: []Dependency{{Name: depName, Version: depVersion, Develop: false}}}
	data, _ := json.MarshalIndent(initialProject, "", "  ")
	if err := os.WriteFile(projectFile, data, 0644); err != nil {
		t.Fatalf("Failed to create initial Project.json: %v", err)
	}
	dataBefore, _ := os.ReadFile(projectFile)

	stdout, _, err := runCommand(t, tempDir, "free", depName)
	checkOutput(t, stdout, "", fmt.Sprintf("Error: Dependency '%s' v%s is not in development mode\n", depName, depVersion), err, true, 1)

	dataAfter, _ := os.ReadFile(projectFile)
	if !bytes.Equal(dataBefore, dataAfter) {
		t.Errorf("Project.json changed unexpectedly")
	}
}

func TestFreeNonExistingDependency(t *testing.T) {
	tempDir := t.TempDir()
	projectName := "myproject"
	depName := "mypkg"

	projectFile := filepath.Join(tempDir, "Project.json")
	initialProject := struct {
		Name         string       `json:"name"`
		Version      string       `json:"version"`
		Dependencies []Dependency `json:"dependencies,omitempty"`
	}{Name: projectName, Version: "v0.1.0", Dependencies: []Dependency{{Name: "otherpkg", Version: "v2.0.0", Develop: false}}}
	data, _ := json.MarshalIndent(initialProject, "", "  ")
	if err := os.WriteFile(projectFile, data, 0644); err != nil {
		t.Fatalf("Failed to create initial Project.json: %v", err)
	}
	dataBefore, _ := os.ReadFile(projectFile)

	stdout, _, err := runCommand(t, tempDir, "free", depName)
	checkOutput(t, stdout, "", fmt.Sprintf("Error: Dependency '%s' not found in project\n", depName), err, true, 1)

	dataAfter, _ := os.ReadFile(projectFile)
	if !bytes.Equal(dataBefore, dataAfter) {
		t.Errorf("Project.json changed unexpectedly")
	}
}

func TestFreeNoProject(t *testing.T) {
	tempDir := t.TempDir()
	depName := "mypkg"

	stdout, _, err := runCommand(t, tempDir, "free", depName)
	checkOutput(t, stdout, "", "Error: No Project.json found in current directory\n", err, true, 1)
}

func TestRegistryStatus(t *testing.T) {
	tempDir := t.TempDir()
	stdout, _, err := runCommand(t, tempDir, "registry", "status", "cosmic-hub")
	checkOutput(t, stdout, "", "Status for registry 'cosmic-hub':\n  Available packages:\n    - cosmic-hub-pkg1 (v1.0.0)\n    - cosmic-hub-pkg2 (v2.1.3)\n  Last updated: 2025-04-05\n", err, false, 0)
}

func TestRegistryStatusInvalid(t *testing.T) {
	tempDir := t.TempDir()
	stdout, _, err := runCommand(t, tempDir, "registry", "status", "invalid-reg")
	checkOutput(t, stdout, "", "Error: 'invalid-reg' is not a valid registry name. Valid options: [cosmic-hub local]\n", err, true, 1)
}

func TestRegistryInit(t *testing.T) {
	tempDir := t.TempDir()
	registryName := "myreg"
	gitURL := "https://git.example.com"
	stdout, _, err := runCommand(t, tempDir, "registry", "init", registryName, gitURL)
	checkOutput(t, stdout, "", fmt.Sprintf("Initialized registry '%s' with Git URL: %s\n", registryName, gitURL), err, false, 0)

	checkRegistriesFile(t, filepath.Join(tempDir, ".cosm", "registries.json"), []struct {
		Name        string              `json:"name"`
		GitURL      string              `json:"giturl"`
		Packages    map[string][]string `json:"packages,omitempty"`
		LastUpdated time.Time           `json:"last_updated,omitempty"`
	}{{Name: registryName, GitURL: gitURL, Packages: make(map[string][]string)}})
}

func TestRegistryInitDuplicate(t *testing.T) {
	tempDir := t.TempDir()
	registryName := "myreg"
	gitURL := "https://git.example.com"
	registriesFile := setupRegistriesFile(t, tempDir, []struct {
		Name        string              `json:"name"`
		GitURL      string              `json:"giturl"`
		Packages    map[string][]string `json:"packages,omitempty"`
		LastUpdated time.Time           `json:"last_updated,omitempty"`
	}{{Name: registryName, GitURL: gitURL, Packages: make(map[string][]string)}})
	dataBefore, _ := os.ReadFile(registriesFile)

	stdout, stderr, err := runCommand(t, tempDir, "registry", "init", registryName, gitURL)
	checkOutput(t, stdout, stderr, fmt.Sprintf("Error: Registry '%s' already exists\n", registryName), err, true, 1)

	dataAfter, _ := os.ReadFile(registriesFile)
	if !bytes.Equal(dataBefore, dataAfter) {
		t.Errorf("registries.json changed unexpectedly")
	}
}

func TestRegistryClone(t *testing.T) {
	tempDir := t.TempDir()
	gitURL := "https://git.example.com/myreg.git"
	expectedName := "myreg.git"

	stdout, _, err := runCommand(t, tempDir, "registry", "clone", gitURL)
	checkOutput(t, stdout, "", fmt.Sprintf("Cloned registry '%s' from %s\n", expectedName, gitURL), err, false, 0)

	checkRegistriesFile(t, filepath.Join(tempDir, ".cosm", "registries.json"), []struct {
		Name        string              `json:"name"`
		GitURL      string              `json:"giturl"`
		Packages    map[string][]string `json:"packages,omitempty"`
		LastUpdated time.Time           `json:"last_updated,omitempty"`
	}{{Name: expectedName, GitURL: gitURL, Packages: make(map[string][]string)}})
}

func TestRegistryCloneDuplicate(t *testing.T) {
	tempDir := t.TempDir()
	gitURL := "https://git.example.com/myreg.git"
	registryName := "myreg.git"
	registriesFile := setupRegistriesFile(t, tempDir, []struct {
		Name        string              `json:"name"`
		GitURL      string              `json:"giturl"`
		Packages    map[string][]string `json:"packages,omitempty"`
		LastUpdated time.Time           `json:"last_updated,omitempty"`
	}{{Name: registryName, GitURL: gitURL, Packages: make(map[string][]string)}})
	dataBefore, _ := os.ReadFile(registriesFile)

	stdout, _, err := runCommand(t, tempDir, "registry", "clone", gitURL)
	checkOutput(t, stdout, "", fmt.Sprintf("Error: Registry '%s' already exists\n", registryName), err, true, 1)

	dataAfter, _ := os.ReadFile(registriesFile)
	if !bytes.Equal(dataBefore, dataAfter) {
		t.Errorf("registries.json changed unexpectedly")
	}
}

func TestRegistryDelete(t *testing.T) {
	tempDir := t.TempDir()
	registryName := "myreg"
	gitURL := "https://git.example.com"
	registriesFile := setupRegistriesFile(t, tempDir, []struct {
		Name        string              `json:"name"`
		GitURL      string              `json:"giturl"`
		Packages    map[string][]string `json:"packages,omitempty"`
		LastUpdated time.Time           `json:"last_updated,omitempty"`
	}{{Name: registryName, GitURL: gitURL, Packages: make(map[string][]string)}})

	stdout, _, err := runCommand(t, tempDir, "registry", "delete", registryName)
	checkOutput(t, stdout, "", fmt.Sprintf("Deleted registry '%s'\n", registryName), err, false, 0)

	checkRegistriesFile(t, registriesFile, []struct {
		Name        string              `json:"name"`
		GitURL      string              `json:"giturl"`
		Packages    map[string][]string `json:"packages,omitempty"`
		LastUpdated time.Time           `json:"last_updated,omitempty"`
	}{})
}

func TestRegistryDeleteForce(t *testing.T) {
	tempDir := t.TempDir()
	registryName := "myreg"
	gitURL := "https://git.example.com"
	registriesFile := setupRegistriesFile(t, tempDir, []struct {
		Name        string              `json:"name"`
		GitURL      string              `json:"giturl"`
		Packages    map[string][]string `json:"packages,omitempty"`
		LastUpdated time.Time           `json:"last_updated,omitempty"`
	}{{Name: registryName, GitURL: gitURL, Packages: make(map[string][]string)}})

	stdout, _, err := runCommand(t, tempDir, "registry", "delete", registryName, "--force")
	checkOutput(t, stdout, "", fmt.Sprintf("Force deleted registry '%s'\n", registryName), err, false, 0)

	checkRegistriesFile(t, registriesFile, []struct {
		Name        string              `json:"name"`
		GitURL      string              `json:"giturl"`
		Packages    map[string][]string `json:"packages,omitempty"`
		LastUpdated time.Time           `json:"last_updated,omitempty"`
	}{})
}

func TestRegistryDeleteNotFound(t *testing.T) {
	tempDir := t.TempDir()
	registryName := "myreg"
	gitURL := "https://git.example.com"
	registriesFile := setupRegistriesFile(t, tempDir, []struct {
		Name        string              `json:"name"`
		GitURL      string              `json:"giturl"`
		Packages    map[string][]string `json:"packages,omitempty"`
		LastUpdated time.Time           `json:"last_updated,omitempty"`
	}{{Name: "otherreg", GitURL: gitURL, Packages: make(map[string][]string)}})
	dataBefore, _ := os.ReadFile(registriesFile)

	stdout, _, err := runCommand(t, tempDir, "registry", "delete", registryName)
	checkOutput(t, stdout, "", fmt.Sprintf("Error: Registry '%s' not found\n", registryName), err, true, 1)

	dataAfter, _ := os.ReadFile(registriesFile)
	if !bytes.Equal(dataBefore, dataAfter) {
		t.Errorf("registries.json changed unexpectedly")
	}
}

func TestRegistryUpdate(t *testing.T) {
	tempDir := t.TempDir()
	registryName := "myreg"
	gitURL := "https://git.example.com"
	registriesFile := setupRegistriesFile(t, tempDir, []struct {
		Name        string              `json:"name"`
		GitURL      string              `json:"giturl"`
		Packages    map[string][]string `json:"packages,omitempty"`
		LastUpdated time.Time           `json:"last_updated,omitempty"`
	}{{Name: registryName, GitURL: gitURL, Packages: make(map[string][]string)}})

	stdout, _, err := runCommand(t, tempDir, "registry", "update", registryName)
	checkOutput(t, stdout, "", fmt.Sprintf("Updated registry '%s'\n", registryName), err, false, 0)

	data, err := os.ReadFile(registriesFile)
	if err != nil {
		t.Fatalf("Failed to read registries.json: %v", err)
	}
	var registries []struct {
		Name        string              `json:"name"`
		GitURL      string              `json:"giturl"`
		Packages    map[string][]string `json:"packages,omitempty"`
		LastUpdated time.Time           `json:"last_updated,omitempty"`
	}
	if err := json.Unmarshal(data, &registries); err != nil {
		t.Fatalf("Failed to parse registries.json: %v", err)
	}
	if len(registries) != 1 || registries[0].Name != registryName || registries[0].GitURL != gitURL {
		t.Errorf("Registry data corrupted: %+v", registries)
	}
	if registries[0].LastUpdated.IsZero() {
		t.Errorf("Expected LastUpdated to be set, got zero")
	}
}

func TestRegistryUpdateNotFound(t *testing.T) {
	tempDir := t.TempDir()
	registryName := "myreg"
	gitURL := "https://git.example.com"
	registriesFile := setupRegistriesFile(t, tempDir, []struct {
		Name        string              `json:"name"`
		GitURL      string              `json:"giturl"`
		Packages    map[string][]string `json:"packages,omitempty"`
		LastUpdated time.Time           `json:"last_updated,omitempty"`
	}{{Name: "otherreg", GitURL: gitURL, Packages: make(map[string][]string)}})
	dataBefore, _ := os.ReadFile(registriesFile)

	stdout, _, err := runCommand(t, tempDir, "registry", "update", registryName)
	checkOutput(t, stdout, "", fmt.Sprintf("Error: Registry '%s' not found\n", registryName), err, true, 1)

	dataAfter, _ := os.ReadFile(registriesFile)
	if !bytes.Equal(dataBefore, dataAfter) {
		t.Errorf("registries.json changed unexpectedly")
	}
}

func TestRegistryUpdateAll(t *testing.T) {
	tempDir := t.TempDir()
	registriesFile := setupRegistriesFile(t, tempDir, []struct {
		Name        string              `json:"name"`
		GitURL      string              `json:"giturl"`
		Packages    map[string][]string `json:"packages,omitempty"`
		LastUpdated time.Time           `json:"last_updated,omitempty"`
	}{
		{Name: "reg1", GitURL: "https://git.example.com/reg1", Packages: make(map[string][]string)},
		{Name: "reg2", GitURL: "https://git.example.com/reg2", Packages: make(map[string][]string)},
	})

	stdout, _, err := runCommand(t, tempDir, "registry", "update", "--all")
	checkOutput(t, stdout, "", "Updated all registries\n", err, false, 0)

	data, err := os.ReadFile(registriesFile)
	if err != nil {
		t.Fatalf("Failed to read registries.json: %v", err)
	}
	var registries []struct {
		Name        string              `json:"name"`
		GitURL      string              `json:"giturl"`
		Packages    map[string][]string `json:"packages,omitempty"`
		LastUpdated time.Time           `json:"last_updated,omitempty"`
	}
	if err := json.Unmarshal(data, &registries); err != nil {
		t.Fatalf("Failed to parse registries.json: %v", err)
	}
	if len(registries) != 2 {
		t.Errorf("Expected 2 registries, got %d", len(registries))
	}
	for _, reg := range registries {
		if reg.LastUpdated.IsZero() {
			t.Errorf("Expected LastUpdated to be set for %s, got zero", reg.Name)
		}
	}
}

func TestRegistryAddNew(t *testing.T) {
	tempDir := t.TempDir()
	registryName := "myreg"
	packageName := "mypkg"
	versionTag := "v1.0.0"
	gitURL := "https://git.example.com/myreg"

	stdout, _, err := runCommand(t, tempDir, "registry", "add", registryName, packageName, versionTag, gitURL)
	checkOutput(t, stdout, "", fmt.Sprintf("Added version '%s' to package '%s' in registry '%s' from %s\n", versionTag, packageName, registryName, gitURL), err, false, 0)

	checkRegistriesFile(t, filepath.Join(tempDir, ".cosm", "registries.json"), []struct {
		Name        string              `json:"name"`
		GitURL      string              `json:"giturl"`
		Packages    map[string][]string `json:"packages,omitempty"`
		LastUpdated time.Time           `json:"last_updated,omitempty"`
	}{{Name: registryName, GitURL: gitURL, Packages: map[string][]string{packageName: {versionTag}}}})
}

func TestRegistryAddExisting(t *testing.T) {
	tempDir := t.TempDir()
	registryName := "myreg"
	packageName := "mypkg"
	versionTag := "v1.0.0"
	gitURL := "https://git.example.com/myreg"
	registriesFile := setupRegistriesFile(t, tempDir, []struct {
		Name        string              `json:"name"`
		GitURL      string              `json:"giturl"`
		Packages    map[string][]string `json:"packages,omitempty"`
		LastUpdated time.Time           `json:"last_updated,omitempty"`
	}{{Name: registryName, GitURL: "https://old.git.example.com", Packages: make(map[string][]string)}})

	stdout, _, err := runCommand(t, tempDir, "registry", "add", registryName, packageName, versionTag, gitURL)
	checkOutput(t, stdout, "", fmt.Sprintf("Added version '%s' to package '%s' in registry '%s' from %s\n", versionTag, packageName, registryName, gitURL), err, false, 0)

	checkRegistriesFile(t, registriesFile, []struct {
		Name        string              `json:"name"`
		GitURL      string              `json:"giturl"`
		Packages    map[string][]string `json:"packages,omitempty"`
		LastUpdated time.Time           `json:"last_updated,omitempty"`
	}{{Name: registryName, GitURL: gitURL, Packages: map[string][]string{packageName: {versionTag}}}})
}

func TestRegistryAddDuplicateVersion(t *testing.T) {
	tempDir := t.TempDir()
	registryName := "myreg"
	packageName := "mypkg"
	versionTag := "v1.0.0"
	gitURL := "https://git.example.com/myreg"
	registriesFile := setupRegistriesFile(t, tempDir, []struct {
		Name        string              `json:"name"`
		GitURL      string              `json:"giturl"`
		Packages    map[string][]string `json:"packages,omitempty"`
		LastUpdated time.Time           `json:"last_updated,omitempty"`
	}{{Name: registryName, GitURL: gitURL, Packages: map[string][]string{packageName: {versionTag}}}})
	dataBefore, _ := os.ReadFile(registriesFile)

	stdout, _, err := runCommand(t, tempDir, "registry", "add", registryName, packageName, versionTag, gitURL)
	checkOutput(t, stdout, "", fmt.Sprintf("Error: Version '%s' already exists in registry '%s' for package '%s'\n", versionTag, registryName, packageName), err, true, 1)

	dataAfter, _ := os.ReadFile(registriesFile)
	if !bytes.Equal(dataBefore, dataAfter) {
		t.Errorf("registries.json changed unexpectedly")
	}
}

func TestRegistryRmVersion(t *testing.T) {
	tempDir := t.TempDir()
	registryName := "myreg"
	packageName := "mypkg"
	versionTag := "v1.0.0"
	gitURL := "https://git.example.com/myreg"
	registriesFile := setupRegistriesFile(t, tempDir, []struct {
		Name        string              `json:"name"`
		GitURL      string              `json:"giturl"`
		Packages    map[string][]string `json:"packages,omitempty"`
		LastUpdated time.Time           `json:"last_updated,omitempty"`
	}{{Name: registryName, GitURL: gitURL, Packages: map[string][]string{packageName: {versionTag, "v1.1.0"}}}})

	stdout, _, err := runCommand(t, tempDir, "registry", "rm", registryName, packageName, versionTag)
	checkOutput(t, stdout, "", fmt.Sprintf("Removed version '%s' from package '%s' in registry '%s'\n", versionTag, packageName, registryName), err, false, 0)

	checkRegistriesFile(t, registriesFile, []struct {
		Name        string              `json:"name"`
		GitURL      string              `json:"giturl"`
		Packages    map[string][]string `json:"packages,omitempty"`
		LastUpdated time.Time           `json:"last_updated,omitempty"`
	}{{Name: registryName, GitURL: gitURL, Packages: map[string][]string{packageName: {"v1.1.0"}}}})
}

func TestRegistryRmVersionForce(t *testing.T) {
	tempDir := t.TempDir()
	registryName := "myreg"
	packageName := "mypkg"
	versionTag := "v1.0.0"
	gitURL := "https://git.example.com/myreg"
	registriesFile := setupRegistriesFile(t, tempDir, []struct {
		Name        string              `json:"name"`
		GitURL      string              `json:"giturl"`
		Packages    map[string][]string `json:"packages,omitempty"`
		LastUpdated time.Time           `json:"last_updated,omitempty"`
	}{{Name: registryName, GitURL: gitURL, Packages: map[string][]string{packageName: {versionTag, "v1.1.0"}}}})

	stdout, _, err := runCommand(t, tempDir, "registry", "rm", registryName, packageName, versionTag, "--force")
	checkOutput(t, stdout, "", fmt.Sprintf("Force removed version '%s' from package '%s' in registry '%s'\n", versionTag, packageName, registryName), err, false, 0)

	checkRegistriesFile(t, registriesFile, []struct {
		Name        string              `json:"name"`
		GitURL      string              `json:"giturl"`
		Packages    map[string][]string `json:"packages,omitempty"`
		LastUpdated time.Time           `json:"last_updated,omitempty"`
	}{{Name: registryName, GitURL: gitURL, Packages: map[string][]string{packageName: {"v1.1.0"}}}})
}

func TestRegistryRmPackage(t *testing.T) {
	tempDir := t.TempDir()
	registryName := "myreg"
	packageName := "mypkg"
	gitURL := "https://git.example.com/myreg"
	registriesFile := setupRegistriesFile(t, tempDir, []struct {
		Name        string              `json:"name"`
		GitURL      string              `json:"giturl"`
		Packages    map[string][]string `json:"packages,omitempty"`
		LastUpdated time.Time           `json:"last_updated,omitempty"`
	}{{Name: registryName, GitURL: gitURL, Packages: map[string][]string{packageName: {"v1.0.0", "v1.1.0"}, "otherpkg": {"v2.0.0"}}}})

	stdout, _, err := runCommand(t, tempDir, "registry", "rm", registryName, packageName)
	checkOutput(t, stdout, "", fmt.Sprintf("Removed package '%s' from registry '%s'\n", packageName, registryName), err, false, 0)

	checkRegistriesFile(t, registriesFile, []struct {
		Name        string              `json:"name"`
		GitURL      string              `json:"giturl"`
		Packages    map[string][]string `json:"packages,omitempty"`
		LastUpdated time.Time           `json:"last_updated,omitempty"`
	}{{Name: registryName, GitURL: gitURL, Packages: map[string][]string{"otherpkg": {"v2.0.0"}}}})
}

func TestRegistryRmPackageForce(t *testing.T) {
	tempDir := t.TempDir()
	registryName := "myreg"
	packageName := "mypkg"
	gitURL := "https://git.example.com/myreg"
	registriesFile := setupRegistriesFile(t, tempDir, []struct {
		Name        string              `json:"name"`
		GitURL      string              `json:"giturl"`
		Packages    map[string][]string `json:"packages,omitempty"`
		LastUpdated time.Time           `json:"last_updated,omitempty"`
	}{{Name: registryName, GitURL: gitURL, Packages: map[string][]string{packageName: {"v1.0.0", "v1.1.0"}, "otherpkg": {"v2.0.0"}}}})

	stdout, _, err := runCommand(t, tempDir, "registry", "rm", registryName, packageName, "--force")
	checkOutput(t, stdout, "", fmt.Sprintf("Force removed package '%s' from registry '%s'\n", packageName, registryName), err, false, 0)

	checkRegistriesFile(t, registriesFile, []struct {
		Name        string              `json:"name"`
		GitURL      string              `json:"giturl"`
		Packages    map[string][]string `json:"packages,omitempty"`
		LastUpdated time.Time           `json:"last_updated,omitempty"`
	}{{Name: registryName, GitURL: gitURL, Packages: map[string][]string{"otherpkg": {"v2.0.0"}}}})
}

func TestRegistryRmVersionNotFound(t *testing.T) {
	tempDir := t.TempDir()
	registryName := "myreg"
	packageName := "mypkg"
	versionTag := "v1.0.0"
	gitURL := "https://git.example.com/myreg"
	registriesFile := setupRegistriesFile(t, tempDir, []struct {
		Name        string              `json:"name"`
		GitURL      string              `json:"giturl"`
		Packages    map[string][]string `json:"packages,omitempty"`
		LastUpdated time.Time           `json:"last_updated,omitempty"`
	}{{Name: registryName, GitURL: gitURL, Packages: map[string][]string{packageName: {"v1.1.0"}}}})
	dataBefore, _ := os.ReadFile(registriesFile)

	stdout, _, err := runCommand(t, tempDir, "registry", "rm", registryName, packageName, versionTag)
	checkOutput(t, stdout, "", fmt.Sprintf("Error: Version '%s' not found for package '%s' in registry '%s'\n", versionTag, packageName, registryName), err, true, 1)

	dataAfter, _ := os.ReadFile(registriesFile)
	if !bytes.Equal(dataBefore, dataAfter) {
		t.Errorf("registries.json changed unexpectedly")
	}
}

func TestRegistryRmPackageNotFound(t *testing.T) {
	tempDir := t.TempDir()
	registryName := "myreg"
	packageName := "mypkg"
	gitURL := "https://git.example.com/myreg"
	registriesFile := setupRegistriesFile(t, tempDir, []struct {
		Name        string              `json:"name"`
		GitURL      string              `json:"giturl"`
		Packages    map[string][]string `json:"packages,omitempty"`
		LastUpdated time.Time           `json:"last_updated,omitempty"`
	}{{Name: registryName, GitURL: gitURL, Packages: map[string][]string{"otherpkg": {"v2.0.0"}}}})
	dataBefore, _ := os.ReadFile(registriesFile)

	stdout, _, err := runCommand(t, tempDir, "registry", "rm", registryName, packageName)
	checkOutput(t, stdout, "", fmt.Sprintf("Error: Package '%s' not found in registry '%s'\n", packageName, registryName), err, true, 1)

	dataAfter, _ := os.ReadFile(registriesFile)
	if !bytes.Equal(dataBefore, dataAfter) {
		t.Errorf("registries.json changed unexpectedly")
	}
}
