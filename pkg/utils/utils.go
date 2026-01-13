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
package utils

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
)

func SleepWithContext(ctx context.Context, d time.Duration) {
	log.Debugf("Sleep %s", d)

	select {
	case <-ctx.Done():
		return
	case <-time.After(d):
		return
	}
}

const minGracePeriodSeconds = 1

// CalculatePodGracePeriod returns the pod grace period in seconds.
// If dynamicGracePeriod is false, returns staticGracePeriodSeconds.
// If dynamicGracePeriod is true, calculates based on notBeforeTime minus buffer.
// Returns minGracePeriodSeconds (1) if notBeforeTime is zero or calculated value is <= 0.
func CalculatePodGracePeriod(dynamicGracePeriod bool, staticGracePeriodSeconds int, notBeforeTime time.Time, buffer time.Duration) int {
	if !dynamicGracePeriod {
		return staticGracePeriodSeconds
	}

	if notBeforeTime.IsZero() {
		return minGracePeriodSeconds
	}

	gracePeriod := time.Until(notBeforeTime) - buffer
	gracePeriodSeconds := int(gracePeriod.Seconds())

	if gracePeriodSeconds < minGracePeriodSeconds {
		return minGracePeriodSeconds
	}

	return gracePeriodSeconds
}
