package system

const failedToGetDiskInfoErr = "failed to get disk info: %w"

const oneMiB = uint64(1024 * 1024)

func BytesToMiB(bytes uint64) uint64 {
	return bytes / oneMiB
}

type DiskUsage struct {
	Path  string
	Free  uint64
	Total uint64
}
