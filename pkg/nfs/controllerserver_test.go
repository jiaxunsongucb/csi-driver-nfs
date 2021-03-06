package nfs

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"golang.org/x/net/context"
	"k8s.io/utils/mount"
)

const (
	testServer    = "test-server"
	testBaseDir   = "test-base-dir"
	testCSIVolume = "test-csi"
	testVolumeId  = "test-server/test-base-dir/test-csi"
)

// for Windows support in the future
var (
	testShare = filepath.Join(string(filepath.Separator), testBaseDir, string(filepath.Separator), testCSIVolume)
)

func initTestController(t *testing.T) *ControllerServer {
	var perm *uint32
	mounter := &mount.FakeMounter{MountPoints: []mount.MountPoint{}}
	driver := NewNFSdriver("", "", perm)
	driver.ns = NewNodeServer(driver, mounter)
	return NewControllerServer(driver)
}

func TestCreateVolume(t *testing.T) {
	cases := []struct {
		name      string
		req       *csi.CreateVolumeRequest
		resp      *csi.CreateVolumeResponse
		expectErr bool
	}{
		{
			name: "valid defaults",
			req: &csi.CreateVolumeRequest{
				Name: testCSIVolume,
				VolumeCapabilities: []*csi.VolumeCapability{
					{
						AccessType: &csi.VolumeCapability_Mount{
							Mount: &csi.VolumeCapability_MountVolume{},
						},
						AccessMode: &csi.VolumeCapability_AccessMode{
							Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
						},
					},
				},
				Parameters: map[string]string{
					paramServer: testServer,
					paramShare:  testBaseDir,
				},
			},
			resp: &csi.CreateVolumeResponse{
				Volume: &csi.Volume{
					VolumeId: testVolumeId,
					VolumeContext: map[string]string{
						paramServer: testServer,
						paramShare:  testShare,
					},
				},
			},
		},
		{
			name: "name empty",
			req: &csi.CreateVolumeRequest{
				VolumeCapabilities: []*csi.VolumeCapability{
					{
						AccessType: &csi.VolumeCapability_Mount{
							Mount: &csi.VolumeCapability_MountVolume{},
						},
						AccessMode: &csi.VolumeCapability_AccessMode{
							Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
						},
					},
				},
				Parameters: map[string]string{
					paramServer: testServer,
					paramShare:  testBaseDir,
				},
			},
			expectErr: true,
		},
		{
			name: "invalid volume capability",
			req: &csi.CreateVolumeRequest{
				Name: testCSIVolume,
				VolumeCapabilities: []*csi.VolumeCapability{
					{
						AccessMode: &csi.VolumeCapability_AccessMode{
							Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
						},
					},
				},
				Parameters: map[string]string{
					paramServer: testServer,
					paramShare:  testBaseDir,
				},
			},
			expectErr: true,
		},
		{
			name: "invalid create context",
			req: &csi.CreateVolumeRequest{
				Name: testCSIVolume,
				VolumeCapabilities: []*csi.VolumeCapability{
					{
						AccessType: &csi.VolumeCapability_Mount{
							Mount: &csi.VolumeCapability_MountVolume{},
						},
						AccessMode: &csi.VolumeCapability_AccessMode{
							Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
						},
					},
				},
				Parameters: map[string]string{
					"unknown-parameter": "foo",
				},
			},
			expectErr: true,
		},
	}

	for _, test := range cases {
		test := test //pin
		t.Run(test.name, func(t *testing.T) {
			// Setup
			cs := initTestController(t)
			// Run
			resp, err := cs.CreateVolume(context.TODO(), test.req)

			// Verify
			if !test.expectErr && err != nil {
				t.Errorf("test %q failed: %v", test.name, err)
			}
			if test.expectErr && err == nil {
				t.Errorf("test %q failed; got success", test.name)
			}
			if !reflect.DeepEqual(resp, test.resp) {
				t.Errorf("test %q failed: got resp %+v, expected %+v", test.name, resp, test.resp)
			}
			if !test.expectErr {
				info, err := os.Stat(filepath.Join(cs.workingMountDir, test.req.Name, test.req.Name))
				if err != nil {
					t.Errorf("test %q failed: couldn't find volume subdirectory: %v", test.name, err)
				}
				if !info.IsDir() {
					t.Errorf("test %q failed: subfile not a directory", test.name)
				}
			}
		})
	}
}
