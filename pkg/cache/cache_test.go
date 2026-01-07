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
package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/vince-riv/aks-node-termination-handler/pkg/cache"
)

func TestCache(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	go cache.SheduleCleaning(ctx)

	const (
		test1sec = "test1sec"
		test3sec = "test3sec"
	)

	cache.Add(test1sec, time.Second)
	cache.Add(test3sec, 0)
	cache.Add(test3sec, 3*time.Second)

	time.Sleep(2 * time.Second)

	if cache.HasKey(test1sec) {
		t.Fatalf("%s not expired", test1sec)
	}

	if !cache.HasKey(test3sec) {
		t.Fatalf("%s expired", test3sec)
	}

	time.Sleep(2 * time.Second)

	if cache.HasKey(test3sec) {
		t.Fatalf("%s expired", test3sec)
	}
}
