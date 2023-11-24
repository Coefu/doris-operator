package clusterctl

import (
	"context"
	dorisv1alpha1 "doris-operator/api/v1alpha1"
	"fmt"
	"math"
	"reflect"
	"sort"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *ClusterReconciler) compareUpdate(ctx context.Context, cluster *dorisv1alpha1.Cluster) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	err := compareUpdateFe(r, ctx, log, cluster)
	if err != nil {
		return ctrl.Result{}, err
	}
	err = compareUpdateBe(r, ctx, log, cluster)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = updateFesStatus(r, ctx, log, cluster)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = updateBesStatus(r, ctx, log, cluster)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = updateClusterStatus(r, ctx, log, cluster)
	if err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func updateFesStatus(r *ClusterReconciler, ctx context.Context, log logr.Logger, cluster *dorisv1alpha1.Cluster) error {
	feList := &dorisv1alpha1.FeList{}
	err := r.List(ctx, feList, &client.ListOptions{Namespace: cluster.Namespace})
	if err != nil {
		log.Error(err, err.Error())
	}
	fe := &dorisv1alpha1.Fe{}
	for _, v := range feList.Items {
		err = r.Get(ctx, types.NamespacedName{Name: v.Name, Namespace: v.Namespace}, fe)
		if err != nil {
			log.Error(err, "Failed to get Fe", "Fe.Namespace", v.Namespace, "Fe.Name", v.Name)
		}
		fe.Status.Cluster = cluster.Name
		err = r.Status().Update(ctx, fe)
		if err != nil {
			log.Error(err, "Failed to update Fe status")
			return err
		}
	}
	return nil
}

func updateBesStatus(r *ClusterReconciler, ctx context.Context, log logr.Logger, cluster *dorisv1alpha1.Cluster) error {
	beList := &dorisv1alpha1.BeList{}
	err := r.List(ctx, beList, &client.ListOptions{Namespace: cluster.Namespace})
	if err != nil {
		log.Error(err, err.Error())
	}
	be := &dorisv1alpha1.Be{}
	for _, v := range beList.Items {
		err = r.Get(ctx, types.NamespacedName{Name: v.Name, Namespace: v.Namespace}, be)
		if err != nil {
			log.Error(err, "Failed to get Fe", "Fe.Namespace", v.Namespace, "Fe.Name", v.Name)
		}
		be.Status.Cluster = cluster.Name
		err = r.Status().Update(ctx, be)
		if err != nil {
			log.Error(err, "Failed to update Fe status")
			return err
		}
	}
	return nil
}

func updateClusterStatus(r *ClusterReconciler, ctx context.Context, log logr.Logger, cluster *dorisv1alpha1.Cluster) error {
	feList := &dorisv1alpha1.FeList{}
	err := r.List(ctx, feList, &client.ListOptions{Namespace: cluster.Namespace})
	if err != nil {
		log.Error(err, err.Error())
	}
	cluster.Status.Fe_replicas = int32(len(feList.Items))

	beList := &dorisv1alpha1.BeList{}
	err = r.List(ctx, beList, &client.ListOptions{Namespace: cluster.Namespace})
	if err != nil {
		log.Error(err, err.Error())
	}

	cluster.Status.Be_replicas = int32(len(beList.Items))
	err = r.Status().Update(ctx, cluster)
	if err != nil {
		log.Error(err, "Failed to update Cluster status")
		return err
	}
	return nil
}

func compareUpdateFe(r *ClusterReconciler, ctx context.Context, log logr.Logger, cluster *dorisv1alpha1.Cluster) error {
	feList := &dorisv1alpha1.FeList{}
	err := r.List(ctx, feList, &client.ListOptions{Namespace: cluster.Namespace})
	if err != nil {
		log.Error(err, err.Error())
	}
	fediff := *cluster.Spec.Fe.Replicas - int32(len(feList.Items))
	if fediff == 0 {
		// update current fe
		err = updateFes(r, ctx, feList, cluster)
		if err != nil {
			log.Error(err, err.Error())
			return err
		}
	}

	if fediff < 0 {
		feLeaderip := "192.168.168.1"

		currentFeips := []string{}
		for _, v := range feList.Items {
			currentFeips = append(currentFeips, v.Spec.InstanceIP)
		}
		sort.Sort(sort.Reverse(sort.StringSlice(currentFeips)))
		// Delete from the end
		needDelFeips := currentFeips[:int32(math.Abs(float64(fediff)))]
		for _, v := range needDelFeips {
			r.deleteFe(ctx, feLeaderip, v)
			// Verify the delete, if v not in fes, del fe resource.
			for {
				if func() bool {
					for _, f := range showFrontends(ctx, feLeaderip) {
						if v == f["IP"] {
							return true
						}
					}
					return false
				}() == false {
					break
				}
			}
			fe := &dorisv1alpha1.Fe{}
			for _, f := range feList.Items {
				if f.Spec.InstanceIP == v {
					err = r.Get(ctx, types.NamespacedName{Name: f.Name, Namespace: f.Namespace}, fe)
					if err != nil {
						log.Error(err, "Failed to get Fe", "Fe.Namespace", f.Namespace, "Fe.Name", f.Name)
					}
					log.Info("Delete Fe", "Fe.namespace", f.Namespace, "Fe.name", f.Name)
					r.Recorder.Event(cluster, corev1.EventTypeWarning, "Delete fe ", fmt.Sprintf("name is %s", f.Name))
					err = r.Delete(ctx, fe)
					if err != nil {
						r.Recorder.Event(cluster, corev1.EventTypeWarning, "Delete fe failed", fmt.Sprintf("name is %s,err is %s", f.Name, err.Error()))
						log.Error(err, "Failed to delete Fe", "Fe.Namespace", f.Namespace, "Fe.Name", f.Name)
						return err
					}
				}
			}
		}
		// update the rest of be
		restOfFeList := &dorisv1alpha1.FeList{}
		err = r.List(ctx, restOfFeList, &client.ListOptions{Namespace: cluster.Namespace})
		if err != nil {
			log.Error(err, err.Error())
		}
		err = updateFes(r, ctx, restOfFeList, cluster)
		if err != nil {
			log.Error(err, err.Error())
			return err
		}
	}
	return nil
}

