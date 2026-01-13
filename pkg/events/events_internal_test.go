/*
Copyright github@vince-riv.io
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

package events

import (
	"testing"
	"time"

	"github.com/vince-riv/aks-node-termination-handler/pkg/config"
	"github.com/vince-riv/aks-node-termination-handler/pkg/types"
)

//nolint:funlen,paralleltest // paralleltest disabled due to global config state
func TestShouldSkipNotBefore(t *testing.T) {
	tests := []struct {
		name      string
		notBefore string
		threshold time.Duration
		wantSkip  bool
		wantErr   bool
	}{
		{
			name:      "threshold disabled (zero) - should not skip",
			notBefore: time.Now().Add(1 * time.Hour).Format(time.RFC1123),
			threshold: 0,
			wantSkip:  false,
			wantErr:   false,
		},
		{
			name:      "threshold negative - should not skip",
			notBefore: time.Now().Add(1 * time.Hour).Format(time.RFC1123),
			threshold: -1 * time.Second,
			wantSkip:  false,
			wantErr:   false,
		},
		{
			name:      "empty NotBefore - should not skip",
			notBefore: "",
			threshold: 5 * time.Minute,
			wantSkip:  false,
			wantErr:   false,
		},
		{
			name:      "NotBefore in past - should not skip",
			notBefore: time.Now().Add(-1 * time.Hour).Format(time.RFC1123),
			threshold: 5 * time.Minute,
			wantSkip:  false,
			wantErr:   false,
		},
		{
			name:      "NotBefore within threshold - should not skip",
			notBefore: time.Now().Add(2 * time.Minute).Format(time.RFC1123),
			threshold: 5 * time.Minute,
			wantSkip:  false,
			wantErr:   false,
		},
		{
			name:      "NotBefore exceeds threshold - should skip",
			notBefore: time.Now().Add(10 * time.Minute).Format(time.RFC1123),
			threshold: 5 * time.Minute,
			wantSkip:  true,
			wantErr:   false,
		},
		{
			name:      "invalid NotBefore format - should return error",
			notBefore: "invalid-date-format",
			threshold: 5 * time.Minute,
			wantSkip:  false,
			wantErr:   true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			threshold := testCase.threshold
			config.Set(config.Type{ //nolint:exhaustruct
				NotBeforeThreshold: &threshold,
			})

			reader := NewReader()
			event := types.ScheduledEventsEvent{
				EventId:           "test-event",
				EventType:         types.EventTypePreempt,
				ResourceType:      "VirtualMachine",
				Resources:         []string{"test-resource"},
				EventStatus:       "Scheduled",
				NotBefore:         testCase.notBefore,
				Description:       "Test event",
				EventSource:       "Platform",
				DurationInSeconds: -1,
			}

			skip, err := reader.shouldSkipNotBefore(event)

			if (err != nil) != testCase.wantErr {
				t.Errorf("shouldSkipNotBefore() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if err != nil {
				return
			}

			if skip != testCase.wantSkip {
				t.Errorf("shouldSkipNotBefore() skip = %v, wantSkip %v", skip, testCase.wantSkip)
			}
		})
	}
}
