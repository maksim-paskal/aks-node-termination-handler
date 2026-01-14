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
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vince-riv/aks-node-termination-handler/pkg/config"
	"github.com/vince-riv/aks-node-termination-handler/pkg/types"
	"github.com/vince-riv/aks-node-termination-handler/pkg/utils"
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
			endpoint := "http://127.0.0.1/test"
			requestTimeout := 1 * time.Second
			period := 1 * time.Second
			config.Set(config.Type{ //nolint:exhaustruct
				Endpoint:           &endpoint,
				RequestTimeout:     &requestTimeout,
				Period:             &period,
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

//nolint:funlen,paralleltest // paralleltest disabled due to global config state
func TestPing(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/ok", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler.HandleFunc("/error", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	handler.HandleFunc("/timeout", func(w http.ResponseWriter, r *http.Request) {
		utils.SleepWithContext(r.Context(), 5*time.Second)
		w.WriteHeader(http.StatusOK)
	})

	testServer := httptest.NewServer(handler)
	defer testServer.Close()

	t.Run("success - 200 response", func(t *testing.T) {
		endpoint := testServer.URL + "/ok"
		requestTimeout := 1 * time.Second
		period := 1 * time.Second
		notBeforeThreshold := time.Duration(0)
		config.Set(config.Type{ //nolint:exhaustruct
			Endpoint:           &endpoint,
			RequestTimeout:     &requestTimeout,
			Period:             &period,
			NotBeforeThreshold: &notBeforeThreshold,
		})

		err := Ping(context.Background())
		if err != nil {
			t.Errorf("Ping() unexpected error: %v", err)
		}
	})

	t.Run("failure - non-200 response", func(t *testing.T) {
		endpoint := testServer.URL + "/error"
		requestTimeout := 1 * time.Second
		period := 1 * time.Second
		notBeforeThreshold := time.Duration(0)
		config.Set(config.Type{ //nolint:exhaustruct
			Endpoint:           &endpoint,
			RequestTimeout:     &requestTimeout,
			Period:             &period,
			NotBeforeThreshold: &notBeforeThreshold,
		})

		err := Ping(context.Background())
		if err == nil {
			t.Error("Ping() expected error for non-200 response")
		}
	})

	t.Run("failure - request timeout", func(t *testing.T) {
		endpoint := testServer.URL + "/timeout"
		requestTimeout := 100 * time.Millisecond
		period := 1 * time.Second
		notBeforeThreshold := time.Duration(0)
		config.Set(config.Type{ //nolint:exhaustruct
			Endpoint:           &endpoint,
			RequestTimeout:     &requestTimeout,
			Period:             &period,
			NotBeforeThreshold: &notBeforeThreshold,
		})

		err := Ping(context.Background())
		if err == nil {
			t.Error("Ping() expected error for timeout")
		}
	})
}
