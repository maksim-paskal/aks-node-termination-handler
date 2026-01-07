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
package logger_test

import (
	"testing"

	"github.com/vince-riv/aks-node-termination-handler/pkg/logger"
)

func TestKubectlLogger(t *testing.T) {
	t.Parallel()

	logger := logger.KubectlLogger{}

	logText := ""

	logger.Log = func(message string) {
		logText = message
	}

	i, err := logger.Write([]byte("test"))
	if err != nil {
		t.Fatal(err)
	}

	if i != 0 {
		t.Fatalf("expected: %d, got: %d", 0, i)
	}

	if logText != "test" {
		t.Fatalf("expected: %s, got: %s", "test", logText)
	}
}
