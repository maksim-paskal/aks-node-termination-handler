package api

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/drain"
)

const AzureProviderID = "^azure:///subscriptions/(.+)/resourceGroups/(.+)/providers/Microsoft.Compute/virtualMachineScaleSets/(.+)/virtualMachines/(.+)$" //nolint:lll

func GetAzureResourceName(ctx context.Context, nodeName string) (string, error) {
	node, err := Clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrap(err, "error in Clientset.CoreV1().Nodes().Get")
	}

	re := regexp.MustCompile(AzureProviderID)

	if !re.MatchString(node.Spec.ProviderID) {
		return "", errors.Wrap(errAzureProviderIDNotValid, node.Spec.ProviderID)
	}

	v := re.FindAllStringSubmatch(node.Spec.ProviderID, 1)
	result := fmt.Sprintf("%s_%s", v[0][3], v[0][4])

	return result, nil
}

func DrainNode(ctx context.Context, nodeName string) error {
	node, err := Clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "error in nodes.get")
	}

	if node.Spec.Unschedulable {
		log.Infof("Node %s is already Unschedulable", nodeName)

		return nil
	}

	logger := &KubectlLogger{}

	helper := &drain.Helper{
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

	if err := drain.RunNodeDrain(helper, nodeName); err != nil {
		return errors.Wrap(err, "error in drain.RunNodeDrain")
	}

	return nil
}
