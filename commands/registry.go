package commands

import (
	"cosm/types"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func Registry(cmd *cobra.Command, args []string) {
	fmt.Println("Registry command requires a subcommand (e.g., 'status', 'init').")
}

// RegistryStatus prints an overview of packages in a registry
func RegistryStatus(cmd *cobra.Command, args []string) {
	registryName := validateStatusArgs(args, cmd)
	cosmDir, err := getCosmDir() // Fixed to handle two return values
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	registriesDir := setupRegistriesDir(cosmDir)
	assertRegistryExists(registriesDir, registryName)
	registry, _ := loadRegistryMetadata(registriesDir, registryName)
	printRegistryStatus(registryName, registry)
}

// RegistryInit initializes a new package registry
func RegistryInit(cmd *cobra.Command, args []string) error { // Changed to return error
	originalDir, registryName, gitURL, registriesDir, err := setupAndParseInitArgs(cmd, args) // Updated to handle error
	if err != nil {
		return err
	}
	registryNames, err := loadAndCheckRegistries(registriesDir, registryName) // Updated to handle error
	if err != nil {
		return err
	}
	registrySubDir, err := cloneAndEnterRegistry(registriesDir, registryName, gitURL, originalDir) // Updated to handle error
	if err != nil {
		return err
	}
	if err := ensureDirectoryEmpty(registrySubDir, gitURL, originalDir); err != nil { // Updated to handle error
		cleanupInit(originalDir, registrySubDir, true)
		return err
	}
	if err := updateRegistriesList(registriesDir, registryNames, registryName, originalDir, registrySubDir); err != nil { // Updated to handle error
		cleanupInit(originalDir, registrySubDir, true)
		return err
	}
	_, err = initializeRegistryMetadata(registrySubDir, registryName, gitURL, originalDir) // Updated to handle error
	if err != nil {
		cleanupInit(originalDir, registrySubDir, true)
		return err
	}
	if err := commitAndPushInitialRegistryChanges(registryName, gitURL, originalDir, registrySubDir); err != nil { // Updated to handle error
		cleanupInit(originalDir, registrySubDir, true)
		return err
	}
	if err := restoreOriginalDir(originalDir, registrySubDir); err != nil { // Updated to handle error
		return err
	}
	fmt.Printf("Initialized registry '%s' with Git URL: %s\n", registryName, gitURL)
	return nil
}

// RegistryAdd adds a package version to a registry
func RegistryAdd(cmd *cobra.Command, args []string) error { // Changed to RunE with error return
	registryName, packageGitURL, cosmDir, registriesDir := parseArgsAndSetup(cmd, args)
	prepareRegistry(registriesDir, registryName)
	registry, registryMetaFile := loadRegistryMetadata(registriesDir, registryName)
	tmpClonePath := clonePackageToTempDir(cosmDir, packageGitURL)
	enterCloneDir(tmpClonePath)
	project, err := validateProjectFile(packageGitURL, tmpClonePath) // Fixed to handle two return values
	if err != nil {
		cleanupTempClone(tmpClonePath)
		return err
	}
	ensurePackageNotRegistered(registry, project.Name, registryName, tmpClonePath)
	validTags := validateAndCollectVersionTags(packageGitURL, project.Version, tmpClonePath)
	packageDir := setupPackageDir(registriesDir, registryName, project.Name, tmpClonePath)
	updatePackageVersions(packageDir, project.Name, project.UUID, packageGitURL, validTags, project, tmpClonePath)
	finalizePackageAddition(cosmDir, tmpClonePath, project.UUID, registriesDir, registryName, project.Name, &registry, registryMetaFile, validTags[0])
	fmt.Printf("Added package '%s' with UUID '%s' to registry '%s'\n", project.Name, project.UUID, registryName)
	return nil
}

// cleanupTempClone removes the temporary clone directory
func cleanupTempClone(tmpClonePath string) {
	if err := os.RemoveAll(tmpClonePath); err != nil {
		fmt.Printf("Warning: Failed to clean up temporary clone directory %s: %v\n", tmpClonePath, err)
	}
}

// clonePackageToTempDir creates a temp clone directly in the clones directory
func clonePackageToTempDir(cosmDir, packageGitURL string) string {
	clonesDir := filepath.Join(cosmDir, "clones")
	if err := os.MkdirAll(clonesDir, 0755); err != nil {
		fmt.Printf("Error creating clones directory: %v\n", err)
		os.Exit(1)
	}
	tmpClonePath := filepath.Join(clonesDir, "tmp-clone-"+uuid.New().String())

	if err := exec.Command("git", "clone", packageGitURL, tmpClonePath).Run(); err != nil {
		cloneOutput, _ := exec.Command("git", "clone", packageGitURL, tmpClonePath).CombinedOutput()
		fmt.Printf("Error cloning package repository at '%s': %v\nOutput: %s\n", packageGitURL, err, cloneOutput)
		cleanupTempClone(tmpClonePath)
		os.Exit(1)
	}
	return tmpClonePath
}

// moveCloneToPermanentDir moves the cloned directory to its permanent location, replacing any existing clone
func moveCloneToPermanentDir(cosmDir, tmpClonePath, packageUUID string) string {
	clonesDir := filepath.Join(cosmDir, "clones")
	packageClonePath := filepath.Join(clonesDir, packageUUID)

	// If the permanent clone directory already exists, remove it
	if _, err := os.Stat(packageClonePath); !os.IsNotExist(err) {
		if err := os.RemoveAll(packageClonePath); err != nil {
			fmt.Printf("Error removing existing clone at %s: %v\n", packageClonePath, err)
			cleanupTempClone(tmpClonePath)
			os.Exit(1)
		}
		fmt.Printf("Warning: Replaced existing clone for UUID '%s' at %s\n", packageUUID, packageClonePath)
	}

	// Move the temporary clone to the permanent location
	if err := os.Rename(tmpClonePath, packageClonePath); err != nil {
		fmt.Printf("Error moving package to %s: %v\n", packageClonePath, err)
		cleanupTempClone(tmpClonePath)
		os.Exit(1)
	}
	return packageClonePath
}

// assertRegistryExists verifies that the specified registry exists in registries.json
func assertRegistryExists(registriesDir, registryName string) {
	registriesFile := filepath.Join(registriesDir, "registries.json")
	if _, err := os.Stat(registriesFile); os.IsNotExist(err) {
		fmt.Println("Error: No registries found (run 'cosm registry init' first)")
		os.Exit(1)
	}
	var registryNames []string
	data, err := os.ReadFile(registriesFile)
	if err != nil {
		fmt.Printf("Error reading registries.json: %v\n", err)
		os.Exit(1)
	}
	if err := json.Unmarshal(data, &registryNames); err != nil {
		fmt.Printf("Error parsing registries.json: %v\n", err)
		os.Exit(1)
	}
	registryExists := false
	for _, name := range registryNames {
		if name == registryName {
			registryExists = true
			break
		}
	}
	if !registryExists {
		fmt.Printf("Error: Registry '%s' not found in registries.json\n", registryName)
		os.Exit(1)
	}
}

// pullRegistryUpdates pulls changes from the registry's remote Git repository
func pullRegistryUpdates(registriesDir, registryName string) {
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		os.Exit(1)
	}

	registryDir := filepath.Join(registriesDir, registryName)
	if err := os.Chdir(registryDir); err != nil {
		restoreDirBeforeExit(currentDir)
		fmt.Printf("Error changing to registry directory %s: %v\n", registryDir, err)
		os.Exit(1)
	}

	pullCmd := exec.Command("git", "pull", "origin", "main")
	pullOutput, err := pullCmd.CombinedOutput()
	if err != nil {
		restoreDirBeforeExit(currentDir)
		fmt.Printf("Error pulling updates from registry '%s': %v\nOutput: %s\n", registryName, err, pullOutput)
		os.Exit(1)
	}

	restoreDirBeforeExit(currentDir)
}

