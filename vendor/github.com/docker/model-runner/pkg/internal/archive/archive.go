package archive

import (
	"fmt"
	"path/filepath"
	"strings"
)

// CheckRelative returns an error if the filename path escapes dir.
// This is used to protect against path traversal attacks when extracting archives.
// It also rejects absolute filename paths.
func CheckRelative(dir, filename string) (string, error) {
	if filepath.IsAbs(filename) {
		return "", fmt.Errorf("archive path has absolute path: %q", filename)
	}
	target := filepath.Join(dir, filename)
	if resolved, err := filepath.EvalSymlinks(target); err == nil {
		target = resolved
		if resolved, err = filepath.EvalSymlinks(dir); err == nil {
			dir = resolved
		}
	}
	rel, err := filepath.Rel(dir, target)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("archive file %q escapes %q", target, dir)
	}
	return target, nil
}

// CheckSymlink returns an error if the link path escapes dir.
// This is used to protect against path traversal attacks when extracting archives.
// It also rejects absolute linkname paths.
func CheckSymlink(dir, name, linkname string) error {
	if filepath.IsAbs(linkname) {
		return fmt.Errorf("archive path has absolute link: %q", linkname)
	}
	_, err := CheckRelative(dir, filepath.Join(filepath.Dir(name), linkname))
	return err
}
