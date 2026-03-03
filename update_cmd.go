package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ReinforceZwei/qb-auto/update"
	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Check for a new qb-auto release",
		Long:  "Checks GitHub for a newer release and prints the command to install it.",
		RunE:  runUpdateCheck,
	}
	cmd.AddCommand(newUpdateInstallCmd())
	return cmd
}

func newUpdateInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Download and install the latest release",
		Long:  "Downloads the latest release binary from GitHub, verifies its checksum, and atomically replaces the running binary.",
		RunE:  runUpdateInstall,
	}
}

func resolveExecPath() (string, error) {
	p, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("cannot determine binary path: %w", err)
	}
	p, err = filepath.EvalSymlinks(p)
	if err != nil {
		return "", fmt.Errorf("cannot resolve binary symlinks: %w", err)
	}
	return p, nil
}

func runUpdateCheck(_ *cobra.Command, _ []string) error {
	fmt.Println("Checking for updates...")

	release, err := update.FetchLatestRelease()
	if err != nil {
		return err
	}

	if release.TagName == version {
		fmt.Printf("Already up to date (%s).\n", version)
		return nil
	}

	fmt.Printf("Update available: %s (current: %s)\n", release.TagName, version)

	execPath, err := resolveExecPath()
	if err != nil {
		return err
	}

	if update.NeedsSudo(execPath) {
		fmt.Println("The binary directory requires elevated permissions. To install:")
		fmt.Println("  sudo qb-auto update install")
	} else {
		fmt.Println("To install:")
		fmt.Println("  qb-auto update install")
	}
	return nil
}

func runUpdateInstall(_ *cobra.Command, _ []string) error {
	execPath, err := resolveExecPath()
	if err != nil {
		return err
	}

	if update.NeedsSudo(execPath) && os.Getuid() != 0 {
		fmt.Fprintln(os.Stderr, "Error: binary is in a directory that requires elevated permissions.")
		fmt.Fprintln(os.Stderr, "Re-run with: sudo qb-auto update install")
		os.Exit(1)
	}

	fmt.Println("Downloading and installing update...")

	if err := update.Do(execPath, version); err != nil {
		if errors.Is(err, update.ErrUpToDate) {
			fmt.Printf("Already up to date (%s).\n", version)
			return nil
		}
		return err
	}

	fmt.Println("Update complete. Please restart qb-auto to apply the new version.")
	return nil
}
