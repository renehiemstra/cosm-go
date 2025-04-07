// cosm --version
// cosm status
// cosm activate

// cosm registry status <registry name>
// cosm registry init <registry name> <giturl>
// cosm registry clone <giturl>
// cosm registry delete <registry name> [--force]
// cosm registry update <registry name>
// cosm registry update --all
// cosm registry add <registry name> v<version tag> <giturl>
// cosm registry rm <registry name> <package name> [--force]
// cosm registry rm <registry name> <package name> v<version> [--force]

// cosm init <package name>
// cosm init <package name> --language <language>
// cosm init <package name> --template <language/template>
// cosm add <name> v<version>
// cosm rm <name>

// cosm release v<version>
// cosm release --patch
// cosm release --minor
// cosm release --major

// cosm develop <package name>
// cosm free <package name>

// cosm upgrade <name>
// cosm upgrade <name> v<x>
// cosm upgrade <name> v<x.y>
// cosm upgrade <name> v<x.y.z>
// cosm upgrade <name> v<x.y.z-alpha>
// cosm upgrade <name> v<x.y>
// cosm upgrade <name> v<x.y.z>
// cosm upgrade <name> --latest
// cosm upgrade --all
// cosm upgrade --all --latest

// cosm downgrade <name> v<version>

package main

import (
	"cosm/commands"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "cosm",
		Short: "A cosmic package manager",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Welcome to Cosm! Use a subcommand like 'status', 'activate', or 'registry'.")
		},
	}

	var versionFlag bool
	rootCmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "Print the version number")
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if versionFlag {
			commands.PrintVersion() // Call the refactored function
		}
	}

	var statusCmd = &cobra.Command{
		Use:   "status",
		Short: "Show the current cosmic status",
		Run:   commands.Status, // Call from commands package,
	}

	var activateCmd = &cobra.Command{
		Use:   "activate",
		Short: "Activate the current project",
		Run:   commands.Activate,
	}

	var initCmd = &cobra.Command{
		Use:   "init [package-name]",
		Short: "Initialize a new project with a Project.json file",
		Args:  cobra.ExactArgs(1),
		Run:   commands.Init,
	}

	var addCmd = &cobra.Command{
		Use:   "add [name] v<version>",
		Short: "Add a dependency to the project",
		Args:  cobra.ExactArgs(2),
		Run:   commands.Add,
	}

	var rmCmd = &cobra.Command{
		Use:   "rm [name]",
		Short: "Remove a dependency from the project",
		Args:  cobra.ExactArgs(1),
		Run:   commands.Rm,
	}

	var releaseCmd = &cobra.Command{
		Use:   "release v<version>",
		Short: "Update the project version",
		Args:  cobra.MaximumNArgs(1),
		Run:   commands.Release,
	}
	releaseCmd.Flags().Bool("patch", false, "Increment the patch version")
	releaseCmd.Flags().Bool("minor", false, "Increment the minor version")
	releaseCmd.Flags().Bool("major", false, "Increment the major version")

	var developCmd = &cobra.Command{
		Use:   "develop [package-name]",
		Short: "Switch an existing dependency to development mode",
		Args:  cobra.ExactArgs(1),
		Run:   commands.Develop,
	}

	var freeCmd = &cobra.Command{
		Use:   "free [package-name]",
		Short: "Close development mode for an existing dependency",
		Args:  cobra.ExactArgs(1),
		Run:   commands.Free,
	}

	var upgradeCmd = &cobra.Command{
		Use:   "upgrade [name] [v<version>]",
		Short: "Upgrade a dependency or all dependencies",
		Args:  cobra.RangeArgs(0, 2),
		Run:   commands.Upgrade,
	}
	upgradeCmd.Flags().Bool("all", false, "Upgrade all direct dependencies")
	upgradeCmd.Flags().Bool("latest", false, "Use the latest version instead of the latest compatible version")

	var downgradeCmd = &cobra.Command{
		Use:   "downgrade [name] v<version>",
		Short: "Downgrade a dependency to an older version",
		Args:  cobra.ExactArgs(2),
		Run:   commands.Downgrade,
	}

	var registryCmd = &cobra.Command{
		Use:   "registry",
		Short: "Manage package registries",
		Run:   commands.Registry,
	}

	var registryStatusCmd = &cobra.Command{
		Use:   "status [registry-name]",
		Short: "Show contents of a registry",
		Args:  cobra.ExactArgs(1),
		Run:   commands.RegistryStatus,
	}

	var registryInitCmd = &cobra.Command{
		Use:   "init [registry-name] [giturl]",
		Short: "Initialize a new registry",
		Args:  cobra.ExactArgs(2),
		Run:   commands.RegistryInit,
	}

	var registryCloneCmd = &cobra.Command{
		Use:   "clone [giturl]",
		Short: "Clone a registry from a Git URL",
		Args:  cobra.ExactArgs(1),
		Run:   commands.RegistryClone,
	}

	var registryDeleteCmd = &cobra.Command{
		Use:   "delete [registry-name]",
		Short: "Delete a registry",
		Args:  cobra.ExactArgs(1),
		Run:   commands.RegistryDelete,
	}
	registryDeleteCmd.Flags().BoolP("force", "f", false, "Force deletion of the registry")

	var registryUpdateCmd = &cobra.Command{
		Use:   "update [registry-name | --all]",
		Short: "Update and synchronize a registry with its remote",
		Args:  cobra.MaximumNArgs(1),
		Run:   commands.RegistryUpdate,
	}
	registryUpdateCmd.Flags().Bool("all", false, "Update all registries")

	var registryAddCmd = &cobra.Command{
		Use:   "add [registry-name] [package-name] v<version> [giturl]",
		Short: "Register a package version to a registry",
		Args:  cobra.ExactArgs(4),
		Run:   commands.RegistryAdd,
	}

	var registryRmCmd = &cobra.Command{
		Use:   "rm [registry-name] [package-name] [v<version>]",
		Short: "Remove a package or version from a registry",
		Args:  cobra.RangeArgs(2, 3),
		Run:   commands.RegistryRm,
	}
	registryRmCmd.Flags().BoolP("force", "f", false, "Force removal of the package or version")

	registryCmd.AddCommand(registryStatusCmd)
	registryCmd.AddCommand(registryInitCmd)
	registryCmd.AddCommand(registryCloneCmd)
	registryCmd.AddCommand(registryDeleteCmd)
	registryCmd.AddCommand(registryUpdateCmd)
	registryCmd.AddCommand(registryAddCmd)
	registryCmd.AddCommand(registryRmCmd)

	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(activateCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(rmCmd)
	rootCmd.AddCommand(releaseCmd)
	rootCmd.AddCommand(developCmd)
	rootCmd.AddCommand(freeCmd)
	rootCmd.AddCommand(upgradeCmd)
	rootCmd.AddCommand(downgradeCmd)
	rootCmd.AddCommand(registryCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
