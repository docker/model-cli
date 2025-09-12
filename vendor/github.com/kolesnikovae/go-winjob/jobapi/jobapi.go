// +build windows

package jobapi

import (
	"os"
	"reflect"
	"syscall"
	"unsafe"
)

var (
	modKernel32               = syscall.NewLazyDLL("kernel32.dll")
	openJobObject             = modKernel32.NewProc("OpenJobObjectW")
	createJobObject           = modKernel32.NewProc("CreateJobObjectW")
	terminateJobObject        = modKernel32.NewProc("TerminateJobObject")
	isProcessInJob            = modKernel32.NewProc("IsProcessInJob")
	assignProcessToJobObject  = modKernel32.NewProc("AssignProcessToJobObject")
	setInformationJobObject   = modKernel32.NewProc("SetInformationJobObject")
	queryInformationJobObject = modKernel32.NewProc("QueryInformationJobObject")
)

// ErrAbandoned specifies that the completion port handle had been closed
// while a GetQueuedCompletionStatus call was outstanding.
// The original error code is ERROR_ABANDONED_WAIT_0 (0x2df).
var ErrAbandoned = syscall.Errno(0x2df)

// Process Security and Access Rights.
//
// https://docs.microsoft.com/en-us/windows/desktop/procthread/process-security-and-access-rights
const (
	PROCESS_TERMINATE                 = 0x000001
	PROCESS_DUP_HANDLE                = 0x000040
	PROCESS_SET_QUOTA                 = 0x000100
	PROCESS_SET_INFORMATION           = 0x000200
	PROCESS_QUERY_INFORMATION         = 0x000400
	PROCESS_QUERY_LIMITED_INFORMATION = 0x001000
	PROCESS_ALL_ACCESS                = 0x1F0FFF
	PROCESS_VM_OPERATION              = 0x000008
	PROCESS_VM_READ                   = 0x000010
	PROCESS_VM_WRITE                  = 0x000020
)

// Job Object Security and Access Rights.
//
// https://docs.microsoft.com/en-us/windows/desktop/ProcThread/job-object-security-and-access-rights
const (
	JOB_OBJECT_ASSIGN_PROCESS = 1 << iota
	JOB_OBJECT_SET_ATTRIBUTES
	JOB_OBJECT_QUERY
	JOB_OBJECT_TERMINATE
	// JOB_OBJECT_SET_SECURITY_ATTRIBUTES flag is not supported. You must set
	// security limitations individually for each process associated with a job.
	// JOB_OBJECT_SET_SECURITY_ATTRIBUTES // 0x000010
	JOB_OBJECT_ALL_ACCESS = 0x1F001F
)

// PriorityClass is a priority class of the process.
//
// https://docs.microsoft.com/en-us/windows/desktop/procthread/scheduling-priorities
type PriorityClass uint32

// Process priority classes.
const (
	NORMAL_PRIORITY_CLASS         PriorityClass = 0x00000020
	IDLE_PRIORITY_CLASS           PriorityClass = 0x00000040
	HIGH_PRIORITY_CLASS           PriorityClass = 0x00000080
	REALTIME_PRIORITY_CLASS       PriorityClass = 0x00000100
	BELOW_NORMAL_PRIORITY_CLASS   PriorityClass = 0x00004000
	ABOVE_NORMAL_PRIORITY_CLASS   PriorityClass = 0x00008000
	PROCESS_MODE_BACKGROUND_BEGIN PriorityClass = 0x00100000
	PROCESS_MODE_BACKGROUND_END   PriorityClass = 0x00200000
)

// JobObjectInformationClass is a information class for the limits to be set or queried.
//
// https://docs.microsoft.com/en-us/windows/desktop/api/jobapi2/nf-jobapi2-setinformationjobobject
// https://docs.microsoft.com/en-us/windows/desktop/api/jobapi2/nf-jobapi2-queryinformationjobobject
type JobObjectInformationClass uint32

