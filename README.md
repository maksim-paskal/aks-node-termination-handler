# AKS Node Termination Handler

Gracefully handle Azure Virtual Machines shutdown within Kubernetes

## Motivation

This tool ensures that kubernetes cluster responds appropriately to events that can cause your Azure Virtual Machines to become unavailable, like evictions Azure Spot Virtual Machines or Reboot. If not handled, your application code may not stop gracefully, take longer to recover full availability, or accidentally schedule work to nodes that are going down. It also can send Telegram or Slack message before Azure Virtual Machines evictions.

Based on [Azure Scheduled Events](https://docs.microsoft.com/en-us/azure/virtual-machines/linux/scheduled-events) and [Safely Drain a Node](https://kubernetes.io/docs/tasks/administer-cluster/safely-drain-node/)

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

## Alerting

To make alerts to Telegram or Slack or Webhook

```bash
helm upgrade aks-node-termination-handler \
--install \
--namespace kube-system \
aks-node-termination-handler/aks-node-termination-handler \
--set priorityClassName=system-node-critical \
--set args[0]=-telegram.token=<telegram token> \
--set args[1]=-telegram.chatID=<telegram chatid> \
--set args[2]=-webhook.url=http://prometheus-pushgateway.prometheus.svc.cluster.local:9091/metrics/job/aks-node-termination-handler \
--set args[3]=-webhook.template='node_termination_event{node="{{ .Node }}"} 1'
```

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
