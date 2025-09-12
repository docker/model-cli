// +build windows

package winjob

import (
	"bytes"
	"encoding/binary"

	"github.com/kolesnikovae/go-winjob/jobapi"
)

// WithCPUHardCapLimit controls CPU rate with hard limit, the value specifies
// the portion of processor cycles that the threads in the job object can use
// during each scheduling interval, as the number of cycles per 10,000 cycles,
// e.g.: to limit a job object with 12.34% CPU rate, the value must be 1234.
// The value must be in range 1-10,000.
//
// The limit cannot be used with WithCPUWeightedLimit or WithCPUMinMaxLimit.
func WithCPUHardCapLimit(v uint32) Limit {
	return LimitCPU.WithValue(CPURate{HardCap: v})
}

// WithCPUWeightedLimit specifies the scheduling weight of the job object,
// which determines the share of processor time given to the job relative to
// other workloads on the processor. This member can be a value from 1 through
// 9, where 1 is the smallest share and 9 is the largest share. The default is
// 5, which should be used for most workloads.
//
// The limit cannot be used with WithCPUHardCapLimit or WithCPUMinMaxLimit.
func WithCPUWeightedLimit(v uint32) Limit {
	return LimitCPU.WithValue(CPURate{Weight: v})
}

// WithCPUMinMaxLimit specifies min and max portions of processor cycles that
// the job object can reserve and use during each scheduling interval.
//
// Specify this rate as a percentage times 100. For example, to set a minimum
// rate of 50%, specify 50 times 100, or 5,000. For the minimum rates to work
// correctly, the sum of the minimum rates for all of the job objects in the
// system cannot exceed 10,000, which is the equivalent of 100%.
//
// The limit cannot be used with WithCPUHardCapLimit or WithCPUWeightedLimit.
func WithCPUMinMaxLimit(min, max uint16) Limit {
	return LimitCPU.WithValue(CPURate{Min: min, Max: max})
}

var LimitCPU cpuLimit

type CPURate struct {
	Min     uint16
	Max     uint16
	Weight  uint32
	HardCap uint32
}

type cpuLimit CPURate

func (l cpuLimit) reset(job *JobObject) {
	job.CPURateControl.ControlFlags = 0
}

func (l cpuLimit) IsSet(job *JobObject) bool {
	return job.CPURateControl.ControlFlags != 0
}

func (l cpuLimit) Value(job *JobObject) interface{} {
	return l.LimitValue(job)
}

func (l cpuLimit) WithValue(x CPURate) cpuLimit {
	return cpuLimit(x)
}

func (l cpuLimit) LimitValue(job *JobObject) CPURate {
	var r CPURate
	switch {
	case job.CPURateControl.ControlFlags&jobapi.JOB_OBJECT_CPU_RATE_CONTROL_HARD_CAP > 0:
		r.HardCap = job.CPURateControl.Value
	case job.CPURateControl.ControlFlags&jobapi.JOB_OBJECT_CPU_RATE_CONTROL_WEIGHT_BASED > 0:
		r.Weight = job.CPURateControl.Value
	case job.CPURateControl.ControlFlags&jobapi.JOB_OBJECT_CPU_RATE_CONTROL_MIN_MAX_RATE > 0:
		var b bytes.Buffer
		_ = binary.Write(&b, binary.LittleEndian, job.CPURateControl.Value)
		r.Min = binary.LittleEndian.Uint16(b.Bytes()[:2])
		r.Max = binary.LittleEndian.Uint16(b.Bytes()[2:])
	}
	return r
}

func (l cpuLimit) set(job *JobObject) {
	var f jobapi.CPUControlFlag
	switch {
	case l.HardCap > 0:
		job.CPURateControl.Value = l.HardCap
		f = jobapi.JOB_OBJECT_CPU_RATE_CONTROL_HARD_CAP
	case l.Weight > 0:
		job.CPURateControl.Value = l.Weight
		f = jobapi.JOB_OBJECT_CPU_RATE_CONTROL_WEIGHT_BASED
	case l.Max > 0:
		f = jobapi.JOB_OBJECT_CPU_RATE_CONTROL_MIN_MAX_RATE
		var b bytes.Buffer
		_ = binary.Write(&b, binary.LittleEndian, l.Min)
		_ = binary.Write(&b, binary.LittleEndian, l.Max)
		job.CPURateControl.Value = binary.LittleEndian.Uint32(b.Bytes())
	}
	job.CPURateControl.ControlFlags = f |
		jobapi.JOB_OBJECT_CPU_RATE_CONTROL_ENABLE |
		jobapi.JOB_OBJECT_CPU_RATE_CONTROL_NOTIFY
}
