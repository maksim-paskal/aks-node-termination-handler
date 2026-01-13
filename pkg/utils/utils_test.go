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
package utils_test

import (
	"context"
	"testing"
	"time"

	"github.com/vince-riv/aks-node-termination-handler/pkg/utils"
)

func TestSleepWithContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	startTime := time.Now()

	utils.SleepWithContext(ctx, 1*time.Second)

	if time.Since(startTime) < 1*time.Second || time.Since(startTime) > 2*time.Second {
		t.Error("SleepWithContext() not working as expected")
	}

	cancel()

	startTime = time.Now()

	utils.SleepWithContext(ctx, 1*time.Second)

	if time.Since(startTime) >= 1*time.Second {
		t.Error("SleepWithContext() not working as expected")
	}
}

func TestCalculatePodGracePeriod(t *testing.T) { //nolint:funlen
	t.Parallel()

	tests := []struct {
		name                     string
		dynamicGracePeriod       bool
		staticGracePeriodSeconds int
		notBeforeTime            time.Time
		buffer                   time.Duration
		want                     int
	}{
		{
			name:                     "dynamic disabled - returns static value",
			dynamicGracePeriod:       false,
			staticGracePeriodSeconds: 30,
			notBeforeTime:            time.Now().Add(1 * time.Hour),
			buffer:                   15 * time.Second,
			want:                     30,
		},
		{
			name:                     "dynamic enabled - zero NotBefore returns minimum",
			dynamicGracePeriod:       true,
			staticGracePeriodSeconds: 30,
			notBeforeTime:            time.Time{},
			buffer:                   15 * time.Second,
			want:                     1,
		},
		{
			name:                     "dynamic enabled - NotBefore in past returns minimum",
			dynamicGracePeriod:       true,
			staticGracePeriodSeconds: 30,
			notBeforeTime:            time.Now().Add(-1 * time.Minute),
			buffer:                   15 * time.Second,
			want:                     1,
		},
		{
			name:                     "dynamic enabled - calculated value below minimum returns minimum",
			dynamicGracePeriod:       true,
			staticGracePeriodSeconds: 30,
			notBeforeTime:            time.Now().Add(10 * time.Second),
			buffer:                   15 * time.Second,
			want:                     1,
		},
		{
			name:                     "dynamic enabled - calculates correctly",
			dynamicGracePeriod:       true,
			staticGracePeriodSeconds: 30,
			notBeforeTime:            time.Now().Add(45 * time.Second),
			buffer:                   15 * time.Second,
			want:                     30,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := utils.CalculatePodGracePeriod(
				testCase.dynamicGracePeriod,
				testCase.staticGracePeriodSeconds,
				testCase.notBeforeTime,
				testCase.buffer,
			)

			// Allow 1 second tolerance for time-based calculations
			if testCase.dynamicGracePeriod && !testCase.notBeforeTime.IsZero() && testCase.want > 1 {
				if got < testCase.want-1 || got > testCase.want+1 {
					t.Errorf("CalculatePodGracePeriod() = %d, want %d (±1)", got, testCase.want)
				}
			} else if got != testCase.want {
				t.Errorf("CalculatePodGracePeriod() = %d, want %d", got, testCase.want)
			}
		})
	}
}
