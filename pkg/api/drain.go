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
package api

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrorrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"k8s.io/kubectl/pkg/drain"
)

const AzureProviderID = "^azure:///subscriptions/(.+)/resourceGroups/(.+)/providers/Microsoft.Compute/virtualMachineScaleSets/(.+)/virtualMachines/(.+)$" //nolint:lll

var (
	errAzureProviderIDNotValid = errors.New("azureProviderID not valid")
	taintKeyPrefix             = "aks-node-termination-handler"
)

func GetAzureResourceName(ctx context.Context, nodeName string) (string, error) {
	node, err := Clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrap(err, "error in Clientset.CoreV1().Nodes().Get")
	}

	regexpObj := regexp.MustCompile(AzureProviderID)

	if !regexpObj.MatchString(node.Spec.ProviderID) {
		return "", errors.Wrap(errAzureProviderIDNotValid, node.Spec.ProviderID)
	}

	v := regexpObj.FindAllStringSubmatch(node.Spec.ProviderID, 1)
	result := fmt.Sprintf("%s_%s", v[0][3], v[0][4])

	return result, nil
}

func DrainNode(ctx context.Context, nodeName string, eventType string, eventID string) error {
	log.Infof("Draining node %s", nodeName)

	node, err := GetNode(ctx, nodeName)
	if err != nil {
		return errors.Wrap(err, "error in nodes.get")
	}

	if node.Spec.Unschedulable {
		log.Infof("Node %s is already Unschedulable", node.Name)

		return nil
	}

	err = addTaint(ctx, node, getTaintKey(eventType), eventID)
	if err != nil {
		return errors.Wrap(err, "failed to taint node")
	}

	logger := &KubectlLogger{}

	helper := &drain.Helper{
		Ctx:                 ctx,
		Client:              Clientset,
		Force:               true,
		GracePeriodSeconds:  -1,
		IgnoreAllDaemonSets: true,
		Out:                 logger,
		ErrOut:              logger,
		DeleteEmptyDirData:  true,
		Timeout:             time.Duration(120) * time.Second, //nolint:gomnd
	}

	if err := drain.RunCordonOrUncordon(helper, node, true); err != nil {
		return errors.Wrap(err, "error in drain.RunCordonOrUncordon")
	}

	if err := drain.RunNodeDrain(helper, node.Name); err != nil {
		return errors.Wrap(err, "error in drain.RunNodeDrain")
	}

	return nil
}

func getTaintKey(eventType string) string {
	return fmt.Sprintf("%s/%s", taintKeyPrefix, strings.ToLower(eventType))
}

func addTaint(ctx context.Context, node *corev1.Node, taintKey string, taintValue string) error {
	freshNode := node.DeepCopy()

	var err error

	updateErr := wait.ExponentialBackoff(retry.DefaultBackoff, func() (bool, error) {
		if freshNode, err = Clientset.CoreV1().Nodes().Get(ctx, freshNode.Name, metav1.GetOptions{}); err != nil {
			nodeErr := errors.Wrap(err, fmt.Sprintf("failed to get node %s", freshNode.Name))
			log.Error(nodeErr)

			return false, nodeErr
		}
		err = updateNodeWith(ctx, taintKey, taintValue, err, freshNode)
		switch {
		case err == nil:
			return true, nil
		case apierrorrs.IsConflict(err):
			return false, nil
		case err != nil:
			return false, errors.Wrap(err, fmt.Sprintf("failed to taint node %s with key %s", freshNode.Name, taintKey))
		}

		return false, nil
	})

	if updateErr != nil {
		return err
	}

	log.Warnf("Successfully added taint %s on node %s", taintKey, freshNode.Name)

	return nil
}

func updateNodeWith(ctx context.Context, taintKey string, taintValue string, err error, node *corev1.Node) error {
	node.Spec.Taints = append(node.Spec.Taints, corev1.Taint{
		Key:    taintKey,
		Value:  taintValue,
		Effect: corev1.TaintEffectNoSchedule,
	})
	_, err = Clientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})

	return errors.Wrap(err, "failed to update node with taint")
}

func GetNode(ctx context.Context, nodeName string) (*corev1.Node, error) {
	node, err := Clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "error in nodes.get")
	}

	return node, nil
}
