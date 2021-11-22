KUBECONFIG=$(HOME)/.kube/azure-dev

build:
	git tag -d `git tag -l "helm-chart-*"`
	goreleaser build --rm-dist --skip-validate --snapshot
	mv ./dist/aks-node-termination-handler_linux_amd64/aks-node-termination-handler aks-node-termination-handler
	docker build --pull . -t paskalmaksim/aks-node-termination-handler:dev

push:
	docker push paskalmaksim/aks-node-termination-handler:dev

deploy:
	helm uninstall aks-node-termination-handler --namespace aks-node-termination-handler || true
	helm upgrade aks-node-termination-handler \
	--install \
	--create-namespace \
	--namespace aks-node-termination-handler \
	./chart \
	--set args[0]=-telegram.token=1072104160:AAH2sFpHELeH5oxMmd-tsVjgTuzoYO6hSLM \
	--set args[1]=-telegram.chatID=-439460552

clean:
	kubectl delete ns aks-node-termination-handler

run:
	# https://t.me/joinchat/iaWV0bPT_Io5NGYy
	go run --race ./cmd \
	-kubeconfig=kubeconfig \
	-node=aks-spotcpu2-24406641-vmss00000e \
	-log.level=DEBUG \
	-log.prety \
	-endpoint=http://localhost:28080/pkg/types/testdata/ScheduledEventsType.json \
	-webhook.url=http://localhost:28080 \
	-telegram.token=1072104160:AAH2sFpHELeH5oxMmd-tsVjgTuzoYO6hSLM \
	-telegram.chatID=-439460552

run-mock:
	go run --race ./mock

test:
	./scripts/validate-license.sh
	go mod tidy
	go fmt ./cmd/... ./pkg/...
	CONFIG=testdata/config_test.yaml go test --race ./cmd/... ./pkg/...
	golangci-lint run -v

test-release:
	goreleaser release --snapshot --skip-publish --rm-dist

upgrade:
	go get -v -u k8s.io/api@v0.20.9 || true
	go get -v -u k8s.io/apimachinery@v0.20.9
	go get -v -u k8s.io/client-go@v0.20.9
	go mod tidy