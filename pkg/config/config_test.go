package config_test

import (
	"testing"

	"github.com/maksim-paskal/aks-node-termination-handler/pkg/config"
)

func TestConfig(t *testing.T) {
	t.Parallel()

	if err := config.Load(); err != nil {
		t.Fatal(err)
	}

	if want := "/some/test/path"; *config.Get().KubeConfigFile != want {
		t.Fatalf("KubeConfigFile != %s", want)
	}
}
