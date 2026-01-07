![codecov](https://codecov.io/gh/vince-riv/aks-node-termination-handler/graph/badge.svg?token=0Z0ENDM8VW)
<!-- ![Docker Pulls](https://img.shields.io/docker/pulls/paskalmaksim/aks-node-termination-handler.svg) -->
<!-- ![Licence](https://img.shields.io/github/license/vince-riv/aks-node-termination-handler.svg) -->

# AKS Node Termination Handler

Gracefully handle Azure Virtual Machines shutdown within Kubernetes

## Motivation

This tool ensures that the Kubernetes cluster responds appropriately to events that can cause your Azure Virtual Machines to become unavailable, such as evictions of Azure Spot Virtual Machines or reboots. If not handled, your application code may not stop gracefully, recovery to full availability may take longer, or work might accidentally be scheduled to nodes that are shutting down. This tool can also send Telegram, Slack or Webhook messages before Azure Virtual Machines evictions occur.

Based on [Azure Scheduled Events](https://docs.microsoft.com/en-us/azure/virtual-machines/linux/scheduled-events) and [Safely Drain a Node](https://kubernetes.io/docs/tasks/administer-cluster/safely-drain-node/)

Support Linux (amd64, arm64) and Windows 2022, 2019* (amd64) nodes.

## Create Azure Kubernetes Cluster

<details>
  <summary>Create basic AKS cluster with Azure CLI</summary>

```bash
# https://learn.microsoft.com/en-us/azure/aks/learn/quick-kubernetes-deploy-cli

# Azure CLI version is 2.50.0
az --version

# Create resource group
az group create \
--name test-aks-group-eastus \
--location eastus

# Create aks cluster, with not spot instances
az aks create \
--resource-group test-aks-group-eastus \
--name MyManagedCluster \
--node-count 1 \
--node-vm-size Standard_DS2_v2 \
--enable-cluster-autoscaler \
--min-count 1 \
--max-count 3

# Create Linux nodepool with Spot Virtual Machines and autoscaling
az aks nodepool add \
--resource-group test-aks-group-eastus \
--cluster-name MyManagedCluster \
--name spotpool \
--priority Spot \
--eviction-policy Delete \
--spot-max-price -1 \
--enable-cluster-autoscaler \
--node-vm-size Standard_DS2_v2 \
--min-count 0 \
--max-count 10

# Create Windows (Windows Server 2022) nodepool with Spot Virtual Machines and autoscaling
az aks nodepool add \
--resource-group test-aks-group-eastus \
--cluster-name MyManagedCluster \
--os-type Windows \
--os-sku Windows2022 \
--priority Spot \
--eviction-policy Delete \
--spot-max-price -1 \
--enable-cluster-autoscaler \
--name spot01 \
--min-count 1 \
--max-count 3

# Create Windows (Windows Server 2019) nodepool with Spot Virtual Machines and autoscaling
az aks nodepool add \
--resource-group test-aks-group-eastus \
--cluster-name MyManagedCluster \
--os-type Windows \
--os-sku Windows2019 \
--priority Spot \
--eviction-policy Delete \
--spot-max-price -1 \
--enable-cluster-autoscaler \
--name spot2 \
--min-count 1 \
--max-count 3

# Get config to connect to cluster
az aks get-credentials \
--resource-group test-aks-group-eastus \
--name MyManagedCluster
```

</details>

## Installation

```bash
helm repo add aks-node-termination-handler https://vince-riv.github.io/aks-node-termination-handler/
helm repo update

helm upgrade aks-node-termination-handler \
--install \
--namespace kube-system \
aks-node-termination-handler/aks-node-termination-handler \
--set priorityClassName=system-node-critical
```

## Send notification events

You can compose your payload with markers that are described [here](pkg/template/README.md)

<details>
  <summary>Send Telegram notification</summary>

```bash
helm upgrade aks-node-termination-handler \
--install \
--namespace kube-system \
aks-node-termination-handler/aks-node-termination-handler \
--set priorityClassName=system-node-critical \
--set 'args[0]=-telegram.token=<telegram token>' \
--set 'args[1]=-telegram.chatID=<telegram chatid>'
```
</details>

<details>
  <summary>Send Slack notification</summary>

```bash
# create payload file
cat <<EOF | tee values.yaml
priorityClassName: system-node-critical

args:
- -webhook.url=https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX
- -webhook.template-file=/files/slack-payload.json
- -webhook.contentType=application/json
- -webhook.method=POST
- -webhook.timeout=30s
- -webhook.retries=3

configMap:
  data:
    slack-payload.json: |
      {
        "channel": "#mychannel",
        "username": "webhookbot",
        "text": "This is message for {{ .NodeName }}, {{ .InstanceType }} from {{ .NodeRegion }}",
        "icon_emoji": ":ghost:"
      }
EOF

# install/upgrade helm chart
helm upgrade aks-node-termination-handler \
--install \
--namespace kube-system \
aks-node-termination-handler/aks-node-termination-handler \
--values values.yaml
```
</details>

<details>
  <summary>Send Prometheus Pushgateway event</summary>

```bash
cat <<EOF | tee values.yaml
priorityClassName: system-node-critical

args:
- -webhook.url=http://prometheus-pushgateway.prometheus.svc.cluster.local:9091/metrics/job/aks-node-termination-handler
- -webhook.template-file=/files/prometheus-pushgateway-payload.txt
- -webhook.contentType=text/plain
- -webhook.method=POST
- -webhook.timeout=30s
- -webhook.retries=3

configMap:
  data:
    prometheus-pushgateway-payload.txt: |
      node_termination_event{node="{{ .NodeName }}"} 1
EOF

# install/upgrade helm chart
helm upgrade aks-node-termination-handler \
--install \
--namespace kube-system \
aks-node-termination-handler/aks-node-termination-handler \
--values values.yaml
```
</details>

<details>
  <summary>Use an HTTP proxy for making webhook requests</summary>

Use the flag `-webhook.http-proxy=http://someproxy:3128` for making requests with a proxy. This flag can use HTTP or HTTPS addresses. You can also use basic auth.

```bash
cat <<EOF | tee values.yaml
priorityClassName: system-node-critical

args:
- -webhook.url=https://someserver/somepath
- -webhook.template-file=/files/payload.json
- -webhook.contentType=text/plain
- -webhook.method=POST
- -webhook.timeout=30s
- -webhook.http-proxy=https://someproxy:3128
- -webhook.retries=3

configMap:
  data:
    payload.json: "This is message for {{ .NodeName }}, {{ .InstanceType }} from {{ .NodeRegion }}"
EOF

# install/upgrade helm chart
helm upgrade aks-node-termination-handler \
--install \
--namespace kube-system \
aks-node-termination-handler/aks-node-termination-handler \
--values values.yaml
```
</details>

## Simulate eviction

### Using Azure CLI

You need to install [Azure Command-Line Interface](https://learn.microsoft.com/en-us/cli/azure/), also you need setup [kubectl](https://learn.microsoft.com/en-us/azure/aks/learn/quick-kubernetes-deploy-cli#connect-to-the-cluster) to your AKS cluster

```bash
# Azure CLI version is 2.61.0
az --version

# Choose your AKS node to simulate eviction
kubectl get no

# Identify your node Azure ID
# subscriptions/{}/resourceGroups/{}/providers/Microsoft.Compute/virtualMachineScaleSets/{}/virtualMachines/{}
kubectl get no aks-nodename-to-simulate-eviction -o json | jq -r '.spec.providerID[9:]'

# Append to your node Azure ID additional path /simulateEviction?api-version=2024-03-01
# And execute this simulation with management.azure.com
az rest --verbose -m post --header "Accept=application/json" -u "https://management.azure.com/{Azure ID}/simulateEviction?api-version=2024-03-01"
```

### Using browser

You can test with [Simulate Eviction API](https://docs.microsoft.com/en-us/rest/api/compute/virtual-machines/simulate-eviction) and change API endpoint to correspond `virtualMachineScaleSets` that are used in AKS.

```bash
POST https://management.azure.com/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Compute/virtualMachineScaleSets/{vmScaleSetName}/virtualMachines/{instanceId}/simulateEviction?api-version=2021-11-01
```

## Metrics

The application exposes Prometheus metrics at the `/metrics` endpoint. Installing the latest chart will add annotations to the pods:

```yaml
annotations:
  prometheus.io/port: "17923"
  prometheus.io/scrape: "true"
```

## Windows 2019 support

If your cluster has (Linux and Windows 2019 nodes), you need to use another image:

```bash
helm upgrade aks-node-termination-handler \
--install \
--namespace kube-system \
aks-node-termination-handler/aks-node-termination-handler \
--set priorityClassName=system-node-critical \
--set image=paskalmaksim/aks-node-termination-handler:latest-ltsc2019
```

If your cluster includes Linux, Windows 2025, Windows 2022, and Windows 2019 nodes, you will need two separate helm installations of `aks-node-termination-handler`, each with different values.

<details>
  <summary>linux-windows2022.values.yaml</summary>

```bash
priorityClassName: system-node-critical

image: paskalmaksim/aks-node-termination-handler:latest

affinity:
  nodeAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      nodeSelectorTerms:
      - matchExpressions:
        - key: kubernetes.azure.com/os-sku
          operator: NotIn
          values:
          - Windows2025
          - Windows2019
```
</details>

<details>
  <summary>linux-windows2019.values.yaml</summary>

```bash
priorityClassName: system-node-critical

image: paskalmaksim/aks-node-termination-handler:latest-ltsc2019

nodeSelector:
  kubernetes.azure.com/os-sku: Windows2019
```
</details>

```bash
# install aks-node-termination-handler for Linux and Windows 2022 nodes
helm upgrade aks-node-termination-handler \
--install \
--namespace kube-system \
aks-node-termination-handler/aks-node-termination-handler \
--values=linux-windows2022.values.yaml

# install aks-node-termination-handler for Windows 2019 nodes
helm upgrade aks-node-termination-handler-windows-2019 \
--install \
--namespace kube-system \
aks-node-termination-handler/aks-node-termination-handler \
--values=linux-windows2019.values.yaml
```

## Red Hat OpenShift support

For OpenShift clusters that use Azure computes for their nodes, you must enable pod hostNetwork support because OpenShift networking has a [restriction](https://docs.openshift.com/container-platform/4.15/networking/understanding-networking.html) for using Azure Metadata Service.

This support can be enabled with `--set hostNetwork=true`

```bash
helm upgrade aks-node-termination-handler \
--install \
--namespace kube-system \
aks-node-termination-handler/aks-node-termination-handler \
--set priorityClassName=system-node-critical \
--set hostNetwork=true
```

## NetworkPolicy support

To limit what the workload can communicate with, Networkpolicy can be added via `--set networkPolicy.enabled=true`. To only allow egress communication towards required endpoints, supply the control plane IP address via  `--set networkPolicy.controlPlaneIP=10.11.12.13`. Additional egress rules can be added via `--set networkPolicy.additionalEgressRules=[]`, see the chart-provided `values.yaml` file for examples.

```bash
helm upgrade aks-node-termination-handler \
--install \
--namespace kube-system \
aks-node-termination-handler/aks-node-termination-handler \
--set networkPolicy.enabled=true \
--set networkPolicy.controlPlaneIP=10.11.12.2
```
