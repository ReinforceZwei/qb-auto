package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// systemd template unit content. %%i produces a literal %i in output,
// which is the systemd instance specifier (the username after @).
const systemdUnitTemplate = `[Unit]
Description=qb-auto torrent automation service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=%%i
ExecStart=%s serve
Restart=on-failure
RestartSec=10s

[Install]
WantedBy=multi-user.target
`

const unitFilePath = "/etc/systemd/system/qb-auto@.service"
const systemBinPath = "/usr/local/bin/qb-auto"

func newInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install systemd service for qb-auto",
		Long:  "Interactively installs a systemd template unit for qb-auto so it can be managed with 'sudo systemctl start qb-auto@<user>'.",
		RunE:  runInstall,
	}
}

func runInstall(_ *cobra.Command, _ []string) error {
	reader := bufio.NewReader(os.Stdin)

	// 1. Root check — writing to /etc/systemd/system requires root.
	if os.Getuid() != 0 {
		fmt.Fprintln(os.Stderr, "Error: this command must be run as root.")
		fmt.Fprintln(os.Stderr, "Re-run with: sudo qb-auto install")
		os.Exit(1)
	}

	// 2. Resolve the current binary path, following any symlinks.
	binaryPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine binary path: %w", err)
	}
	binaryPath, err = filepath.EvalSymlinks(binaryPath)
	if err != nil {
		return fmt.Errorf("cannot resolve binary symlinks: %w", err)
	}

	// 3. Check if the binary is reachable via $PATH.
	execPath := binaryPath
	if _, lookErr := exec.LookPath("qb-auto"); lookErr != nil {
		fmt.Printf("qb-auto is not found in $PATH (current location: %s)\n", binaryPath)
		fmt.Println("  [c] Copy to /usr/local/bin/qb-auto (recommended)")
		fmt.Println("  [k] Keep current path in service unit")
		fmt.Print("Choice [c/k]: ")

		choice := readLine(reader)
		if choice == "c" || choice == "" {
			fmt.Printf("Copying %s → %s\n", binaryPath, systemBinPath)
			if err := copyFile(binaryPath, systemBinPath, 0755); err != nil {
				return fmt.Errorf("failed to copy binary: %w", err)
			}
			execPath = systemBinPath
			fmt.Println("Binary copied.")
		}
	}

	// 4. Render and write the systemd unit file.
	unitContent := fmt.Sprintf(systemdUnitTemplate, execPath)
	fmt.Printf("\nWriting unit file: %s\n", unitFilePath)
	if err := os.WriteFile(unitFilePath, []byte(unitContent), 0644); err != nil {
		return fmt.Errorf("failed to write unit file: %w", err)
	}

	// 5. Reload systemd so it picks up the new unit.
	fmt.Println("Running: systemctl daemon-reload")
	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl daemon-reload failed: %w\n%s", err, string(out))
	}

	fmt.Printf("\nUnit file installed: %s\n", unitFilePath)
	fmt.Println("Daemon reloaded.")

	// 6. Determine the target username. $SUDO_USER is set when running via sudo.
	username := os.Getenv("SUDO_USER")
	if username == "" {
		fmt.Print("\nCould not detect user from $SUDO_USER. Enter the username to configure the service for: ")
		username = readLine(reader)
	}

	if username == "" {
		fmt.Println("\nNo username provided — skipping enable step.")
		printUsage(username)
		return nil
	}

	// 7. Ask the user whether to enable and start the service now.
	fmt.Printf("\nEnable and start qb-auto@%s now?\n", username)
	fmt.Printf("  Will run: systemctl enable --now qb-auto@%s\n", username)
	fmt.Print("Proceed? [Y/n]: ")

	answer := readLine(reader)
	if answer == "y" || answer == "" {
		fmt.Printf("Running: systemctl enable --now qb-auto@%s\n", username)
		if out, err := exec.Command("systemctl", "enable", "--now", "qb-auto@"+username).CombinedOutput(); err != nil {
			return fmt.Errorf("systemctl enable failed: %w\n%s", err, string(out))
		}
		fmt.Println("Service enabled and started.")
	}

	printUsage(username)
	return nil
}

func printUsage(username string) {
	if username == "" {
		username = "<user>"
	}
	fmt.Printf(`
Manage your service:
  sudo systemctl start   qb-auto@%s
  sudo systemctl stop    qb-auto@%s
  sudo systemctl status  qb-auto@%s
  sudo systemctl enable  qb-auto@%s   # auto-start on boot
`, username, username, username, username)
}

func readLine(r *bufio.Reader) string {
	line, _ := r.ReadString('\n')
	return strings.TrimSpace(strings.ToLower(line))
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
