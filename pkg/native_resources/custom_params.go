package native_resources

import corev1 "k8s.io/api/core/v1"

type Params struct {
	Name              string                        `json:"name"`
	Namespace         string                        `json:"namespace"`
	Labels            map[string]string             `json:"labels"`
	Image             string                        `json:"image"`
	Command           []string                      `json:"command"`
	Args              []string                      `json:"args"`
	PodPorts          []corev1.ContainerPort        `json:"ports"`
	VolumeMounts      []corev1.VolumeMount          `json:"volumeMounts"`
	Volumes           []corev1.Volume               `json:"volume"`
	Storage           string                        `json:"storage"` // samples: 5Gi
	ServicePorts      []corev1.ServicePort          `json:"servicePorts"`
	ContainerResource corev1.ResourceRequirements   `json:"containeResource"`
	InstanceType      string                        `json:"instanceType"` // only fe or be
	StorageClass      *string                       `json:"storageClass,omitempty"`
	ImagePullSecrets  []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
}
