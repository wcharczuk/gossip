package main

import (
	"gossip/pkg/kube"
	"gossip/pkg/kube/clifactory"

	v1 "k8s.io/api/core/v1"
)

func main() {
	clifactory.Resources{
		"service": kube.ServiceVirtualIP("metric-sink", "metric-sink", v1.ServicePort{Name: "http", Protocol: v1.ProtocolTCP, Port: 3000}),
		"deployment": kube.Deployment(
			"metric-sink",
			"sf-microk8s.hawk-bluegill.ts.net:32000/metric-sink:latest",
			kube.OptDeploymentReplicas(1),
			kube.OptDeploymentPort("http", 3000, v1.ProtocolTCP),
		),
	}.Main()
}
