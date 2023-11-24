package clusterctl

import (
	"context"
	dorisv1alpha1 "doris-operator/api/v1alpha1"
	"fmt"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *ClusterReconciler) checkFe(ctx context.Context, cluster *dorisv1alpha1.Cluster) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)
	fe := &dorisv1alpha1.Fe{}
	if *cluster.Spec.Fe.Replicas == int32(1) {
		err := checkFe1(ctx, log, r, cluster, fe)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	if *cluster.Spec.Fe.Replicas > int32(1) {
		//TODO: fe replicas > 1,like 3,5,7,9
		err := checkFe1(ctx, log, r, cluster, fe)
		if err != nil {
			return ctrl.Result{}, err
		}
		fe1Svc := fe.Name + "." + fe.Namespace
		for i := int32(2); i < *cluster.Spec.Fe.Replicas+int32(1); i++ {
			feName := "doris-fe" + strconv.Itoa(int(i))
			err = r.Get(ctx, types.NamespacedName{Name: feName, Namespace: cluster.Namespace}, fe)
			if err != nil {
				if errors.IsNotFound(err) {
					r.addFe(ctx, fe1Svc, fe.Name+"."+fe.Namespace)
					isMasterFeIP := isMasterFeip(ctx, fe1Svc)
					cmdArgs := []string{"--helper", isMasterFeIP + ":9010"}
					err = createFe(ctx, log, r, cluster, feName, cmdArgs)
					if err != nil {
						return ctrl.Result{}, err
					}
				}
			}
		}
		go updateFeCmdArgs(ctx, log, r, cluster, fe1Svc, clusterIPs, fe)
	}
	// Fe created successfully - return and requeue
	return ctrl.Result{Requeue: true, RequeueAfter: time.Minute * 1}, nil
}

func isMasterFeip(ctx context.Context, feLeaderip string) string {
	frontendsinfo := showFrontends(ctx, feLeaderip)
	for _, v := range frontendsinfo {
		if v["IsMaster"] == "true" {
			return v["IP"].(string)
		}
	}
	return ""
}

