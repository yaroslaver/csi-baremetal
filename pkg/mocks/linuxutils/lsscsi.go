/*
Copyright © 2020 Dell Inc. or its subsidiaries. All Rights Reserved.

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

package linuxutils

import (
	"github.com/stretchr/testify/mock"

	"github.com/dell/csi-baremetal/pkg/base/linuxutils/lsscsi"
)

// MockWrapLsscsi is a mock implementation of WrapLsscsi interface from lsscsi package
type MockWrapLsscsi struct {
	mock.Mock
}

// GetSCSIDevices is a mock implementations
func (m *MockWrapLsscsi) GetSCSIDevices() ([]*lsscsi.SCSIDevice, error) {
	args := m.Mock.Called()

	return args.Get(0).([]*lsscsi.SCSIDevice), args.Error(1)
}
