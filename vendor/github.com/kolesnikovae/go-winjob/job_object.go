// +build windows

package winjob

import (
	"os"
	"syscall"

	"github.com/kolesnikovae/go-winjob/jobapi"
)

// JobObject represents windows job object. Microsoft documentation says the
// following: A job object allows groups of processes to be managed as a unit.
// Job objects are namable, securable, sharable objects that control attributes
// of the processes associated with them. Operations performed on a job object
// affect all processes associated with the job object. Examples include
// enforcing limits such as working set size and process priority or
// terminating all processes associated with a job.
//
// https://docs.microsoft.com/en-us/windows/desktop/procthread/job-objects.
type JobObject struct {
	Name   string
	Handle syscall.Handle
	JobInfo
}

// Limit manages a job object limits.
//
// Microsoft documentation says the following: A job can enforce limits such as
// working set size, process priority, and end-of-job time limit on each
// process that is associated with the job. If a process associated with a
// job attempts to increase its working set size or process priority from the
// limit established by the job, the function calls succeed but are silently
// ignored. A job can also set limits that trigger a notification when they
// are exceeded but allow the job to continue to run.
//
// https://docs.microsoft.com/en-us/windows/desktop/procthread/job-objects
type Limit interface {

	// IsSet reports whether the limit is set for the job object.
	IsSet(*JobObject) bool

	// Value returns actual limit value for the job object.
	// A Limit implementation may provide specific methods for
	// accessing its values of a concrete type, if applicable.
	Value(*JobObject) interface{}

	set(*JobObject)
	reset(*JobObject)
}

// Counters contains basic accounting information and I/O counters
// of a job object.
type Counters struct {
	TotalUserTime             uint64
	TotalKernelTime           uint64
	ThisPeriodTotalUserTime   uint64
	ThisPeriodTotalKernelTime uint64
	TotalPageFaultCount       uint32
	TotalProcesses            uint32
	ActiveProcesses           uint32
	TotalTerminatedProcesses  uint32

	ReadOperationCount  uint64
	WriteOperationCount uint64
	OtherOperationCount uint64
	ReadTransferCount   uint64
	WriteTransferCount  uint64
	OtherTransferCount  uint64
}

type JobInfo struct {
	ExtendedLimits jobapi.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	UIRestrictions jobapi.JOBOBJECT_BASIC_UI_RESTRICTIONS
	AccountingInfo jobapi.JOBOBJECT_BASIC_AND_IO_ACCOUNTING_INFORMATION
	CPURateControl jobapi.JOBOBJECT_CPU_RATE_CONTROL_INFORMATION
	NetRateControl jobapi.JOBOBJECT_NET_RATE_CONTROL_INFORMATION
}

// Create creates a new job object. An anonymous job object will be created,
// if a name is not provided. One or more job object limits may be specified:
// refer to limits documentation for details. If limits fail to apply, created
// job object will be disposed.
func Create(name string, limits ...Limit) (*JobObject, error) {
	hJobObject, err := jobapi.CreateJobObject(name, jobapi.MakeSA())
	if err != nil {
		return nil, err
	}
	job := JobObject{
		Name:   name,
		Handle: hJobObject,
	}
	if len(limits) != 0 {
		if err := job.SetLimit(limits...); err != nil {
			_ = job.Close()
			return nil, err
		}
	}
	return &job, nil
}

// Open opens existing job object by its name. A job is being opened with
// JOB_OBJECT_ALL_ACCESS access rights.
func Open(name string) (*JobObject, error) {
	return OpenWithAccess(name, jobapi.JOB_OBJECT_ALL_ACCESS)
}

// Open opens existing job object by its name with access rights specified.
func OpenWithAccess(name string, access uintptr) (*JobObject, error) {
	hJobObject, err := jobapi.OpenJobObject(access, 0, name)
	if err != nil {
		return nil, err
	}
	job := JobObject{
		Name:   name,
		Handle: hJobObject,
	}
	return &job, nil
}

// Close closes job object handle.
func (job *JobObject) Close() error {
	return syscall.Close(job.Handle)
}

