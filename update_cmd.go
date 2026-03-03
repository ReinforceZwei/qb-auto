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
	return &cobra.Command{
		Use:   "update",
		Short: "Update qb-auto to the latest release",
		Long:  "Downloads the latest release binary from GitHub, verifies its checksum, and replaces the running binary.",
		RunE:  runUpdate,
	}
}

func runUpdate(_ *cobra.Command, _ []string) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine binary path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("cannot resolve binary symlinks: %w", err)
	}

	if update.NeedsSudo(execPath) && os.Getuid() != 0 {
		fmt.Fprintln(os.Stderr, "Error: binary is in a directory that requires elevated permissions.")
		fmt.Fprintln(os.Stderr, "Re-run with: sudo qb-auto update")
		os.Exit(1)
	}

	fmt.Println("Checking for updates...")

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
