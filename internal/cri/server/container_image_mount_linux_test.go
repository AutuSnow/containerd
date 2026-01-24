/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package server

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	runtime "k8s.io/cri-api/pkg/apis/runtime/v1"

	"github.com/containerd/containerd/v2/internal/cri/store/label"
	sandboxstore "github.com/containerd/containerd/v2/internal/cri/store/sandbox"
)

func TestGetImageVolumeSnapshotOpts(t *testing.T) {
	ctx := context.Background()

	for _, test := range []struct {
		name          string
		sandboxConfig *runtime.PodSandboxConfig
		expectOpts    bool
		expectError   bool
	}{
		{
			name: "with user namespace enabled",
			sandboxConfig: &runtime.PodSandboxConfig{
				Metadata: &runtime.PodSandboxMetadata{
					Name:      "test-sandbox",
					Uid:       "test-uid",
					Namespace: "test-ns",
				},
				Linux: &runtime.LinuxPodSandboxConfig{
					SecurityContext: &runtime.LinuxSandboxSecurityContext{
						NamespaceOptions: &runtime.NamespaceOption{
							UsernsOptions: &runtime.UserNamespace{
								Mode: runtime.NamespaceMode_POD,
								Uids: []*runtime.IDMapping{
									{
										ContainerId: 0,
										HostId:      65536,
										Length:      65536,
									},
								},
								Gids: []*runtime.IDMapping{
									{
										ContainerId: 0,
										HostId:      65536,
										Length:      65536,
									},
								},
							},
						},
					},
				},
			},
			expectOpts:  true,
			expectError: false,
		},
		{
			name: "without user namespace",
			sandboxConfig: &runtime.PodSandboxConfig{
				Metadata: &runtime.PodSandboxMetadata{
					Name:      "test-sandbox",
					Uid:       "test-uid",
					Namespace: "test-ns",
				},
				Linux: &runtime.LinuxPodSandboxConfig{
					SecurityContext: &runtime.LinuxSandboxSecurityContext{
						NamespaceOptions: &runtime.NamespaceOption{
							UsernsOptions: &runtime.UserNamespace{
								Mode: runtime.NamespaceMode_NODE,
							},
						},
					},
				},
			},
			expectOpts:  false,
			expectError: false,
		},
		{
			name: "with nil Linux config",
			sandboxConfig: &runtime.PodSandboxConfig{
				Metadata: &runtime.PodSandboxMetadata{
					Name:      "test-sandbox",
					Uid:       "test-uid",
					Namespace: "test-ns",
				},
				Linux: nil,
			},
			expectOpts:  false,
			expectError: false,
		},
		{
			name: "with nil namespace options",
			sandboxConfig: &runtime.PodSandboxConfig{
				Metadata: &runtime.PodSandboxMetadata{
					Name:      "test-sandbox",
					Uid:       "test-uid",
					Namespace: "test-ns",
				},
				Linux: &runtime.LinuxPodSandboxConfig{
					SecurityContext: &runtime.LinuxSandboxSecurityContext{
						NamespaceOptions: nil,
					},
				},
			},
			expectOpts:  false,
			expectError: false,
		},
		{
			name: "with nil userns options",
			sandboxConfig: &runtime.PodSandboxConfig{
				Metadata: &runtime.PodSandboxMetadata{
					Name:      "test-sandbox",
					Uid:       "test-uid",
					Namespace: "test-ns",
				},
				Linux: &runtime.LinuxPodSandboxConfig{
					SecurityContext: &runtime.LinuxSandboxSecurityContext{
						NamespaceOptions: &runtime.NamespaceOption{
							UsernsOptions: nil,
						},
					},
				},
			},
			expectOpts:  false,
			expectError: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			// Create a test CRI service with a sandbox store
			labels := label.NewStore()
			c := &criService{
				sandboxStore: sandboxstore.NewStore(labels, nil),
			}

			// Create and add a sandbox to the store
			sandboxID := "test-sandbox-id"
			sandbox := sandboxstore.NewSandbox(
				sandboxstore.Metadata{
					ID:     sandboxID,
					Name:   test.sandboxConfig.Metadata.Name,
					Config: test.sandboxConfig,
				},
				sandboxstore.Status{},
			)
			c.sandboxStore.Add(sandbox)

			opts, err := c.getImageVolumeSnapshotOpts(ctx, sandboxID)

			if test.expectError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			if test.expectOpts {
				assert.NotEmpty(t, opts, "expected snapshot options to be returned")
			} else {
				assert.Empty(t, opts, "expected no snapshot options")
			}
		})
	}
}

func TestGetImageVolumeSnapshotOpts_SandboxNotFound(t *testing.T) {
	ctx := context.Background()

	labels := label.NewStore()
	c := &criService{
		sandboxStore: sandboxstore.NewStore(labels, nil),
	}

	// Try to get options for a non-existent sandbox
	_, err := c.getImageVolumeSnapshotOpts(ctx, "non-existent-sandbox")
	assert.Error(t, err, "expected error when sandbox not found")
	assert.Contains(t, err.Error(), "failed to get sandbox", "error should mention sandbox not found")
}

func TestGetImageVolumeSnapshotOpts_InvalidUserNamespace(t *testing.T) {
	ctx := context.Background()

	// Test with invalid user namespace configuration (POD mode without mappings)
	sandboxConfig := &runtime.PodSandboxConfig{
		Metadata: &runtime.PodSandboxMetadata{
			Name:      "test-sandbox",
			Uid:       "test-uid",
			Namespace: "test-ns",
		},
		Linux: &runtime.LinuxPodSandboxConfig{
			SecurityContext: &runtime.LinuxSandboxSecurityContext{
				NamespaceOptions: &runtime.NamespaceOption{
					UsernsOptions: &runtime.UserNamespace{
						Mode: runtime.NamespaceMode_POD,
						// Missing Uids and Gids - this should cause an error
					},
				},
			},
		},
	}

	labels := label.NewStore()
	c := &criService{
		sandboxStore: sandboxstore.NewStore(labels, nil),
	}

	sandboxID := "test-sandbox-id"
	sandbox := sandboxstore.NewSandbox(
		sandboxstore.Metadata{
			ID:     sandboxID,
			Name:   sandboxConfig.Metadata.Name,
			Config: sandboxConfig,
		},
		sandboxstore.Status{},
	)
	c.sandboxStore.Add(sandbox)

	_, err := c.getImageVolumeSnapshotOpts(ctx, sandboxID)
	assert.Error(t, err, "expected error with invalid user namespace configuration")
	assert.Contains(t, err.Error(), "failed to get snapshotter remap options", "error should mention remap options failure")
}
