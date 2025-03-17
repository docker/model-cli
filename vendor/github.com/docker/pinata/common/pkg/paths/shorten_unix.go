//go:build !windows

package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

// ShortenUnixSocketPath converts a long absolute path into a shorter path relative
// to the current working directory to work around size limits in the socket address
// struct.
func ShortenUnixSocketPath(path string) (string, error) {
	if len(path) <= maxUnixSocketPathLen {
		return path, nil
	}

	// absolute path is too long, attempt to use a relative path
	p, err := relative(path)
	if err != nil {
		return "", err
	}

	if len(p) > maxUnixSocketPathLen {
		return "", fmt.Errorf("absolute and relative socket path %s longer than %d characters", p, maxUnixSocketPathLen)
	}
	return p, nil
}

func relative(p string) (string, error) {
	path2 := filepath.Dir(p)
	if _, err := os.Stat(path2); err == nil {
		path2, err = filepath.EvalSymlinks(path2)
		if err != nil {
			return "", err
		}
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir2, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(dir2, path2)
	if err != nil {
		return "", err
	}
	return filepath.Join(rel, filepath.Base(p)), nil
}
