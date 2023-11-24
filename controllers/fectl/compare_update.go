package fectl

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

func (r *FeReconciler) compareUpdate(ctx context.Context, fe *dorisv1alpha1.Fe) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)
	i := 0

	// Fetch current fe statefulset
	sts := &appsv1.StatefulSet{}
	err := r.Get(ctx, types.NamespacedName{Name: fe.Name, Namespace: fe.Namespace}, sts)
	if err != nil {
		log.Error(err, err.Error())
	}

	if fe.Spec.Image != sts.Spec.Template.Spec.Containers[0].Image {
		sts.Spec.Template.Spec.Containers[0].Image = fe.Spec.Image
		i++
	}

	if !reflect.DeepEqual(fe.Spec.Command, sts.Spec.Template.Spec.Containers[0].Command) {
		sts.Spec.Template.Spec.Containers[0].Command = fe.Spec.Command
		i++
	}

	if !reflect.DeepEqual(fe.Spec.ImagePullSecrets, sts.Spec.Template.Spec.ImagePullSecrets) {
		sts.Spec.Template.Spec.ImagePullSecrets = fe.Spec.ImagePullSecrets
		i++
	}

	if !reflect.DeepEqual(fe.Spec.Resources, sts.Spec.Template.Spec.Containers[0].Resources) {
		sts.Spec.Template.Spec.Containers[0].Resources = fe.Spec.Resources
		i++
	}

	if !reflect.DeepEqual(fe.Spec.Args, sts.Spec.Template.Spec.Containers[0].Args) {
		sts.Spec.Template.Spec.Containers[0].Args = fe.Spec.Args
		i++
	}

	if i != 0 {
		err = r.Update(ctx, sts)
		if err != nil {
			log.Error(err, "Failed to update Doris-fe statefulset", "Fe.statefulset.Namespace", sts.Namespace, "Fe.statefulset.Name", sts.Name)
			return ctrl.Result{}, err
		}
	}
	// Ask to requeue after 1 minute in order to give enough time for the
	// pods be created on the cluster side and the operand be able
	// to do the next update step accurately.
	return ctrl.Result{RequeueAfter: time.Minute}, nil
}