// Job object information classes.
const (
	JobObjectBasicAccountingInformation JobObjectInformationClass = iota + 1
	JobObjectBasicLimitInformation
	JobObjectBasicProcessIdList
	JobObjectBasicUIRestrictions
	JobObjectSecurityLimitInformation
	JobObjectEndOfJobTimeInformation
	JobObjectAssociateCompletionPortInformation
	JobObjectBasicAndIoAccountingInformation
	JobObjectExtendedLimitInformation
	JobObjectJobSetInformation
	JobObjectGroupInformation
	JobObjectNotificationLimitInformation
	JobObjectLimitViolationInformation
	JobObjectGroupInformationEx
	JobObjectCpuRateControlInformation
	JobObjectCompletionFilter
	JobObjectCompletionCounter
	JobObjectFreezeInformation
	JobObjectExtendedAccountingInformation
	JobObjectWakeInformation
	JobObjectBackgroundInformation
	JobObjectSchedulingRankBiasInformation
	JobObjectTimerVirtualizationInformation
	JobObjectCycleTimeNotification
	JobObjectClearEvent
	JobObjectInterferenceInformation
	JobObjectClearPeakJobMemoryUsed
	JobObjectMemoryUsageInformation
	JobObjectSharedCommit
	JobObjectContainerId
	JobObjectIoRateControlInformation
	JobObjectNetRateControlInformation
	JobObjectNotificationLimitInformation2
	JobObjectLimitViolationInformation2
	JobObjectCreateSilo
	JobObjectSiloBasicInformation
	JobObjectSiloRootDirectory
	JobObjectServerSiloBasicInformation
	JobObjectServerSiloUserSharedData
	JobObjectServerSiloInitialize
	JobObjectServerSiloRunningState
	JobObjectIoAttribution
	JobObjectMemoryPartitionInformation
	JobObjectContainerTelemetryId
	JobObjectSiloSystemRoot
	JobObjectEnergyTrackingState
	JobObjectThreadImpersonationInformation
)

// LimitFlag specifies the limit flags that are in effect. This type is a
// bitfield that defines whether other structure members of JOBOBJECT_BASIC_LIMIT_INFORMATION
// are used.
//
// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-jobobject_basic_limit_information
// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-jobobject_extended_limit_information
type LimitFlag uint32

// Job object limit flags.
const (
	JOB_OBJECT_LIMIT_WORKINGSET LimitFlag = 1 << iota
	JOB_OBJECT_LIMIT_PROCESS_TIME
	JOB_OBJECT_LIMIT_JOB_TIME
	JOB_OBJECT_LIMIT_ACTIVE_PROCESS
	JOB_OBJECT_LIMIT_AFFINITY
	JOB_OBJECT_LIMIT_PRIORITY_CLASS
	JOB_OBJECT_LIMIT_PRESERVE_JOB_TIME
	JOB_OBJECT_LIMIT_SCHEDULING_CLASS
	JOB_OBJECT_LIMIT_PROCESS_MEMORY
	JOB_OBJECT_LIMIT_JOB_MEMORY
	JOB_OBJECT_LIMIT_DIE_ON_UNHANDLED_EXCEPTION
	JOB_OBJECT_LIMIT_BREAKAWAY_OK
	JOB_OBJECT_LIMIT_SILENT_BREAKAWAY_OK
	JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	JOB_OBJECT_LIMIT_SUBSET_AFFINITY
	JOB_OBJECT_LIMIT_JOB_MEMORY_LOW
	JOB_OBJECT_LIMIT_JOB_READ_BYTES
	JOB_OBJECT_LIMIT_JOB_WRITE_BYTES
	JOB_OBJECT_LIMIT_RATE_CONTROL
	JOB_OBJECT_LIMIT_IO_RATE_CONTROL
	JOB_OBJECT_LIMIT_NET_RATE_CONTROL

	JOB_OBJECT_LIMIT_CPU_RATE_CONTROL = JOB_OBJECT_LIMIT_RATE_CONTROL
	JOB_OBJECT_LIMIT_JOB_MEMORY_HIGH  = JOB_OBJECT_LIMIT_JOB_MEMORY
)

// UIRestrictionsClass is a restriction class for the user interface.
//
// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-jobobject_basic_ui_restrictions
type UIRestrictionsClass uint32

