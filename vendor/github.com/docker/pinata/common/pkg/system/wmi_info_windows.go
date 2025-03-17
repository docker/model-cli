package system

import (
	"fmt"

	"github.com/yusufpapurcu/wmi"
)

type Win32_Process struct {
	Name           string
	ProcessID      uint32
	UserModeTime   uint64 // CPU time in user mode 100 nanoseconds units
	KernelModeTime uint64 // CPU time in kernel mode 100 nanoseconds units
	WorkingSetSize uint64 // Memory usage
	CreationDate   string // Time at which the process was created (DATETIME format)
}

func GetWin32_Process(names []string) (Win32_Process, error) {
	var ret []Win32_Process
	query := ""
	for idx, name := range names {
		if idx == 0 {
			query = "WHERE "
		} else {
			query += " OR "
		}
		query += "Name = '" + name + "'"
	}
	q := wmi.CreateQuery(&ret, query)
	err := wmi.Query(q, &ret)
	if err != nil {
		return Win32_Process{}, fmt.Errorf("querying WMI process '%s': %w", names, err)
	}
	// Pick the process with the priority defined by the order in the names
	for _, name := range names {
		for _, process := range ret {
			if process.Name == name {
				return process, nil
			}
		}
	}
	return Win32_Process{}, fmt.Errorf("querying WMI process '%s' : unexpected result : %v", names, ret)
}

func GetWin32_ProcessByPID(pid int) (Win32_Process, error) {
	var ret []Win32_Process
	q := wmi.CreateQuery(&ret, fmt.Sprintf("WHERE ProcessID = '%d'", pid))
	err := wmi.Query(q, &ret)
	if err != nil {
		return Win32_Process{}, fmt.Errorf("querying WMI process '%d': %w", pid, err)
	}
	if len(ret) != 1 {
		return Win32_Process{}, fmt.Errorf("querying WMI process '%d' : unexpected result : %v", pid, ret)
	}
	return ret[0], nil
}
