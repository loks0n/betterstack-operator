package credentials

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FetchAPIToken resolves the token string stored in the referenced secret.
func FetchAPIToken(ctx context.Context, cl client.Client, namespace string, selector corev1.SecretKeySelector) (string, error) {
	if selector.Name == "" {
		return "", errors.New("apiTokenSecretRef.name must be specified")
	}

	key := types.NamespacedName{Name: selector.Name, Namespace: namespace}
	secret := &corev1.Secret{}
	if err := cl.Get(ctx, key, secret); err != nil {
		return "", err
	}

	tokenBytes, ok := secret.Data[selector.Key]
	if !ok {
		return "", fmt.Errorf("secret %s/%s missing key %s", namespace, selector.Name, selector.Key)
	}

	if len(tokenBytes) == 0 {
		return "", fmt.Errorf("secret %s/%s key %s is empty", namespace, selector.Name, selector.Key)
	}

	return string(tokenBytes), nil
}
