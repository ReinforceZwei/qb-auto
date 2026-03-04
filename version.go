//go:build ignore

package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const usage = `Usage:
  go run version.go                  Print current version
  go run version.go [flags] <bump>   Bump version and create git tag

Arguments:
  bump    major | minor | patch | <explicit version e.g. v1.2.3>

Flags:
  -p, --push   Push the new tag to origin after creating it
`

func main() {
	var push bool
	flag.BoolVar(&push, "p", false, "Push new tag to origin")
	flag.BoolVar(&push, "push", false, "Push new tag to origin")
	flag.Usage = func() { fmt.Print(usage) }
	flag.Parse()

	current := currentVersion()

	if flag.NArg() == 0 {
		fmt.Printf("Current version: %s\n\n", current)
		fmt.Print(usage)
		os.Exit(0)
	}

	arg := flag.Arg(0)
	next, err := bump(current, arg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := createTag(next); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating tag: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("%s → %s\n", current, next)

	if push {
		if err := pushTag(next); err != nil {
			fmt.Fprintf(os.Stderr, "Error pushing tag: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Tag %s pushed to origin.\n", next)
	} else {
		fmt.Printf("To push: git push origin %s\n", next)
	}
}

// currentVersion returns the latest git tag, or "v0.0.0" if none exist.
func currentVersion() string {
	out, err := exec.Command("git", "describe", "--tags", "--abbrev=0").Output()
	if err != nil {
		return "v0.0.0"
	}
	return strings.TrimSpace(string(out))
}

// bump computes the next version string given the current version and a bump
// argument (major, minor, patch, or an explicit vX.Y.Z string).
func bump(current, arg string) (string, error) {
	switch arg {
	case "major", "minor", "patch":
		major, minor, patch, err := parse(current)
		if err != nil {
			return "", fmt.Errorf("cannot parse current version %q: %w", current, err)
		}
		switch arg {
		case "major":
			major++
			minor, patch = 0, 0
		case "minor":
			minor++
			patch = 0
		case "patch":
			patch++
		}
		return fmt.Sprintf("v%d.%d.%d", major, minor, patch), nil
	default:
		// Treat as explicit version; must start with 'v' and be parseable.
		if !strings.HasPrefix(arg, "v") {
			return "", fmt.Errorf("explicit version must start with 'v' (e.g. v1.2.3), got %q", arg)
		}
		if _, _, _, err := parse(arg); err != nil {
			return "", fmt.Errorf("invalid explicit version %q: %w", arg, err)
		}
		return arg, nil
	}
}

// parse splits a vX.Y.Z string into its three numeric components.
func parse(v string) (major, minor, patch int, err error) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("expected vX.Y.Z format")
	}
	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid major: %w", err)
	}
	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid minor: %w", err)
	}
	patch, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid patch: %w", err)
	}
	return major, minor, patch, nil
}

// createTag creates an annotated git tag for the given version.
func createTag(version string) error {
	cmd := exec.Command("git", "tag", "-a", version, "-m", version)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// pushTag pushes the given tag to the origin remote.
func pushTag(version string) error {
	cmd := exec.Command("git", "push", "origin", version)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
