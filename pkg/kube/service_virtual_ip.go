package kube

import (
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HeadlessService return a headless service spec with a given name, targeting a given deployment by name.
//
// HeadlessServices do not allocate virtual IPs for the service, instead rely on DNS to hold the addresses of ready pods.
func ServiceVirtualIP(name, deploymentName string, port corev1.ServicePort) *corev1.Service {
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
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				port,
			},
		},
	}
}
