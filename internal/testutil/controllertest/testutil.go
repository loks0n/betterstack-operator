package controllertest

import (
	"context"
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	monitoringv1alpha1 "loks0n/betterstack-operator/api/v1alpha1"
)

// NewScheme constructs a runtime.Scheme populated with the APIs used in controller tests.
func NewScheme(t testing.TB) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add corev1 to scheme: %v", err)
	}
	if err := monitoringv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add api to scheme: %v", err)
	}
	return scheme
}

// FindCondition locates a condition by type.
func FindCondition(conditions []metav1.Condition, condType string) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == condType {
			return &conditions[i]
		}
	}
	return nil
}

// FailingStatusClient decorates the status writer to induce failures after N calls.
type FailingStatusClient struct {
	client.Client
	FailOn int
	calls  int
}

func (f *FailingStatusClient) Status() client.StatusWriter {
	return &FailingStatusWriter{
		StatusWriter: f.Client.Status(),
		failOn:       f.FailOn,
		calls:        &f.calls,
	}
}

// Calls returns how many status operations have been attempted.
func (f *FailingStatusClient) Calls() int {
	return f.calls
}

// FailingStatusWriter represents a status writer that fails on the Nth call.
type FailingStatusWriter struct {
	client.StatusWriter
	failOn int
	calls  *int
}

func (w *FailingStatusWriter) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	*w.calls++
	if *w.calls == w.failOn {
		return fmt.Errorf("status patch failed")
	}
	return w.StatusWriter.Patch(ctx, obj, patch, opts...)
}

func (w *FailingStatusWriter) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	*w.calls++
	if *w.calls == w.failOn {
		return fmt.Errorf("status patch failed")
	}
	return w.StatusWriter.Update(ctx, obj, opts...)
}
