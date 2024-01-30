![codecov](https://codecov.io/gh/maksim-paskal/aks-node-termination-handler/graph/badge.svg?token=0Z0ENDM8VW)
![Docker Pulls](https://img.shields.io/docker/pulls/paskalmaksim/aks-node-termination-handler.svg)
![Licence](https://img.shields.io/github/license/maksim-paskal/aks-node-termination-handler.svg)

# AKS Node Termination Handler

Gracefully handle Azure Virtual Machines shutdown within Kubernetes

## Motivation

This tool ensures that kubernetes cluster responds appropriately to events that can cause your Azure Virtual Machines to become unavailable, like evictions Azure Spot Virtual Machines or Reboot. If not handled, your application code may not stop gracefully, take longer to recover full availability, or accidentally schedule work to nodes that are going down. It also can send Telegram or Slack message before Azure Virtual Machines evictions.

Based on [Azure Scheduled Events](https://docs.microsoft.com/en-us/azure/virtual-machines/linux/scheduled-events) and [Safely Drain a Node](https://kubernetes.io/docs/tasks/administer-cluster/safely-drain-node/)

Support Linux (amd64, arm64) and Windows (amd64) nodes.

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

# Create Windows nodepool with Spot Virtual Machines and autoscaling
az aks nodepool add \
--resource-group test-aks-group-eastus \
--cluster-name MyManagedCluster \
--os-type Windows \
--priority Spot \
--eviction-policy Delete \
--spot-max-price -1 \
--enable-cluster-autoscaler \
--name spot01 \
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
helm repo add aks-node-termination-handler https://maksim-paskal.github.io/aks-node-termination-handler/
helm repo update

helm upgrade aks-node-termination-handler \
--install \
--namespace kube-system \
aks-node-termination-handler/aks-node-termination-handler \
--set priorityClassName=system-node-critical
```

## Send notification events

You can compose your payload with markers that described [here](pkg/template/README.md)

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

## Simulate eviction

You can test with [Simulate Eviction API](https://docs.microsoft.com/en-us/rest/api/compute/virtual-machines/simulate-eviction) and change API endpoint to correspond `virtualMachineScaleSets` that used in AKS

```bash
POST https://management.azure.com/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Compute/virtualMachineScaleSets/{vmScaleSetName}/virtualMachines/{instanceId}/simulateEviction?api-version=2021-11-01
```

## Metrics

Application expose Prometheus metrics in `/metrics` endpoint. Installing latest chart will add annotations to pods:

```yaml
annotations:
  prometheus.io/port: "17923"
  prometheus.io/scrape: "true"
```