// Terminate destroys the job object and all the associated processes.
// If the job is nested, this function terminates all child jobs in the
// hierarchy. All the processes and threads in the job object will use
// exit code 1.
func (job *JobObject) Terminate() error {
	return job.TerminateWithExitCode(1)
}

// TerminateWithExitCode terminates the job object. All the processes and
// threads in the job object will use the exit code provided.
func (job *JobObject) TerminateWithExitCode(exitCode uint32) error {
	return jobapi.TerminateJobObject(job.Handle, exitCode)
}

// Assign opens specified process by PID and adds it to the job object.
// When a process is associated with a job, the association cannot be
// broken. A process can be associated with more than one job object in a
// hierarchy of nested jobs (OS-dependent). The process is opened with
// PROCESS_ALL_ACCESS access rights.
func (job *JobObject) Assign(p *os.Process) error {
	desiredAccess := jobapi.PROCESS_ALL_ACCESS
	return withProcessHandle(p.Pid, desiredAccess, func(h syscall.Handle) error {
		return jobapi.AssignProcessToJobObject(job.Handle, h)
	})
}

// Contains returns true if the process is running in the job object.
// The process is opened with PROCESS_QUERY_LIMITED_INFORMATION access
// rights.
func (job *JobObject) Contains(p *os.Process) (found bool, err error) {
	desiredAccess := jobapi.PROCESS_QUERY_LIMITED_INFORMATION
	err = withProcessHandle(p.Pid, desiredAccess, func(h syscall.Handle) error {
		found, err = jobapi.IsProcessInJob(h, job.Handle)
		return err
	})
	return found, err
}

func withProcessHandle(pid, access int, fn func(h syscall.Handle) error) error {
	hProcess, err := syscall.OpenProcess(uint32(access), false, uint32(pid))
	if err != nil {
		return err
	}
	defer func() {
		_ = syscall.CloseHandle(hProcess)
	}()
	return fn(hProcess)
}

// Counters creates a new Counters and queries the given job object for basic
// and I/O accounting information. If job counters are queried on interval,
// returned Counters should be used with consequent QueryCounters calls in
// order to reduce the number of allocations.
func (job *JobObject) Counters() (*Counters, error) {
	var c Counters
	if err := job.QueryCounters(&c); err != nil {
		return nil, err
	}
	return &c, nil
}

// QueryCounters queries the job object for basic and I/O accounting
// information and fills provided Counters with the data retrieved.
func (job *JobObject) QueryCounters(c *Counters) error {
	err := job.sync(jobapi.QueryInfo, jobapi.JobObjectBasicAndIoAccountingInformation)
	if err != nil {
		return err
	}

	c.TotalUserTime = job.AccountingInfo.TotalUserTime
	c.TotalKernelTime = job.AccountingInfo.TotalUserTime
	c.ThisPeriodTotalUserTime = job.AccountingInfo.TotalUserTime
	c.ThisPeriodTotalKernelTime = job.AccountingInfo.TotalUserTime

	c.TotalPageFaultCount = job.AccountingInfo.TotalPageFaultCount
	c.TotalProcesses = job.AccountingInfo.TotalProcesses
	c.ActiveProcesses = job.AccountingInfo.ActiveProcesses
	c.TotalTerminatedProcesses = job.AccountingInfo.TotalTerminatedProcesses

	c.ReadOperationCount = job.AccountingInfo.ReadOperationCount
	c.WriteOperationCount = job.AccountingInfo.WriteOperationCount
	c.OtherOperationCount = job.AccountingInfo.OtherOperationCount
	c.ReadTransferCount = job.AccountingInfo.ReadTransferCount
	c.WriteTransferCount = job.AccountingInfo.WriteTransferCount
	c.OtherTransferCount = job.AccountingInfo.OtherOperationCount

	return nil
}

// QueryLimits queries all supported limit information for the job object.
func (job *JobObject) QueryLimits() error {
	return job.sync(jobapi.QueryInfo,
		jobapi.JobObjectExtendedLimitInformation,
		jobapi.JobObjectBasicUIRestrictions,
		jobapi.JobObjectCpuRateControlInformation,
		jobapi.JobObjectNetRateControlInformation)
}

