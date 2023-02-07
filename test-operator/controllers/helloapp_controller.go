/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	appsv1 "github.com/sourcecd/operators/test-operator/api/v1"
	a "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// HelloAppReconciler reconciles a HelloApp object
type HelloAppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=apps.sourcecd.com,resources=helloapps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.sourcecd.com,resources=helloapps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps.sourcecd.com,resources=helloapps/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the HelloApp object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *HelloAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	helloApp := &appsv1.HelloApp{}
	err := r.Client.Get(ctx, req.NamespacedName, helloApp)
	if err != nil {
		if errors2.IsNotFound(err) {
			log.Log.Info("HelloApp resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Log.Error(err, "Failed to get HelloApp")
		return ctrl.Result{}, err
	}
	dpl := &a.Deployment{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: helloApp.Name, Namespace: helloApp.Namespace}, dpl)
	if err != nil && errors2.IsNotFound(err) {
		dep := r.deployHelloApp(helloApp)
		log.Log.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.Client.Create(ctx, dep)
		if err != nil {
			log.Log.Error(err, "Failed to create new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Log.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}

	size := helloApp.Spec.Size
	if *dpl.Spec.Replicas != size {
		dpl.Spec.Replicas = &size
		err = r.Client.Update(ctx, dpl)
		if err != nil {
			log.Log.Error(err, "Failed to update Deployment", "Deployment.Namespace", dpl.Namespace, "Deployment.Name", dpl.Name)
			return ctrl.Result{}, err
		}
		// Spec updated - return and requeue
		return ctrl.Result{Requeue: true}, nil
	}
	return ctrl.Result{}, nil
}

func (r *HelloAppReconciler) deployHelloApp(ha *appsv1.HelloApp) *a.Deployment {
	replicas := ha.Spec.Size
	labels := map[string]string{"app": "mock-containers"}
	image := ha.Spec.Image
	dep := &a.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ha.Name,
			Namespace: ha.Namespace,
		},
		Spec: a.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: image,
						Name:  ha.Name,
					}},
				},
			},
		},
	}
	_ = ctrl.SetControllerReference(ha, dep, r.Scheme)
	return dep
}

// SetupWithManager sets up the controller with the Manager.
func (r *HelloAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.HelloApp{}).
		Complete(r)
}
