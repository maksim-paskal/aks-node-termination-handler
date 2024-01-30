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
package main_test

import (
	"context"
	"flag"
	"testing"

	"github.com/maksim-paskal/aks-node-termination-handler/pkg/alert"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/api"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/client"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/config"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/template"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/types"
)

var ctx = context.TODO()

func TestDrain(t *testing.T) {
	t.Parallel()

	_ = flag.Set("config", "./testdata/config_test.yaml")

	flag.Parse()

	if err := config.Load(); err != nil {
		t.Fatal(err)
	}

	if err := client.Init(); err != nil {
		t.Fatal(err)
	}

	if err := alert.Init(); err != nil {
		t.Fatal(err)
	}

	if err := alert.SendTelegram(&template.MessageType{Template: "e2e"}); err != nil {
		t.Fatal(err)
	}

	if err := api.DrainNode(ctx, *config.Get().NodeName, "Preempt", "manual"); err != nil {
		t.Fatal(err)
	}

	if err := api.AddNodeEvent(ctx, &types.EventMessage{
		Type:    "Info",
		Reason:  "TestDrain",
		Message: "TestDrain",
	}); err != nil {
		t.Fatal(err)
	}
}
