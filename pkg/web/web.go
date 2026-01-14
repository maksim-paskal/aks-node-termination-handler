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
package web

import (
	"context"
	"net/http"
	"net/http/pprof"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vince-riv/aks-node-termination-handler/pkg/alert"
	"github.com/vince-riv/aks-node-termination-handler/pkg/api"
	"github.com/vince-riv/aks-node-termination-handler/pkg/config"
	"github.com/vince-riv/aks-node-termination-handler/pkg/events"
	"github.com/vince-riv/aks-node-termination-handler/pkg/metrics"
)

func Start(ctx context.Context) {
	const (
		readTimeout    = 5 * time.Second
		requestTimeout = 10 * time.Second
		writeTimeout   = 20 * time.Second
	)

	server := &http.Server{
		Addr:         *config.Get().WebHTTPAddress,
		Handler:      http.TimeoutHandler(GetHandler(), requestTimeout, "timeout"),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	log.Info("web.address=", server.Addr)

	go func() {
		<-ctx.Done()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), config.Get().GracePeriod())
		defer shutdownCancel()

		_ = server.Shutdown(shutdownCtx) //nolint:contextcheck
	}()

	err := server.ListenAndServe()
	if err != nil && ctx.Err() == nil {
		log.WithError(err).Fatal()
	}
}

func GetHandler() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", handlerHealthz)
	mux.HandleFunc("/drainNode", handlerDrainNode)

	mux.Handle("/metrics", metrics.GetHandler())

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	return mux
}

func handlerHealthz(w http.ResponseWriter, r *http.Request) {
	// check alerts transports
	err := alert.Ping()
	if err != nil {
		log.WithError(err).Error("alerts transport is not working")
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	// check instance metadata API
	err = events.Ping(r.Context())
	if err != nil {
		log.WithError(err).Error("instance metadata API is not available")
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	// check kubernetes API
	if _, err := api.GetNode(r.Context(), *config.Get().NodeName); err != nil {
		log.WithError(err).Error("kubernetes API is not available")
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	_, _ = w.Write([]byte("LIVE"))
}

func handlerDrainNode(w http.ResponseWriter, r *http.Request) {
	err := api.DrainNode(r.Context(), *config.Get().NodeName, "Preempt", "manual", *config.Get().PodGracePeriodSeconds)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	_, _ = w.Write([]byte("done"))
}
