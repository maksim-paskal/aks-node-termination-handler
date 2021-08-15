/*
Copyright paskal.maksim@gmail.com
Licensed under the Apache License, Version 2.0 (the "License")
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package config_test

import (
	"testing"

	"github.com/maksim-paskal/aks-node-termination-handler/pkg/config"
)

func TestConfig(t *testing.T) {
	t.Parallel()

	if err := config.Load(); err != nil {
		t.Fatal(err)
	}

	if want := "/some/test/path"; *config.Get().KubeConfigFile != want {
		t.Fatalf("KubeConfigFile != %s", want)
	}
}
