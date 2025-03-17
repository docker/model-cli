package system

import (
	"context"
	"fmt"
	"os"
)

func HostHasVirtualizationSupport(_ context.Context) error {
	_, err := os.Stat("/dev/kvm")
	return err
}

// UserCanAccessDevKVM checks if user can access /dev/kvm,
// permissions can be set via the acl / group membership
func UserCanAccessDevKVM(_ context.Context) error {
	f, err := os.OpenFile("/dev/kvm", os.O_RDWR, 0o666)
	if err == nil {
		_ = f.Close()
		return nil
	}
	return fmt.Errorf("user must be added to the kvm group to access the kvm device, see https://docs.docker.com/desktop/install/linux-install/#kvm-virtualization-support")
}