// loadRegistryMetadata loads and validates the registry metadata from registry.json
func loadRegistryMetadata(registriesDir, registryName string) (types.Registry, string) {
	registryMetaFile := filepath.Join(registriesDir, registryName, "registry.json")
	data, err := os.ReadFile(registryMetaFile)
	if err != nil {
		fmt.Printf("Error reading registry.json for '%s': %v\n", registryName, err)
		os.Exit(1)
	}
	var registry types.Registry
	if err := json.Unmarshal(data, &registry); err != nil {
		fmt.Printf("Error parsing registry.json for '%s': %v\n", registryName, err)
		os.Exit(1)
	}
	if registry.Packages == nil {
		registry.Packages = make(map[string]string)
	}
	return registry, registryMetaFile
}

// updateRegistryMetadata updates and writes the registry metadata to registry.json
func updateRegistryMetadata(registry *types.Registry, packageName, packageUUID, registryMetaFile string) {
	registry.Packages[packageName] = packageUUID
	data, err := json.MarshalIndent(*registry, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling registry.json: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(registryMetaFile, data, 0644); err != nil {
		fmt.Printf("Error writing registry.json: %v\n", err)
		os.Exit(1)
	}
}

// commitAndPushRegistryChanges commits and pushes changes to the registry's Git repository
func commitAndPushRegistryChanges(registriesDir, registryName, packageName, versionTag string) {
	registryDir := filepath.Join(registriesDir, registryName)
	if err := os.Chdir(registryDir); err != nil {
		fmt.Printf("Error changing to registry directory %s: %v\n", registryDir, err)
		os.Exit(1)
	}

	addCmd := exec.Command("git", "add", ".")
	addOutput, err := addCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error staging changes in registry: %v\nOutput: %s\n", err, addOutput)
		os.Exit(1)
	}

	commitMsg := fmt.Sprintf("Added package %s version %s", packageName, versionTag)
	commitCmd := exec.Command("git", "commit", "-m", commitMsg)
	commitOutput, err := commitCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error committing changes in registry: %v\nOutput: %s\n", err, commitOutput)
		os.Exit(1)
	}

	pushCmd := exec.Command("git", "push", "origin", "main")
	pushOutput, err := pushCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error pushing changes to registry: %v\nOutput: %s\n", err, pushOutput)
		os.Exit(1)
	}
}

