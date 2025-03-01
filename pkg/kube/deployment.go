package kube

import (
	apiv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func Deployment(name, image string, opts ...DeploymentOption) *apiv1.Deployment {
	obj := apiv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app": name,
			},
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		Spec: apiv1.DeploymentSpec{
			Replicas: Ref[int32](3),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": name,
				},
			},
			Strategy: apiv1.DeploymentStrategy{
				Type: apiv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &apiv1.RollingUpdateDeployment{
					MaxSurge: &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": name,
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyAlways,
					Containers: []corev1.Container{
						{
							Name:  name,
							Image: image,
						},
					},
				},
			},
		},
	}
	for _, opt := range opts {
		opt(&obj)
	}
	return &obj
}

func OptDeploymentReplicas(replicas int32) DeploymentOption {
	return func(d *apiv1.Deployment) {
		d.Spec.Replicas = Ref(replicas)
	}
}

func OptDeploymentPort(name string, port int32, protocol corev1.Protocol) DeploymentOption {
	return func(d *apiv1.Deployment) {
		d.Spec.Template.Spec.Containers[0].Ports = []corev1.ContainerPort{
			{
				Name:          name,
				ContainerPort: port,
				Protocol:      protocol,
			},
		}
	}
}

type DeploymentOption func(*apiv1.Deployment)

func Ref[A any](v A) *A {
	return &v
}