// User-interface restrictions for a job object.
const (
	JOB_OBJECT_UILIMIT_HANDLES UIRestrictionsClass = 1 << iota
	JOB_OBJECT_UILIMIT_READCLIPBOARD
	JOB_OBJECT_UILIMIT_WRITECLIPBOARD
	JOB_OBJECT_UILIMIT_SYSTEMPARAMETERS
	JOB_OBJECT_UILIMIT_DISPLAYSETTINGS
	JOB_OBJECT_UILIMIT_GLOBALATOMS
	JOB_OBJECT_UILIMIT_DESKTOP
	JOB_OBJECT_UILIMIT_EXITWINDOWS
)

// CPUControlFlag is a scheduling policy for CPU rate control.
//
// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-jobobject_cpu_rate_control_information
type CPUControlFlag uint32

// CPU rate control flags.
const (
	JOB_OBJECT_CPU_RATE_CONTROL_ENABLE CPUControlFlag = 1 << iota
	JOB_OBJECT_CPU_RATE_CONTROL_WEIGHT_BASED
	JOB_OBJECT_CPU_RATE_CONTROL_HARD_CAP
	JOB_OBJECT_CPU_RATE_CONTROL_NOTIFY
	JOB_OBJECT_CPU_RATE_CONTROL_MIN_MAX_RATE
)

// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-jobobject_associate_completion_port
type CompletionPortMessage uint32

// Job object completion port message types.
const (
	JOB_OBJECT_MSG_END_OF_JOB_TIME CompletionPortMessage = iota + 1
	JOB_OBJECT_MSG_END_OF_PROCESS_TIME
	JOB_OBJECT_MSG_ACTIVE_PROCESS_LIMIT
	JOB_OBJECT_MSG_ACTIVE_PROCESS_ZERO
	_
	JOB_OBJECT_MSG_NEW_PROCESS
	JOB_OBJECT_MSG_EXIT_PROCESS
	JOB_OBJECT_MSG_ABNORMAL_EXIT_PROCESS
	JOB_OBJECT_MSG_PROCESS_MEMORY_LIMIT
	JOB_OBJECT_MSG_JOB_MEMORY_LIMIT
	JOB_OBJECT_MSG_NOTIFICATION_LIMIT
	JOB_OBJECT_MSG_JOB_CYCLE_TIME_LIMIT
	JOB_OBJECT_MSG_SILO_TERMINATED

	JOB_OBJECT_MSG_MINIMUM = JOB_OBJECT_MSG_END_OF_JOB_TIME
	JOB_OBJECT_MSG_MAXIMUM = JOB_OBJECT_MSG_SILO_TERMINATED
)

// EndOfJobTimeAction specifies the action that the system will perform when
// the end-of-job time limit has been exceeded.
//
// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-jobobject_end_of_job_time_information
type EndOfJobTimeAction uint32

// End-of-time actions.
const (
	JOB_OBJECT_TERMINATE_AT_END_OF_JOB EndOfJobTimeAction = iota
	JOB_OBJECT_POST_AT_END_OF_JOB
)

// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-jobobject_notification_limit_information
// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-jobobject_notification_limit_information_2
type JOBOBJECT_RATE_CONTROL_TOLERANCE uint32

// Rate control tolerance levels.
const (
	ToleranceLow JOBOBJECT_RATE_CONTROL_TOLERANCE = iota + 1
	ToleranceMedium
	ToleranceHigh
)

// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-jobobject_notification_limit_information
// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-jobobject_notification_limit_information_2
type JOBOBJECT_RATE_CONTROL_TOLERANCE_INTERVAL uint32

// Rate control tolerance intervals.
const (
	ToleranceIntervalShort JOBOBJECT_RATE_CONTROL_TOLERANCE_INTERVAL = iota + 1
	ToleranceIntervalMedium
	ToleranceIntervalLong
)

