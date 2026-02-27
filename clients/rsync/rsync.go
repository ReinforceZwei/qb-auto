package rsync

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/ReinforceZwei/qb-auto/config"
)

// Client wraps the rsync binary to copy files to a remote NAS via the rsync
// daemon protocol (rsync://).
type Client struct {
	binaryPath   string
	host         string
	port         int
	module       string
	user         string
	passwordFile string
}

// NewClient creates a Client from application config.
func NewClient(cfg *config.Config) *Client {
	return &Client{
		binaryPath:   cfg.RsyncBinaryPath,
		host:         cfg.RsyncHost,
		port:         cfg.RsyncPort,
		module:       cfg.RsyncModule,
		user:         cfg.RsyncUser,
		passwordFile: cfg.RsyncPasswordFile,
	}
}

// Copy transfers src (a local path) into dest (a path relative to the rsync
// module on the NAS). The source folder itself is copied into dest, preserving
// its name as the final path segment.
//
// Example:
//
//	src  = "/mnt/4tb/pool/anime/[LoliHouse] Acro Trip [01-12][...]"
//	dest = "anime/Acro Trip 頂尖惡路"
//
// Result on NAS: <module>/<dest>/<src-folder-name>
//
//	→ anime/anime/Acro Trip 頂尖惡路/[LoliHouse] Acro Trip [01-12][...]
func (c *Client) Copy(ctx context.Context, src, dest string) error {
	destURL := fmt.Sprintf(
		"rsync://%s@%s:%d/%s/%s/",
		c.user,
		c.host,
		c.port,
		c.module,
		strings.Trim(dest, "/"),
	)

	args := []string{
		"-a",
		"--password-file=" + c.passwordFile,
		src,
		destURL,
	}

	cmd := exec.CommandContext(ctx, c.binaryPath, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return fmt.Errorf("rsync: %w: %s", err, msg)
		}
		return fmt.Errorf("rsync: %w", err)
	}

	return nil
}
