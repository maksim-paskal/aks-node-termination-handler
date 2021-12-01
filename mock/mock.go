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
	"fmt"
	"io"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

const port = ":28080"

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
		log.Fatal(err)
	}

	defer r.Body.Close()

	request = append(request, "--BODY--")
	request = append(request, string(bodyBytes))

	_, _ = w.Write([]byte(strings.Join(request, "\n")))
}

// simple server for test env.
func main() {
	http.HandleFunc("/debug", debugHandler)
	http.Handle("/", http.FileServer(http.Dir(".")))
	log.Infof("Listen %s", port)

	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.WithError(err).Fatal()
	}
}