func compareUpdateBe(r *ClusterReconciler, ctx context.Context, log logr.Logger, cluster *dorisv1alpha1.Cluster) error {
	beList := &dorisv1alpha1.BeList{}
	err := r.List(ctx, beList, &client.ListOptions{Namespace: cluster.Namespace})
	if err != nil {
		log.Error(err, err.Error())
	}

	bediff := *cluster.Spec.Be.Replicas - int32(len(beList.Items))

	if bediff == 0 {
		// update current be
		err = updateBes(r, ctx, beList, cluster)
		if err != nil {
			log.Error(err, err.Error())
			return err
		}
	}

	if bediff < 0 {
		feLeaderip := "192.168.168.1"

		currentBeips := []string{}
		for _, v := range beList.Items {
			currentBeips = append(currentBeips, v.Spec.InstanceIP)
		}
		sort.Sort(sort.Reverse(sort.StringSlice(currentBeips)))
		// Delete from the end
		needDelBeips := currentBeips[:int32(math.Abs(float64(bediff)))]
		for _, v := range needDelBeips {
			r.deleteBe(ctx, feLeaderip, v)

			var tempTabletNum string
			var i = int32(0)
			// The isDecommission status of the be object is true. Indicates that the node is being offline.
		delBe:
			for {
				backendsinfo := showBackends(ctx, feLeaderip)

				var tempBeips []interface{}
				for _, b := range backendsinfo {
					tempBeips = append(tempBeips, b["IP"])
				}

				if IsContain(v, tempBeips) {
					for _, b := range backendsinfo {
						if b["IP"] == v {
							if b["SystemDecommissioned"] == "true" {
								i++
								if tempTabletNum == "" {
									tempTabletNum = b["TabletNum"].(string)
								} else {
									if b["TabletNum"].(string) < tempTabletNum {
										tempTabletNum = b["TabletNum"].(string)
										time.Sleep(time.Second * 1)
										break
									}
									if b["TabletNum"].(string) == tempTabletNum {
										time.Sleep(time.Second * 1)
										/***
										The order does not necessarily carry out successfully. For example, when the
										remaining BE storage space is insufficient to accommodate the data on the offline
										BE, or when the number of remaining machines does not meet the minimum number of
										replicas, the command cannot be completed, and the BE will always be in the state
										of isDecommission as true.
										*/
										if i == int32(60) {
											break delBe
										}
										break
									}
								}
							}
						}
					}
				} else {
					be := &dorisv1alpha1.Be{}
					for _, b := range beList.Items {
						if v == b.Spec.InstanceIP {
							err = r.Get(ctx, types.NamespacedName{Name: b.Name, Namespace: b.Namespace}, be)
							if err != nil {
								log.Error(err, "Failed to get Be", "Be.Namespace", b.Namespace, "Be.Name", b.Name)
							}
							log.Info("Delete be", "Be.namespace", b.Namespace, "Be.name", b.Name)
							r.Recorder.Event(cluster, corev1.EventTypeWarning, "Delete be ", fmt.Sprintf("name is %s", b.Name))
							err = r.Delete(ctx, be)
							if err != nil {
								r.Recorder.Event(cluster, corev1.EventTypeWarning, "Delete be failed", fmt.Sprintf("name is %s,err is %s", b.Name, err.Error()))
								log.Error(err, "Failed to delete Be", "Be.Namespace", b.Namespace, "Be.Name", b.Name)
								return err
							}
							break delBe
						}
					}
				}
			}
		}

		// update the rest of be
		restOfBeList := &dorisv1alpha1.BeList{}
		err = r.List(ctx, restOfBeList, &client.ListOptions{Namespace: cluster.Namespace})
		if err != nil {
			log.Error(err, err.Error())
		}
		err = updateBes(r, ctx, restOfBeList, cluster)
		if err != nil {
			log.Error(err, err.Error())
			return err
		}
	}
	return nil
}