func checkFe1(ctx context.Context, log logr.Logger, r *ClusterReconciler, cluster *dorisv1alpha1.Cluster, fe *dorisv1alpha1.Fe) error {
	feName := "doris-fe1"
	err := r.Get(ctx, types.NamespacedName{Name: feName, Namespace: cluster.Namespace}, fe)
	if err != nil {
		if errors.IsNotFound(err) {
			cmdArgs := []string{}
			err = createFe(ctx, log, r, cluster, feName, cmdArgs)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func createFe(ctx context.Context, log logr.Logger, r *ClusterReconciler, cluster *dorisv1alpha1.Cluster, feName string, cmdArgs []string) error {
	newFe := r.feForCluster(cluster, feName, cmdArgs)
	// Set cluster instance as the owner and controller
	err := ctrl.SetControllerReference(cluster, newFe, r.Scheme)
	if err != nil {
		log.Error(err, err.Error())
	}
	log.Info("Creating a new Fe", "Fe.Namespaces", newFe.Namespace, "Fe.Name", newFe.Name)
	r.Recorder.Event(cluster, corev1.EventTypeNormal, "Add fe", fmt.Sprintf("name is %s", newFe.Name))
	err = r.Create(ctx, newFe)
	if err != nil {
		log.Error(err, "Failed to create new Fe", "Fe.Namespace", newFe.Namespace, "Fe.Name", newFe.Name)
		r.Recorder.Event(cluster, corev1.EventTypeWarning, "Add fe failed", fmt.Sprintf("name is %s,err is %s", newFe.Name, err.Error()))
		return err
	}
	return nil
}

func updateFeCmdArgs(ctx context.Context, log logr.Logger, r *ClusterReconciler, cluster *dorisv1alpha1.Cluster, feLeaderip string, fe *dorisv1alpha1.Fe) {
	// if all fe is alive, 10 minutes Max
	for k := 0; k < 600; k++ {
		i := 0
		fesinfo := showFrontends(ctx, feLeaderip)
		for _, v := range fesinfo {
			if v["Alive"] != "true" {
				break
			}
			i++
		}
		if i == len(fesinfo) {
			break
		}
		time.Sleep(time.Second)
	}
	frontendsinfo := showFrontends(ctx, feLeaderip)
	for _, v := range frontendsinfo {
		if v["IsMaster"] == "false" {
			for j, k := range clusterIPs["feIPs"] {
				if v["IP"] == k {
					err := r.Get(ctx, types.NamespacedName{Name: j, Namespace: cluster.Namespace}, fe)
					if err != nil {
						log.Error(err, err.Error())
					}
					if len(fe.Spec.Args) != 0 {
						fe.Spec.Args = []string{}
						err = r.Update(ctx, fe)
						if err != nil {
							log.Error(err, "Failed to update Doris-fe", "Fe.Namespace", fe.Namespace, "Fe.Name", fe.Name)
						}
					}
				}
			}
		}
	}
}

func (r *ClusterReconciler) checkBe(ctx context.Context, cluster *dorisv1alpha1.Cluster) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)
	be := &dorisv1alpha1.Be{}
	for i := int32(1); i < *cluster.Spec.Be.Replicas+int32(1); i++ {
		beName := "doris-be" + strconv.Itoa(int(i))
		err := r.Get(ctx, types.NamespacedName{Name: beName, Namespace: cluster.Namespace}, be)
		if err != nil {
			if errors.IsNotFound(err) {
				newBe := r.beForCluster(cluster, beName)
				// Set cluster instance as the owner and controller
				err = ctrl.SetControllerReference(cluster, newBe, r.Scheme)
				if err != nil {
					log.Error(err, err.Error())
				}
				log.Info("Creating a new Be", "Be.Namespaces", newBe.Namespace, "Be.Name", newBe.Name)
				r.Recorder.Event(cluster, corev1.EventTypeNormal, "Add be", fmt.Sprintf("name is %s", newBe.Name))
				err = r.Create(ctx, newBe)
				if err != nil {
					r.Recorder.Event(cluster, corev1.EventTypeWarning, "Add be failed", fmt.Sprintf("name is %s,err is %s", newBe.Name, err.Error()))
					log.Error(err, "Failed to create new Be", "Be.Namespace", newBe.Namespace, "Be.Name", newBe.Name)
					return ctrl.Result{}, err
				}
				// fe registry be
				go r.feRegistryBe(ctx, feLeaderip, beip, cluster, newBe)
			}
		}
	}
	// Be created successfully - return and requeue
	return ctrl.Result{Requeue: true, RequeueAfter: time.Minute * 1}, nil
}

func (r *ClusterReconciler) feForCluster(c *dorisv1alpha1.Cluster, name string, cmdArgs []string) *dorisv1alpha1.Fe {
	return &dorisv1alpha1.Fe{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.Namespace,
		},
		Spec: dorisv1alpha1.FeSpec{
			Image:            c.Spec.Fe.Image,
			Command:          c.Spec.Fe.Command,
			Args:             cmdArgs,
			Domain:           c.Spec.Fe.Domain,
			Storage:          c.Spec.Fe.Storage,
			Resources:        c.Spec.Fe.Resources,
			ImagePullSecrets: c.Spec.ImagePullSecrets,
			StorageClass:     c.Spec.StorageClass,
			//InstanceIP:       ip,
		},
	}
}

func (r *ClusterReconciler) beForCluster(c *dorisv1alpha1.Cluster, name string) *dorisv1alpha1.Be {
	return &dorisv1alpha1.Be{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.Namespace,
		},
		Spec: dorisv1alpha1.BeSpec{
			Image:            c.Spec.Be.Image,
			Command:          c.Spec.Be.Command,
			Autoscaling:      c.Spec.Be.Autoscaling,
			Storage:          c.Spec.Be.Storage,
			Resources:        c.Spec.Be.Resources,
			ImagePullSecrets: c.Spec.ImagePullSecrets,
			StorageClass:     c.Spec.StorageClass,
			//InstanceIP:       ip,
		},
	}
}
