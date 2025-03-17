//go:build !windows

package system

import (
	"fmt"
	"syscall"
)

func GetDiskUsage(path string) (DiskUsage, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return DiskUsage{}, fmt.Errorf(failedToGetDiskInfoErr, err)
	}
	return DiskUsage{
		Path:  path,
		Free:  BytesToMiB(stat.Bavail * uint64(stat.Bsize)),
		Total: BytesToMiB(stat.Blocks * uint64(stat.Bsize)),
	}, nil
}
