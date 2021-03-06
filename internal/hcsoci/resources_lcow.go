// +build windows

package hcsoci

// Contains functions relating to a LCOW container, as opposed to a utility VM

import (
	"fmt"
	"path"
	"strconv"

	"github.com/Microsoft/hcsshim/internal/schema2"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
)

const rootfsPath = "rootfs"
const mountPathPrefix = "m"

func allocateLinuxResources(coi *createOptionsInternal, resources *Resources) error {
	if coi.Spec.Root == nil {
		coi.Spec.Root = &specs.Root{}
	}
	if coi.Spec.Root.Path == "" {
		logrus.Debugln("hcsshim::allocateLinuxResources mounting storage")
		mcl, err := mountContainerLayers(coi.Spec.Windows.LayerFolders, resources.GuestRoot, coi.HostingSystem)
		if err != nil {
			return fmt.Errorf("failed to mount container storage: %s", err)
		}
		if coi.HostingSystem == nil {
			coi.Spec.Root.Path = mcl.(string) // Argon v1 or v2
		} else {
			coi.Spec.Root.Path = mcl.(schema2.CombinedLayersV2).ContainerRootPath // v2 Xenon LCOW
		}
		resources.Layers = coi.Spec.Windows.LayerFolders
	} else {
		hostPath := coi.Spec.Root.Path
		guestPath := path.Join(resources.GuestRoot, rootfsPath)
		flags := int32(0)
		if coi.Spec.Root.Readonly {
			flags |= schema2.VPlan9FlagReadOnly
		}
		err := coi.HostingSystem.AddPlan9(hostPath, guestPath, flags)
		if err != nil {
			return fmt.Errorf("adding plan9 root: %s", err)
		}
		coi.Spec.Root.Path = guestPath
		resources.Plan9Mounts = append(resources.Plan9Mounts, hostPath)
	}

	for i, mount := range coi.Spec.Mounts {
		if mount.Type != "bind" {
			continue
		}
		if mount.Destination == "" || mount.Source == "" {
			return fmt.Errorf("invalid OCI spec - a mount must have both source and a destination: %+v", mount)
		}
		if mount.Type != "" {
			return fmt.Errorf("invalid OCI spec - Type '%s' must not be set", mount.Type)
		}

		if coi.HostingSystem != nil {
			logrus.Debugf("hcsshim::allocateLinuxResources Hot-adding Plan9 for OCI mount %+v", mount)

			hostPath := mount.Source
			guestPath := path.Join(resources.GuestRoot, mountPathPrefix+strconv.Itoa(i))

			// TODO: Read-only
			var flags int32
			err := coi.HostingSystem.AddPlan9(hostPath, guestPath, flags)
			if err != nil {
				return fmt.Errorf("adding plan9 mount %+v: %s", mount, err)
			}
			coi.Spec.Mounts[i].Source = guestPath
			resources.Plan9Mounts = append(resources.Plan9Mounts, hostPath)
		}
	}

	return nil
}
