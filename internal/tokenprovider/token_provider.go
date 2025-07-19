package tokenprovider

import (
	"context"
	"fmt"

	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/utils/ptr"
)

type K8s struct {
	targetNamespace       string
	k8sServiceAccountName string
	corev1                v1.CoreV1Interface
}

func NewK8s(
	targetNamespace, k8sServiceAccountName string,
	corev1 v1.CoreV1Interface,
) K8s {
	return K8s{
		targetNamespace:       targetNamespace,
		k8sServiceAccountName: k8sServiceAccountName,
		corev1:                corev1,
	}
}

func (a K8s) GetToken(ctx context.Context, audience string) (string, error) {
	tokenRequest := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			Audiences:         []string{audience},
			ExpirationSeconds: ptr.To(int64(900)),
		},
	}

	tokenResp, err := a.corev1.
		ServiceAccounts(a.targetNamespace).
		CreateToken(ctx, a.k8sServiceAccountName, tokenRequest, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create token for service account %s in namespace %s: %w", a.k8sServiceAccountName, a.targetNamespace, err)
	}
	return tokenResp.Status.Token, nil
}
