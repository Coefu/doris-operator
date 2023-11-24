package native_resources

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func MakeService(p *Params, serviceType string) *corev1.Service {
	if serviceType == "headless" {
		return &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      p.Name,
				Namespace: p.Namespace,
			},
			Spec: corev1.ServiceSpec{
				Selector:  p.Labels,
				Ports:     p.ServicePorts,
				ClusterIP: "None",
			},
		}
	}
	if serviceType == "nodeport" {
		return &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      p.Name + "-nodeport",
				Namespace: p.Namespace,
			},
			Spec: corev1.ServiceSpec{
				Selector: p.Labels,
				Ports:    p.ServicePorts,
				Type:     corev1.ServiceTypeNodePort,
			},
		}
	}
	return &corev1.Service{}
}
