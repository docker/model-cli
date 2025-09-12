// +build windows

package winjob

import (
	"time"

	"github.com/kolesnikovae/go-winjob/jobapi"
)

// WithBreakawayOK allows any process associated with the job to create child
// processes that are not associated with the job, if the process created with
// CREATE_BREAKAWAY_FROM_JOB flag.
func WithBreakawayOK() Limit {
	return LimitBreakawayOK
}

// WithSilentBreakawayOK allows any process associated with the job to create
// child processes that are not associated with the job. If the job is nested
// and its immediate job object allows breakaway, the child process breaks away
// from the immediate job object and from each job in the parent job chain,
// moving up the hierarchy until it reaches a job that does not permit breakaway.
// If the immediate job object does not allow breakaway, the child process does
// not break away even if jobs in its parent job chain allow it.
func WithSilentBreakawayOK() Limit {
	return LimitSilentBreakawayOK
}

// WithDieOnUnhandledException forces a call to the SetErrorMode function with
// the SEM_NOGPFAULTERRORBOX flag for each process associated with the job.
//
// If an exception occurs and the system calls the UnhandledExceptionFilter
// function, the debugger will be given a chance to act. If there is no
// debugger, the functions returns EXCEPTION_EXECUTE_HANDLER. Normally, this
// will cause termination of the process with the exception code as the exit
// status.
func WithDieOnUnhandledException() Limit {
	return LimitDieOnUnhandledException
}

// WithKillOnJobClose causes all processes associated with the job to terminate
// when the last handle to the job is closed.
func WithKillOnJobClose() Limit {
	return LimitKillOnJobClose
}

// WithPreserveJobTime preserves any job time limits you previously set. As
// long as this limit is set, you can establish a per-job time limit once, then
// alter other limits in subsequent calls.
//
// This flag cannot be used with LimitJobMemory.
func WithPreserveJobTime() Limit {
	return LimitPreserveJobTime
}

// WithSubsetAffinity allows processes to use a subset of the processor
// affinity for all processes associated with the job.
//
// This limit must be combined with LimitAffinity.
func WithSubsetAffinity() Limit {
	return LimitSubsetAffinity
}

// WithAffinity causes all processes associated with the job to use the same
// processor affinity.
//
// The value specified must be a subset of the system affinity mask obtained by
// calling the GetProcessAffinityMask function. The affinity of each thread is
// set to this value, but threads are free to subsequently set their affinity,
// as long as it is a subset of the specified affinity mask. Processes cannot
// set their own affinity mask.
//
// A process affinity mask is a bit vector in which each bit represents the
// processors that a process is allowed to run on. A system affinity mask is a
// bit vector in which each bit represents the processors that are configured
// into a system.
//
// A process affinity mask is a subset of the system affinity mask. A process
// is only allowed to run on the processors configured into a system.
// Therefore, the process affinity mask cannot specify a 1 bit for a processor
// when the system affinity mask specifies a 0 bit for that processor.
//
// If the job is nested, the specified processor affinity must be a subset of
// the effective affinity of the parent job. If the specified affinity a
// superset of the affinity of the parent job, it is ignored and the affinity
// of the parent job is used.
//
// This limit must be combined with LimitSubsetAffinity.
func WithAffinity(x uintptr) Limit {
	return LimitAffinity.WithValue(x)
}

// WithJobMemoryLimit causes all processes associated with the job to limit the
// job-wide sum of their committed memory. When a process attempts to commit
// memory that would exceed the job-wide limit, it fails.
//
// The value specifies the limit for the virtual memory that can be committed
// for the job object in bytes.
//
// If the job object is associated with a completion port, a
// JOB_OBJECT_MSG_JOB_MEMORY_LIMIT message is sent to the completion port.
func WithJobMemoryLimit(x uintptr) Limit {
	return LimitJobMemory.WithValue(x)
}

// WithJobTimeLimit establishes a user-mode execution time limit for the job.
//
// The system adds the current time of the processes associated with the job to
// this limit. For example, if you set this limit to 1 minute, and the job has
// a process that has accumulated 5 minutes of user-mode time, the limit
// actually enforced is 6 minutes.
//
// The system periodically checks to determine whether the sum of the user-mode
// execution time for all processes is greater than this end-of-job limit.
// By default, all processes are terminated and the status code is set to
// ERROR_NOT_ENOUGH_QUOTA.
func WithJobTimeLimit(x time.Duration) Limit {
	return LimitJobTime.WithValue(x)
}

