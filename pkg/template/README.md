# Templating Options

| Template  | Description | Example |
| --------- | ----------- | ------- |
| `{{ .Event.EventId }}` | Globally unique identifier for this event. | 602d9444-d2cd-49c7-8624-8643e7171297 |
| `{{ .Event.EventType }}` | Impact this event causes. | Reboot |
| `{{ .Event.ResourceType }}` | Type of resource this event affects. | VirtualMachine |
| `{{ .Event.Resources }}` | List of resources this event affects. | [ FrontEnd_IN_0 ...] |
| `{{ .Event.EventStatus }}` | Status of this event. | Scheduled |
| `{{ .Event.NotBefore }}` | Time after which this event can start. The event is guaranteed to not start before this time. Will be blank if the event has already started | Mon, 19 Sep 2016 18:29:47 GMT |
| `{{ .Event.Description }}` | Description of this event. | Host server is undergoing maintenance |
| `{{ .Event.EventSource }}` | Initiator of the event. | Platform |
| `{{ .Event.DurationInSeconds }}` | The expected duration of the interruption caused by the event. | -1 |
| `{{ .NodeLabels }}` | Node labels | kubernetes.azure.com/agentpool:spotcpu4m16n ... |
| `{{ .NodeName }}` | Node name | aks-spotcpu4m16n-41289323-vmss0000ny |
| `{{ .ClusterName }}` | Node label kubernetes.azure.com/cluster | MC_EAST-US-RC-STAGE_stage-cluster_eastus |
| `{{ .InstanceType }}` | Node label node.kubernetes.io/instance-type | Standard_D4as_v5 |
| `{{ .NodeArch }}` | Node label kubernetes.io/arch | amd64 |
| `{{ .NodeOS }}` | Node label kubernetes.io/os | linux |
| `{{ .NodeRole }}` | Node label kubernetes.io/role | agent |
| `{{ .NodeRegion }}` | Node label topology.kubernetes.io/region | eastus |
| `{{ .NodeZone }}` | Node label topology.kubernetes.io/zone | 0 |
| `{{ .NodePods }}` | List of pods on node | [ pod1 ...] |
