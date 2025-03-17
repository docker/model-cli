package system

import (
	"fmt"
)

type Version struct {
	Major string
	Minor string
	Patch string
}

func (v Version) String() string {
	var s string

	if v.Patch != "0" {
		s = fmt.Sprintf("%s.%s.%s", v.Major, v.Minor, v.Patch)
	} else if v.Minor != "0" {
		s = fmt.Sprintf("%s.%s", v.Major, v.Minor)
	} else {
		s = v.Major
	}

	return s
}