// JOBOBJECT_BASIC_LIMIT_INFORMATION contains basic limit information for a job object.
//
// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-jobobject_basic_limit_information
type JOBOBJECT_BASIC_LIMIT_INFORMATION struct {
	PerProcessUserTimeLimit int64
	PerJobUserTimeLimit     int64
	LimitFlags              LimitFlag
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint32
	Affinity                uintptr
	PriorityClass           PriorityClass
	SchedulingClass         uint32
}

// JOBOBJECT_BASIC_UI_RESTRICTIONS contains basic user-interface restrictions for a job object.
//
// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-jobobject_basic_ui_restrictions
type JOBOBJECT_BASIC_UI_RESTRICTIONS struct {
	UIRestrictionsClass
}

// JOBOBJECT_BASIC_AND_IO_ACCOUNTING_INFORMATION contains basic accounting
// and I/O accounting information for a job object.
//
// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-jobobject_basic_and_io_accounting_information
type JOBOBJECT_BASIC_AND_IO_ACCOUNTING_INFORMATION struct {
	JOBOBJECT_BASIC_ACCOUNTING_INFORMATION
	IO_COUNTERS
}

// JOBOBJECT_BASIC_ACCOUNTING_INFORMATION contains basic accounting information for a job object.
//
// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-jobobject_basic_accounting_information
type JOBOBJECT_BASIC_ACCOUNTING_INFORMATION struct {
	TotalUserTime             uint64
	TotalKernelTime           uint64
	ThisPeriodTotalUserTime   uint64
	ThisPeriodTotalKernelTime uint64
	TotalPageFaultCount       uint32
	TotalProcesses            uint32
	ActiveProcesses           uint32
	TotalTerminatedProcesses  uint32
}

// IO_COUNTERS contains I/O accounting information for a process or a job
// object. For a job object, the counters include all operations performed
// by all processes that have ever been associated with the job, in addition to
// all processes currently associated with the job.
//
// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-io_counters
type IO_COUNTERS struct {
	ReadOperationCount  uint64
	WriteOperationCount uint64
	OtherOperationCount uint64
	ReadTransferCount   uint64
	WriteTransferCount  uint64
	OtherTransferCount  uint64
}

// JOBOBJECT_BASIC_PROCESS_ID_LIST contains the process identifier list for
// a job object. If the job is nested, the process identifier list consists
// of all processes associated with the job and its child jobs.
//
// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-jobobject_basic_process_id_list
type JOBOBJECT_BASIC_PROCESS_ID_LIST struct {
	NumberOfAssignedProcesses uint32
	NumberOfProcessIdsInList  uint32
	ProcessIDList             uintptr
}

// JOBOBJECT_CPU_RATE_CONTROL_INFORMATION contains CPU rate control information
// for a job object. The original structure contains a union that was replaced
// with a single Value member.
//
// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-jobobject_cpu_rate_control_information
type JOBOBJECT_CPU_RATE_CONTROL_INFORMATION struct {
	ControlFlags CPUControlFlag
	Value        uint32
}

// JOBOBJECT_NET_RATE_CONTROL_INFORMATION contains information used to control
// the network traffic for a job. This structure is used by the
// SetInformationJobObject and QueryInformationJobObject functions with the
// JobObjectNetRateControlInformation information class.
//
// https://docs.microsoft.com/en-us/windows/win32/api/winnt/ns-winnt-jobobject_net_rate_control_information.
type JOBOBJECT_NET_RATE_CONTROL_INFORMATION struct {
	MaxBandwidth uint64
	ControlFlags JOB_OBJECT_NET_RATE_CONTROL_FLAGS
	DscpTag      byte
	_            [3]byte
}

type JOB_OBJECT_NET_RATE_CONTROL_FLAGS uint32

const (
	JOB_OBJECT_NET_RATE_CONTROL_ENABLE JOB_OBJECT_NET_RATE_CONTROL_FLAGS = 1 << iota
	JOB_OBJECT_NET_RATE_CONTROL_MAX_BANDWIDTH
	JOB_OBJECT_NET_RATE_CONTROL_DSCP_TAG
	JOB_OBJECT_NET_RATE_CONTROL_VALID_FLAGS JOB_OBJECT_NET_RATE_CONTROL_FLAGS = 7
)

