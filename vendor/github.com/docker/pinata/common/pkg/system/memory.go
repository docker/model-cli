package system

import (
	"github.com/shirou/gopsutil/v3/mem"
)

type Memory struct {
	Total uint64
	Free  uint64
	Used  uint64
}

func MemoryInfo() Memory {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return Memory{}
	}
	return Memory{
		Total: BytesToMiB(vmStat.Total),
		Free:  BytesToMiB(vmStat.Free),
		Used:  BytesToMiB(vmStat.Used),
	}
}
