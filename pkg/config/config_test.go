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
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/types"
	"github.com/stretchr/testify/assert"
)

//nolint:paralleltest
func TestConfigDefaults(t *testing.T) {
	if err := config.Load(); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "http://169.254.169.254/metadata/scheduledevents?api-version=2020-07-01", *config.Get().Endpoint)
}

//nolint:paralleltest
func TestValidConfigFile(t *testing.T) {
	configFile := "./testdata/config_test.yaml"
	newConfig := config.Type{ConfigFile: &configFile}
	config.Set(newConfig)

	err := config.Load()
	assert.NoError(t, err)

	assert.Equal(t, "/some/test/path", *config.Get().KubeConfigFile)
}

//nolint:paralleltest
func TestInvalidConfigFile(t *testing.T) {
	configFile := "testdata/config_yaml_fake.yaml"
	newConfig := config.Type{ConfigFile: &configFile}
	config.Set(newConfig)

	err := config.Load()
	assert.Error(t, err)
}

//nolint:paralleltest
func TestVersion(t *testing.T) {
	if config.GetVersion() != "dev" {
		t.Fatal("version is not dev")
	}
}

//nolint:paralleltest,funlen
func TestConfig(t *testing.T) {
	testCases := []struct {
		taintEffect string
		nodeName    string
		telegramID  string
		err         bool
		testName    string
	}{
		{
			testName:    "noSchedule",
			taintEffect: "NoSchedule",
			telegramID:  "1",
			nodeName:    "validNode",
			err:         false,
		},
		{
			testName:    "noExecute",
			taintEffect: "NoExecute",
			nodeName:    "validNode",
			telegramID:  "1",
			err:         false,
		},
		{
			testName:    "preferNoSchedule",
			taintEffect: "PreferNoSchedule",
			nodeName:    "validNode",
			telegramID:  "1",
			err:         false,
		},
		{
			testName:    "invalidNodeName",
			taintEffect: "NoSchedule",
			nodeName:    "",
			telegramID:  "1",
			err:         true,
		},
		{
			testName:    "InvalidTelegramId",
			taintEffect: "NoSchedule",
			nodeName:    "validNode",
			telegramID:  "invalidTelegramId",
			err:         true,
		},
		{
			testName:    "InvalidNodeName",
			taintEffect: "NoSchedule",
			nodeName:    "",
			telegramID:  "1",
			err:         true,
		},
	}
	//nolint:scopelint
	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			newConfig := config.Type{
				TaintEffect:    &tc.taintEffect,
				NodeName:       &tc.nodeName,
				TelegramChatID: &tc.telegramID,
			}
			config.Set(newConfig)
			err := config.Check()
			if tc.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsExcludedEvent(t *testing.T) {
	t.Parallel()

	trueValue := true
	falseValue := false

	testConfigValid := config.Type{
		DrainOnFreezeEvent: &falseValue,
	}

	// test DrainOnFreezeEvent logic
	testConfigValid.DrainOnFreezeEvent = &falseValue
	if b := testConfigValid.IsExcludedEvent(types.EventTypeFreeze); b != true {
		t.Fatal("when DrainOnFreezeEvent is false, IsExcludedEvent must be true")
	}

	testConfigValid.DrainOnFreezeEvent = &trueValue
	if b := testConfigValid.IsExcludedEvent(types.EventTypeFreeze); b == true {
		t.Fatal("when DrainOnFreezeEvent is true, IsExcludedEvent must be false")
	}
}
