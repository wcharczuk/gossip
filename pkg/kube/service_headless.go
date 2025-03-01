package kube

import (
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceHeadless return a headless service spec with a given name, targeting a given deployment by name.
//
// Headless services do not allocate virtual IPs for the service, instead rely on DNS to hold the addresses of ready pods.
func ServiceHeadless(name, deploymentName string, port corev1.ServicePort) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		},
		TypeMeta: v1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": deploymentName,
			},
			ClusterIP: "None",
			Ports: []corev1.ServicePort{
				port,
			},
		},
	}
}
