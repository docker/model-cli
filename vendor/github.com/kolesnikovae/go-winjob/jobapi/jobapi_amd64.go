// +build windows

package jobapi

// JOBOBJECT_EXTENDED_LIMIT_INFORMATION contains basic and extended limit information for a job object.
//
// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-jobobject_extended_limit_information
type JOBOBJECT_EXTENDED_LIMIT_INFORMATION struct {
	BasicLimitInformation JOBOBJECT_BASIC_LIMIT_INFORMATION
	IoInfo                IO_COUNTERS // Reserved.
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}
