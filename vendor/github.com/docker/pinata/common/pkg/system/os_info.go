package system

import (
	"sync"
)

type InfoGetter interface {
	GetSystemInfo() (OsInfo, error)
	MemoryInfo() Memory
}

func NewInfoGetter() InfoGetter {
	return &infoGetter{}
}

type infoGetter struct{}

func (infoGetter) GetSystemInfo() (OsInfo, error) { return GetSystemInfo() }

func (infoGetter) MemoryInfo() Memory {
	return MemoryInfo()
}

type LanguageInfo struct {
	Locale         string
	ShortName      string
	ActiveCodePage uint32
}

type OsInfo struct {
	PlatformSpecific
	Name           string
	ReleaseId      string
	BuildNumber    string
	Language       LanguageInfo
	BuildLabName   string
	Edition        string
	Version        Version
	Vendor         string
	VendorSuper    string
	DisplayVersion string // needed for Win
}

var (
	systemInfo     OsInfo
	systemInfoErr  error
	systemInfoOnce sync.Once
)

// GetSystemInfo returns host OS version
func GetSystemInfo() (OsInfo, error) {
	systemInfoOnce.Do(func() {
		systemInfo, systemInfoErr = getSystemInfo()
	})
	return systemInfo, systemInfoErr
}
