package native_resources

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func MakeStatefulSet(p *Params) *appsv1.StatefulSet {
	var replicas int32 = 1
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.Name,
			Namespace: p.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: p.Name,
			Replicas:    &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: p.Labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					//Annotations: annotationForSts(p.InstanceIP),
					Labels: p.Labels,
				},
				Spec: corev1.PodSpec{
					Containers:       containersForSts(p),
					Volumes:          p.Volumes,
					ImagePullSecrets: p.ImagePullSecrets,
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				pvcForSts(p),
			},
		},
	}
	return sts
}

func containersForSts(p *Params) []corev1.Container {
	containers := []corev1.Container{}
	app := corev1.Container{
		Name:         p.Name,
		Image:        p.Image,
		Command:      p.Command,
		Args:         p.Args,
		Ports:        p.PodPorts,
		VolumeMounts: p.VolumeMounts,
		Resources:    p.ContainerResource,
	}
	containers = append(containers, app)
	return containers
}

func pvcForSts(p *Params) corev1.PersistentVolumeClaim {
	return corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-pvc", p.Name),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{"ReadWriteOnce"},
			StorageClassName: p.StorageClass,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse(p.Storage)},
				Limits:   corev1.ResourceList{corev1.ResourceStorage: resource.MustParse(p.Storage)},
			},
		},
	}
}
