package logger

import (
	"os"
	"strings"
	"syscall"
)

// openFile on Windows setting the sharing mode to allow the file to be re-opened.
func openFile(path string) (*os.File, error) {
	path = longPath(path)
	pathp, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	h, err := syscall.CreateFile(pathp, openWrite, shareReadWriteDelete, nil, dispositionOpenAlways, 0, 0)
	if s, ok := err.(syscall.Errno); ok && s == errInvalidName {
		// The filename, directory name, or volume label syntax is incorrect.
		return nil, os.ErrNotExist
	}
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(h), path)
	if _, err := f.Seek(0, seekRelativeToEnd); err != nil {
		return nil, err
	}
	return f, nil
}

const (
	shareReadWriteDelete  = uint32(syscall.FILE_SHARE_READ | syscall.FILE_SHARE_WRITE | syscall.FILE_SHARE_DELETE)
	openWrite             = uint32(syscall.GENERIC_WRITE)
	dispositionOpenAlways = uint32(syscall.OPEN_ALWAYS)
	errInvalidName        = 123 // filename is invalid, e.g. "foo:"
	seekRelativeToEnd     = 2
)

// don't break if the path is > 260 chars
func longPath(path string) string {
	if !strings.HasPrefix(path, `\\?\`) {
		return `\\?\` + path
	}
	return path
}
