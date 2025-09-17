# jobapi

Go bindings for [Windows Job Objects](https://docs.microsoft.com/en-us/windows/win32/procthread/job-objects):
> A job object allows groups of processes to be managed as a unit. Job objects are namable, securable, sharable objects that control attributes of the processes associated with them. Operations performed on a job object affect all processes associated with the job object. Examples include enforcing limits such as working set size and process priority or terminating all processes associated with a job.

The package holds supplemental types and functions for low-level interactions with the operating system.

Golang naming convention is sacrificed in favor of straight name mapping: WinAPI uses mixed ALL_CAPS and CaseCamel. To avoid any confusion, naming conforms Microsoft documentation.

### Limitations
 - Sessions/Terminal Services: all processes within a job must run within the same session as the job. An attempt to assign a process from another session will fail with `ERROR_ACCESS_DENIED`.
 - [Nested jobs](https://docs.microsoft.com/en-us/windows/win32/procthread/nested-jobs) were introduced in Windows 8 and Windows Server 2012. On Windows 7 `AssignProcessToJob` call will fail with `ERROR_ACCESS_DENIED`, if the process is already assigned to a job.
 - IO rate controls had been introduced into api-ms-win-core-job-l2-1-1.dll in 10.0.10240 and removed in 10.0.15063.
 - Support for `JOBOBJECT_SECURITY_LIMIT_INFORMATION` was removed starting with Windows Vista.
