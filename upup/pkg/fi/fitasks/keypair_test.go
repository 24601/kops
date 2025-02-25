/*
Copyright 2017 The Kubernetes Authors.

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

package fitasks

import (
	"strings"
	"testing"

	"k8s.io/kops/upup/pkg/fi"
)

func TestKeypairDeps(t *testing.T) {
	ca := &Keypair{
		Name: fi.PtrTo("ca"),
	}
	cert := &Keypair{
		Name:   fi.PtrTo("cert"),
		Signer: ca,
	}

	tasks := make(map[string]fi.Task)
	tasks["ca"] = ca
	tasks["cert"] = cert

	deps := fi.FindTaskDependencies(tasks)

	if strings.Join(deps["ca"], ",") != "" {
		t.Errorf("unexpected dependencies for ca: %v", deps["ca"])
	}

	if strings.Join(deps["cert"], ",") != "ca" {
		t.Errorf("unexpected dependencies for cert: %v", deps["cert"])
	}
}
