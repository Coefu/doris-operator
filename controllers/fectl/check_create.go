package fectl

import (
	"context"
	dorisv1alpha1 "doris-operator/api/v1alpha1"
	"doris-operator/pkg/cluster"

	ctrl "sigs.k8s.io/controller-runtime"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *FeReconciler) check(ctx context.Context, fe *dorisv1alpha1.Fe) []string {
	isNotFound := []string{}

	sts := &appsv1.StatefulSet{}
	err := r.Get(ctx, types.NamespacedName{Name: fe.Name, Namespace: fe.Namespace}, sts)
	if err != nil && errors.IsNotFound(err) {
		isNotFound = append(isNotFound, "statefulset")
	}

	svc := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: fe.Name, Namespace: fe.Namespace}, svc)
	if err != nil && errors.IsNotFound(err) {
		isNotFound = append(isNotFound, "headless_service")
	}
	err = r.Get(ctx, types.NamespacedName{Name: fe.Name + "-nodeport", Namespace: fe.Namespace}, svc)
	if err != nil && errors.IsNotFound(err) {
		isNotFound = append(isNotFound, "nodeport_service")
	}

	cm := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{Name: fe.Name + "-config", Namespace: fe.Namespace}, cm)
	if err != nil && errors.IsNotFound(err) {
		isNotFound = append(isNotFound, "configmap")
	}

	if fe.Spec.Domain != "" {
		ingress := &networking.Ingress{}
		err = r.Get(ctx, types.NamespacedName{Name: fe.Name, Namespace: fe.Namespace}, ingress)
		if err != nil && errors.IsNotFound(err) {
			isNotFound = append(isNotFound, "ingress")
		}
	}
	return isNotFound
}

func (r *FeReconciler) createDepResource(ctx context.Context, fe *dorisv1alpha1.Fe, isnotfound []string) error {
	log := ctrllog.FromContext(ctx)
	feCustomParam := cluster.Params{}

	feCustomParam.Name = fe.Name
	feCustomParam.Namespace = fe.Namespace
	feCustomParam.Labels = map[string]string{"app": fe.Name}

	for _, v := range isnotfound {
		if v == "statefulset" {
			// convert params: cluster feconfig -> fe custom param -> fe statefulset
			feCustomParam.StorageClass = fe.Spec.StorageClass
			feCustomParam.ImagePullSecrets = fe.Spec.ImagePullSecrets
			feCustomParam.InstanceIP = fe.Spec.InstanceIP
			feCustomParam.Image = fe.Spec.Image
			feCustomParam.Command = fe.Spec.Command
			feCustomParam.Args = fe.Spec.Args
			feCustomParam.PodPorts = containerPortForFe()
			feCustomParam.VolumeMounts = volumeMountsForSts(fe.Name)
			feCustomParam.Volumes = volumesForSts(fe.Name)
			feCustomParam.Storage = fe.Spec.Storage
			feCustomParam.ContainerResource = fe.Spec.Resources

			// Define a new statefulset
			sts := cluster.MakeStatefulSet(&feCustomParam)
			// Set fe instance as the owner and controller
			err := ctrl.SetControllerReference(fe, sts, r.Scheme)
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
			feCustomParam.ServicePorts = portForHeadlessService()
			headless_svc := cluster.MakeService(&feCustomParam, "headless")
			// Set fe instance as the owner and controller
			err := ctrl.SetControllerReference(fe, headless_svc, r.Scheme)
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
		if v == "nodeport_service" {
			feCustomParam.ServicePorts = portForNodePortService()
			nodePort_svc := cluster.MakeService(&feCustomParam, "nodeport")
			// Set fe instance as the owner and controller
			err := ctrl.SetControllerReference(fe, nodePort_svc, r.Scheme)
			if err != nil {
				log.Error(err, err.Error())
			}
			log.Info("Creating a new nodePort service", "service.Namespace", nodePort_svc.Namespace, "service.Name", nodePort_svc.Name)
			err = r.Create(ctx, nodePort_svc)
			if err != nil {
				log.Error(err, "Failed to create new nodePort service", "service.Namespace", nodePort_svc.Namespace, "service.Name", nodePort_svc.Name)
				return err
			}
		}
		if v == "configmap" {
			feCustomParam.InstanceType = "fe"
			cm := cluster.MakeConfigMap(&feCustomParam)
			// Set fe instance as the owner and controller
			err := ctrl.SetControllerReference(fe, cm, r.Scheme)
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
		if v == "ingress" {
			// TODO: add ingress create.
		}
	}
	return nil
}

func containerPortForFe() []corev1.ContainerPort {
	return []corev1.ContainerPort{
		{
			Name:          "http",
			ContainerPort: 8030,
		},
		{
			Name:          "rpc",
			ContainerPort: 9020,
		},
		{
			Name:          "query",
			ContainerPort: 9030,
		},
		{
			Name:          "edit-log",
			ContainerPort: 9010,
		},
	}
}

func volumeMountsForSts(feName string) []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      feName + "-pvc",
			MountPath: "/root/doris-meta",
		},
		{
			Name:      feName + "-config",
			MountPath: "/root/conf/fe.conf",
			SubPath:   "fe.conf",
		},
	}
}

func volumesForSts(feName string) []corev1.Volume {
	return []corev1.Volume{
		{
			Name: feName + "-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: feName + "-config",
					},
				},
			},
		},
	}
}

func portForHeadlessService() []corev1.ServicePort {
	return []corev1.ServicePort{
		{
			Name: "http",
			Port: 8030,
		},
		{
			Name: "rpc",
			Port: 9020,
		},
		{
			Name: "query",
			Port: 9030,
		},
		{
			Name: "edit-log",
			Port: 9010,
		},
	}
}

func portForNodePortService() []corev1.ServicePort {
	return []corev1.ServicePort{
		{
			Name: "http",
			Port: 8030,
		},
		{
			Name: "rpc",
			Port: 9020,
		},
		{
			Name: "query",
			Port: 9030,
			//NodePort: 30030,
		},
		{
			Name: "edit-log",
			Port: 9010,
		},
	}
}