// validateProjectFile reads and validates Project.json, returning the project
func validateProjectFile(packageGitURL, tmpClonePath string) (types.Project, error) { // Changed to return (types.Project, error)
	data, err := os.ReadFile("Project.json")
	if err != nil {
		return types.Project{}, fmt.Errorf("repository at '%s' does not contain a Project.json file: %v", packageGitURL, err)
	}
	var project types.Project
	if err := json.Unmarshal(data, &project); err != nil {
		return types.Project{}, fmt.Errorf("invalid Project.json in repository at '%s': %v", packageGitURL, err)
	}
	if project.Name == "" {
		return types.Project{}, fmt.Errorf("Project.json in repository at '%s' does not contain a valid package name", packageGitURL)
	}
	if project.UUID == "" {
		return types.Project{}, fmt.Errorf("Project.json in repository at '%s' does not contain a valid UUID", packageGitURL)
	}
	if _, err := uuid.Parse(project.UUID); err != nil {
		return types.Project{}, fmt.Errorf("invalid UUID '%s' in Project.json at '%s': %v", project.UUID, packageGitURL, err)
	}
	if project.Version == "" {
		return types.Project{}, fmt.Errorf("Project.json at '%s' does not contain a version", packageGitURL)
	}
	// Validate version parsing
	_, err = parseSemVer(project.Version) // Fixed to handle both return values
	if err != nil {
		return types.Project{}, fmt.Errorf("invalid version in Project.json at '%s': %v", packageGitURL, err)
	}
	return project, nil
}

