KUBECONFIG=$(HOME)/.kube/azure-dev
tag=dev
image=paskalmaksim/aks-node-termination-handler:$(tag)

build:
	git tag -d `git tag -l "helm-chart-*"`
	go run github.com/goreleaser/goreleaser@latest build --rm-dist --skip-validate --snapshot
	mv ./dist/aks-node-termination-handler_linux_amd64/aks-node-termination-handler aks-node-termination-handler
	docker build --pull . -t $(image)

push:
	docker push $(image)

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
	-node=aks-spotcpu2-24406641-vmss00002v \
	-log.level=DEBUG \
	-log.prety \
	-endpoint=http://localhost:28080/pkg/types/testdata/ScheduledEventsType.json \
	-webhook.url=http://localhost:9091/metrics/job/aks-node-termination-handler \
	-webhook.template='node_termination_event{node="{{ .Node }}"} 1' \
	-telegram.token=1072104160:AAH2sFpHELeH5oxMmd-tsVjgTuzoYO6hSLM \
	-telegram.chatID=-439460552

run-mock:
	go run --race ./mock

test:
	./scripts/validate-license.sh
	go mod tidy
	go fmt ./cmd/... ./pkg/...
	CONFIG=testdata/config_test.yaml go test --race ./cmd/... ./pkg/...
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run -v

test-release:
	go run github.com/goreleaser/goreleaser@latest release --snapshot --skip-publish --rm-dist

upgrade:
	go get -v -u k8s.io/client-go@v0.21.10
	go get -v -u k8s.io/kubectl@v0.21.10
	go get -v -u k8s.io/api@v0.21.10 || true
	go get -v -u k8s.io/apimachinery@v0.21.10
	go mod tidy

scan:
	@trivy image \
	-ignore-unfixed --no-progress --severity HIGH,CRITICAL \
	$(image)