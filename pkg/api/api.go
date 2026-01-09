/*
Copyright paskal.maksim@gmail.com (Original Author 2021-2025)
Copyright github@vince-riv.io (Modifications 2026-present)
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
package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/vince-riv/aks-node-termination-handler/pkg/client"
	"github.com/vince-riv/aks-node-termination-handler/pkg/config"
	"github.com/vince-riv/aks-node-termination-handler/pkg/logger"
	"github.com/vince-riv/aks-node-termination-handler/pkg/types"
	corev1 "k8s.io/api/core/v1"
	apierrorrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"k8s.io/kubectl/pkg/drain"
)

const taintKeyPrefix = "aks-node-termination-handler"

func GetAzureResourceName(ctx context.Context, nodeName string) (string, error) {
	// return user defined resource name
	if len(*config.Get().ResourceName) > 0 {
		return *config.Get().ResourceName, nil
	}

	node, err := client.GetKubernetesClient().CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrap(err, "error in Clientset.CoreV1().Nodes().Get")
	}

	azureResourceName, err := types.NewAzureResource(node.Spec.ProviderID)
	if err != nil {
		return "", errors.Wrap(err, "error in types.NewAzureResource")
	}

	return azureResourceName.EventResourceName, nil
}

func DrainNode(ctx context.Context, nodeName string, eventType string, eventID string) error { //nolint:cyclop
	log.Infof("Draining node %s", nodeName)

	node, err := GetNode(ctx, nodeName)
	if err != nil {
		return errors.Wrap(err, "error in nodes.get")
	}

	if node.Spec.Unschedulable {
		log.Infof("Node %s is already Unschedulable", node.Name)

		return nil
	}

	// taint node before draining if effect is NoSchedule or TaintEffectPreferNoSchedule
	if *config.Get().TaintNode && *config.Get().TaintEffect != string(corev1.TaintEffectNoExecute) {
		err = addTaint(ctx, node, getTaintKey(eventType), eventID)
		if err != nil {
			return errors.Wrap(err, "failed to taint node")
		}
	}

	logger := &logger.KubectlLogger{}
	logger.Log = func(message string) {
		log.Info(message)
	}

	helper := &drain.Helper{
		Ctx:                 ctx,
		Client:              client.GetKubernetesClient(),
		Force:               true,
		GracePeriodSeconds:  *config.Get().PodGracePeriodSeconds,
		IgnoreAllDaemonSets: true,
		Out:                 logger,
		ErrOut:              logger,
		DeleteEmptyDirData:  true,
		Timeout:             config.Get().NodeGracePeriod(),
		DisableEviction:     *config.Get().DisableEviction,
	}

	if *config.Get().DryRun {
		log.Infof("DRY RUN ENABLED; skipping cordoning and draining of node %s", node.Name)
	} else {
		if err := drain.RunCordonOrUncordon(helper, node, true); err != nil {
			return errors.Wrap(err, "error in drain.RunCordonOrUncordon")
		}

		if err := drain.RunNodeDrain(helper, node.Name); err != nil {
			return errors.Wrap(err, "error in drain.RunNodeDrain")
		}
	}

	// taint node after draining if effect is TaintEffectNoExecute
	// this NoExecute taint effect will stop all daemonsents on the node that can not handle this effect
	if *config.Get().TaintNode && *config.Get().TaintEffect == string(corev1.TaintEffectNoExecute) {
		err = addTaint(ctx, node, getTaintKey(eventType), eventID)
		if err != nil {
			return errors.Wrap(err, "failed to taint node")
		}
	}

	return nil
}

func getTaintKey(eventType string) string {
	return fmt.Sprintf("%s/%s", taintKeyPrefix, strings.ToLower(eventType))
}

func addTaint(ctx context.Context, node *corev1.Node, taintKey string, taintValue string) error {
	log.Infof("Adding taint %s=%s on node %s", taintKey, taintValue, node.Name)

	freshNode := node.DeepCopy()

	var err error

	updateErr := wait.ExponentialBackoff(retry.DefaultBackoff, func() (bool, error) {
		if freshNode, err = client.GetKubernetesClient().CoreV1().Nodes().Get(ctx, freshNode.Name, metav1.GetOptions{}); err != nil {
			nodeErr := errors.Wrapf(err, "failed to get node %s", freshNode.Name)
			log.Error(nodeErr)

			return false, nodeErr
		}

		err = updateNodeWith(ctx, taintKey, taintValue, freshNode)

		switch {
		case err == nil:
			return true, nil
		case apierrorrs.IsConflict(err):
			return false, nil
		case err != nil:
			return false, errors.Wrapf(err, "failed to taint node %s with key %s", freshNode.Name, taintKey)
		}

		return false, nil
	})

	if updateErr != nil {
		return err
	}

	log.Warnf("Successfully added taint %s on node %s", taintKey, freshNode.Name)

	return nil
}

func updateNodeWith(ctx context.Context, taintKey string, taintValue string, node *corev1.Node) error {
	if *config.Get().DryRun {
		log.Infof("DRY RUN ENABLED; skipping adding taint %s=%s on node %s", taintKey, taintValue, node.Name)
		return nil
	}
	node.Spec.Taints = append(node.Spec.Taints, corev1.Taint{
		Key:    taintKey,
		Value:  taintValue,
		Effect: corev1.TaintEffect(*config.Get().TaintEffect),
	})
	_, err := client.GetKubernetesClient().CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})

	return errors.Wrap(err, "failed to update node with taint")
}

func GetNode(ctx context.Context, nodeName string) (*corev1.Node, error) {
	node, err := client.GetKubernetesClient().CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "error in nodes.get")
	}

	return node, nil
}

func AddNodeEvent(ctx context.Context, eventType, eventReason, eventMessage string) error {
	message := &types.EventMessage{
		Type:    eventType,
		Reason:  eventReason,
		Message: eventMessage,
	}

	return AddNodeEventMessage(ctx, message)
}

func AddNodeEventMessage(ctx context.Context, message *types.EventMessage) error {
	node, err := GetNode(ctx, *config.Get().NodeName)
	if err != nil {
		return errors.Wrap(err, "error in GetNode")
	}

	event := corev1.Event{
		InvolvedObject: corev1.ObjectReference{
			APIVersion:      "v1",
			Kind:            "Node",
			Name:            node.Name,
			UID:             node.UID,
			ResourceVersion: node.ResourceVersion,
		},
		Count:          1,
		FirstTimestamp: metav1.Now(),
		LastTimestamp:  metav1.Now(),
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s.%s", *config.Get().NodeName, uuid.New().String()),
		},
		Type:    message.Type,
		Reason:  message.Reason,
		Message: message.Message,
		Source: corev1.EventSource{
			Component: "aks-node-termination-handler",
		},
	}

	err = wait.ExponentialBackoff(retry.DefaultBackoff, func() (bool, error) {
		_, err = client.GetKubernetesClient().CoreV1().Events("default").Create(ctx, &event, metav1.CreateOptions{})

		switch {
		case err == nil:
			return true, nil
		case apierrorrs.IsConflict(err):
			return false, nil
		case err != nil:
			return false, errors.Wrap(err, "failed to create event")
		}

		return false, nil
	})
	if err != nil {
		return errors.Wrap(err, "failed to add event")
	}

	return nil
}

func GetNodeLabels(ctx context.Context, nodeName string) (map[string]string, error) {
	// this need for unit tests
	if nodeName == "!!invalid!!GetNodeLabels" {
		return nil, errors.New("invalid node name")
	}

	// this need for unit tests
	if client.GetKubernetesClient() == nil {
		return make(map[string]string), nil
	}

	node, err := client.GetKubernetesClient().CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "error in nodes.get")
	}

	return node.Labels, nil
}

func GetNodePods(ctx context.Context, nodeName string) ([]string, error) {
	// this need for unit tests
	if nodeName == "!!invalid!!GetNodePods" {
		return nil, errors.New("invalid node name")
	}

	// this need for unit tests
	if client.GetKubernetesClient() == nil {
		return []string{}, nil
	}

	pods, err := client.GetKubernetesClient().CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "error in pods.list")
	}

	result := make([]string, 0)

	for _, pod := range pods.Items {
		// ignore DaemonSet pods from pods list, because they are not affected by node termination
		if getPodReferenceKind(pod) == "DaemonSet" {
			continue
		}

		if pod.Spec.NodeName == nodeName {
			result = append(result, pod.Name)
		}
	}

	return result, nil
}

func getPodReferenceKind(pod corev1.Pod) string {
	for _, ownerReference := range pod.OwnerReferences {
		if len(ownerReference.Kind) > 0 {
			return ownerReference.Kind
		}
	}

	return ""
}
