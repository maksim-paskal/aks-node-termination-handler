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
package template

import (
	"bytes"
	"context"
	"html/template"

	"github.com/pkg/errors"
	"github.com/vince-riv/aks-node-termination-handler/pkg/api"
	"github.com/vince-riv/aks-node-termination-handler/pkg/types"
)

type MessageType struct {
	Event        types.ScheduledEventsEvent
	Template     string
	NodeLabels   map[string]string `description:"Node labels"`
	NodeName     string            `description:"Node name"`
	ClusterName  string            `description:"Node label kubernetes.azure.com/cluster"`
	InstanceType string            `description:"Node label node.kubernetes.io/instance-type"`
	NodeArch     string            `description:"Node label kubernetes.io/arch"`
	NodeOS       string            `description:"Node label kubernetes.io/os"`
	NodeRole     string            `description:"Node label kubernetes.io/role"`
	NodeRegion   string            `description:"Node label topology.kubernetes.io/region"`
	NodeZone     string            `description:"Node label topology.kubernetes.io/zone"`
	NodePods     []string          `description:"List of pods on node"`
}

func NewMessageType(ctx context.Context, nodeName string, event types.ScheduledEventsEvent) (*MessageType, error) {
	nodeLabels, err := api.GetNodeLabels(ctx, nodeName)
	if err != nil {
		return nil, errors.Wrap(err, "error in nodes.get")
	}

	nodePods, err := api.GetNodePods(ctx, nodeName)
	if err != nil {
		return nil, errors.Wrap(err, "error in getNodePods")
	}

	return &MessageType{
		Event:        event,
		NodeName:     nodeName,
		NodeLabels:   nodeLabels,
		ClusterName:  nodeLabels["kubernetes.azure.com/cluster"],
		InstanceType: nodeLabels["node.kubernetes.io/instance-type"],
		NodeArch:     nodeLabels["kubernetes.io/arch"],
		NodeOS:       nodeLabels["kubernetes.io/os"],
		NodeRole:     nodeLabels["kubernetes.io/role"],
		NodeRegion:   nodeLabels["topology.kubernetes.io/region"],
		NodeZone:     nodeLabels["topology.kubernetes.io/zone"],
		NodePods:     nodePods,
	}, nil
}

func Message(obj *MessageType) (string, error) {
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
