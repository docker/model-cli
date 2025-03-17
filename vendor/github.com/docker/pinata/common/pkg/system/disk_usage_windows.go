package system

import (
	"fmt"
	"path/filepath"

	"golang.org/x/sys/windows"
)

func GetDiskUsage(path string) (DiskUsage, error) {
	var freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes uint64
	if err := windows.GetDiskFreeSpaceEx(
		windows.StringToUTF16Ptr(filepath.VolumeName(path)),
		&freeBytesAvailable,
		&totalNumberOfBytes,
		&totalNumberOfFreeBytes); err != nil {
		return DiskUsage{}, fmt.Errorf(failedToGetDiskInfoErr, err)
	}
	return DiskUsage{
		Path:  path,
		Free:  BytesToMiB(freeBytesAvailable),
		Total: BytesToMiB(totalNumberOfBytes),
	}, nil
}
