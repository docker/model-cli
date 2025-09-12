// +build windows

package winjob

import "github.com/kolesnikovae/go-winjob/jobapi"

// WithDesktopLimit prevents processes associated with the job from creating
// desktops and switching desktops using the CreateDesktop and SwitchDesktop
// functions.
func WithDesktopLimit() Limit {
	return LimitDesktop
}

// WithDisplaySettingsLimit prevents processes associated with the job from
// calling the ChangeDisplaySettings function.
func WithDisplaySettingsLimit() Limit {
	return LimitDisplaySettings
}

// WithExitWindowsLimit prevents processes associated with the job from calling
// the ExitWindows or ExitWindowsEx function.
func WithExitWindowsLimit() Limit {
	return LimitExitWindows
}

// WithGlobalAtomsLimit prevents processes associated with the job from
// accessing global atoms. When this limit is used, each job has its own atom
// table.
func WithGlobalAtomsLimit() Limit {
	return LimitGlobalAtoms
}

// WithHandlesLimit prevents processes associated with the job from using USER
// handles owned by processes not associated with the same job.
func WithHandlesLimit() Limit {
	return LimitHandles
}

// WithReadClipboardLimit prevents processes associated with the job from
// reading data from the clipboard.
func WithReadClipboardLimit() Limit {
	return LimitReadClipboard
}

// WithSystemParametersLimit prevents processes associated with the job from
// changing system parameters by using the SystemParametersInfo function.
func WithSystemParametersLimit() Limit {
	return LimitSystemParameters
}

// WithWriteClipboardLimit prevents processes associated with the job from
// writing data to the clipboard.
func WithWriteClipboardLimit() Limit {
	return LimitWriteClipboard
}

var (
	LimitDesktop          = uiRestriction(jobapi.JOB_OBJECT_UILIMIT_DESKTOP)
	LimitDisplaySettings  = uiRestriction(jobapi.JOB_OBJECT_UILIMIT_DISPLAYSETTINGS)
	LimitExitWindows      = uiRestriction(jobapi.JOB_OBJECT_UILIMIT_EXITWINDOWS)
	LimitGlobalAtoms      = uiRestriction(jobapi.JOB_OBJECT_UILIMIT_GLOBALATOMS)
	LimitHandles          = uiRestriction(jobapi.JOB_OBJECT_UILIMIT_HANDLES)
	LimitSystemParameters = uiRestriction(jobapi.JOB_OBJECT_UILIMIT_SYSTEMPARAMETERS)
	LimitWriteClipboard   = uiRestriction(jobapi.JOB_OBJECT_UILIMIT_WRITECLIPBOARD)
	LimitReadClipboard    = uiRestriction(jobapi.JOB_OBJECT_UILIMIT_READCLIPBOARD)
)

type uiRestriction jobapi.UIRestrictionsClass

func (r uiRestriction) set(job *JobObject) {
	job.UIRestrictions.UIRestrictionsClass |= jobapi.UIRestrictionsClass(r)
}

func (r uiRestriction) reset(job *JobObject) {
	job.UIRestrictions.UIRestrictionsClass &^= jobapi.UIRestrictionsClass(r)
}

func (r uiRestriction) IsSet(job *JobObject) bool {
	return job.UIRestrictions.UIRestrictionsClass&jobapi.UIRestrictionsClass(r) > 0
}

func (r uiRestriction) Value(job *JobObject) interface{} {
	return r.IsSet(job)
}
