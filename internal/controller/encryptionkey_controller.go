/*
Copyright 2025.

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

package controller

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	sijomav1alpha1 "github.com/sijoma.io/key-encryption-granter/api/v1alpha1"
	aws2 "github.com/sijoma.io/key-encryption-granter/internal/aws"
	"github.com/sijoma.io/key-encryption-granter/internal/tokenprovider"
)

// EncryptionKeyReconciler reconciles a EncryptionKey object
type EncryptionKeyReconciler struct {
	client.Client
	tokenProvider
	Scheme *runtime.Scheme
}

// This does a K8s TokenRequest to get the token for the service account, we can also pass the audience
// for example to query cloud APIs of AWS.
type tokenProvider interface {
	GetToken(ctx context.Context, targetNamespace, k8sServiceAccountName, audience string) (string, error)
}

// +kubebuilder:rbac:groups=sijoma.sijoma.io,resources=encryptionkeys,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sijoma.sijoma.io,resources=encryptionkeys/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=sijoma.sijoma.io,resources=encryptionkeys/finalizers,verbs=update

// Allow getting k8s service accounts, required to request tokens
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get
// +kubebuilder:rbac:groups="",resources=serviceaccounts/token,verbs=create

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *EncryptionKeyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	var encryptionKey = sijomav1alpha1.EncryptionKey{}
	err := r.Get(ctx, req.NamespacedName, &encryptionKey)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	awskms := aws2.NewAwsKMS(
		aws2.WithTenantRoleARN(encryptionKey.Spec.AccountID),
		aws2.WithKeyARN(encryptionKey.Spec.KeyID),
		aws2.WithRoleSessionName("encryption-observer"),
		aws2.WithTokenProvider(r.tokenProvider),
	)

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	info, err := awskms.DescribeKey(
		ctxWithTimeout,
		encryptionKey.Namespace,
		encryptionKey.Spec.KubernetesServiceAccount,
	)
	if err != nil {
		logger.Error(err, "Failed to describe KMS key")
		return ctrl.Result{}, fmt.Errorf("failed to describe KMS key: %w", err)
	}

	if info != nil && info.KeyMetadata != nil {
		// Update the status with key information
		encryptionKey.Status.KeyState = string(info.KeyMetadata.KeyState)
		encryptionKey.Status.Arn = *info.KeyMetadata.Arn
		encryptionKey.Status.LastReconciledTime = ptr.To(metav1.Now())
		// Update the status in Kubernetes
		if err := r.Status().Update(ctx, &encryptionKey); err != nil {
			logger.Error(err, "Failed to update EncryptionKey status")
			return ctrl.Result{}, fmt.Errorf("failed to update status: %w", err)
		}

		logger.Info("Successfully updated EncryptionKey status", "keyState", info.KeyMetadata.KeyState)
	}

	return ctrl.Result{RequeueAfter: time.Minute}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EncryptionKeyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	cfg, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		return fmt.Errorf("EncryptionKeyReconciler: failed to create kubernetes config: %w", err)
	}

	r.tokenProvider = tokenprovider.NewK8s(cfg.CoreV1())
	return ctrl.NewControllerManagedBy(mgr).
		For(&sijomav1alpha1.EncryptionKey{}).
		Named("encryptionkey").
		Complete(r)
}