// JOBOBJECT_END_OF_JOB_TIME_INFORMATION specifies the action the system will
// perform when an end-of-job time limit is exceeded.
//
// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-jobobject_end_of_job_time_information
type JOBOBJECT_END_OF_JOB_TIME_INFORMATION struct {
	EndOfJobTimeAction EndOfJobTimeAction
}

// JOBOBJECT_ASSOCIATE_COMPLETION_PORT contains information used to associate
// a completion port with a job. You can associate one completion port with a job.
//
// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-jobobject_associate_completion_port
type JOBOBJECT_ASSOCIATE_COMPLETION_PORT struct {
	CompletionKey  syscall.Handle
	CompletionPort syscall.Handle
}

// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-jobobject_notification_limit_information
type JOBOBJECT_NOTIFICATION_LIMIT_INFORMATION struct {
	IoReadBytesLimit             uint64
	IoWriteBytesLimit            uint64
	PerJobUserTimeLimit          int64
	JobMemoryLimit               uint64
	RateControlTolerance         JOBOBJECT_RATE_CONTROL_TOLERANCE
	RateControlToleranceInterval JOBOBJECT_RATE_CONTROL_TOLERANCE_INTERVAL
	LimitFlags                   LimitFlag
}

// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-jobobject_notification_limit_information_2
type JOBOBJECT_NOTIFICATION_LIMIT_INFORMATION_2 struct {
	IoReadBytesLimit                uint64
	IoWriteBytesLimit               uint64
	PerJobUserTimeLimit             int64
	JobHighMemoryLimit              uint64
	CpuRateControlTolerance         JOBOBJECT_RATE_CONTROL_TOLERANCE
	CpuRateControlToleranceInterval JOBOBJECT_RATE_CONTROL_TOLERANCE_INTERVAL
	LimitFlags                      LimitFlag
	IoRateControlTolerance          JOBOBJECT_RATE_CONTROL_TOLERANCE
	JobLowMemoryLimit               uint64
	IoRateControlToleranceInterval  JOBOBJECT_RATE_CONTROL_TOLERANCE_INTERVAL
	NetRateControlTolerance         JOBOBJECT_RATE_CONTROL_TOLERANCE
	NetRateControlToleranceInterval JOBOBJECT_RATE_CONTROL_TOLERANCE_INTERVAL
}

// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-jobobject_limit_violation_information
type JOBOBJECT_LIMIT_VIOLATION_INFORMATION struct {
	LimitFlags                LimitFlag
	ViolationLimitFlags       LimitFlag
	IoReadBytes               uint64
	IoReadBytesLimit          uint64
	IoWriteBytes              uint64
	IoWriteBytesLimit         uint64
	PerJobUserTime            int64
	PerJobUserTimeLimit       int64
	JobMemory                 uint64
	JobMemoryLimit            uint64
	RateControlTolerance      JOBOBJECT_RATE_CONTROL_TOLERANCE
	RateControlToleranceLimit JOBOBJECT_RATE_CONTROL_TOLERANCE
}

// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-jobobject_limit_violation_information_2
type JOBOBJECT_LIMIT_VIOLATION_INFORMATION_2 struct {
	LimitFlags                   LimitFlag
	ViolationLimitFlags          LimitFlag
	IoReadBytes                  uint64
	IoReadBytesLimit             uint64
	IoWriteBytes                 uint64
	IoWriteBytesLimit            uint64
	PerJobUserTime               uint64
	PerJobUserTimeLimit          uint64
	JobMemory                    uint64
	JobHighMemoryLimit           uint64
	CpuRateControlTolerance      JOBOBJECT_RATE_CONTROL_TOLERANCE
	CpuRateControlToleranceLimit JOBOBJECT_RATE_CONTROL_TOLERANCE
	JobLowMemoryLimit            uint64
	IORateControlTolerance       JOBOBJECT_RATE_CONTROL_TOLERANCE
	IORateControlToleranceLimit  JOBOBJECT_RATE_CONTROL_TOLERANCE
	NetRateControlTolerance      JOBOBJECT_RATE_CONTROL_TOLERANCE
	NetRateControlToleranceLimit JOBOBJECT_RATE_CONTROL_TOLERANCE
}