// WithProcessMemoryLimit causes all processes associated with the job to limit
// their committed memory. When a process attempts to commit memory that would
// exceed the per-process limit, it fails.
//
// If the job object is associated with a completion port, a
// JOB_OBJECT_MSG_PROCESS_MEMORY_LIMIT message is sent to the completion port.
//
// If the job is nested, the effective memory limit is the most restrictive
// memory limit in the job chain.
func WithProcessMemoryLimit(x uintptr) Limit {
	return LimitProcessMemory.WithValue(x)
}

// WithProcessTimeLimit establishes a user-mode execution time limit for each
// currently active process and for all future processes associated with the
// job.
//
// The system periodically checks to determine whether each process associated
// with the job has accumulated more user-mode time than the set limit. If it
// has, the process is terminated.
//
// If the job is nested, the effective limit is the most restrictive limit in
// the job chain.
func WithProcessTimeLimit(x time.Duration) Limit {
	return LimitProcessTime.WithValue(x)
}

// WithActiveProcessLimit establishes a maximum number of simultaneously active
// processes associated with the job. The ActiveProcessLimit member contains
// additional information.
//
// If you try to associate a process with a job, and this causes the active
// process count to exceed this limit, the process is terminated and the
// association fails.
func WithActiveProcessLimit(x uint32) Limit {
	return LimitActiveProcess.WithValue(x)
}

// WithWorkingSetLimit causes all processes associated with the job to use the
// same minimum and maximum working set sizes (specified in bytes).
//
// If maximum working set size is nonzero, minimum working set cannot be zero,
// and vice-versa. The actual limit values can be retrieved with
// MinWorkingSetSize() and MaxWorkingSetSize() methods of LimitWorkingSet.
//
// If the job is nested, the effective working set size is the smallest
// working set size in the job chain.
//
// Note: processes can still empty their working sets using
// SetProcessWorkingSetSize, even when is used. However, you cannot use
// SetProcessWorkingSetSize change the minimum or maximum working set size
// of a process in a job object.
func WithWorkingSetLimit(min, max uintptr) Limit {
	return LimitWorkingSet.WithValue(min, max)
}

// WithPriorityClassLimit causes all processes associated with the job to use
// the same priority class.
//
// If the job is nested, the effective priority class is the lowest priority
// class in the job chain.
//
// Processes and threads cannot modify their priority class. The calling
// process must enable the SE_INC_BASE_PRIORITY_NAME privilege.
func WithPriorityClassLimit(x jobapi.PriorityClass) Limit {
	return LimitPriorityClass.WithValue(x)
}

// WithSchedulingClassLimit causes all processes in the job to use the same
// scheduling class.
//
// If the job is nested, the effective scheduling class is the lowest
// scheduling class in the job chain.
//
// The valid values are 0 to 9. Use 0 for the least favorable scheduling class
// relative to other threads, and 9 for the most favorable scheduling class
// relative to other threads. By default, this value is 5. To use a scheduling
// class greater than 5, the calling process must enable the
// SE_INC_BASE_PRIORITY_NAME privilege.
func WithSchedulingClassLimit(x uint32) Limit {
	return LimitSchedulingClass.WithValue(x)
}

