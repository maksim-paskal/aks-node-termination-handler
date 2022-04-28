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

	"github.com/maksim-paskal/aks-node-termination-handler/pkg/alert"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/api"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/config"
	log "github.com/sirupsen/logrus"
)

func Start() {
	log.Info("http.address=", *config.Get().WebHTTPAddress)

	if err := http.ListenAndServe(*config.Get().WebHTTPAddress, GetHandler()); err != nil {
		log.WithError(err).Fatal()
	}
}

func GetHandler() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", handlerHealthz)
	mux.HandleFunc("/drainNode", handlerDrainNode)

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	return mux
}

func handlerHealthz(w http.ResponseWriter, r *http.Request) {
	// check alerts transports
	if err := alert.Ping(); err != nil {
		log.WithError(err).Error("alerts transport is not working")
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	// check kubernetes API
	if err := api.Ping(); err != nil {
		log.WithError(err).Error("kubernetes API is not available")
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	_, _ = w.Write([]byte("LIVE"))
}

func handlerDrainNode(w http.ResponseWriter, r *http.Request) {
	err := api.DrainNode(context.Background(), *config.Get().NodeName, "Preempt", "manual")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	_, _ = w.Write([]byte("done"))
}