// validateAndCollectVersionTags fetches Git tags, or releases the current version if none exist
func validateAndCollectVersionTags(packageGitURL string, packageVersion string, tmpClonePath string) []string {
	tagOutput, err := exec.Command("git", "tag").CombinedOutput()
	if err != nil || len(strings.TrimSpace(string(tagOutput))) == 0 {
		// No tags found, use Project.json packageVersion and tag it
		if packageVersion == "" {
			fmt.Printf("Error: Project.json at '%s' has no version specified\n", packageGitURL)
			cleanupTempClone(tmpClonePath)
			os.Exit(1)
		}

		// Tag the current version
		if err := exec.Command("git", "tag", packageVersion).Run(); err != nil {
			fmt.Printf("Error tagging version '%s' in repository at '%s': %v\n", packageVersion, packageGitURL, err)
			cleanupTempClone(tmpClonePath)
			os.Exit(1)
		}
		// Push the tag to the remote
		if err := exec.Command("git", "push", "origin", packageVersion).Run(); err != nil {
			fmt.Printf("Error pushing tag '%s' to origin for repository at '%s': %v\n", packageVersion, packageGitURL, err)
			cleanupTempClone(tmpClonePath)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "No valid tags found; released version '%s' from Project.json to repository at '%s'\n", packageVersion, packageGitURL)
		return []string{packageVersion}
	}

	tags := strings.Split(strings.TrimSpace(string(tagOutput)), "\n")
	var validTags []string
	for _, tag := range tags {
		if strings.HasPrefix(tag, "v") && len(strings.Split(tag, ".")) >= 2 {
			validTags = append(validTags, tag)
		}
	}
	if len(validTags) == 0 {
		fmt.Printf("Error: No valid version tags (e.g., vX.Y.Z) found in repository at '%s'\n", packageGitURL)
		cleanupTempClone(tmpClonePath)
		os.Exit(1)
	}
	return validTags
}

// updateVersionsList loads and writes versions.json, updating with new tags
func updateVersionsList(packageDir string, tagsToAdd *[]string, tmpClonePath string) {
	versionsFile := filepath.Join(packageDir, "versions.json")
	var existingVersions []string
	if data, err := os.ReadFile(versionsFile); err == nil {
		if err := json.Unmarshal(data, &existingVersions); err != nil {
			fmt.Printf("Error parsing versions.json at %s: %v\n", versionsFile, err)
			cleanupTempClone(tmpClonePath)
			os.Exit(1)
		}
	}
	for _, versionTag := range *tagsToAdd {
		versionExists := false
		for _, v := range existingVersions {
			if v == versionTag {
				versionExists = true
				break
			}
		}
		if !versionExists {
			existingVersions = append(existingVersions, versionTag)
		}
	}
	data, err := json.MarshalIndent(existingVersions, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling versions.json: %v\n", err)
		cleanupTempClone(tmpClonePath)
		os.Exit(1)
	}
	if err := os.WriteFile(versionsFile, data, 0644); err != nil {
		fmt.Printf("Error writing versions.json: %v\n", err)
		cleanupTempClone(tmpClonePath)
		os.Exit(1)
	}
}

