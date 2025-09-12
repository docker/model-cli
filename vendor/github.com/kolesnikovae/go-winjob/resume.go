// +build windows

package winjob

import (
	"fmt"
	"os/exec"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Start creates a job object with the limits specified and starts the given
// command within the job. The process is created with suspended threads which
// are resumed when the process has been added to the job object.
func Start(cmd *exec.Cmd, limits ...Limit) (*JobObject, error) {
	job, err := Create("", limits...)
	if err != nil {
		return nil, err
	}
	if err := StartInJobObject(cmd, job); err != nil {
		_ = job.Close()
		return nil, err
	}
	return job, nil
}

// StartInJobObject starts the given command within the job objects specified.
// The process is created with suspended threads which are resumed when the
// process is added to the job.
func StartInJobObject(cmd *exec.Cmd, job *JobObject) error {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = new(windows.SysProcAttr)
	}
	cmd.SysProcAttr.CreationFlags |= windows.CREATE_SUSPENDED
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := job.Assign(cmd.Process); err != nil {
		return err
	}
	return Resume(cmd)
}

// Resume resumes the process of the given command. The command should be
// created with CREATE_SUSPENDED flag:
//
//  cmd.SysProcAttr = &windows.SysProcAttr{
//    CreationFlags: windows.CREATE_SUSPENDED,
//  }
//
// CREATE_SUSPENDED specifies that the primary thread of the new process is
// created in a suspended state, and does not run until the ResumeThread
// windows function is called.
func Resume(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return fmt.Errorf("process is nil")
	}
	return ResumeProcess(cmd.Process.Pid)
}

// ResumeProcess resumes the first found thread of the process.
func ResumeProcess(pid int) (err error) {
	s, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, uint32(pid))
	if err != nil {
		return fmt.Errorf("CreateToolhelp32Snapshot: %w", err)
	}
	defer func() {
		_ = windows.Close(s)
	}()

	var e windows.ThreadEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	if err := windows.Thread32First(s, &e); err != nil {
		return fmt.Errorf("Thread32First: %w", err)
	}

	for {
		err := windows.Thread32Next(s, &e)
		switch err {
		default:
			return fmt.Errorf("Thread32Next: %w", err)
		case windows.ERROR_NO_MORE_FILES:
			return fmt.Errorf("no threads found")
		case nil:
			if int(e.OwnerProcessID) == pid && e.ThreadID != 0 {
				return ResumeThread(e.ThreadID)
			}
		}
	}
}

// ResumeThread resumes given thread.
func ResumeThread(tid uint32) (err error) {
	hThread, err := windows.OpenThread(windows.THREAD_SUSPEND_RESUME, false, tid)
	if err != nil {
		return fmt.Errorf("OpenThread: %w", err)
	}
	defer func() {
		_ = windows.Close(hThread)
	}()
	if _, err = windows.ResumeThread(hThread); err != nil {
		return fmt.Errorf("ResumeThread: %w", err)
	}
	return nil
}
