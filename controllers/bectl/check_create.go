package bectl

import (
	"context"
	dorisv1alpha1 "doris-operator/api/v1alpha1"
	"doris-operator/pkg/native_resources"

	ctrl "sigs.k8s.io/controller-runtime"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *BeReconciler) check(ctx context.Context, be *dorisv1alpha1.Be) []string {
	isNotFound := []string{}

	sts := &appsv1.StatefulSet{}
	err := r.Get(ctx, types.NamespacedName{Name: be.Name, Namespace: be.Namespace}, sts)
	if err != nil && errors.IsNotFound(err) {
		isNotFound = append(isNotFound, "statefulset")
	}

	svc := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: be.Name, Namespace: be.Namespace}, svc)
	if err != nil && errors.IsNotFound(err) {
		isNotFound = append(isNotFound, "headless_service")
	}

	cm := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{Name: be.Name + "-config", Namespace: be.Namespace}, cm)
	if err != nil && errors.IsNotFound(err) {
		isNotFound = append(isNotFound, "configmap")
	}

	return isNotFound
}

func (r *BeReconciler) createDepResource(ctx context.Context, be *dorisv1alpha1.Be, isnotfound []string) error {
	log := ctrllog.FromContext(ctx)
	beCustomParam := native_resources.Params{}

	beCustomParam.Name = be.Name
	beCustomParam.Namespace = be.Namespace
	beCustomParam.Labels = map[string]string{"app": be.Name}
	beCustomParam.ServicePorts = portForService()

	for _, v := range isnotfound {
		if v == "statefulset" {
			// convert params: cluster beconfig -> be custom param -> be statefulset
			beCustomParam.StorageClass = be.Spec.StorageClass
			beCustomParam.ImagePullSecrets = be.Spec.ImagePullSecrets
			//beCustomParam.InstanceIP = be.Spec.InstanceIP
			beCustomParam.Image = be.Spec.Image
			beCustomParam.Command = be.Spec.Command
			beCustomParam.PodPorts = containerPortForBe()
			beCustomParam.VolumeMounts = volumeMountsForSts(be.Name)
			beCustomParam.Volumes = volumesForSts(be.Name)
			beCustomParam.Storage = be.Spec.Storage
			beCustomParam.ContainerResource = be.Spec.Resources

			// Define a new statefulset
			sts := native_resources.MakeStatefulSet(&beCustomParam)
			// Set be instance as the owner and controller
			err := ctrl.SetControllerReference(be, sts, r.Scheme)
			if err != nil {
				log.Error(err, err.Error())
			}
			log.Info("Creating a new StatefulSet", "StatefulSet.Namespace", sts.Namespace, "StatefulSet.Name", sts.Name)
			err = r.Create(ctx, sts)
			if err != nil {
				log.Error(err, "Failed to create new StatefulSet", "StatefulSet.Namespace", sts.Namespace, "StatefulSet.Name", sts.Name)
				return err
			}
		}
		if v == "headless_service" {
			headless_svc := native_resources.MakeService(&beCustomParam, "headless")
			// Set be instance as the owner and controller
			err := ctrl.SetControllerReference(be, headless_svc, r.Scheme)
			if err != nil {
				log.Error(err, err.Error())
			}
			log.Info("Creating a new headless service", "service.Namespace", headless_svc.Namespace, "service.Name", headless_svc.Name)
			err = r.Create(ctx, headless_svc)
			if err != nil {
				log.Error(err, "Failed to create new headless service", "service.Namespace", headless_svc.Namespace, "service.Name", headless_svc.Name)
				return err
			}
		}
		if v == "configmap" {
			beCustomParam.InstanceType = "be"
			cm := native_resources.MakeConfigMap(&beCustomParam)
			// Set be instance as the owner and controller
			err := ctrl.SetControllerReference(be, cm, r.Scheme)
			if err != nil {
				log.Error(err, err.Error())
			}
			log.Info("Creating a new configmap", "configmap.Namespace", cm.Namespace, "configmap.Name", cm.Name)
			err = r.Create(ctx, cm)
			if err != nil {
				log.Error(err, "Failed to create new configmap", "configmap.Namespace", cm.Namespace, "configmap.Name", cm.Name)
				return err
			}
		}
	}
	return nil
}

func containerPortForBe() []corev1.ContainerPort {
	return []corev1.ContainerPort{
		{
			Name:          "be",
			ContainerPort: 9060,
		},
		{
			Name:          "webserver",
			ContainerPort: 8040,
		},
		{
			Name:          "heartbeat",
			ContainerPort: 9050,
		},
		{
			Name:          "brpc",
			ContainerPort: 8060,
		},
	}
}

func volumeMountsForSts(beName string) []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      beName + "-pvc",
			MountPath: "/root/storage/doris",
		},
		{
			Name:      beName + "-config",
			MountPath: "/root/conf/be.conf",
			SubPath:   "be.conf",
		},
	}
}

func volumesForSts(beName string) []corev1.Volume {
	return []corev1.Volume{
		{
			Name: beName + "-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: beName + "-config",
					},
				},
			},
		},
	}
}

func portForService() []corev1.ServicePort {
	return []corev1.ServicePort{
		{
			Name: "be",
			Port: 9060,
		},
		{
			Name: "webserver",
			Port: 8040,
		},
		{
			Name: "heartbeat-service",
			Port: 9050,
		},
		{
			Name: "brpc",
			Port: 8060,
		},
	}
}
