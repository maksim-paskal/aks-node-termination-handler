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

	"github.com/google/uuid"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/config"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrorrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
)

type KubectlLogger struct{}

func (b *KubectlLogger) Write(p []byte) (int, error) {
	log.Info(string(p))

	return 0, nil
}

type eventMessage struct {
	Type    string
	Reason  string
	Message string
}

func addNodeEvent(ctx context.Context, message *eventMessage) error {
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
		_, err = Clientset.CoreV1().Events("default").Create(ctx, &event, metav1.CreateOptions{})
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