// SetLimit applies given limits to the job object.
func (job *JobObject) SetLimit(limits ...Limit) error {
	return job.applyLimit(true, limits...)
}

// HasLimits returns true if any limit is set on the job object.
func (job *JobObject) HasLimits() (bool, error) {
	if err := job.QueryLimits(); err != nil {
		return false, err
	}
	return len(job.limitInfoClassesSet()) > 0, nil
}

// ResetLimits resets all the job object limits.
func (job *JobObject) ResetLimits() error {
	if err := job.QueryLimits(); err != nil {
		return err
	}
	infoClasses := job.limitInfoClassesSet()
	job.JobInfo = JobInfo{}
	return job.sync(jobapi.SetInfo, infoClasses...)
}

// ResetLimit resets given limits of the job object.
func (job *JobObject) ResetLimit(limits ...Limit) error {
	return job.applyLimit(false, limits...)
}

// applyLimits queries required limit information and sets or resets
// the limits specified.
func (job *JobObject) applyLimit(set bool, limits ...Limit) error {
	classesSet := make(map[jobapi.JobObjectInformationClass]struct{})
	for _, limit := range limits {
		infoClass := resolveRequiredInfoClass(limit)
		if _, queried := classesSet[infoClass]; !queried {
			if err := job.sync(jobapi.QueryInfo, infoClass); err != nil {
				return err
			}
		}
		classesSet[infoClass] = struct{}{}
		if set {
			limit.set(job)
			continue
		}
		limit.reset(job)
	}

	infoClasses := make([]jobapi.JobObjectInformationClass, 0)
	for k := range classesSet {
		infoClasses = append(infoClasses, k)
	}

	return job.sync(jobapi.SetInfo, infoClasses...)
}

func resolveRequiredInfoClass(limit Limit) jobapi.JobObjectInformationClass {
	switch limit.(type) {
	default:
		return jobapi.JobObjectExtendedLimitInformation
	case uiRestriction:
		return jobapi.JobObjectBasicUIRestrictions
	case cpuLimit:
		return jobapi.JobObjectCpuRateControlInformation
	case netBandwidthLimit, netDSCPTagLimit:
		return jobapi.JobObjectNetRateControlInformation
	}
}

func (job *JobObject) infoPtr(infoClass jobapi.JobObjectInformationClass) interface{} {
	switch infoClass {
	case jobapi.JobObjectBasicAndIoAccountingInformation:
		return &job.AccountingInfo
	case jobapi.JobObjectExtendedLimitInformation:
		return &job.ExtendedLimits
	case jobapi.JobObjectBasicUIRestrictions:
		return &job.UIRestrictions
	case jobapi.JobObjectCpuRateControlInformation:
		return &job.CPURateControl
	case jobapi.JobObjectNetRateControlInformation:
		return &job.NetRateControl
	default:
		return nil
	}
}

func (job *JobObject) limitInfoClassesSet() (classes []jobapi.JobObjectInformationClass) {
	for _, info := range []struct {
		isSet bool
		class jobapi.JobObjectInformationClass
	}{
		{
			job.ExtendedLimits.BasicLimitInformation.LimitFlags > 0,
			jobapi.JobObjectExtendedLimitInformation,
		},
		{
			job.UIRestrictions.UIRestrictionsClass > 0,
			jobapi.JobObjectBasicUIRestrictions,
		},
		{
			job.CPURateControl.ControlFlags > 0,
			jobapi.JobObjectCpuRateControlInformation,
		},
		{
			job.NetRateControl.ControlFlags > 0,
			jobapi.JobObjectNetRateControlInformation,
		},
	} {
		if info.isSet {
			classes = append(classes, info.class)
		}
	}
	return classes
}

type infoClassSync func(syscall.Handle, jobapi.JobObjectInformationClass, interface{}) error

func (job *JobObject) sync(fn infoClassSync, infoClasses ...jobapi.JobObjectInformationClass) error {
	for _, infoClass := range infoClasses {
		if err := fn(job.Handle, infoClass, job.infoPtr(infoClass)); err != nil {
			return err
		}
	}
	return nil
}
