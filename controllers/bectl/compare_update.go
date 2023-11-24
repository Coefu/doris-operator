package bectl

import (
	"context"
	dorisv1alpha1 "doris-operator/api/v1alpha1"
	"reflect"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"

	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *BeReconciler) compareUpdate(ctx context.Context, be *dorisv1alpha1.Be) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)
	i := 0

	// Fetch current fe statefulset
	sts := &appsv1.StatefulSet{}
	err := r.Get(ctx, types.NamespacedName{Name: be.Name, Namespace: be.Namespace}, sts)
	if err != nil {
		log.Error(err, err.Error())
	}

	if be.Spec.Image != sts.Spec.Template.Spec.Containers[0].Image {
		sts.Spec.Template.Spec.Containers[0].Image = be.Spec.Image
		i++
	}

	if !reflect.DeepEqual(be.Spec.Command, sts.Spec.Template.Spec.Containers[0].Command) {
		sts.Spec.Template.Spec.Containers[0].Command = be.Spec.Command
		i++
	}

	if !reflect.DeepEqual(be.Spec.ImagePullSecrets, sts.Spec.Template.Spec.ImagePullSecrets) {
		sts.Spec.Template.Spec.ImagePullSecrets = be.Spec.ImagePullSecrets
		i++
	}

	if !reflect.DeepEqual(be.Spec.Resources, sts.Spec.Template.Spec.Containers[0].Resources) {
		sts.Spec.Template.Spec.Containers[0].Resources = be.Spec.Resources
		i++
	}

	if i != 0 {
		err = r.Update(ctx, sts)
		if err != nil {
			log.Error(err, "Failed to update Doris-be statefulset", "Be.statefulset.Namespace", sts.Namespace, "Be.statefulset.Name", sts.Name)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{RequeueAfter: time.Minute}, nil
}
