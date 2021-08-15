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