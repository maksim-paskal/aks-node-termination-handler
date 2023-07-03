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
package cache

import (
	"context"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

var data = sync.Map{}

func Add(key string, ttl time.Duration) {
	data.Store(key, time.Now().Add(ttl))
}

func HasKey(key string) bool {
	_, exists := data.Load(key)

	return exists
}

func SheduleCleaning(ctx context.Context) {
	for ctx.Err() == nil {
		data.Range(func(key, value interface{}) bool {
			expireTime, ok := value.(time.Time)

			if ok && expireTime.Before(time.Now()) {
				log.Infof("delete %s", key)

				data.Delete(key)
			}

			return true
		})

		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
		}
	}
}
