// +build windows

package winjob

import (
	"errors"
	"fmt"
	"sync"
	"syscall"

	"github.com/kolesnikovae/go-winjob/jobapi"
)

// Port is a wrapper for a job object IoCompletionPort.
//
// The system sends messages to the I/O completion port associated with a job
// when certain events occur. If the job is nested, the message is sent to
// every I/O completion port associated with any job in the parent job chain of
// the job that triggered the message. All messages are sent directly from the
// job as if the job had called the PostQueuedCompletionStatus function.
//
// Note that, except for limits set with JobObjectNotificationLimitInformation
// information class, message delivery to a completion port is not guaranteed.
// Notifications for limits set with JobObjectNotificationLimitInformation are
// guaranteed to arrive at the completion port.
type Port syscall.Handle

// Subscription is created when a new completion port is being associated
// with a job object. Refer to Notify function.
type Subscription struct {
	Port
	mu     sync.Mutex
	err    error
	closed bool
}

// Notification is a CompletionPort message related to a job object.
type Notification struct {
	Type NotificationType
	// If a message does not concern a particular process, the PID will be 0.
	PID int
}

type NotificationType string

const (
	NotificationEndOfJobTime        = "EndOfJobTime"
	NotificationEndOfProcessTime    = "EndOfProcessTime"
	NotificationActiveProcessLimit  = "ActiveProcessLimit"
	NotificationActiveProcessZero   = "ActiveProcessZero"
	NotificationNewProcess          = "NewProcess"
	NotificationExitProcess         = "ExitProcess"
	NotificationAbnormalExitProcess = "AbnormalExitProcess"
	NotificationProcessMemoryExit   = "ProcessMemoryExit"
	NotificationJobMemoryLimit      = "JobMemoryLimit"
	NotificationNotificationLimit   = "NotificationLimit"
	NotificationJobCycleLimit       = "JobCycleLimit"
	NotificationSiloTerminated      = "SiloTerminated"
)

var notificationTypes = map[jobapi.CompletionPortMessage]NotificationType{
	jobapi.JOB_OBJECT_MSG_END_OF_JOB_TIME:       NotificationEndOfJobTime,
	jobapi.JOB_OBJECT_MSG_END_OF_PROCESS_TIME:   NotificationEndOfProcessTime,
	jobapi.JOB_OBJECT_MSG_ACTIVE_PROCESS_LIMIT:  NotificationActiveProcessLimit,
	jobapi.JOB_OBJECT_MSG_ACTIVE_PROCESS_ZERO:   NotificationActiveProcessZero,
	jobapi.JOB_OBJECT_MSG_NEW_PROCESS:           NotificationNewProcess,
	jobapi.JOB_OBJECT_MSG_EXIT_PROCESS:          NotificationExitProcess,
	jobapi.JOB_OBJECT_MSG_ABNORMAL_EXIT_PROCESS: NotificationAbnormalExitProcess,
	jobapi.JOB_OBJECT_MSG_PROCESS_MEMORY_LIMIT:  NotificationProcessMemoryExit,
	jobapi.JOB_OBJECT_MSG_JOB_MEMORY_LIMIT:      NotificationJobMemoryLimit,
	jobapi.JOB_OBJECT_MSG_NOTIFICATION_LIMIT:    NotificationNotificationLimit,
	jobapi.JOB_OBJECT_MSG_JOB_CYCLE_TIME_LIMIT:  NotificationJobCycleLimit,
	jobapi.JOB_OBJECT_MSG_SILO_TERMINATED:       NotificationSiloTerminated,
}

func resolveNotificationType(mType jobapi.CompletionPortMessage) (NotificationType, bool) {
	t, ok := notificationTypes[mType]
	return t, ok
}

// CreatePort creates a new job object completion port for notifications and
// associates it with the given job object. If an association can not be
// established, the port handle is closed, and returned Port handle represents
// the actual handle state. Created Port must be disposed with a Close call.
func CreatePort(job *JobObject) (p Port, err error) {
	// https://docs.microsoft.com/en-us/windows/win32/fileio/createiocompletionport
	handle, err := syscall.CreateIoCompletionPort(
		syscall.InvalidHandle, // Ignore ExistingCompletionPort and CompletionKey.
		0,                     // ExistingCompletionPort
		0,                     // CompletionKey
		1,                     // NumberOfConcurrentThreads
	)
	if err != nil {
		return p, err
	}
	err = jobapi.AssociateCompletionPort(job.Handle, handle)
	if err != nil {
		_ = syscall.CloseHandle(handle)
	}
	return Port(handle), err
}

// Close disposes completion port handle.
func (p Port) Close() error {
	return syscall.CloseHandle(syscall.Handle(p))
}

// NextMessage blocks until the next completion port message is received,
// or a Close call, whichever occurs first. If a subscription is closed
// while the underlying GetQueuedCompletionStatus call was outstanding,
// a wrapped ErrAbandoned error will be returned.
func (p Port) NextMessage() (Notification, error) {
	mType, pid, err := jobapi.GetQueuedCompletionStatus(syscall.Handle(p), syscall.INFINITE)
	if err != nil {
		return Notification{}, err
	}
	typ, ok := resolveNotificationType(jobapi.CompletionPortMessage(mType))
	if !ok {
		typ = NotificationType(fmt.Sprintf("%v", mType))
	}
	m := Notification{
		Type: typ,
		PID:  int(pid),
	}
	return m, nil
}

// Notify causes job to relay notifications to the channel given. The channel
// is closed either on completion port polling error, or on subscription Close
// call.
func Notify(c chan<- Notification, job *JobObject) (*Subscription, error) {
	p, err := CreatePort(job)
	if err != nil {
		return nil, err
	}
	s := Subscription{Port: p}
	go s.notify(c)
	return &s, nil
}

// Close interrupts completion port polling, closes port handle and a channel
// provided to Notify call. The call is thread-safe and supposed to be
// performed concurrently with notification handling.
func (s *Subscription) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	if err := s.Port.Close(); err != nil {
		return err
	}
	s.closed = true
	return nil
}

// Err reports an error encountered during completion polling, if any.
// The call should be done after Notify channel close.
func (s *Subscription) Err() error {
	s.mu.Lock()
	err := s.err
	s.mu.Unlock()
	return err
}

func (s *Subscription) notify(c chan<- Notification) {
	defer close(c)
	for {
		m, err := s.Port.NextMessage()
		if err != nil {
			s.handlePortErr(err)
			return
		}
		c <- m
	}
}

func (s *Subscription) handlePortErr(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if errors.Is(err, jobapi.ErrAbandoned) && s.closed {
		return
	}
	s.err = err
}
