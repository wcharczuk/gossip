package main

import (
	"gossip/pkg/kube"
	"gossip/pkg/kube/clifactory"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func main() {
	clifactory.Resources{
		"service":    kube.ServiceHeadless("gossip-members", "gossip", v1.ServicePort{Name: "http", Protocol: v1.ProtocolTCP, Port: 7946, TargetPort: intstr.FromInt32(7946)}),
		"deployment": kube.Deployment("gossip", "sf-microk8s.hawk-bluegill.ts.net:32000/gossip:latest", kube.OptDeploymentPort("http", 7946, v1.ProtocolTCP)),
	}.Main()
}
