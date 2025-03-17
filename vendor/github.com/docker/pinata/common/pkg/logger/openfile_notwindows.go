//go:build !windows

package logger

import "os"

func openFile(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
}
