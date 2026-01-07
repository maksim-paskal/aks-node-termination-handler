KUBECONFIG=$(HOME)/.kube/azure-stage
tag=dev
image=paskalmaksim/aks-node-termination-handler:$(tag)
#telegramToken=1072104160:AAH2sFpHELeH5oxMmd-tsVjgTuzoYO6hSLM
#telegramChatID=-439460552
telegramToken=
telegramChatID=
node=`kubectl get no -lkubernetes.azure.com/scalesetpriority=spot | awk '{print $$1}' | tail -1` 

chart-lint:
	ct lint --all
	helm template ./charts/aks-node-termination-handler | kubectl apply --dry-run=client -f -

build:
	git tag -d `git tag -l "helm-chart-*"`
	go run github.com/goreleaser/goreleaser@latest build --clean --skip=validate --snapshot
	mv ./dist/aks-node-termination-handler_linux_amd64_v1/aks-node-termination-handler aks-node-termination-handler
	docker build --pull --push . -t $(image)

push:
	docker push $(image)

deploy:
	helm uninstall aks-node-termination-handler --namespace kube-system || true
	helm upgrade aks-node-termination-handler \
	--install \
	--namespace kube-system \
	./charts/aks-node-termination-handler \
	--set image=paskalmaksim/aks-node-termination-handler:dev \
	--set imagePullPolicy=Always \
	--set priorityClassName=system-node-critical \
	--set args[0]=-telegram.token=${telegramToken} \
	--set args[1]=-telegram.chatID=${telegramChatID} \
	--set args[2]=-taint.node \
	--set args[3]=-taint.effect=NoExecute \
	--set args[4]=-podGracePeriodSeconds=30 \

clean:
	kubectl delete ns aks-node-termination-handler

run:
	# https://t.me/joinchat/iaWV0bPT_Io5NGYy
	go run --race ./cmd \
	-kubeconfig=${KUBECONFIG} \
	-node=$(node) \
	-log.level=DEBUG \
	-log.pretty \
	-taint.node \
	-taint.effect=NoExecute \
	-podGracePeriodSeconds=30 \
	-gracePeriodSeconds=0 \
	-endpoint=http://localhost:28080/pkg/types/testdata/ScheduledEventsType.json \
	-webhook.url=http://localhost:9091/metrics/job/aks-node-termination-handler \
	-webhook.template='node_termination_event{node="{{ .NodeName }}"} 1' \
	-telegram.token=${telegramToken} \
	-telegram.chatID=${telegramChatID} \
	-web.address=127.0.0.1:17923

run-mock:
	go run --race ./mock -address=127.0.0.1:28080

test:
	./scripts/validate-license.sh
	go mod tidy
	go fmt ./cmd/... ./pkg/... ./internal/...
	go vet ./cmd/... ./pkg/... ./internal/...
	go test --race -coverprofile coverage.out ./cmd/... ./pkg/...
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run -v

.PHONY: e2e
e2e:
	go test -v -race ./e2e \
	-kubeconfig=$(KUBECONFIG) \
	-node=${node} \
	-telegram.token=${telegramToken} \
	-telegram.chatID=${telegramChatID}

coverage:
	go tool cover -html=coverage.out

test-release:
	go run github.com/goreleaser/goreleaser@latest release --snapshot --skip-publish --rm-dist

heap:
	go tool pprof -http=127.0.0.1:8080 http://localhost:17923/debug/pprof/heap

upgrade:
	go get -v -u k8s.io/client-go@v0.21.11
	go get -v -u k8s.io/kubectl@v0.21.11
	go get -v -u k8s.io/api@v0.21.11 || true
	go get -v -u k8s.io/apimachinery@v0.21.11
	go mod tidy

scan:
	@trivy image \
	-ignore-unfixed --no-progress --severity HIGH,CRITICAL \
	$(image)
	@helm template ./charts/aks-node-termination-handler > /tmp/aks-node-termination-handler.yaml
	@trivy config /tmp/aks-node-termination-handler.yaml