// addPackageVersion adds a single version to the package directory
func addPackageVersion(packageDir, packageName, packageUUID, packageGitURL string, versionTag string, project types.Project, tmpClonePath string) {
	versionDir := filepath.Join(packageDir, versionTag)
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		fmt.Printf("Error creating version directory %s: %v\n", versionDir, err)
		cleanupTempClone(tmpClonePath)
		os.Exit(1)
	}

	sha1Output, err := exec.Command("git", "rev-list", "-n", "1", versionTag).Output()
	if err != nil {
		fmt.Printf("Error getting SHA1 for tag '%s': %v\n", versionTag, err)
		cleanupTempClone(tmpClonePath)
		os.Exit(1)
	}
	sha1 := strings.TrimSpace(string(sha1Output))

	specs := types.Specs{
		Name:    packageName,
		UUID:    packageUUID,
		Version: versionTag,
		GitURL:  packageGitURL,
		SHA1:    sha1,
		Deps:    project.Deps,
	}
	data, err := json.MarshalIndent(specs, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling specs.json for version '%s': %v\n", versionTag, err)
		cleanupTempClone(tmpClonePath)
		os.Exit(1)
	}
	specsFile := filepath.Join(versionDir, "specs.json")
	if err := os.WriteFile(specsFile, data, 0644); err != nil {
		fmt.Printf("Error writing specs.json for version '%s': %v\n", versionTag, err)
		cleanupTempClone(tmpClonePath)
		os.Exit(1)
	}
}

// setupAndParseInitArgs validates arguments and sets up directories for RegistryInit
func setupAndParseInitArgs(cmd *cobra.Command, args []string) (string, string, string, string, error) { // Changed to return error
	originalDir, err := os.Getwd()
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to get original directory: %v", err)
	}

	if len(args) != 2 {
		return "", "", "", "", fmt.Errorf("exactly two arguments required (e.g., cosm registry init <registry name> <giturl>)")
	}
	registryName := args[0]
	gitURL := args[1]

	if registryName == "" {
		return "", "", "", "", fmt.Errorf("registry name cannot be empty")
	}
	if gitURL == "" {
		return "", "", "", "", fmt.Errorf("git URL cannot be empty")
	}

	cosmDir, err := getGlobalCosmDir()
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to get global .cosm directory: %v", err)
	}
	registriesDir := filepath.Join(cosmDir, "registries")
	if err := os.MkdirAll(registriesDir, 0755); err != nil {
		return "", "", "", "", fmt.Errorf("failed to create %s directory: %v", registriesDir, err)
	}

	return originalDir, registryName, gitURL, registriesDir, nil
}

// loadAndCheckRegistries loads registries.json and checks for duplicate registry names
func loadAndCheckRegistries(registriesDir, registryName string) ([]string, error) { // Changed to return ([]string, error)
	registriesFile := filepath.Join(registriesDir, "registries.json")
	var registryNames []string
	if data, err := os.ReadFile(registriesFile); err == nil {
		if err := json.Unmarshal(data, &registryNames); err != nil {
			return nil, fmt.Errorf("failed to parse registries.json: %v", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read registries.json: %v", err)
	}

	for _, name := range registryNames {
		if name == registryName {
			return nil, fmt.Errorf("registry '%s' already exists", registryName)
		}
	}

	return registryNames, nil
}

// cleanupInit reverts to the original directory and removes the registrySubDir if needed
func cleanupInit(originalDir, registrySubDir string, removeDir bool) {
	if err := os.Chdir(originalDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error returning to original directory %s: %v\n", originalDir, err)
		// Don’t exit here; let the caller handle the exit after cleanup
	}
	if removeDir {
		if err := os.RemoveAll(registrySubDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to clean up registry directory %s: %v\n", registrySubDir, err)
		}
	}
}

// cloneAndEnterRegistry clones the repository into registries/<registryName> and changes to it
func cloneAndEnterRegistry(registriesDir, registryName, gitURL, originalDir string) (string, error) { // Changed to return (string, error)
	registrySubDir := filepath.Join(registriesDir, registryName)
	cloneCmd := exec.Command("git", "clone", gitURL, registrySubDir)
	cloneOutput, err := cloneCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to clone repository at '%s' into %s: %v\nOutput: %s", gitURL, registrySubDir, err, cloneOutput)
	}

	// Change to the cloned directory
	if err := os.Chdir(registrySubDir); err != nil {
		return "", fmt.Errorf("failed to change to registry directory %s: %v", registrySubDir, err)
	}
	return registrySubDir, nil
}

// ensureDirectoryEmpty checks if the cloned directory is empty except for .git
func ensureDirectoryEmpty(dir, gitURL, originalDir string) error { // Changed to return error
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %v", dir, err)
	}
	for _, file := range files {
		if file.Name() != ".git" { // Ignore .git directory
			return fmt.Errorf("repository at '%s' cloned into %s is not empty (contains %s)", gitURL, dir, file.Name())
		}
	}
	return nil
}