// IsProcessInJob determines whether the process is running in a job object.
//
// https://docs.microsoft.com/en-us/windows/desktop/api/jobapi/nf-jobapi-isprocessinjob
func IsProcessInJob(hProcess, hJobObject syscall.Handle) (bool, error) {
	var found bool
	ret, _, lastErr := isProcessInJob.Call(
		uintptr(hProcess),
		uintptr(hJobObject),
		uintptr(unsafe.Pointer(&found)))
	if ret == 0 {
		return found, os.NewSyscallError("IsProcessInJob", lastErr)
	}
	return found, nil
}

// OpenJobObject opens an existing job object.
//
// https://docs.microsoft.com/en-us/windows/desktop/api/jobapi2/nf-jobapi2-openjobobjectw
func OpenJobObject(desiredAccess, inheritHandles uintptr, jobName string) (syscall.Handle, error) {
	n, err := syscall.UTF16PtrFromString(jobName)
	if err != nil {
		return syscall.InvalidHandle, err
	}
	h, _, lastErr := openJobObject.Call(
		desiredAccess,
		inheritHandles,
		uintptr(unsafe.Pointer(n)))
	if h == 0 {
		return syscall.InvalidHandle, os.NewSyscallError("OpenJobObject", lastErr)
	}
	return syscall.Handle(h), nil
}

// CreateJobObject creates or opens a job object.
//
// https://docs.microsoft.com/en-us/windows/desktop/api/jobapi2/nf-jobapi2-createjobobjectw
func CreateJobObject(jobName string, sa *syscall.SecurityAttributes) (syscall.Handle, error) {
	n, err := syscall.UTF16PtrFromString(jobName)
	if err != nil {
		return syscall.InvalidHandle, err
	}
	h, _, lastErr := createJobObject.Call(
		uintptr(unsafe.Pointer(sa)),
		uintptr(unsafe.Pointer(n)))
	if h == 0 {
		return syscall.InvalidHandle, os.NewSyscallError("CreateJobObject", lastErr)
	}
	return syscall.Handle(h), nil
}

// TerminateJobObject terminates all processes currently associated with the
// job and removes the job object. exitCode it the exit code to be used by
// all processes and threads in the job object.
//
// https://docs.microsoft.com/en-us/windows/desktop/api/jobapi2/nf-jobapi2-terminatejobobject
func TerminateJobObject(hJobObject syscall.Handle, exitCode uint32) error {
	ret, _, lastErr := terminateJobObject.Call(
		uintptr(hJobObject),
		uintptr(exitCode))
	if ret == 0 {
		return os.NewSyscallError("TerminateJobObject", lastErr)
	}
	return nil
}

// AssignProcessToJobObject assigns a process to an existing job object.
//
// https://docs.microsoft.com/en-us/windows/desktop/api/jobapi2/nf-jobapi2-assignprocesstojobobject
func AssignProcessToJobObject(hJobObject, hProcess syscall.Handle) error {
	ret, _, lastErr := assignProcessToJobObject.Call(
		uintptr(hJobObject),
		uintptr(hProcess))
	if ret == 0 {
		return os.NewSyscallError("AssignProcessToJobObject", lastErr)
	}
	return nil
}

// QueryInformationJobObject retrieves limit and job state information for a job object.
//
// https://docs.microsoft.com/en-us/windows/desktop/api/jobapi2/nf-jobapi2-queryinformationjobobject
func QueryInformationJobObject(
	hJobObject syscall.Handle,
	infoClass JobObjectInformationClass,
	jobObjectInfo unsafe.Pointer,
	length uint32,
	retLen unsafe.Pointer) error {
	ret, _, lastErr := queryInformationJobObject.Call(
		uintptr(hJobObject),
		uintptr(infoClass),
		uintptr(jobObjectInfo),
		uintptr(length),
		uintptr(retLen))
	if ret == 0 {
		return os.NewSyscallError("QueryInformationJobObject", lastErr)
	}
	return nil
}

