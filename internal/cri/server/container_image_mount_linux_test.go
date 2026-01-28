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
)

func TestGetImageVolumeSnapshotOpts(t *testing.T) {
	ctx := context.Background()

	for _, test := range []struct {
		name        string
		mount       *runtime.Mount
		expectOpts  bool
		expectError bool
	}{
		{
			name: "with user namespace mappings",
			mount: &runtime.Mount{
				ContainerPath: "/data",
				UidMappings: []*runtime.IDMapping{
					{
						ContainerId: 0,
						HostId:      65536,
						Length:      65536,
					},
				},
				GidMappings: []*runtime.IDMapping{
					{
						ContainerId: 0,
						HostId:      65536,
						Length:      65536,
					},
				},
			},
			expectOpts:  true,
			expectError: false,
		},
		{
			name: "without user namespace mappings",
			mount: &runtime.Mount{
				ContainerPath: "/data",
			},
			expectOpts:  false,
			expectError: false,
		},
		{
			name: "with nil UID mappings",
			mount: &runtime.Mount{
				ContainerPath: "/data",
				UidMappings:   nil,
				GidMappings: []*runtime.IDMapping{
					{
						ContainerId: 0,
						HostId:      65536,
						Length:      65536,
					},
				},
			},
			expectOpts:  false,
			expectError: false,
		},
		{
			name: "with nil GID mappings",
			mount: &runtime.Mount{
				ContainerPath: "/data",
				UidMappings: []*runtime.IDMapping{
					{
						ContainerId: 0,
						HostId:      65536,
						Length:      65536,
					},
				},
				GidMappings: nil,
			},
			expectOpts:  false,
			expectError: false,
		},
		{
			name: "with empty UID mappings",
			mount: &runtime.Mount{
				ContainerPath: "/data",
				UidMappings:   []*runtime.IDMapping{},
				GidMappings: []*runtime.IDMapping{
					{
						ContainerId: 0,
						HostId:      65536,
						Length:      65536,
					},
				},
			},
			expectOpts:  false,
			expectError: false,
		},
		{
			name: "with empty GID mappings",
			mount: &runtime.Mount{
				ContainerPath: "/data",
				UidMappings: []*runtime.IDMapping{
					{
						ContainerId: 0,
						HostId:      65536,
						Length:      65536,
					},
				},
				GidMappings: []*runtime.IDMapping{},
			},
			expectOpts:  false,
			expectError: false,
		},
		{
			name: "with multiple UID mappings (should error)",
			mount: &runtime.Mount{
				ContainerPath: "/data",
				UidMappings: []*runtime.IDMapping{
					{
						ContainerId: 0,
						HostId:      65536,
						Length:      32768,
					},
					{
						ContainerId: 32768,
						HostId:      98304,
						Length:      32768,
					},
				},
				GidMappings: []*runtime.IDMapping{
					{
						ContainerId: 0,
						HostId:      65536,
						Length:      65536,
					},
				},
			},
			expectOpts:  false,
			expectError: true,
		},
		{
			name: "with invalid UID mapping length (should error)",
			mount: &runtime.Mount{
				ContainerPath: "/data",
				UidMappings: []*runtime.IDMapping{
					{
						ContainerId: 0,
						HostId:      65536,
						Length:      0, // Invalid length
					},
				},
				GidMappings: []*runtime.IDMapping{
					{
						ContainerId: 0,
						HostId:      65536,
						Length:      65536,
					},
				},
			},
			expectOpts:  false,
			expectError: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := &criService{}

			opts, err := c.getImageVolumeSnapshotOpts(ctx, test.mount)

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
