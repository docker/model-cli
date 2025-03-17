package system

import (
	"fmt"
	"strings"
	"unsafe"

	"github.com/docker/pinata/common/pkg/logger"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var (
	log               = logger.Default.WithComponent("system")
	knownHomeEditions = []string{
		"Home",
		"HomeN",
		"HomeEval",
		"Core",
		"CoreN",
		"CoreSingleLanguage",
		"CoreCountrySpecific",
	}
)

type PlatformSpecific struct {
	IsWindowsHome bool
}

var kernel32 = windows.NewLazySystemDLL("kernel32.dll")

func getSystemLocale() (string, error) {
	proc := kernel32.NewProc("GetSystemDefaultLocaleName")
	maxLength := uint32(100)
	lang := make([]uint16, maxLength)
	_, _, err := proc.Call(
		uintptr(unsafe.Pointer(&lang[0])),
		uintptr(unsafe.Pointer(&maxLength)),
	)
	if err != windows.Errno(0) {
		return "", err
	}
	locale := windows.UTF16ToString(lang)
	return locale, nil
}

// GetACP returns the active code page for the OS.
func getACP() (uint32, error) {
	proc := kernel32.NewProc("GetACP")
	acp, _, err := proc.Call()
	if err != windows.Errno(0) {
		return 0, err
	}
	return uint32(acp), nil
}

func getLocaleShortName(locale string) string {
	shortName := locale
	localeSplit := strings.Split(locale, "-")
	if len(localeSplit) > 1 {
		shortName = localeSplit[0]
	}
	return shortName
}

func getLanguageInfo() LanguageInfo {
	locale, err := getSystemLocale()
	if err != nil {
		log.Warnln("retrieving system locale:", err)
		locale = "unknown"
	}

	codePage, err := getACP()
	if err != nil {
		log.Warnln("retrieving system code page:", err)
		codePage = 0
	}

	return LanguageInfo{
		Locale:         locale,
		ShortName:      getLocaleShortName(locale),
		ActiveCodePage: codePage,
	}
}

func getSystemInfo() (OsInfo, error) {
	softwareKey, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		return OsInfo{}, fmt.Errorf("retrieving registry key: %w", err)
	}
	defer softwareKey.Close()

	name, _, err := softwareKey.GetStringValue("ProductName")
	if err != nil {
		return OsInfo{}, fmt.Errorf("retrieving product name: %w", err)
	}
	releaseID, _, err := softwareKey.GetStringValue("ReleaseId")
	if err != nil {
		return OsInfo{}, fmt.Errorf("retrieving ReleaseId: %w", err)
	}
	build, _, err := softwareKey.GetStringValue("CurrentBuild")
	if err != nil {
		return OsInfo{}, fmt.Errorf("retrieving CurrentBuild: %w", err)
	}
	edition, _, err := softwareKey.GetStringValue("EditionID")
	if err != nil {
		return OsInfo{}, fmt.Errorf("retrieving EditionID: %w", err)
	}

	labName, _, err := softwareKey.GetStringValue("BuildLabEx")
	if err != nil {
		return OsInfo{}, fmt.Errorf("retrieving BuildLabEx: %w", err)
	}

	displayVersion, _, err := softwareKey.GetStringValue("DisplayVersion")
	if err != nil {
		// https://docs.microsoft.com/en-us/answers/questions/470070/windows-version.html
		log.Warnln("Windows version might not be up-to-date:", err)
		displayVersion = releaseID
	}

	name = strings.ToLower(name)
	return OsInfo{
		Name:         name,
		ReleaseId:    releaseID,
		BuildNumber:  build,
		Language:     getLanguageInfo(),
		Edition:      edition,
		BuildLabName: labName,
		Version: Version{
			Major: name,
			Minor: releaseID,
			Patch: build,
		},
		DisplayVersion: displayVersion,
		PlatformSpecific: PlatformSpecific{
			IsWindowsHome: isHome(edition),
		},
	}, nil
}

func isHome(edition string) bool {
	for _, knownEdition := range knownHomeEditions {
		if edition == knownEdition {
			return true
		}
	}
	return false
}
