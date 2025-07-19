package tokenprovider

import (
	"context"
	"fmt"
	"sync"
	"time"

	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/utils/ptr"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type K8s struct {
	corev1 v1.CoreV1Interface

	mu    sync.Mutex
	cache map[string]tokenCacheEntry // key: namespace|serviceAccount|audience
}

type tokenCacheEntry struct {
	token     string
	expiresAt time.Time
}

func NewK8s(
	corev1 v1.CoreV1Interface,
) *K8s {
	return &K8s{
		corev1: corev1,
		cache:  make(map[string]tokenCacheEntry),
	}
}

func tokenCacheKey(namespace, sa, audience string) string {
	return fmt.Sprintf("%s|%s|%s", namespace, sa, audience)
}

func (a *K8s) GetToken(ctx context.Context, targetNamespace, k8sServiceAccountName, audience string) (string, error) {
	key := tokenCacheKey(targetNamespace, k8sServiceAccountName, audience)
	logger := logf.FromContext(ctx).WithValues("targetNamespace", targetNamespace, "k8sServiceAccountName", k8sServiceAccountName, "audience", audience)
	a.mu.Lock()
	entry, found := a.cache[key]
	a.mu.Unlock()

	if found && time.Until(entry.expiresAt) > 30*time.Second {
		logger.Info("cache hit", "key", key)
		return entry.token, nil
	}

	logger.Info("cache miss, requesting new token")
	expiration := 900
	tokenRequest := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			Audiences:         []string{audience},
			ExpirationSeconds: ptr.To(int64(expiration)),
		},
	}

	tokenResp, err := a.corev1.
		ServiceAccounts(targetNamespace).
		CreateToken(ctx, k8sServiceAccountName, tokenRequest, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create token for service account %s in targetNamespace %s: %w", k8sServiceAccountName, targetNamespace, err)
	}

	expiresAt := time.Now().Add(time.Second * time.Duration(expiration))

	a.mu.Lock()
	a.cache[key] = tokenCacheEntry{
		token:     tokenResp.Status.Token,
		expiresAt: expiresAt,
	}
	a.mu.Unlock()

	return tokenResp.Status.Token, nil
}
