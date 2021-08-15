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
