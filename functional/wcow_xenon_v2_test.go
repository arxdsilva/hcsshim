// +build functional wcow wcowv2 wcowv2xenon

package functional

import (
	"os"
	"testing"

	"github.com/Microsoft/hcsshim/functional/utilities"
	"github.com/Microsoft/hcsshim/internal/guid"
	"github.com/Microsoft/hcsshim/internal/hcsoci"
	"github.com/Microsoft/hcsshim/internal/osversion"
	"github.com/Microsoft/hcsshim/internal/schemaversion"
	"github.com/Microsoft/hcsshim/internal/uvm"
	"github.com/Microsoft/hcsshim/internal/uvmfolder"
	"github.com/Microsoft/hcsshim/internal/wclayer"
	"github.com/Microsoft/hcsshim/internal/wcow"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// TODO: This is just copied out of the old wcow_xenon_test.go under hcsoci. Needs re-implementing.
// TODO: Also need to copy out the wcow v1
// TODO: Also need to copy of the wcow argon (v1/v2)

// Helper to create a utility VM. Returns the UtilityVM and folder used as its scratch
func createv2WCOWUVM(t *testing.T, uvmLayers []string, uvmId string, resources *specs.WindowsResources) (*uvm.UtilityVM, string) {
	if uvmId == "" {
		uvmId = guid.New().String()
	}

	uvmImageFolder, err := uvmfolder.LocateUVMFolder(uvmLayers)
	if err != nil {
		t.Fatal("Failed to locate UVM folder from layers")
	}

	scratchFolder := testutilities.CreateTempDir(t)
	if err := wcow.CreateUVMScratch(uvmImageFolder, scratchFolder, uvmId); err != nil {
		t.Fatalf("failed to create scratch: %s", err)
	}

	wcowUVM, err := uvm.Create(&uvm.UVMOptions{
		ID:              uvmId,
		OperatingSystem: "windows",
		LayerFolders:    append(uvmLayers, scratchFolder),
		Resources:       resources,
	})
	if err != nil {
		t.Fatalf("Failed create WCOW v2 UVM: %s", err)
	}
	if err := wcowUVM.Start(); err != nil {
		wcowUVM.Terminate()
		t.Fatalf("Failed start WCOW v2 UVM: %s", err)

	}
	return wcowUVM, scratchFolder
}

// Simple v2 Xenon
func TestV2XenonWCOW(t *testing.T) {
	testutilities.RequiresBuild(t, osversion.RS5)
	nanoserverLayers := testutilities.LayerFolders(t, "microsoft/nanoserver")
	wcowUVM, uvmScratchDir := createv2WCOWUVM(t, nanoserverLayers, "", nil)
	defer os.RemoveAll(uvmScratchDir)
	defer wcowUVM.Terminate()

	// Container scratch
	containerScratchDir := testutilities.CreateTempDir(t)
	defer os.RemoveAll(containerScratchDir)
	if err := wclayer.CreateScratchLayer(containerScratchDir, nanoserverLayers); err != nil {
		t.Fatalf("failed to create container scratch layer: %s", err)
	}

	// Containers OCI document
	spec := testutilities.GetDefaultWindowsSpec(t)
	spec.Process = &specs.Process{Args: []string{"cmd", "/s", "/c", "echo", "hello"}}
	spec.Windows.LayerFolders = append(nanoserverLayers, containerScratchDir)

	// Create and Start the container
	container, resources, err := CreateContainerTestWrapper(&hcsoci.CreateOptions{
		ID:            "container",
		Spec:          spec,
		SchemaVersion: schemaversion.SchemaV20(),
		HostingSystem: wcowUVM,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer container.Terminate()
	if err := container.Start(); err != nil {
		t.Fatal(err)
	}

	// Orderly cleanup
	hcsoci.ReleaseResources(resources, wcowUVM, true)
	container.Terminate()
}

//// TODO: Have a similar test where the UVM scratch folder does not exist.
//// A single WCOW xenon but where the container sandbox folder is not pre-created by the client
//func TestV2XenonWCOWContainerSandboxFolderDoesNotExist(t *testing.T) {
//	t.Skip("Skipping for now")
//	uvm, uvmScratchDir := createv2WCOWUVM(t, layersNanoserver, "TestV2XenonWCOWContainerSandboxFolderDoesNotExist_UVM", nil)
//	defer os.RemoveAll(uvmScratchDir)
//	defer uvm.Terminate()

//	// This is the important bit for this test. It's deleted here. We call the helper only to allocate a temporary directory
//	containerScratchDir := createTempDir(t)
//	os.RemoveAll(containerScratchDir)
//	defer os.RemoveAll(containerScratchDir) // As auto-created

//	layerFolders := append(layersBusybox, containerScratchDir)
//	hostedContainer, err := CreateContainer(&CreateOptions{
//		Id:            "container",
//		HostingSystem: uvm,
//		Spec:          &specs.Spec{Windows: &specs.Windows{LayerFolders: layerFolders}},
//	})
//	if err != nil {
//		t.Fatalf("CreateContainer failed: %s", err)
//	}
//	defer unmountContainerLayers(layerFolders, uvm, unmountOperationAll)

//	// Start/stop the container
//	startContainer(t, hostedContainer)
//	runCommand(t, hostedContainer, "cmd /s /c echo TestV2XenonWCOW", `c:\`, "TestV2XenonWCOW")
//	stopContainer(t, hostedContainer)
//	hostedContainer.Terminate()
//}

//// TODO What about mount. Test with the client doing the mount.
//// TODO Test as above, but where sandbox for UVM is entirely created by a client to show how it's done.

//// Two v2 WCOW containers in the same UVM, each with a single base layer
//func TestV2XenonWCOWTwoContainers(t *testing.T) {
//	t.Skip("Skipping for now")
//	uvm, uvmScratchDir := createv2WCOWUVM(t, layersNanoserver, "TestV2XenonWCOWTwoContainers_UVM", nil)
//	defer os.RemoveAll(uvmScratchDir)
//	defer uvm.Terminate()

//	// First hosted container
//	firstContainerScratchDir := createWCOWTempDirWithSandbox(t)
//	defer os.RemoveAll(firstContainerScratchDir)
//	firstLayerFolders := append(layersNanoserver, firstContainerScratchDir)
//	firstHostedContainer, err := CreateContainer(&CreateOptions{
//		Id:            "FirstContainer",
//		HostingSystem: uvm,
//		SchemaVersion: schemaversion.SchemaV20(),
//		Spec:          &specs.Spec{Windows: &specs.Windows{LayerFolders: firstLayerFolders}},
//	})
//	if err != nil {
//		t.Fatalf("CreateContainer failed: %s", err)
//	}
//	defer unmountContainerLayers(firstLayerFolders, uvm, unmountOperationAll)

//	// Second hosted container
//	secondContainerScratchDir := createWCOWTempDirWithSandbox(t)
//	defer os.RemoveAll(firstContainerScratchDir)
//	secondLayerFolders := append(layersNanoserver, secondContainerScratchDir)
//	secondHostedContainer, err := CreateContainer(&CreateOptions{
//		Id:            "SecondContainer",
//		HostingSystem: uvm,
//		SchemaVersion: schemaversion.SchemaV20(),
//		Spec:          &specs.Spec{Windows: &specs.Windows{LayerFolders: secondLayerFolders}},
//	})
//	if err != nil {
//		t.Fatalf("CreateContainer failed: %s", err)
//	}
//	defer unmountContainerLayers(secondLayerFolders, uvm, unmountOperationAll)

//	startContainer(t, firstHostedContainer)
//	runCommand(t, firstHostedContainer, "cmd /s /c echo FirstContainer", `c:\`, "FirstContainer")
//	startContainer(t, secondHostedContainer)
//	runCommand(t, secondHostedContainer, "cmd /s /c echo SecondContainer", `c:\`, "SecondContainer")
//	stopContainer(t, firstHostedContainer)
//	stopContainer(t, secondHostedContainer)
//	firstHostedContainer.Terminate()
//	secondHostedContainer.Terminate()
//}

////// This verifies the container storage is unmounted correctly so that a second
////// container can be started from the same storage.
////func TestV2XenonWCOWWithRemount(t *testing.T) {
//////	//t.Skip("Skipping for now")
////	uvmID := "Testv2XenonWCOWWithRestart_UVM"
////	uvmScratchDir, err := ioutil.TempDir("", "uvmScratch")
////	if err != nil {
////		t.Fatalf("Failed create temporary directory: %s", err)
////	}
////	if err := CreateWCOWSandbox(layersNanoserver[0], uvmScratchDir, uvmID); err != nil {
////		t.Fatalf("Failed create Windows UVM Sandbox: %s", err)
////	}
////	defer os.RemoveAll(uvmScratchDir)

////	uvm, err := CreateContainer(&CreateOptions{
////		Id:              uvmID,
////		Owner:           "unit-test",
////		SchemaVersion:   SchemaV20(),
////		IsHostingSystem: true,
////		Spec: &specs.Spec{
////			Windows: &specs.Windows{
////				LayerFolders: []string{uvmScratchDir},
////				HyperV:       &specs.WindowsHyperV{UtilityVMPath: filepath.Join(layersNanoserver[0], `UtilityVM\Files`)},
////			},
////		},
////	})
////	if err != nil {
////		t.Fatalf("Failed create UVM: %s", err)
////	}
////	defer uvm.Terminate()
////	if err := uvm.Start(); err != nil {
////		t.Fatalf("Failed start utility VM: %s", err)
////	}

////	// Mount the containers storage in the utility VM
////	containerScratchDir := createWCOWTempDirWithSandbox(t)
////	layerFolders := append(layersNanoserver, containerScratchDir)
////	cls, err := Mount(layerFolders, uvm, SchemaV20())
////	if err != nil {
////		t.Fatalf("failed to mount container storage: %s", err)
////	}
////	combinedLayers := cls.(CombinedLayersV2)
////	mountedLayers := &ContainersResourcesStorageV2{
////		Layers: combinedLayers.Layers,
////		Path:   combinedLayers.ContainerRootPath,
////	}
////	defer func() {
////		if err := Unmount(layerFolders, uvm, SchemaV20(), unmountOperationAll); err != nil {
////			t.Fatalf("failed to unmount container storage: %s", err)
////		}
////	}()

////	// Create the first container
////	defer os.RemoveAll(containerScratchDir)
////	xenon, err := CreateContainer(&CreateOptions{
////		Id:            "container",
////		Owner:         "unit-test",
////		HostingSystem: uvm,
////		SchemaVersion: SchemaV20(),
////		Spec:          &specs.Spec{Windows: &specs.Windows{}}, // No layerfolders as we mounted them ourself.
////	})
////	if err != nil {
////		t.Fatalf("CreateContainer failed: %s", err)
////	}

////	// Start/stop the first container
////	startContainer(t, xenon)
////	runCommand(t, xenon, "cmd /s /c echo TestV2XenonWCOWFirstStart", `c:\`, "TestV2XenonWCOWFirstStart")
////	stopContainer(t, xenon)
////	xenon.Terminate()

////	// Now unmount and remount to exactly the same places
////	if err := Unmount(layerFolders, uvm, SchemaV20(), unmountOperationAll); err != nil {
////		t.Fatalf("failed to unmount container storage: %s", err)
////	}
////	if _, err = Mount(layerFolders, uvm, SchemaV20()); err != nil {
////		t.Fatalf("failed to mount container storage: %s", err)
////	}

////	// Create an identical second container and verify it works too.
////	xenon2, err := CreateContainer(&CreateOptions{
////		Id:            "container",
////		Owner:         "unit-test",
////		HostingSystem: uvm,
////		SchemaVersion: SchemaV20(),
////		Spec:          &specs.Spec{Windows: &specs.Windows{LayerFolders: layerFolders}},
////		MountedLayers: mountedLayers,
////	})
////	if err != nil {
////		t.Fatalf("CreateContainer failed: %s", err)
////	}
////	startContainer(t, xenon2)
////	runCommand(t, xenon2, "cmd /s /c echo TestV2XenonWCOWAfterRemount", `c:\`, "TestV2XenonWCOWAfterRemount")
////	stopContainer(t, xenon2)
////	xenon2.Terminate()
////}

//// Lots of v2 WCOW containers in the same UVM, each with a single base layer. Containers aren't
//// actually started, but it stresses the SCSI controller hot-add logic.
//func TestV2XenonWCOWCreateLots(t *testing.T) {
//	t.Skip("Skipping for now")
//	uvm, uvmScratchDir := createv2WCOWUVM(t, layersNanoserver, "TestV2XenonWCOWCreateLots", nil)
//	defer os.RemoveAll(uvmScratchDir)
//	defer uvm.Terminate()

//	// 63 as 0:0 is already taken as the UVMs scratch. So that leaves us with 64-1 left for container scratches on SCSI
//	for i := 0; i < 63; i++ {
//		containerScratchDir := createWCOWTempDirWithSandbox(t)
//		defer os.RemoveAll(containerScratchDir)
//		layerFolders := append(layersNanoserver, containerScratchDir)
//		hostedContainer, err := CreateContainer(&CreateOptions{
//			Id:            fmt.Sprintf("container%d", i),
//			HostingSystem: uvm,
//			SchemaVersion: schemaversion.SchemaV20(),
//			Spec:          &specs.Spec{Windows: &specs.Windows{LayerFolders: layerFolders}},
//		})
//		if err != nil {
//			t.Fatalf("CreateContainer failed: %s", err)
//		}
//		defer hostedContainer.Terminate()
//		defer unmountContainerLayers(layerFolders, uvm, unmountOperationAll)
//	}

//	// TODO: Should check the internal structures here for VSMB and SCSI

//	// TODO: Push it over 63 now and will get a failure.
//}

//// TestV2XenonWCOWMultiLayer creates a V2 Xenon having multiple image layers
//func TestV2XenonWCOWMultiLayer(t *testing.T) {
//	t.Skip("for now")

//	uvmMemory := uint64(1 * 1024 * 1024 * 1024)
//	uvmCPUCount := uint64(2)
//	resources := &specs.WindowsResources{
//		Memory: &specs.WindowsMemoryResources{
//			Limit: &uvmMemory,
//		},
//		CPU: &specs.WindowsCPUResources{
//			Count: &uvmCPUCount,
//		},
//	}
//	uvm, uvmScratchDir := createv2WCOWUVM(t, layersNanoserver, "TestV2XenonWCOWMultiLayer_UVM", resources)
//	defer os.RemoveAll(uvmScratchDir)
//	defer uvm.Terminate()

//	// Create a sandbox for the hosted container
//	containerScratchDir := createWCOWTempDirWithSandbox(t)
//	defer os.RemoveAll(containerScratchDir)

//	// Create the container. Note that this will auto-mount for us.
//	containerLayers := append(layersBusybox, containerScratchDir)
//	xenon, err := CreateContainer(&CreateOptions{
//		Id:            "container",
//		HostingSystem: uvm,
//		Spec:          &specs.Spec{Windows: &specs.Windows{LayerFolders: containerLayers}},
//	})
//	if err != nil {
//		t.Fatalf("CreateContainer failed: %s", err)
//	}

//	// Start/stop the container
//	startContainer(t, xenon)
//	runCommand(t, xenon, "echo Container", `c:\`, "Container")
//	stopContainer(t, xenon)
//	xenon.Terminate()
//	// TODO Move this to a defer function to fail if it fails.
//	if err := unmountContainerLayers(containerLayers, uvm, unmountOperationAll); err != nil {
//		t.Fatalf("unmount failed: %s", err)
//	}

//}

//// TestV2XenonWCOWSingleMappedDirectory tests a V2 Xenon WCOW with a single mapped directory
//func TestV2XenonWCOWSingleMappedDirectory(t *testing.T) {
//	t.Skip("Skipping for now")
//	uvm, uvmScratchDir := createv2WCOWUVM(t, layersNanoserver, "", nil)
//	defer os.RemoveAll(uvmScratchDir)
//	defer uvm.Terminate()

//	// Create the container hosted inside the utility VM
//	containerScratchDir := createWCOWTempDirWithSandbox(t)
//	defer os.RemoveAll(containerScratchDir)
//	layerFolders := append(layersNanoserver, containerScratchDir)

//	// Create a temp folder containing foo.txt which will be used for the bind-mount test.
//	source := createTempDir(t)
//	defer os.RemoveAll(source)
//	mount := specs.Mount{
//		Source:      source,
//		Destination: `c:\foo`,
//	}
//	f, err := os.OpenFile(filepath.Join(source, "foo.txt"), os.O_RDWR|os.O_CREATE, 0755)
//	f.Close()

//	hostedContainer, err := CreateContainer(&CreateOptions{
//		HostingSystem: uvm,
//		Spec: &specs.Spec{
//			Windows: &specs.Windows{LayerFolders: layerFolders},
//			Mounts:  []specs.Mount{mount},
//		},
//	})
//	if err != nil {
//		t.Fatalf("CreateContainer failed: %s", err)
//	}
//	defer unmountContainerLayers(layerFolders, uvm, unmountOperationAll)

//	// TODO BUGBUG NEED TO UNMOUNT TO VSMB SHARE FOR THE CONTAINER

//	// Start/stop the container
//	startContainer(t, hostedContainer)
//	runCommand(t, hostedContainer, `cmd /s /c dir /b c:\foo`, `c:\`, "foo.txt")
//	stopContainer(t, hostedContainer)
//	hostedContainer.Terminate()
//}