// updateRegistriesList adds the registry name to registries.json
func updateRegistriesList(registriesDir string, registryNames []string, registryName, originalDir, registrySubDir string) error { // Changed to return error
	registryNames = append(registryNames, registryName)
	data, err := json.MarshalIndent(registryNames, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registries.json: %v", err)
	}
	registriesFile := filepath.Join(registriesDir, "registries.json")
	if err := os.WriteFile(registriesFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write registries.json: %v", err)
	}
	return nil
}

// initializeRegistryMetadata creates and writes the registry.json file
func initializeRegistryMetadata(registrySubDir, registryName, gitURL, originalDir string) (string, error) { // Changed to return (string, error)
	registryMetaFile := filepath.Join(registrySubDir, "registry.json")
	registry := types.Registry{
		Name:     registryName,
		UUID:     uuid.New().String(),
		GitURL:   gitURL,
		Packages: make(map[string]string),
	}
	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal registry.json: %v", err)
	}
	if err := os.WriteFile(registryMetaFile, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write registry.json: %v", err)
	}
	return registryMetaFile, nil
}

// commitAndPushInitialRegistryChanges stages, commits, and pushes the initial registry changes
func commitAndPushInitialRegistryChanges(registryName, gitURL, originalDir, registrySubDir string) error { // Changed to return error
	addCmd := exec.Command("git", "add", "registry.json")
	addOutput, err := addCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stage registry.json: %v\nOutput: %s", err, addOutput)
	}
	commitCmd := exec.Command("git", "commit", "-m", fmt.Sprintf("Initialized registry %s", registryName))
	commitOutput, err := commitCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to commit initial registry setup: %v\nOutput: %s", err, commitOutput)
	}
	pushCmd := exec.Command("git", "push", "origin", "main")
	pushOutput, err := pushCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to push initial commit to %s: %v\nOutput: %s", gitURL, err, pushOutput)
	}
	return nil
}

// restoreOriginalDir returns to the original directory without removing the registry subdir
func restoreOriginalDir(originalDir, registrySubDir string) error { // Changed to return error
	if err := os.Chdir(originalDir); err != nil {
		return fmt.Errorf("failed to return to original directory %s: %v", originalDir, err)
	}
	return nil
}

// parseArgsAndSetup validates arguments and sets up directories for RegistryAdd
func parseArgsAndSetup(cmd *cobra.Command, args []string) (string, string, string, string) {
	if len(args) != 2 {
		fmt.Println("Error: Exactly two arguments required (e.g., cosm registry add <registry_name> <package giturl>)")
		cmd.Usage()
		os.Exit(1)
	}
	registryName := args[0]
	packageGitURL := args[1]

	if registryName == "" {
		fmt.Println("Error: Registry name cannot be empty")
		os.Exit(1)
	}
	if packageGitURL == "" {
		fmt.Println("Error: Package Git URL cannot be empty")
		os.Exit(1)
	}

	cosmDir, err := getGlobalCosmDir()
	if err != nil {
		fmt.Printf("Error getting global .cosm directory: %v\n", err)
		os.Exit(1)
	}
	registriesDir := filepath.Join(cosmDir, "registries")

	return registryName, packageGitURL, cosmDir, registriesDir
}