func IsContain(str interface{}, array []interface{}) bool {
	for _, v := range array {
		if v == str {
			return true
		}
	}
	return false
}

func updateFes(r *ClusterReconciler, ctx context.Context, feList *dorisv1alpha1.FeList, cluster *dorisv1alpha1.Cluster) error {
	// update current be
	pod := &corev1.Pod{}
	for _, v := range feList.Items {
		err := r.compareUpdateFe(ctx, cluster, v.Name)
		if err != nil {
			return err
		}
		// rolling update
		for {
			err = r.Get(ctx, types.NamespacedName{Name: v.Name + "-0", Namespace: cluster.Namespace}, pod)
			if err != nil {
				if errors.IsNotFound(err) {
					continue
				} else {
					return err
				}
			} else {
				if pod.Status.Phase == "Running" {
					break
				}
			}
		}
	}
	return nil
}

func updateBes(r *ClusterReconciler, ctx context.Context, beList *dorisv1alpha1.BeList, cluster *dorisv1alpha1.Cluster) error {
	// update current be
	pod := &corev1.Pod{}
	for _, v := range beList.Items {
		err := r.compareUpdateBe(ctx, cluster, v.Name)
		if err != nil {
			return err
		}
		// rolling update
		for {
			err = r.Get(ctx, types.NamespacedName{Name: v.Name + "-0", Namespace: cluster.Namespace}, pod)
			if err != nil {
				return err
			}
			if pod.Status.Phase == "Running" {
				break
			}
		}
	}
	return nil
}

func (r *ClusterReconciler) compareUpdateFe(ctx context.Context, cluster *dorisv1alpha1.Cluster, feName string) error {
	log := ctrllog.FromContext(ctx)
	i := 0

	fe := &dorisv1alpha1.Fe{}

	err := r.Get(ctx, types.NamespacedName{Name: feName, Namespace: cluster.Namespace}, fe)
	if err != nil {
		log.Error(err, err.Error())
	}

	if cluster.Spec.Fe.Image != fe.Spec.Image {
		fe.Spec.Image = cluster.Spec.Fe.Image
		i++
	}

	if cluster.Spec.Fe.Domain != fe.Spec.Domain {
		fe.Spec.Domain = cluster.Spec.Fe.Domain
		i++
	}

	if !reflect.DeepEqual(cluster.Spec.Fe.Command, fe.Spec.Command) {
		fe.Spec.Command = cluster.Spec.Fe.Command
		i++
	}

	if !reflect.DeepEqual(cluster.Spec.ImagePullSecrets, fe.Spec.ImagePullSecrets) {
		fe.Spec.ImagePullSecrets = cluster.Spec.ImagePullSecrets
		i++
	}

	if !reflect.DeepEqual(cluster.Spec.Fe.Resources, fe.Spec.Resources) {
		fe.Spec.Resources = cluster.Spec.Fe.Resources
		i++
	}

	// update only when it changes
	if i != 0 {
		log.Info("Update doris-fe", "fe.name", fe.Name)
		err = r.Update(ctx, fe)
		if err != nil {
			log.Error(err, "Failed to update Doris-fe", "Fe.Namespace", fe.Namespace, "Fe.Name", fe.Name)
			return err
		}
	}
	return nil
}

func (r *ClusterReconciler) compareUpdateBe(ctx context.Context, cluster *dorisv1alpha1.Cluster, beName string) error {
	log := ctrllog.FromContext(ctx)
	i := 0

	be := &dorisv1alpha1.Be{}

	err := r.Get(ctx, types.NamespacedName{Name: beName, Namespace: cluster.Namespace}, be)
	if err != nil {
		log.Error(err, err.Error())
	}

	if cluster.Spec.Be.Image != be.Spec.Image {
		be.Spec.Image = cluster.Spec.Be.Image
		i++
	}

	if !reflect.DeepEqual(cluster.Spec.Be.Command, be.Spec.Command) {
		be.Spec.Command = cluster.Spec.Be.Command
		i++
	}

	if !reflect.DeepEqual(cluster.Spec.ImagePullSecrets, be.Spec.ImagePullSecrets) {
		be.Spec.ImagePullSecrets = cluster.Spec.ImagePullSecrets
		i++
	}

	if !reflect.DeepEqual(cluster.Spec.Be.Resources, be.Spec.Resources) {
		be.Spec.Resources.Requests = cluster.Spec.Be.Resources.Requests
		i++
	}

	// update only when it changes
	if i != 0 {
		log.Info("update...", "", be.Name)
		err = r.Update(ctx, be)
		if err != nil {
			log.Error(err, "Failed to update Doris-be", "Be.Namespace", be.Namespace, "Be.Name", be.Name)
			return err
		}
	}
	return nil
}
