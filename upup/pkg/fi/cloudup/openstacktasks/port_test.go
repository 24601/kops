/*
Copyright 2019 The Kubernetes Authors.

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

package openstacktasks

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	sg "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kops/pkg/apis/kops"
	"k8s.io/kops/upup/pkg/fi"
	"k8s.io/kops/upup/pkg/fi/cloudup/openstack"
)

func Test_Port_GetDependencies(t *testing.T) {
	tasks := map[string]fi.Task{
		"foo": &SecurityGroup{Name: fi.PtrTo("security-group")},
		"bar": &Subnet{Name: fi.PtrTo("subnet")},
		"baz": &Instance{Name: fi.PtrTo("instance")},
		"qux": &FloatingIP{Name: fi.PtrTo("fip")},
		"xxx": &Network{Name: fi.PtrTo("network")},
	}

	port := &Port{}

	actual := port.GetDependencies(tasks)

	expected := []fi.Task{
		&Subnet{Name: fi.PtrTo("subnet")},
		&Network{Name: fi.PtrTo("network")},
		&SecurityGroup{Name: fi.PtrTo("security-group")},
	}

	actualSorted := sortedTasks(actual)
	expectedSorted := sortedTasks(expected)
	sort.Sort(actualSorted)
	sort.Sort(expectedSorted)

	if !reflect.DeepEqual(expectedSorted, actualSorted) {
		t.Errorf("Dependencies differ:\n%v\n\tinstead of\n%v", actualSorted, expectedSorted)
	}
}

func Test_NewPortTaskFromCloud(t *testing.T) {
	tests := []struct {
		desc              string
		lifecycle         fi.Lifecycle
		cloud             openstack.OpenstackCloud
		cloudPort         *ports.Port
		foundPort         *Port
		modifiedFoundPort *Port
		expectedPortTask  *Port
		expectedError     error
	}{
		{
			desc:              "empty cloud port found port nil",
			lifecycle:         fi.LifecycleSync,
			cloud:             &portCloud{},
			cloudPort:         &ports.Port{},
			foundPort:         nil,
			modifiedFoundPort: nil,
			expectedPortTask: &Port{
				ID:             fi.PtrTo(""),
				Name:           fi.PtrTo(""),
				Network:        &Network{ID: fi.PtrTo("")},
				SecurityGroups: []*SecurityGroup{},
				Subnets:        []*Subnet{},
				Lifecycle:      fi.LifecycleSync,
			},
			expectedError: nil,
		},
		{
			desc:              "empty cloud port found port not nil",
			lifecycle:         fi.LifecycleSync,
			cloud:             &portCloud{},
			cloudPort:         &ports.Port{},
			foundPort:         &Port{},
			modifiedFoundPort: &Port{ID: fi.PtrTo("")},
			expectedPortTask: &Port{
				ID:             fi.PtrTo(""),
				Name:           fi.PtrTo(""),
				Network:        &Network{ID: fi.PtrTo("")},
				SecurityGroups: []*SecurityGroup{},
				Subnets:        []*Subnet{},
				Lifecycle:      fi.LifecycleSync,
			},
			expectedError: nil,
		},
		{
			desc:      "fully populated cloud port found port not nil",
			lifecycle: fi.LifecycleSync,
			cloud:     &portCloud{},
			cloudPort: &ports.Port{
				ID:        "id",
				Name:      "name",
				NetworkID: "networkID",
				FixedIPs: []ports.IP{
					{SubnetID: "subnet-a"},
					{SubnetID: "subnet-b"},
				},
				SecurityGroups: []string{
					"sg-1",
					"sg-2",
				},
			},
			foundPort:         &Port{},
			modifiedFoundPort: &Port{ID: fi.PtrTo("id")},
			expectedPortTask: &Port{
				ID:      fi.PtrTo("id"),
				Name:    fi.PtrTo("name"),
				Network: &Network{ID: fi.PtrTo("networkID")},
				SecurityGroups: []*SecurityGroup{
					{ID: fi.PtrTo("sg-1"), Lifecycle: fi.LifecycleSync},
					{ID: fi.PtrTo("sg-2"), Lifecycle: fi.LifecycleSync},
				},
				Subnets: []*Subnet{
					{ID: fi.PtrTo("subnet-a"), Lifecycle: fi.LifecycleSync},
					{ID: fi.PtrTo("subnet-b"), Lifecycle: fi.LifecycleSync},
				},
				Lifecycle: fi.LifecycleSync,
			},
			expectedError: nil,
		},
		{
			desc:      "fully populated cloud port found port nil",
			lifecycle: fi.LifecycleSync,
			cloud:     &portCloud{},
			cloudPort: &ports.Port{
				ID:        "id",
				Name:      "name",
				NetworkID: "networkID",
				FixedIPs: []ports.IP{
					{SubnetID: "subnet-a"},
					{SubnetID: "subnet-b"},
				},
				SecurityGroups: []string{
					"sg-1",
					"sg-2",
				},
				Tags: []string{
					"cluster",
				},
			},
			foundPort:         nil,
			modifiedFoundPort: nil,
			expectedPortTask: &Port{
				ID:        fi.PtrTo("id"),
				Lifecycle: fi.LifecycleSync,
				Name:      fi.PtrTo("name"),
				Network:   &Network{ID: fi.PtrTo("networkID")},
				SecurityGroups: []*SecurityGroup{
					{ID: fi.PtrTo("sg-1"), Lifecycle: fi.LifecycleSync},
					{ID: fi.PtrTo("sg-2"), Lifecycle: fi.LifecycleSync},
				},
				Subnets: []*Subnet{
					{ID: fi.PtrTo("subnet-a"), Lifecycle: fi.LifecycleSync},
					{ID: fi.PtrTo("subnet-b"), Lifecycle: fi.LifecycleSync},
				},
				Tags: []string{
					"cluster",
				},
			},
			expectedError: nil,
		},
		{
			desc:      "fully populated cloud port found port not nil populates the InstanceGroupName",
			lifecycle: fi.LifecycleSync,
			cloud:     &portCloud{},
			cloudPort: &ports.Port{
				ID:        "id",
				Name:      "name",
				NetworkID: "networkID",
				FixedIPs: []ports.IP{
					{SubnetID: "subnet-a"},
					{SubnetID: "subnet-b"},
				},
				SecurityGroups: []string{
					"sg-1",
					"sg-2",
				},
				Tags: []string{
					"KopsInstanceGroup=node-ig",
				},
			},
			foundPort: &Port{
				InstanceGroupName: fi.PtrTo("node-ig"),
				Tags: []string{
					"KopsInstanceGroup=node-ig",
				},
			},
			modifiedFoundPort: &Port{
				ID:                fi.PtrTo("id"),
				InstanceGroupName: fi.PtrTo("node-ig"),
				Tags: []string{
					"KopsInstanceGroup=node-ig",
				},
			},
			expectedPortTask: &Port{
				ID:                fi.PtrTo("id"),
				InstanceGroupName: fi.PtrTo("node-ig"),
				Lifecycle:         fi.LifecycleSync,
				Name:              fi.PtrTo("name"),
				Network:           &Network{ID: fi.PtrTo("networkID")},
				SecurityGroups: []*SecurityGroup{
					{ID: fi.PtrTo("sg-1"), Lifecycle: fi.LifecycleSync},
					{ID: fi.PtrTo("sg-2"), Lifecycle: fi.LifecycleSync},
				},
				Subnets: []*Subnet{
					{ID: fi.PtrTo("subnet-a"), Lifecycle: fi.LifecycleSync},
					{ID: fi.PtrTo("subnet-b"), Lifecycle: fi.LifecycleSync},
				},
				Tags: []string{
					"KopsInstanceGroup=node-ig",
				},
			},
			expectedError: nil,
		},
		{
			desc:      "fully populated cloud port found port nil populates the InstanceGroupName if found",
			lifecycle: fi.LifecycleSync,
			cloud:     &portCloud{},
			cloudPort: &ports.Port{
				ID:        "id",
				Name:      "name",
				NetworkID: "networkID",
				FixedIPs: []ports.IP{
					{SubnetID: "subnet-a"},
					{SubnetID: "subnet-b"},
				},
				SecurityGroups: []string{
					"sg-1",
					"sg-2",
				},
				Tags: []string{
					"cluster",
					"KopsInstanceGroup=node-ig",
				},
			},
			foundPort:         nil,
			modifiedFoundPort: nil,
			expectedPortTask: &Port{
				ID:                fi.PtrTo("id"),
				InstanceGroupName: fi.PtrTo("node-ig"),
				Lifecycle:         fi.LifecycleSync,
				Name:              fi.PtrTo("name"),
				Network:           &Network{ID: fi.PtrTo("networkID")},
				SecurityGroups: []*SecurityGroup{
					{ID: fi.PtrTo("sg-1"), Lifecycle: fi.LifecycleSync},
					{ID: fi.PtrTo("sg-2"), Lifecycle: fi.LifecycleSync},
				},
				Subnets: []*Subnet{
					{ID: fi.PtrTo("subnet-a"), Lifecycle: fi.LifecycleSync},
					{ID: fi.PtrTo("subnet-b"), Lifecycle: fi.LifecycleSync},
				},
				Tags: []string{
					"cluster",
					"KopsInstanceGroup=node-ig",
				},
			},
			expectedError: nil,
		},
		{
			desc:      "cloud port found port not nil honors additional security groups",
			lifecycle: fi.LifecycleSync,
			cloud: &portCloud{
				listSecurityGroups: map[string][]sg.SecGroup{
					"add-1": {
						{ID: "add-1", Name: "add-1"},
					},
					"add-2": {
						{ID: "add-2", Name: "add-2"},
					},
				},
			},
			cloudPort: &ports.Port{
				ID:        "id",
				Name:      "name",
				NetworkID: "networkID",
				FixedIPs: []ports.IP{
					{SubnetID: "subnet-a"},
					{SubnetID: "subnet-b"},
				},
				SecurityGroups: []string{
					"sg-1",
					"sg-2",
					"add-1",
					"add-2",
				},
			},
			foundPort: &Port{
				AdditionalSecurityGroups: []string{
					"add-1",
					"add-2",
				},
			},
			modifiedFoundPort: &Port{
				ID: fi.PtrTo("id"),
				AdditionalSecurityGroups: []string{
					"add-1",
					"add-2",
				},
			},
			expectedPortTask: &Port{
				ID:      fi.PtrTo("id"),
				Name:    fi.PtrTo("name"),
				Network: &Network{ID: fi.PtrTo("networkID")},
				SecurityGroups: []*SecurityGroup{
					{ID: fi.PtrTo("sg-1"), Lifecycle: fi.LifecycleSync},
					{ID: fi.PtrTo("sg-2"), Lifecycle: fi.LifecycleSync},
				},
				AdditionalSecurityGroups: []string{
					"add-1",
					"add-2",
				},
				Subnets: []*Subnet{
					{ID: fi.PtrTo("subnet-a"), Lifecycle: fi.LifecycleSync},
					{ID: fi.PtrTo("subnet-b"), Lifecycle: fi.LifecycleSync},
				},
				Lifecycle: fi.LifecycleSync,
			},
			expectedError: nil,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.desc, func(t *testing.T) {
			actual, err := newPortTaskFromCloud(testCase.cloud, testCase.lifecycle, testCase.cloudPort, testCase.foundPort)

			compareErrors(t, err, testCase.expectedError)

			if !reflect.DeepEqual(actual, testCase.expectedPortTask) {
				t.Errorf("Port task differs:\n%v\n\tinstead of\n%v", actual, testCase.expectedPortTask)
			}

			if !reflect.DeepEqual(testCase.foundPort, testCase.modifiedFoundPort) {
				t.Errorf("Found Port task differs:\n%v\n\tinstead of\n%v", testCase.foundPort, testCase.modifiedFoundPort)
			}
		})
	}
}

func Test_Port_Find(t *testing.T) {
	tests := []struct {
		desc             string
		context          *fi.Context
		port             *Port
		expectedPortTask *Port
		expectedError    error
	}{
		{
			desc: "nothing found",
			context: &fi.Context{
				Cluster: &kops.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "clusterName",
					},
				},
				Cloud: &portCloud{},
			},
			port: &Port{
				Name:      fi.PtrTo("name"),
				Lifecycle: fi.LifecycleSync,
			},
			expectedPortTask: nil,
			expectedError:    nil,
		},
		{
			desc: "port found no tags",
			context: &fi.Context{
				Cluster: &kops.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "clusterName",
					},
				},
				Cloud: &portCloud{
					listPorts: []ports.Port{
						{
							ID:        "id",
							Name:      "name",
							NetworkID: "networkID",
							FixedIPs: []ports.IP{
								{SubnetID: "subnet-a"},
								{SubnetID: "subnet-b"},
							},
							SecurityGroups: []string{
								"sg-1",
								"sg-2",
							},
							Tags: []string{"clusterName"},
						},
					},
				},
			},
			port: &Port{
				Name:      fi.PtrTo("name"),
				Lifecycle: fi.LifecycleSync,
			},
			expectedPortTask: &Port{
				ID:      fi.PtrTo("id"),
				Name:    fi.PtrTo("name"),
				Network: &Network{ID: fi.PtrTo("networkID")},
				SecurityGroups: []*SecurityGroup{
					{ID: fi.PtrTo("sg-1"), Lifecycle: fi.LifecycleSync},
					{ID: fi.PtrTo("sg-2"), Lifecycle: fi.LifecycleSync},
				},
				Subnets: []*Subnet{
					{ID: fi.PtrTo("subnet-a"), Lifecycle: fi.LifecycleSync},
					{ID: fi.PtrTo("subnet-b"), Lifecycle: fi.LifecycleSync},
				},
				Lifecycle: fi.LifecycleSync,
			},
			expectedError: nil,
		},
		{
			desc: "port found with tags",
			context: &fi.Context{
				Cluster: &kops.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "clusterName",
					},
				},
				Cloud: &portCloud{
					listPorts: []ports.Port{
						{
							ID:        "id",
							Name:      "name",
							NetworkID: "networkID",
							FixedIPs: []ports.IP{
								{SubnetID: "subnet-a"},
								{SubnetID: "subnet-b"},
							},
							SecurityGroups: []string{
								"sg-1",
								"sg-2",
							},
							Tags: []string{"clusterName"},
						},
					},
				},
			},
			port: &Port{
				Name:      fi.PtrTo("name"),
				Lifecycle: fi.LifecycleSync,
				Tags:      []string{"clusterName"},
			},
			expectedPortTask: &Port{
				ID:      fi.PtrTo("id"),
				Name:    fi.PtrTo("name"),
				Network: &Network{ID: fi.PtrTo("networkID")},
				SecurityGroups: []*SecurityGroup{
					{ID: fi.PtrTo("sg-1"), Lifecycle: fi.LifecycleSync},
					{ID: fi.PtrTo("sg-2"), Lifecycle: fi.LifecycleSync},
				},
				Subnets: []*Subnet{
					{ID: fi.PtrTo("subnet-a"), Lifecycle: fi.LifecycleSync},
					{ID: fi.PtrTo("subnet-b"), Lifecycle: fi.LifecycleSync},
				},
				Lifecycle: fi.LifecycleSync,
				Tags:      []string{"clusterName"},
			},
			expectedError: nil,
		},
		{
			desc: "multiple ports found",
			context: &fi.Context{
				Cluster: &kops.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "clusterName",
					},
				},
				Cloud: &portCloud{
					listPorts: []ports.Port{
						{
							ID:   "id-1",
							Name: "name",
							Tags: []string{"clusterName"},
						},
						{
							ID:   "id-2",
							Name: "name",
							Tags: []string{"clusterName"},
						},
					},
				},
			},
			port: &Port{
				Name:      fi.PtrTo("name"),
				Lifecycle: fi.LifecycleSync,
			},
			expectedPortTask: nil,
			expectedError:    fmt.Errorf("found multiple ports with name: name"),
		},
		{
			desc: "error listing ports",
			context: &fi.Context{
				Cluster: &kops.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "clusterName",
					},
				},
				Cloud: &portCloud{
					listPorts: []ports.Port{
						{
							ID:   "id-1",
							Name: "name",
						},
					},
					listPortsError: fmt.Errorf("list error"),
				},
			},
			port: &Port{
				Name:      fi.PtrTo("name"),
				Lifecycle: fi.LifecycleSync,
			},
			expectedPortTask: nil,
			expectedError:    fmt.Errorf("list error"),
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.desc, func(t *testing.T) {
			actual, err := testCase.port.Find(testCase.context)

			compareErrors(t, err, testCase.expectedError)

			if !reflect.DeepEqual(actual, testCase.expectedPortTask) {
				t.Errorf("Port task differs:\n%v\n\tinstead of\n%v", actual, testCase.expectedPortTask)
			}
		})
	}
}

func Test_Port_CheckChanges(t *testing.T) {
	tests := []struct {
		desc          string
		actual        *Port
		expected      *Port
		changes       *Port
		expectedError error
	}{
		{
			desc:   "actual nil all required fields set",
			actual: nil,
			expected: &Port{
				Name:    fi.PtrTo("name"),
				Network: &Network{ID: fi.PtrTo("networkID")},
			},
			expectedError: nil,
		},
		{
			desc:   "actual nil required field Name nil",
			actual: nil,
			expected: &Port{
				Name:    nil,
				Network: &Network{ID: fi.PtrTo("networkID")},
			},
			expectedError: fi.RequiredField("Name"),
		},
		{
			desc:   "actual nil required field Network nil",
			actual: nil,
			expected: &Port{
				Name:    fi.PtrTo("name"),
				Network: nil,
			},
			expectedError: fi.RequiredField("Network"),
		},
		{
			desc: "actual not nil all changeable fields set",
			actual: &Port{
				Name:    fi.PtrTo("name"),
				Network: nil,
			},
			expected: &Port{
				Name:    fi.PtrTo("name"),
				Network: nil,
			},
			changes: &Port{
				Name:    nil,
				Network: nil,
			},
			expectedError: nil,
		},
		{
			desc: "actual not nil all changeable fields set",
			actual: &Port{
				Name:    fi.PtrTo("name"),
				Network: nil,
			},
			expected: &Port{
				Name:    fi.PtrTo("name"),
				Network: nil,
			},
			changes: &Port{
				Name:    nil,
				Network: &Network{ID: fi.PtrTo("networkID")},
			},
			expectedError: fi.CannotChangeField("Network"),
		},
		{
			desc: "actual not nil unchangeable field Name set",
			actual: &Port{
				Name:    fi.PtrTo("name"),
				Network: nil,
			},
			expected: &Port{
				Name:    fi.PtrTo("name"),
				Network: &Network{ID: fi.PtrTo("networkID")},
			},
			changes: &Port{
				Name:    fi.PtrTo("name"),
				Network: nil,
			},
			expectedError: fi.CannotChangeField("Name"),
		},
		{
			desc: "actual not nil unchangeable field Network set",
			actual: &Port{
				Name:    fi.PtrTo("name"),
				Network: nil,
			},
			expected: &Port{
				Name:    nil,
				Network: &Network{ID: fi.PtrTo("networkID")},
			},
			changes: &Port{
				Name:    nil,
				Network: &Network{ID: fi.PtrTo("networkID")},
			},
			expectedError: fi.CannotChangeField("Network"),
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.desc, func(t *testing.T) {
			var port Port
			err := (&port).CheckChanges(testCase.actual, testCase.expected, testCase.changes)

			compareErrors(t, err, testCase.expectedError)
		})
	}
}

func Test_Port_RenderOpenstack(t *testing.T) {
	tests := []struct {
		desc              string
		target            *openstack.OpenstackAPITarget
		actual            *Port
		expected          *Port
		changes           *Port
		expectedCloudPort *ports.Port
		expectedAfter     *Port
		expectedError     error
	}{
		{
			desc: "actual not nil",
			actual: &Port{
				ID:      fi.PtrTo("actual-id"),
				Name:    fi.PtrTo("name"),
				Network: &Network{ID: fi.PtrTo("networkID")},
			},
			expected: &Port{
				ID:      fi.PtrTo("expected-id"),
				Name:    fi.PtrTo("name"),
				Network: &Network{ID: fi.PtrTo("networkID")},
			},
			expectedAfter: &Port{
				ID:      fi.PtrTo("actual-id"),
				Name:    fi.PtrTo("name"),
				Network: &Network{ID: fi.PtrTo("networkID")},
			},
			expectedCloudPort: nil,
			expectedError:     nil,
		},
		{
			desc: "actual nil success",
			target: &openstack.OpenstackAPITarget{
				Cloud: &portCloud{
					createPort: &ports.Port{
						ID:        "cloud-id",
						Name:      "name",
						NetworkID: "networkID",
						FixedIPs: []ports.IP{
							{SubnetID: "subnet-a"},
							{SubnetID: "subnet-b"},
						},
						SecurityGroups: []string{
							"sg-1",
							"sg-2",
						},
					},
				},
			},
			actual: nil,
			expected: &Port{
				ID:      fi.PtrTo("expected-id"),
				Name:    fi.PtrTo("name"),
				Network: &Network{ID: fi.PtrTo("networkID")},
				SecurityGroups: []*SecurityGroup{
					{ID: fi.PtrTo("sg-1")},
					{ID: fi.PtrTo("sg-2")},
				},
				Subnets: []*Subnet{
					{ID: fi.PtrTo("subnet-a")},
					{ID: fi.PtrTo("subnet-b")},
				},
			},
			expectedAfter: &Port{
				ID:      fi.PtrTo("cloud-id"),
				Name:    fi.PtrTo("name"),
				Network: &Network{ID: fi.PtrTo("networkID")},
				SecurityGroups: []*SecurityGroup{
					{ID: fi.PtrTo("sg-1")},
					{ID: fi.PtrTo("sg-2")},
				},
				Subnets: []*Subnet{
					{ID: fi.PtrTo("subnet-a")},
					{ID: fi.PtrTo("subnet-b")},
				},
			},
			expectedCloudPort: &ports.Port{
				ID:        "id",
				Name:      "name",
				NetworkID: "networkID",
				FixedIPs: []ports.IP{
					{SubnetID: "subnet-a"},
					{SubnetID: "subnet-b"},
				},
				SecurityGroups: []string{
					"sg-1",
					"sg-2",
				},
			},
			expectedError: nil,
		},
		{
			desc: "actual nil cloud error",
			target: &openstack.OpenstackAPITarget{
				Cloud: &portCloud{
					createPortError: fmt.Errorf("port create error"),
				},
			},
			actual: nil,
			expected: &Port{
				ID:      fi.PtrTo("expected-id"),
				Name:    fi.PtrTo("name"),
				Network: &Network{ID: fi.PtrTo("networkID")},
			},
			expectedAfter: &Port{
				ID:      fi.PtrTo("expected-id"),
				Name:    fi.PtrTo("name"),
				Network: &Network{ID: fi.PtrTo("networkID")},
			},
			expectedCloudPort: nil,
			expectedError:     fmt.Errorf("Error creating port: port create error"),
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.desc, func(t *testing.T) {
			var port Port
			err := (&port).RenderOpenstack(testCase.target, testCase.actual, testCase.expected, testCase.changes)

			compareErrors(t, err, testCase.expectedError)

			if !reflect.DeepEqual(testCase.expected, testCase.expectedAfter) {
				t.Errorf("Expected Port task differs:\n%v\n\tinstead of\n%v", testCase.expected, testCase.expectedAfter)
			}
		})
	}
}

func Test_Port_createOptsFromPortTask(t *testing.T) {
	tests := []struct {
		desc               string
		target             *openstack.OpenstackAPITarget
		actual             *Port
		expected           *Port
		changes            *Port
		expectedCreateOpts ports.CreateOptsBuilder
		expectedError      error
	}{
		{
			desc: "all fields set",
			target: &openstack.OpenstackAPITarget{
				Cloud: &portCloud{
					listSecurityGroups: map[string][]sg.SecGroup{
						"add-1": {
							{ID: "add-1-id", Name: "add-1"},
						},
						"add-2": {
							{ID: "add-2-id", Name: "add-2"},
						},
					},
				},
			},
			expected: &Port{
				ID:      fi.PtrTo("expected-id"),
				Name:    fi.PtrTo("name"),
				Network: &Network{ID: fi.PtrTo("networkID")},
				SecurityGroups: []*SecurityGroup{
					{ID: fi.PtrTo("sg-1")},
					{ID: fi.PtrTo("sg-2")},
				},
				AdditionalSecurityGroups: []string{
					"add-1",
					"add-2",
				},
				Subnets: []*Subnet{
					{ID: fi.PtrTo("subnet-a")},
					{ID: fi.PtrTo("subnet-b")},
				},
			},
			expectedCreateOpts: ports.CreateOpts{
				Name:      "name",
				NetworkID: "networkID",
				SecurityGroups: &[]string{
					"sg-1",
					"sg-2",
					"add-1-id",
					"add-2-id",
				},
				FixedIPs: []ports.IP{
					{SubnetID: "subnet-a"},
					{SubnetID: "subnet-b"},
				},
			},
		},
		{
			desc: "nonexisting additional security groups",
			target: &openstack.OpenstackAPITarget{
				Cloud: &portCloud{
					listSecurityGroups: map[string][]sg.SecGroup{
						"add-1": {
							{ID: "add-1-id", Name: "add-1"},
						},
					},
				},
			},
			expected: &Port{
				ID:      fi.PtrTo("expected-id"),
				Name:    fi.PtrTo("name"),
				Network: &Network{ID: fi.PtrTo("networkID")},
				SecurityGroups: []*SecurityGroup{
					{ID: fi.PtrTo("sg-1")},
					{ID: fi.PtrTo("sg-2")},
				},
				AdditionalSecurityGroups: []string{
					"add-2",
				},
				Subnets: []*Subnet{
					{ID: fi.PtrTo("subnet-a")},
					{ID: fi.PtrTo("subnet-b")},
				},
			},
			expectedError: fmt.Errorf("Additional SecurityGroup not found for name add-2"),
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.desc, func(t *testing.T) {
			opts, err := portCreateOptsFromPortTask(testCase.target, testCase.actual, testCase.expected, testCase.changes)

			compareErrors(t, err, testCase.expectedError)

			if !reflect.DeepEqual(testCase.expectedCreateOpts, opts) {
				t.Errorf("Port create opts differs:\n%v\n\tinstead of\n%v", opts, testCase.expectedCreateOpts)
			}
		})
	}
}

type portCloud struct {
	openstack.OpenstackCloud
	listPorts               []ports.Port
	listPortsError          error
	createPort              *ports.Port
	createPortError         error
	listSecurityGroups      map[string][]sg.SecGroup
	listSecurityGroupsError error
}

func (p *portCloud) ListPorts(opt ports.ListOptsBuilder) ([]ports.Port, error) {
	return p.listPorts, p.listPortsError
}

func (p *portCloud) CreatePort(opt ports.CreateOptsBuilder) (*ports.Port, error) {
	return p.createPort, p.createPortError
}

func (p *portCloud) ListSecurityGroups(opt sg.ListOpts) ([]sg.SecGroup, error) {
	return p.listSecurityGroups[opt.Name], p.listSecurityGroupsError
}

type sortedTasks []fi.Task

func (s sortedTasks) Len() int           { return len(s) }
func (s sortedTasks) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s sortedTasks) Less(i, j int) bool { return fmt.Sprintf("%v", s[i]) < fmt.Sprintf("%v", s[j]) }

func compareErrors(t *testing.T, actual, expected error) {
	t.Helper()
	if pointersAreBothNil(t, "errors", actual, expected) {
		return
	}
	a := fmt.Sprintf("%v", actual)
	e := fmt.Sprintf("%v", expected)
	if a != e {
		t.Errorf("error differs: %+v instead of %+v", actual, expected)
	}
}

func pointersAreBothNil(t *testing.T, name string, actual, expected interface{}) bool {
	t.Helper()
	if actual == nil && expected == nil {
		return true
	}
	if !reflect.ValueOf(actual).IsValid() {
		return false
	}
	if reflect.ValueOf(actual).IsNil() && reflect.ValueOf(expected).IsNil() {
		return true
	}
	if actual == nil && expected != nil {
		t.Fatalf("%s differ: actual is nil, expected is not", name)
	}
	if actual != nil && expected == nil {
		t.Fatalf("%s differ: expected is nil, actual is not", name)
	}
	return false
}
