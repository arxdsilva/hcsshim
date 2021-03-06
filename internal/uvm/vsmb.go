package uvm

import (
	"fmt"
	"strconv"

	"github.com/Microsoft/hcsshim/internal/schema2"
	"github.com/sirupsen/logrus"
)

func (share *vsmbShare) GuestPath() string {
	return `\\?\VMSMB\VSMB-{dcc079ae-60ba-4d07-847c-3493609c0870}\` + share.name
}

// AddVSMB adds a VSMB share to a utility VM. Each VSMB share is ref-counted and
// only added if it isn't already.
func (uvm *UtilityVM) AddVSMB(hostPath string, uvmPath string, flags int32) error {
	if uvm.operatingSystem != "windows" {
		return errNotSupported
	}

	logrus.Debugf("uvm::AddVSMB %s %s %d id:%s", hostPath, uvmPath, flags, uvm.id)
	uvm.m.Lock()
	defer uvm.m.Unlock()
	if uvm.vsmbShares == nil {
		uvm.vsmbShares = make(map[string]*vsmbShare)
	}
	share := uvm.vsmbShares[hostPath]
	if share == nil {
		uvm.vsmbCounter++
		shareName := "s" + strconv.FormatUint(uvm.vsmbCounter, 16)

		modification := &schema2.ModifySettingsRequestV2{
			ResourceType: schema2.ResourceTypeVSmbShare,
			RequestType:  schema2.RequestTypeAdd,
			Settings: schema2.VirtualMachinesResourcesStorageVSmbShareV2{
				Name:  shareName,
				Flags: flags,
				Path:  hostPath,
			},
			ResourceUri: fmt.Sprintf("virtualmachine/devices/virtualsmbshares/" + shareName),
		}

		// TODO: Hosted settings to support mapped directories on Windows
		if uvmPath != "" {
			panic("not yet implemented TODO TODO TODO - hostedSettings for VSMB")
		}

		if err := uvm.Modify(modification); err != nil {
			return err
		}
		share = &vsmbShare{
			name:    shareName,
			uvmPath: uvmPath,
		}
		uvm.vsmbShares[hostPath] = share
	}
	share.refCount++
	logrus.Debugf("hcsshim::AddVSMB Success %s: refcount=%d %+v", hostPath, share.refCount, share)
	return nil
}

// RemoveVSMB removes a VSMB share from a utility VM. Each VSMB share is ref-counted
// and only actually removed when the ref-count drops to zero.
func (uvm *UtilityVM) RemoveVSMB(hostPath string) error {
	if uvm.operatingSystem != "windows" {
		return errNotSupported
	}
	logrus.Debugf("uvm::RemoveVSMB %s id:%s", hostPath, uvm.id)
	uvm.m.Lock()
	defer uvm.m.Unlock()
	share := uvm.vsmbShares[hostPath]
	if share == nil {
		return fmt.Errorf("%s is not present as a VSMB share in %s, cannot remove", hostPath, uvm.id)
	}

	share.refCount--
	if share.refCount > 0 {
		logrus.Debugf("uvm::RemoveVSMB Success %s id:%s Ref-count now %d. It is still present in the utility VM", hostPath, uvm.id, share.refCount)
		return nil
	}
	logrus.Debugf("uvm::RemoveVSMB Zero ref-count, removing. %s id:%s", hostPath, uvm.id)
	modification := &schema2.ModifySettingsRequestV2{
		ResourceType: schema2.ResourceTypeVSmbShare,
		RequestType:  schema2.RequestTypeRemove,
		Settings:     schema2.VirtualMachinesResourcesStorageVSmbShareV2{Name: share.name},
		ResourceUri:  "virtualmachine/devices/virtualsmbshares/" + share.name,
	}
	if err := uvm.Modify(modification); err != nil {
		return fmt.Errorf("failed to remove vsmb share %s from %s: %s: %s", hostPath, uvm.id, modification, err)
	}
	logrus.Debugf("uvm::RemoveVSMB Success %s id:%s successfully removed from utility VM", hostPath, uvm.id)
	delete(uvm.vsmbShares, hostPath)
	return nil
}

// GetVSMBGuestPath returns the guest path of a VSMB mount.
func (uvm *UtilityVM) GetVSMBGuestPath(hostPath string) (string, error) {
	if hostPath == "" {
		return "", fmt.Errorf("no hostPath passed to GetVSMBShareCounter")
	}
	uvm.m.Lock()
	defer uvm.m.Unlock()
	share := uvm.vsmbShares[hostPath]
	if share == nil {
		return "", fmt.Errorf("%s not found as VSMB share in %s", hostPath, uvm.id)
	}
	path := share.GuestPath()
	logrus.Debugf("uvm::GetVSMBGuestPath Success %s id:%s path:%s", hostPath, uvm.id, path)
	return path, nil
}
