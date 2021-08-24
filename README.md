# AKS Node Termination Handler
Gracefully handle Azure Virtual Machines shutdown within Kubernetes

## Motivation
This tool ensures that kubernetes cluster responds appropriately to events that can cause your Azure Virtual Machines to become unavailable, like evictions Azure Spot Virtual Machines or Reboot. If not handled, your application code may not stop gracefully, take longer to recover full availability, or accidentally schedule work to nodes that are going down. It also can send Telegram or Slack message before Azure Virtual Machines evictions.

Based on [Azure Scheduled Events](https://docs.microsoft.com/en-us/azure/virtual-machines/linux/scheduled-events) and [Safely Drain a Node](https://kubernetes.io/docs/tasks/administer-cluster/safely-drain-node/)

## Installation
```bash
git clone git@github.com:maksim-paskal/aks-node-termination-handler.git

helm upgrade aks-node-termination-handler \
--install \
--create-namespace \
--namespace aks-node-termination-handler \
./chart
```

## Alerting
To make alerts to Telegram or Slack
```
helm upgrade aks-node-termination-handler \
--install \
--create-namespace \
--namespace aks-node-termination-handler \
./chart \
--set args[0]=-webhook.url=https://hooks.slack.com/services/ID/ID/ID \
--set args[1]=-telegram.token=<telegram token> \
--set args[2]=-telegram.chatID=<telegram chatid> \
```
## Simulate eviction
You can test with [Simulate Eviction API](https://docs.microsoft.com/en-us/rest/api/compute/virtual-machines/simulate-eviction) and change API endpoint to correspond `virtualMachineScaleSets` that used in AKS
```
POST https://management.azure.com/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Compute/virtualMachineScaleSets/{vmName}/virtualMachines/{vmID}/simulateEviction?api-version=2021-03-01
```