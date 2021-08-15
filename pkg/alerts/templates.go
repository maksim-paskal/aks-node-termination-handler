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
package alerts

import (
	"bytes"
	"html/template"

	"github.com/maksim-paskal/aks-node-termination-handler/pkg/types"
	"github.com/pkg/errors"
)

type TemplateMessageType struct {
	Node     string
	Event    types.ScheduledEventsEvent
	Template string
}

func TemplateMessage(obj TemplateMessageType) (string, error) {
	tmpl, err := template.New("message").Parse(obj.Template)
	if err != nil {
		return "", errors.Wrap(err, "error in template.Parse")
	}

	var tpl bytes.Buffer

	err = tmpl.Execute(&tpl, obj)
	if err != nil {
		return "", errors.Wrap(err, "error in template.Execute")
	}

	return tpl.String(), nil
}
