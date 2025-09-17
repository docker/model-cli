# go-winjob
[![GoDoc](https://godoc.org/github.com/kolesnikovae/go-winjob?status.svg)](https://godoc.org/github.com/kolesnikovae/go-winjob/)
[![Go Report Card](https://goreportcard.com/badge/github.com/kolesnikovae/go-winjob)](https://goreportcard.com/report/github.com/kolesnikovae/go-winjob)
[![Build status](https://ci.appveyor.com/api/projects/status/yim6v5uws84x8ip6/branch/master?svg=true)](https://ci.appveyor.com/project/kolesnikovae/go-winjob/branch/master)
[![CodeCov](https://codecov.io/gh/kolesnikovae/go-winjob/branch/master/graph/badge.svg)](https://codecov.io/gh/kolesnikovae/go-winjob)

Go bindings for [Windows Job Objects](https://docs.microsoft.com/en-us/windows/win32/procthread/job-objects):
> A job object allows groups of processes to be managed as a unit. Job objects are namable, securable, sharable objects that control attributes of the processes associated with them. Operations performed on a job object affect all processes associated with the job object. Examples include enforcing limits such as working set size and process priority or terminating all processes associated with a job.

The package provides means to manage windows jobs. **jobapi** sub-package holds supplemental types and functions for low-level interactions with the operating system.

## Installation

To start using **go-winjob**, install Go 1.11 or above and run go get:
```
$ go get github.com/kolesnikovae/go-winjob
```

## Usage
### Creating and Managing Jobs

The example below demonstrates an efficient way to ensure no descendant processes will be left after the process exit:
```go
cmd := exec.Cmd("app.exe")
job, err := winjob.Start(cmd,
    winjob.LimitKillOnJobClose,
    winjob.LimitBreakawayOK)

if err != nil {
    // ...
}

defer job.Close()
if err := cmd.Wait(); err != nil {
    // ...
}
```

`LimitKillOnJobClose` acts similarly to `prctl(PR_SET_PDEATHSIG, SIGKILL)` in Linux: the job is destroyed when its last handle has been closed and all associated processes have been terminated. However, if the job has the `LimitKillOnJobClose`, closing the last job object handle terminates all associated processes and then destroys the job object itself.

The same result can be achieved by manual assignment:
<details>
  <summary>Show example</summary>  

   ```go
   job, _ := winjob.Create("",
       winjob.LimitKillOnJobClose,
       winjob.LimitBreakawayOK)
   
   cmd := exec.Cmd("app.exe")
   cmd.SysProcAttr = &windows.SysProcAttr{
       CreationFlags: windows.CREATE_SUSPENDED,
   }
   
   if err := cmd.Start(); err != nil {
       // ...
   }
   
   if err := job.Assign(cmd.Process); err != nil {
       // ...
   }
   
   if err := winjob.ResumeProcess(cmd.Process); err != nil {
       // ...
   }
   
   if err := cmd.Wait(); err != nil {
       // ...
   }
   ```

</details>

### Job Limits

**go-winjob** manages limits of the following types:
 - [x] Basic Limits
 - [x] Extended Limits
 - [x] UI Restriction
 - [x] CPU Rate Control
 - [x] Net Rate Control
 - [ ] IO Rate Control (Deprecated)
 - [ ] Notifications Limits
 - [ ] Violations Limits

Limits can be applied to a job object at any time either by one, or all together
(a full list can be found in the package documentation):
```go
limits := []winjob.Limit{
    winjob.WithKillOnJobClose(),
    winjob.WithWorkingSetLimit(1<<20, 8<<20),
    winjob.WithCPUHardCapLimit(5000),
    winjob.WithDSCPTag(0x14),
}

if err := job.SetLimits(limits...); err != nil {
    // ...
}

if err := job.ResetLimit(winjob.LimitKillOnJobClose); err != nil {
    // ...
}
```

Also, a particular limit value can be examined:
```go
if err := job.QueryLimits(); err != nil {
    // ...
}

winjob.LimitCPU(job).LimitValue()
// Output: {Min:0 Max:0 Weight:0 HardCap:500}
```
Alternatively, limit values are accessible via `JobInfo` member of a `JobObject`.

**Note**: limits should be explicitly queried with `job.QueryLimits()` before accessing their values.

### Job Notifications

A job can also set limits that trigger a notification when they are exceeded but allow the job to continue to run.

It is best to do this when the job is inactive, to reduce the chance of missing notifications for processes whose states change during the association of the completion port.
```go
c := make(chan winjob.Notification, 1)
s, err := winjob.Notify(c, job)
if err != nil {
    // ...
}

go func() {
    defer s.Close()
    for {
        select {
        case <-ctx.Done():
            return
        case n := <-c:
            switch n.Type {
            case winjob.NotificationNewProcess:
            	// ...
            case winjob.NotificationExitProcess:
            	// ...
            case winjob.NotificationNotificationLimit:
            	// Query limit violations.
            default:
            	log.Println(n.Type, n.PID)
            }
        }
    }
}()

if err := winjob.Start(cmd, limits...); err != nil {
    // ...
}
```

A full list of supported notification types can be found in the package documentation.

Note that, with the exception of limits set with the `JobObjectNotificationLimitInformation` information class explicitly, delivery of messages to the completion port is not guaranteed; failure of a message to arrive does not necessarily mean that the event did not occur.

Refer to `examples/job` for a full example.

### Resource Accounting for Jobs

A job object records basic and IO accounting information for all its associated processes, including those that have terminated:
```go
c, err := job.Counters()
if err != nil {
    // ...
}
```

JSON output:
```json
{
    "TotalUserTime": 156250,
    "TotalKernelTime": 156250,
    "ThisPeriodTotalUserTime": 156250,
    "ThisPeriodTotalKernelTime": 156250,
    "TotalPageFaultCount": 7900,
    "TotalProcesses": 2,
    "ActiveProcesses": 0,
    "TotalTerminatedProcesses": 0,
    "ReadOperationCount": 52,
    "WriteOperationCount": 0,
    "OtherOperationCount": 638,
    "ReadTransferCount": 202300,
    "WriteTransferCount": 0,
    "OtherTransferCount": 638
}
```
In order to avoid unnecessary allocations, `QueryCounters` method can be used instead:
```go
var counters winjob.Counters
if err := job.QueryCounters(&counters); err != nil {
	// ...
}
```
