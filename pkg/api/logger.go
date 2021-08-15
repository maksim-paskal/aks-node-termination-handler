package api

import (
	log "github.com/sirupsen/logrus"
)

type KubectlLogger struct{}

func (b *KubectlLogger) Write(p []byte) (nn int, err error) {
	log.Info(string(p))

	return 0, nil
}