// prepareRegistry ensures the registry exists and is up-to-date
func prepareRegistry(registriesDir, registryName string) {
	assertRegistryExists(registriesDir, registryName)
	pullRegistryUpdates(registriesDir, registryName)
}

// enterCloneDir changes to the temporary clone directory
func enterCloneDir(tmpClonePath string) {
	if err := os.Chdir(tmpClonePath); err != nil {
		fmt.Printf("Error changing to cloned directory: %v\n", err)
		cleanupTempClone(tmpClonePath)
		os.Exit(1)
	}
}

// ensurePackageNotRegistered checks if the package is already in the registry
func ensurePackageNotRegistered(registry types.Registry, packageName, registryName, tmpClonePath string) {
	if _, exists := registry.Packages[packageName]; exists {
		fmt.Fprintf(os.Stderr, "Error: Package '%s' is already registered in registry '%s'\n", packageName, registryName)
		cleanupTempClone(tmpClonePath)
		os.Exit(1)
	}
}

// setupPackageDir creates the package directory structure
func setupPackageDir(registriesDir, registryName, packageName, tmpClonePath string) string {
	packageFirstLetter := strings.ToUpper(string(packageName[0]))
	packageDir := filepath.Join(registriesDir, registryName, packageFirstLetter, packageName)
	if err := os.MkdirAll(packageDir, 0755); err != nil {
		fmt.Printf("Error creating package directory %s: %v\n", packageDir, err)
		cleanupTempClone(tmpClonePath)
		os.Exit(1)
	}
	return packageDir
}

// updatePackageVersions updates the versions list and adds version specs
func updatePackageVersions(packageDir, packageName, packageUUID, packageGitURL string, validTags []string, project types.Project, tmpClonePath string) {
	updateVersionsList(packageDir, &validTags, tmpClonePath)
	for _, versionTag := range validTags {
		addPackageVersion(packageDir, packageName, packageUUID, packageGitURL, versionTag, project, tmpClonePath)
	}
}

// finalizePackageAddition completes the package addition process
func finalizePackageAddition(cosmDir, tmpClonePath, packageUUID, registriesDir, registryName, packageName string, registry *types.Registry, registryMetaFile string, firstVersionTag string) {
	moveCloneToPermanentDir(cosmDir, tmpClonePath, packageUUID)
	updateRegistryMetadata(registry, packageName, packageUUID, registryMetaFile)
	commitAndPushRegistryChanges(registriesDir, registryName, packageName, firstVersionTag)
}

// validateStatusArgs checks the command-line arguments for validity
func validateStatusArgs(args []string, cmd *cobra.Command) string {
	if len(args) != 1 {
		fmt.Println("Error: Exactly one argument required (e.g., cosm registry status <registry_name>)")
		cmd.Usage()
		os.Exit(1)
	}
	registryName := args[0]
	if registryName == "" {
		fmt.Println("Error: Registry name cannot be empty")
		os.Exit(1)
	}
	return registryName
}

// setupRegistriesDir constructs the registries directory path
func setupRegistriesDir(cosmDir string) string {
	return filepath.Join(cosmDir, "registries")
}

// printRegistryStatus displays the registry's package information
func printRegistryStatus(registryName string, registry types.Registry) {
	fmt.Printf("Registry Status for '%s':\n", registryName)
	if len(registry.Packages) == 0 {
		fmt.Println("  No packages registered.")
	} else {
		fmt.Println("  Packages:")
		for pkgName, pkgUUID := range registry.Packages {
			fmt.Printf("    - %s (UUID: %s)\n", pkgName, pkgUUID)
		}
	}
}

func RegistryClone(cmd *cobra.Command, args []string) {
}

func RegistryDelete(cmd *cobra.Command, args []string) {
}

func RegistryUpdate(cmd *cobra.Command, args []string) {
}

func RegistryRm(cmd *cobra.Command, args []string) {
}
