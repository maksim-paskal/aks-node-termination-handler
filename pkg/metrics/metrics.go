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
package metrics

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const namespace = "aks_node_termination_handler"

type Instrumenter struct {
	subsystemIdentifier string
}

// New creates a new Instrumenter. The subsystemIdentifier will be used as part of
// the metric names (e.g. http_<identifier>_requests_total).
func NewInstrumenter(subsystemIdentifier string) *Instrumenter {
	return &Instrumenter{subsystemIdentifier: subsystemIdentifier}
}

// InstrumentedRoundTripper returns an instrumented round tripper.
func (i *Instrumenter) InstrumentedRoundTripper() http.RoundTripper {
	inFlightRequestsGauge := promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      fmt.Sprintf("http_%s_in_flight_requests", i.subsystemIdentifier),
		Help:      fmt.Sprintf("A gauge of in-flight requests to the http %s.", i.subsystemIdentifier),
	})

	requestsPerEndpointCounter := promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      fmt.Sprintf("http_%s_requests_total", i.subsystemIdentifier),
			Help:      fmt.Sprintf("A counter for requests to the http %s per endpoint.", i.subsystemIdentifier),
		},
		[]string{"code", "method", "endpoint"},
	)

	requestLatencyHistogram := promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      fmt.Sprintf("http_%s_request_duration_seconds", i.subsystemIdentifier),
			Help:      fmt.Sprintf("A histogram of request latencies to the http %s .", i.subsystemIdentifier),
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method"},
	)

	return promhttp.InstrumentRoundTripperInFlight(inFlightRequestsGauge,
		promhttp.InstrumentRoundTripperDuration(requestLatencyHistogram,
			i.instrumentRoundTripperEndpoint(requestsPerEndpointCounter,
				http.DefaultTransport,
			),
		),
	)
}

func (i *Instrumenter) instrumentRoundTripperEndpoint(counter *prometheus.CounterVec, next http.RoundTripper) promhttp.RoundTripperFunc { //nolint:lll
	return func(r *http.Request) (*http.Response, error) {
		resp, err := next.RoundTrip(r)
		if err == nil {
			statusCode := strconv.Itoa(resp.StatusCode)
			counter.WithLabelValues(statusCode, strings.ToLower(resp.Request.Method), resp.Request.URL.Path).Inc()
		}

		return resp, errors.Wrap(err, "error making roundtrip")
	}
}

var ErrorReadingEndpoint = promauto.NewCounter(
	prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "error_reading_endpoint_total",
		Help:      "A counter for errored reading endpoint",
	},
)

var ScheduledEventsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "scheduled_events_total",
		Help:      "Scheduled Events from Azure",
	},
	[]string{"type"},
)

func GetHandler() http.Handler {
	return promhttp.Handler()
}
