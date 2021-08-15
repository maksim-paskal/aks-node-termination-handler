package api

import (
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/config"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	Clientset  *kubernetes.Clientset
	Restconfig *rest.Config
)

func MakeAuth() error {
	var err error

	if len(*config.Get().KubeConfigFile) > 0 {
		Restconfig, err = clientcmd.BuildConfigFromFlags("", *config.Get().KubeConfigFile)
		if err != nil {
			return errors.Wrap(err, "error in clientcmd.BuildConfigFromFlags")
		}
	} else {
		log.Info("No kubeconfig file use incluster")
		Restconfig, err = rest.InClusterConfig()
		if err != nil {
			return errors.Wrap(err, "error in rest.InClusterConfig")
		}
	}

	Clientset, err = kubernetes.NewForConfig(Restconfig)
	if err != nil {
		log.WithError(err).Fatal()
	}

	return nil
}