var (
	LimitBreakawayOK             = basicLimit(jobapi.JOB_OBJECT_LIMIT_BREAKAWAY_OK)
	LimitDieOnUnhandledException = basicLimit(jobapi.JOB_OBJECT_LIMIT_DIE_ON_UNHANDLED_EXCEPTION)
	LimitKillOnJobClose          = basicLimit(jobapi.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE)
	LimitPreserveJobTime         = basicLimit(jobapi.JOB_OBJECT_LIMIT_PRESERVE_JOB_TIME)
	LimitSubsetAffinity          = basicLimit(jobapi.JOB_OBJECT_LIMIT_SUBSET_AFFINITY)
	LimitSilentBreakawayOK       = basicLimit(jobapi.JOB_OBJECT_LIMIT_SILENT_BREAKAWAY_OK)

	LimitAffinity        = affinityLimit{basicLimit: basicLimit(jobapi.JOB_OBJECT_LIMIT_AFFINITY)}
	LimitJobMemory       = jobMemoryLimit{basicLimit: basicLimit(jobapi.JOB_OBJECT_LIMIT_JOB_MEMORY)}
	LimitJobTime         = jobTimeLimit{basicLimit: basicLimit(jobapi.JOB_OBJECT_LIMIT_JOB_TIME)}
	LimitProcessMemory   = processMemoryLimit{basicLimit: basicLimit(jobapi.JOB_OBJECT_LIMIT_PROCESS_MEMORY)}
	LimitProcessTime     = processTimeLimit{basicLimit: basicLimit(jobapi.JOB_OBJECT_LIMIT_PROCESS_TIME)}
	LimitActiveProcess   = activeProcessLimit{basicLimit: basicLimit(jobapi.JOB_OBJECT_LIMIT_ACTIVE_PROCESS)}
	LimitWorkingSet      = workingSetLimit{basicLimit: basicLimit(jobapi.JOB_OBJECT_LIMIT_WORKINGSET)}
	LimitPriorityClass   = priorityClassLimit{basicLimit: basicLimit(jobapi.JOB_OBJECT_LIMIT_PRIORITY_CLASS)}
	LimitSchedulingClass = schedulingClassLimit{basicLimit: basicLimit(jobapi.JOB_OBJECT_LIMIT_SCHEDULING_CLASS)}
)

type basicLimit jobapi.LimitFlag

func (l basicLimit) set(job *JobObject) {
	job.ExtendedLimits.BasicLimitInformation.LimitFlags |= jobapi.LimitFlag(l)
}

func (l basicLimit) reset(job *JobObject) {
	job.ExtendedLimits.BasicLimitInformation.LimitFlags &^= jobapi.LimitFlag(l)
}

func (l basicLimit) IsSet(job *JobObject) bool {
	return job.ExtendedLimits.BasicLimitInformation.LimitFlags&jobapi.LimitFlag(l) > 0
}

func (l basicLimit) Value(job *JobObject) interface{} {
	return l.IsSet(job)
}

type affinityLimit struct {
	basicLimit
	affinity uintptr
}

func (l affinityLimit) WithValue(x uintptr) affinityLimit {
	l.affinity = x
	return l
}

func (l affinityLimit) LimitValue(job *JobObject) uintptr {
	return job.ExtendedLimits.BasicLimitInformation.Affinity
}

func (l affinityLimit) set(job *JobObject) {
	job.ExtendedLimits.BasicLimitInformation.Affinity = l.affinity
	l.basicLimit.set(job)
}

func (l affinityLimit) Value(job *JobObject) interface{} {
	return l.LimitValue(job)
}

type jobMemoryLimit struct {
	basicLimit
	jobMemory uintptr
}

func (l jobMemoryLimit) WithValue(x uintptr) jobMemoryLimit {
	l.jobMemory = x
	return l
}

func (l jobMemoryLimit) LimitValue(job *JobObject) uintptr {
	return job.ExtendedLimits.JobMemoryLimit
}

func (l jobMemoryLimit) set(job *JobObject) {
	job.ExtendedLimits.JobMemoryLimit = l.jobMemory
	l.basicLimit.set(job)
}

func (l jobMemoryLimit) Value(job *JobObject) interface{} {
	return l.LimitValue(job)
}

type jobTimeLimit struct {
	basicLimit
	jobTime int64
}

// Time limits and counters are specified in 100-nanosecond ticks.
const timeFraction = 100

func (l jobTimeLimit) WithValue(x time.Duration) jobTimeLimit {
	l.jobTime = x.Nanoseconds() / timeFraction
	return l
}

func (l jobTimeLimit) LimitValue(job *JobObject) time.Duration {
	return time.Duration(job.ExtendedLimits.BasicLimitInformation.PerJobUserTimeLimit * timeFraction)
}

func (l jobTimeLimit) set(job *JobObject) {
	job.ExtendedLimits.BasicLimitInformation.PerJobUserTimeLimit = l.jobTime
	l.basicLimit.set(job)
}

func (l jobTimeLimit) Value(job *JobObject) interface{} {
	return l.LimitValue(job)
}

type processMemoryLimit struct {
	basicLimit
	processMemory uintptr
}