// SetInformationJobObject sets information for a job object.
//
// https://docs.microsoft.com/en-us/windows/desktop/api/jobapi2/nf-jobapi2-setinformationjobobject
func SetInformationJobObject(
	hJobObject syscall.Handle,
	infoClass JobObjectInformationClass,
	jobObjectInfo unsafe.Pointer,
	length uint32) error {
	ret, _, lastErr := setInformationJobObject.Call(
		uintptr(hJobObject),
		uintptr(infoClass),
		uintptr(jobObjectInfo),
		uintptr(length))
	if ret == 0 {
		return os.NewSyscallError("SetInformationJobObject", lastErr)
	}
	return nil
}

// QueryInfo performs QueryInformationJobObject call for the information class specified.
// A pointer to the appropriate information type must be provided.
func QueryInfo(hJobObject syscall.Handle, infoClass JobObjectInformationClass, v interface{}) error {
	var retLen uint32
	return QueryInformationJobObject(hJobObject, infoClass,
		unsafe.Pointer(reflect.ValueOf(v).Pointer()),
		uint32(reflect.TypeOf(v).Elem().Size()),
		unsafe.Pointer(&retLen))
}

// QueryInfo performs SetInformationJobObject call for the information class specified.
// A pointer to the appropriate information type must be provided.
func SetInfo(hJobObject syscall.Handle, infoClass JobObjectInformationClass, v interface{}) error {
	return SetInformationJobObject(hJobObject, infoClass,
		unsafe.Pointer(reflect.ValueOf(v).Pointer()),
		uint32(reflect.TypeOf(v).Elem().Size()))
}

// MakeSA creates a SECURITY_ATTRIBUTES structure that specifies the
// security descriptor for the job object and determines that child
// processes can not inherit the handle.
//
// https://docs.microsoft.com/en-us/previous-versions/windows/desktop/legacy/aa379560(v=vs.85)
func MakeSA() *syscall.SecurityAttributes {
	var sa syscall.SecurityAttributes
	sa.Length = uint32(unsafe.Sizeof(sa))
	sa.InheritHandle = 0
	return &sa
}

// AssociateCompletionPort associates a job object with a completion port.
// The system sends messages to the I/O completion port associated with a job
// when certain events occur. If the job is nested, the message is sent to
// every I/O completion port associated with any job in the parent job chain of
// the job that triggered the message.
//
// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-jobobject_associate_completion_port
func AssociateCompletionPort(hJobObject, hPort syscall.Handle) error {
	jacp := JOBOBJECT_ASSOCIATE_COMPLETION_PORT{
		CompletionKey:  hJobObject,
		CompletionPort: hPort,
	}
	err := SetInformationJobObject(
		hJobObject,
		JobObjectAssociateCompletionPortInformation,
		unsafe.Pointer(&jacp),
		uint32(unsafe.Sizeof(jacp)))
	if err != nil {
		return os.NewSyscallError("SetInformationJobObject", err)
	}
	return err
}

// GetQueuedCompletionStatus attempts to dequeue an I/O completion packet from
// the specified I/O completion port. If there is no completion packet queued,
// the function waits for a pending I/O operation associated with the
// completion port to complete.
//
// Timout is the number of milliseconds that the caller is willing to wait for
// a completion packet to appear at the completion port. If a completion packet
// does not appear within the specified time, the function times out. For
// infinite timeout syscall.INFINITE should be used.
//
// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-jobobject_associate_completion_port
func GetQueuedCompletionStatus(hPort syscall.Handle, timeout uint32) (mType uint32, pid uintptr, err error) {
	var (
		completionKey uint32
		overlapped    *syscall.Overlapped
	)
	err = syscall.GetQueuedCompletionStatus(
		hPort,
		&mType,
		&completionKey,
		&overlapped,
		timeout)
	if err != nil {
		return 0, 0, os.NewSyscallError("GetQueuedCompletionStatus", err)
	}
	return mType, uintptr(unsafe.Pointer(overlapped)), nil
}
