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
package types_test

import (
	"encoding/json"
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/vince-riv/aks-node-termination-handler/pkg/types"
)

func TestScheduledEventsType(t *testing.T) {
	t.Parallel()

	messageBytes, err := os.ReadFile("testdata/ScheduledEventsType.json")
	if err != nil {
		t.Fatal(err)
	}

	message := types.ScheduledEventsType{}

	err = json.Unmarshal(messageBytes, &message)
	if err != nil {
		t.Fatal(err)
	}

	if len(message.Events) == 0 {
		t.Fatal("events is empty")
	}

	if want := "VirtualMachine"; message.Events[0].ResourceType != want {
		t.Fatalf("want=%s, got=%s", want, message.Events[0].ResourceType)
	}
}

func TestAzureResource(t *testing.T) {
	t.Parallel()

	type azureResourceTest struct {
		providerID string
		want       *types.AzureResource
	}

	tests := make([]azureResourceTest, 0)

	tests = append(tests, azureResourceTest{
		providerID: "azure:///subscriptions/12345a05-1234-1234-12345-922b47912341/resourceGroups/mc_prod_prod_eastus/providers/Microsoft.Compute/virtualMachineScaleSets/aks-spotcpu2v2-19654750-vmss/virtualMachines/2768", //nolint:lll
		want: &types.AzureResource{
			EventResourceName: "aks-spotcpu2v2-19654750-vmss_2768",
			SubscriptionID:    "12345a05-1234-1234-12345-922b47912341",
			ResourceGroup:     "mc_prod_prod_eastus",
		},
	})

	tests = append(tests, azureResourceTest{
		providerID: "azure:///subscriptions/12345a05-1234-1234-12345-922b47912342/resourceGroups/aro-infra-lth8qmzr-test-openshift-cluster1/providers/Microsoft.Compute/virtualMachines/test-openshift-cluste-t98dd-master-0", //nolint:lll
		want: &types.AzureResource{
			EventResourceName: "test-openshift-cluste-t98dd-master-0",
			SubscriptionID:    "12345a05-1234-1234-12345-922b47912342",
			ResourceGroup:     "aro-infra-lth8qmzr-test-openshift-cluster1",
		},
	})

	tests = append(tests, azureResourceTest{
		providerID: "azure:///subscriptions/12345a05-1234-1234-12345-922b47912343/resourceGroups/aro-infra-lth8qmzr-test-openshift-cluster2/providers/Microsoft.Compute/virtualMachines/test-openshift-cluste-t98dd-worker-eastus1-rz2t8", //nolint:lll
		want: &types.AzureResource{
			EventResourceName: "test-openshift-cluste-t98dd-worker-eastus1-rz2t8",
			SubscriptionID:    "12345a05-1234-1234-12345-922b47912343",
			ResourceGroup:     "aro-infra-lth8qmzr-test-openshift-cluster2",
		},
	})

	for testID, test := range tests {
		t.Run("Test"+strconv.Itoa(testID), func(t *testing.T) {
			t.Parallel()

			azureResource, err := types.NewAzureResource(test.providerID)
			if err != nil {
				t.Fatal(err)
			}

			// need to set providerID for comparison
			test.want.ProviderID = test.providerID

			if !reflect.DeepEqual(azureResource, test.want) {
				t.Fatalf("want=%+v, got=%+v", test.want, azureResource)
			}
		})
	}

	// test invalid providerID
	if _, err := types.NewAzureResource("azure://fake"); err == nil {
		t.Fatal("error expected")
	}
}

func TestNotBeforeTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		notBefore string
		wantZero  bool
		wantErr   bool
	}{
		{
			name:      "valid RFC1123 time",
			notBefore: "Mon, 19 Sep 2016 18:29:47 GMT",
			wantZero:  false,
			wantErr:   false,
		},
		{
			name:      "empty string returns zero time",
			notBefore: "",
			wantZero:  true,
			wantErr:   false,
		},
		{
			name:      "invalid format returns error",
			notBefore: "2016-09-19T18:29:47Z",
			wantZero:  false,
			wantErr:   true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			event := types.ScheduledEventsEvent{
				EventId:           "",
				EventType:         "",
				ResourceType:      "",
				Resources:         nil,
				EventStatus:       "",
				NotBefore:         testCase.notBefore,
				Description:       "",
				EventSource:       "",
				DurationInSeconds: 0,
			}
			got, err := event.NotBeforeTime()

			if (err != nil) != testCase.wantErr {
				t.Errorf("NotBeforeTime() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if err != nil {
				return
			}

			if got.IsZero() != testCase.wantZero {
				t.Errorf("NotBeforeTime() isZero = %v, wantZero %v", got.IsZero(), testCase.wantZero)
			}
		})
	}
}
