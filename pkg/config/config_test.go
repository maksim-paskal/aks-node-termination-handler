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
	"flag"
	"strings"
	"testing"

	"github.com/maksim-paskal/aks-node-termination-handler/pkg/config"
)

func TestConfig(t *testing.T) { //nolint:cyclop,funlen
	if err := flag.Set("config", "testdata/config_test.yaml"); err != nil {
		t.Fatal(err)
	}

	t.Parallel()

	if err := config.Load(); err != nil {
		t.Fatal(err)
	}

	if want := "/some/test/path"; *config.Get().KubeConfigFile != want {
		t.Fatalf("KubeConfigFile != %s", want)
	}

	if !strings.Contains(config.String(), "endpoint: http://169.254.169.254/metadata/scheduledevents?api-version=2020-07-01") { //nolint:lll
		t.Fatal("config not equal to default config")
	}

	// set to fake config
	if err := flag.Set("config", "/tmp"); err != nil {
		t.Fatal(err)
	}

	if err := config.Load(); err == nil {
		t.Fatal("config must errored")
	}

	// set to fake config
	if err := flag.Set("config", "testdata/config_yaml_fake.yaml"); err != nil {
		t.Fatal(err)
	}

	if err := config.Load(); err == nil {
		t.Fatal("config must errored")
	}

	// set to nil config
	if err := flag.Set("config", ""); err != nil {
		t.Fatal(err)
	}

	if err := config.Load(); err != nil {
		t.Fatal("config must be nil")
	}

	// test check node
	if err := flag.Set("node", ""); err != nil {
		t.Fatal(err)
	}

	if err := config.Load(); err != nil {
		t.Fatal("config must be loaded")
	}

	if err := config.Check(); err == nil {
		t.Fatal("config must be nil")
	}

	// test check chatID
	if err := flag.Set("node", "some node"); err != nil {
		t.Fatal(err)
	}

	if err := flag.Set("telegram.chatID", "qweqweqwe"); err != nil {
		t.Fatal(err)
	}

	if err := config.Load(); err != nil {
		t.Fatal("config must be loaded")
	}

	if err := config.Check(); err == nil {
		t.Fatal("config must be nil")
	}

	// test all ok
	if err := flag.Set("node", "some node"); err != nil {
		t.Fatal(err)
	}

	if err := flag.Set("telegram.chatID", "12345"); err != nil {
		t.Fatal(err)
	}

	if err := config.Load(); err != nil {
		t.Fatal("config must be loaded")
	}

	if err := config.Check(); err != nil {
		t.Fatal("config must be nil")
	}
}

func TestVersion(t *testing.T) {
	t.Parallel()

	if config.GetVersion() != "dev" {
		t.Fatal("version is not dev")
	}
}
