package system

const (
	// The minimum version of macOS that will allow Docker Desktop to launch and to update.
	MacOSMinimumVersion = "12.0.0"
	// The minimum version of macOS that will allow Docker Desktop to be supported.
	MacOSRecommendedVersion = "12.0.0"

	// ProCondition for Windows Pro.
	ProCondition = "[EditionID] != Home && [EditionID] != HomeEval && [EditionID] != Core && [EditionID] != CoreN && [EditionID] != CoreSingleLanguage && [EditionID] != CoreCountrySpecific && [BuildNumber] >= 19044"
	// MinimumWindowsBuildCondition states the minimum Windows build required to run Docker Desktop.
	MinimumWindowsBuildCondition = "[BuildNumber] >= 19044"
	// The minimum version of Windows that will allow Docker Desktop to install, launch and update.
	WindowsOSCondition = "( " + ProCondition + " ) || ( " + MinimumWindowsBuildCondition + " )"
)
