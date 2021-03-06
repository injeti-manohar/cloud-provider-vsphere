/*
Copyright 2018 The Kubernetes Authors.

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

package fcd

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"

	cm "k8s.io/cloud-provider-vsphere/pkg/common/connectionmanager"
	"k8s.io/cloud-provider-vsphere/pkg/common/vclib"
)

const (
	NUM_OF_CONNECTION_ATTEMPTS     int = 3
	RETRY_ATTEMPT_DELAY_IN_SECONDS int = 1

	MINIMUM_SUPPORTED_VCENTER_MAJOR int = 6
	MINIMUM_SUPPORTED_VCENTER_MINOR int = 5
)

func checkAPI(version string) error {
	items := strings.Split(version, ".")
	if len(items) <= 1 {
		return fmt.Errorf("Invalid API Version format")
	}

	major, err := strconv.Atoi(items[0])
	if err != nil {
		return fmt.Errorf("Invalid Major Version value invalid")
	}
	minor, err := strconv.Atoi(items[1])
	if err != nil {
		return fmt.Errorf("Invalid Minor Version value invalid")
	}

	if major < MINIMUM_SUPPORTED_VCENTER_MAJOR {
		return fmt.Errorf("The minimum supported vCenter is 6.5")
	}
	if major == MINIMUM_SUPPORTED_VCENTER_MAJOR && minor < MINIMUM_SUPPORTED_VCENTER_MINOR {
		return fmt.Errorf("The minimum supported vCenter is 6.5")
	}
	return nil
}

func removePortFromHost(host string) string {
	result := host
	index := strings.IndexAny(host, ":")
	if index != -1 {
		result = host[:index]
	}
	return result
}

// getAllFCDs returns all FCDs in all VC/DC sorted by UUID
func getAllFCDs(ctx context.Context, cm *cm.ConnectionManager) []*vclib.FirstClassDiskInfo {

	firstClassDisks := make([]*vclib.FirstClassDiskInfo, 0)

	for vc, vsi := range cm.VsphereInstanceMap {

		var err error
		for i := 0; i < NUM_OF_CONNECTION_ATTEMPTS; i++ {
			err = cm.ConnectByInstance(ctx, vsi)
			if err == nil {
				break
			}
			time.Sleep(time.Duration(RETRY_ATTEMPT_DELAY_IN_SECONDS) * time.Second)
		}
		if err != nil {
			log.Errorf("Failed to connection to vCenter: %s with err: %v", vc, err)
			continue
		}

		datacenters, err := vclib.GetAllDatacenter(ctx, vsi.Conn)
		if err != nil {
			log.Errorf("GetAllDatacenter failed vc=%s err=%v", vc, err)
			continue
		}

		for _, datacenter := range datacenters {
			firstClassDisksSubset, err := datacenter.GetAllFirstClassDisks(ctx)
			if err != nil {
				log.Errorf("GetAllFirstClassDisks failed vc=%s err=%v", vc, err)
				continue
			}

			firstClassDisks = append(firstClassDisks, firstClassDisksSubset...)
		}
	}

	sort.Slice(firstClassDisks, func(i, j int) bool {
		return firstClassDisks[i].Config.Id.Id > firstClassDisks[j].Config.Id.Id
	})

	return firstClassDisks
}
