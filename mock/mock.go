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
package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

func debugHandler(w http.ResponseWriter, r *http.Request) {
	// Create return string
	request := []string{}
	// Add the request string
	url := fmt.Sprintf("%v %v %v", r.Method, r.URL, r.Proto)
	request = append(request, url)
	// Add the host
	request = append(request, fmt.Sprintf("Host: %v", r.Host))

	request = append(request, "--HEADERS--")
	// Loop through headers
	for name, headers := range r.Header {
		name = strings.ToLower(name)

		for _, h := range headers {
			request = append(request, fmt.Sprintf("%v: %v", name, h))
		}
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.WithError(err).Fatal()
	}

	defer r.Body.Close()

	request = append(request, "--BODY--")
	request = append(request, string(bodyBytes))

	_, _ = w.Write([]byte(strings.Join(request, "\n")))
}

// simple server for test env.
func main() {
	address := flag.String("address", ":28080", "address")
	flag.Parse()

	http.HandleFunc("/debug", debugHandler)
	http.Handle("/", http.FileServer(http.Dir(".")))

	scheduledEventsType, err := filepath.Abs("pkg/types/testdata/ScheduledEventsType.json")
	if err != nil {
		log.WithError(err).Fatal()
	}

	log.Infof("edit %s file to test events", scheduledEventsType)

	const (
		readTimeout  = 5 * time.Second
		writeTimeout = 10 * time.Second
	)

	server := &http.Server{
		Addr:         *address,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	log.Infof("Listen %s", server.Addr)

	err = server.ListenAndServe()
	if err != nil {
		log.WithError(err).Fatal()
	}
}