func (l processMemoryLimit) WithValue(x uintptr) processMemoryLimit {
	l.processMemory = x
	return l
}

func (l processMemoryLimit) LimitValue(job *JobObject) uintptr {
	return job.ExtendedLimits.ProcessMemoryLimit
}

func (l processMemoryLimit) set(job *JobObject) {
	job.ExtendedLimits.ProcessMemoryLimit = l.processMemory
	l.basicLimit.set(job)
}

func (l processMemoryLimit) Value(job *JobObject) interface{} {
	return l.LimitValue(job)
}

type processTimeLimit struct {
	basicLimit
	processTime int64
}

func (l processTimeLimit) WithValue(x time.Duration) processTimeLimit {
	l.processTime = x.Nanoseconds() / timeFraction
	return l
}

func (l processTimeLimit) LimitValue(job *JobObject) time.Duration {
	return time.Duration(job.ExtendedLimits.BasicLimitInformation.PerProcessUserTimeLimit * timeFraction)
}

func (l processTimeLimit) set(job *JobObject) {
	job.ExtendedLimits.BasicLimitInformation.PerProcessUserTimeLimit = l.processTime
	l.basicLimit.set(job)
}

func (l processTimeLimit) Value(job *JobObject) interface{} {
	return l.LimitValue(job)
}

type activeProcessLimit struct {
	basicLimit
	procs uint32
}

func (l activeProcessLimit) WithValue(x uint32) activeProcessLimit {
	l.procs = x
	return l
}

func (l activeProcessLimit) LimitValue(job *JobObject) uint32 {
	return job.ExtendedLimits.BasicLimitInformation.ActiveProcessLimit
}

func (l activeProcessLimit) set(job *JobObject) {
	job.ExtendedLimits.BasicLimitInformation.ActiveProcessLimit = uint32(l.procs)
	l.basicLimit.set(job)
}

func (l activeProcessLimit) Value(job *JobObject) interface{} {
	return l.LimitValue(job)
}

type workingSetLimit struct {
	basicLimit
	wsMin uintptr
	wsMax uintptr
}

func (l workingSetLimit) WithValue(min, max uintptr) workingSetLimit {
	l.wsMin = min
	l.wsMax = max
	return l
}

func (l workingSetLimit) MinWorkingSetSize(job *JobObject) uintptr {
	return job.ExtendedLimits.BasicLimitInformation.MinimumWorkingSetSize
}

func (l workingSetLimit) MaxWorkingSetSize(job *JobObject) uintptr {
	return job.ExtendedLimits.BasicLimitInformation.MaximumWorkingSetSize
}

func (l workingSetLimit) set(job *JobObject) {
	job.ExtendedLimits.BasicLimitInformation.MinimumWorkingSetSize = l.wsMin
	job.ExtendedLimits.BasicLimitInformation.MaximumWorkingSetSize = l.wsMax
	l.basicLimit.set(job)
}

type priorityClassLimit struct {
	basicLimit
	prio jobapi.PriorityClass
}

func (l priorityClassLimit) WithValue(x jobapi.PriorityClass) priorityClassLimit {
	l.prio = x
	return l
}

func (l priorityClassLimit) LimitValue(job *JobObject) jobapi.PriorityClass {
	return job.ExtendedLimits.BasicLimitInformation.PriorityClass
}

func (l priorityClassLimit) set(job *JobObject) {
	job.ExtendedLimits.BasicLimitInformation.PriorityClass = l.prio
	l.basicLimit.set(job)
}

func (l priorityClassLimit) Value(job *JobObject) interface{} {
	return l.LimitValue(job)
}

type schedulingClassLimit struct {
	basicLimit
	schedClass uint32
}

func (l schedulingClassLimit) WithValue(x uint32) schedulingClassLimit {
	l.schedClass = x
	return l
}

func (l schedulingClassLimit) LimitValue(job *JobObject) uint32 {
	return job.ExtendedLimits.BasicLimitInformation.SchedulingClass
}

func (l schedulingClassLimit) set(job *JobObject) {
	job.ExtendedLimits.BasicLimitInformation.SchedulingClass = l.schedClass
	l.basicLimit.set(job)
}

func (l schedulingClassLimit) Value(job *JobObject) interface{} {
	return l.LimitValue(job)
}
