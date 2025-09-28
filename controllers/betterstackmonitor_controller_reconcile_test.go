package controllers

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	monitoringv1alpha1 "loks0n/betterstack-operator/api/v1alpha1"
	"loks0n/betterstack-operator/pkg/betterstack"
)

func TestReconcileAddsFinalizer(t *testing.T) {
	scheme := newTestScheme(t)

	monitor := &monitoringv1alpha1.BetterStackMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example",
			Namespace: "default",
		},
		Spec: monitoringv1alpha1.BetterStackMonitorSpec{
			APITokenSecretRef: corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "api"},
				Key:                  "token",
			},
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(monitor).
		WithObjects(monitor.DeepCopy()).
		Build()

	r := &BetterStackMonitorReconciler{
		Client: client,
		Scheme: scheme,
	}

	ctx := context.Background()
	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: monitor.Name, Namespace: monitor.Namespace}})
	if err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}
	if res.Requeue || res.RequeueAfter != 0 {
		t.Fatalf("expected no explicit requeue, got %#v", res)
	}

	updated := &monitoringv1alpha1.BetterStackMonitor{}
	if err := client.Get(ctx, types.NamespacedName{Name: monitor.Name, Namespace: monitor.Namespace}, updated); err != nil {
		t.Fatalf("failed to fetch updated monitor: %v", err)
	}
	if !controllerutil.ContainsFinalizer(updated, monitoringv1alpha1.BetterStackMonitorFinalizer) {
		t.Fatalf("expected finalizer to be present")
	}
}

func TestReconcileHandlesMissingCredentials(t *testing.T) {
	scheme := newTestScheme(t)

	monitor := &monitoringv1alpha1.BetterStackMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "example",
			Namespace:  "default",
			Generation: 7,
			Finalizers: []string{monitoringv1alpha1.BetterStackMonitorFinalizer},
		},
		Spec: monitoringv1alpha1.BetterStackMonitorSpec{
			APITokenSecretRef: corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "api"},
				Key:                  "token",
			},
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(monitor).
		WithObjects(monitor.DeepCopy()).
		Build()

	r := &BetterStackMonitorReconciler{
		Client: client,
		Scheme: scheme,
	}

	ctx := context.Background()
	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: monitor.Name, Namespace: monitor.Namespace}})
	if err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}
	if res.RequeueAfter != requeueIntervalOnError {
		t.Fatalf("expected requeue after %v, got %v", requeueIntervalOnError, res.RequeueAfter)
	}

	updated := &monitoringv1alpha1.BetterStackMonitor{}
	if err := client.Get(ctx, types.NamespacedName{Name: monitor.Name, Namespace: monitor.Namespace}, updated); err != nil {
		t.Fatalf("failed to fetch updated monitor: %v", err)
	}

	creds := findCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionCredentials)
	if creds == nil || creds.Status != metav1.ConditionFalse || creds.Reason != "TokenUnavailable" {
		t.Fatalf("unexpected credentials condition: %+v", creds)
	}
	ready := findCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionReady)
	if ready == nil || ready.Status != metav1.ConditionFalse || ready.Reason != "TokenUnavailable" {
		t.Fatalf("unexpected ready condition: %+v", ready)
	}
}

func TestReconcileCreatesMonitorWhenRemoteMissing(t *testing.T) {
	scheme := newTestScheme(t)

	monitor := &monitoringv1alpha1.BetterStackMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "example",
			Namespace:  "default",
			Generation: 3,
			Finalizers: []string{monitoringv1alpha1.BetterStackMonitorFinalizer},
		},
		Spec: monitoringv1alpha1.BetterStackMonitorSpec{
			URL:           "https://example.com",
			MonitorType:   "status",
			RequestMethod: "GET",
			APITokenSecretRef: corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "api"},
				Key:                  "token",
			},
			BaseURL: "https://api.test",
		},
		Status: monitoringv1alpha1.BetterStackMonitorStatus{
			MonitorID: "remote-123",
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
		Data:       map[string][]byte{"token": []byte("abcd")},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(monitor).
		WithObjects(monitor.DeepCopy(), secret.DeepCopy()).
		Build()

	service := &fakeMonitorService{
		t: t,
		getFn: func(ctx context.Context, id string) (betterstack.Monitor, error) {
			if id != "remote-123" {
				t.Fatalf("unexpected get id %s", id)
			}
			return betterstack.Monitor{}, &betterstack.APIError{StatusCode: http.StatusNotFound}
		},
		updateFn: func(ctx context.Context, id string, req betterstack.MonitorUpdateRequest) (betterstack.Monitor, error) {
			if id != "remote-123" {
				t.Fatalf("unexpected update id %s", id)
			}
			return betterstack.Monitor{}, &betterstack.APIError{StatusCode: http.StatusNotFound}
		},
		createFn: func(ctx context.Context, req betterstack.MonitorCreateRequest) (betterstack.Monitor, error) {
			if req.URL == nil || *req.URL != "https://example.com" {
				t.Fatalf("unexpected create url %+v", req.URL)
			}
			if req.MonitorType == nil || *req.MonitorType != "status" {
				t.Fatalf("unexpected monitor type %+v", req.MonitorType)
			}
			if req.HTTPMethod == nil || *req.HTTPMethod != "get" {
				t.Fatalf("expected request method get, got %+v", req.HTTPMethod)
			}
			return betterstack.Monitor{ID: "new-id"}, nil
		},
	}
	factory := &fakeMonitorFactory{service: service}

	r := &BetterStackMonitorReconciler{
		Client:         client,
		Scheme:         scheme,
		MonitorFactory: factory.build,
	}

	ctx := context.Background()
	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: monitor.Name, Namespace: monitor.Namespace}})
	if err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}
	if res.Requeue || res.RequeueAfter != 0 {
		t.Fatalf("expected no requeue, got %#v", res)
	}

	updated := &monitoringv1alpha1.BetterStackMonitor{}
	if err := client.Get(ctx, types.NamespacedName{Name: monitor.Name, Namespace: monitor.Namespace}, updated); err != nil {
		t.Fatalf("failed to fetch updated monitor: %v", err)
	}

	if updated.Status.MonitorID != "new-id" {
		t.Fatalf("expected monitor id to be new-id, got %q", updated.Status.MonitorID)
	}
	if updated.Status.ObservedGeneration != monitor.Generation {
		t.Fatalf("expected observed generation %d, got %d", monitor.Generation, updated.Status.ObservedGeneration)
	}
	ready := findCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionReady)
	if ready == nil || ready.Status != metav1.ConditionTrue || ready.Reason != "MonitorSynced" {
		t.Fatalf("unexpected ready condition: %+v", ready)
	}
	if factory.calls != 1 {
		t.Fatalf("expected monitor factory to be invoked once, got %d", factory.calls)
	}
	if factory.lastToken != "abcd" {
		t.Fatalf("expected token 'abcd', got %q", factory.lastToken)
	}
}

func TestReconcileHandlesUpdateError(t *testing.T) {
	scheme := newTestScheme(t)

	monitor := &monitoringv1alpha1.BetterStackMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "example",
			Namespace:  "default",
			Generation: 4,
			Finalizers: []string{monitoringv1alpha1.BetterStackMonitorFinalizer},
		},
		Spec: monitoringv1alpha1.BetterStackMonitorSpec{
			URL:         "https://example.com",
			MonitorType: "status",
			APITokenSecretRef: corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "api"},
				Key:                  "token",
			},
			BaseURL: "https://api.test",
		},
		Status: monitoringv1alpha1.BetterStackMonitorStatus{MonitorID: "remote-123"},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
		Data:       map[string][]byte{"token": []byte("abcd")},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(monitor).
		WithObjects(monitor.DeepCopy(), secret.DeepCopy()).
		Build()
	service := &fakeMonitorService{
		t: t,
		getFn: func(ctx context.Context, id string) (betterstack.Monitor, error) {
			return betterstack.Monitor{ID: id}, nil
		},
		updateFn: func(ctx context.Context, id string, req betterstack.MonitorUpdateRequest) (betterstack.Monitor, error) {
			if id != "remote-123" {
				t.Fatalf("unexpected update id %s", id)
			}
			return betterstack.Monitor{}, &betterstack.APIError{StatusCode: http.StatusInternalServerError, Message: "boom"}
		},
	}
	factory := &fakeMonitorFactory{service: service}

	r := &BetterStackMonitorReconciler{
		Client:         client,
		Scheme:         scheme,
		MonitorFactory: factory.build,
	}

	ctx := context.Background()
	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: monitor.Name, Namespace: monitor.Namespace}})
	if err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}
	if res.RequeueAfter != requeueIntervalOnError {
		t.Fatalf("expected requeue after error, got %#v", res)
	}

	updated := &monitoringv1alpha1.BetterStackMonitor{}
	if err := client.Get(ctx, types.NamespacedName{Name: monitor.Name, Namespace: monitor.Namespace}, updated); err != nil {
		t.Fatalf("failed to fetch updated monitor: %v", err)
	}
	if updated.Status.MonitorID != "remote-123" {
		t.Fatalf("monitor id should remain unchanged, got %q", updated.Status.MonitorID)
	}

	syncCond := findCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionSync)
	if syncCond == nil || syncCond.Status != metav1.ConditionFalse || syncCond.Reason != "SyncFailed" {
		t.Fatalf("unexpected sync condition: %+v", syncCond)
	}
	readyCond := findCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionReady)
	if readyCond == nil || readyCond.Status != metav1.ConditionFalse || readyCond.Reason != "SyncFailed" {
		t.Fatalf("unexpected ready condition: %+v", readyCond)
	}
}

func TestReconcileHandlesCreateError(t *testing.T) {
	scheme := newTestScheme(t)

	monitor := &monitoringv1alpha1.BetterStackMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "example",
			Namespace:  "default",
			Generation: 2,
			Finalizers: []string{monitoringv1alpha1.BetterStackMonitorFinalizer},
		},
		Spec: monitoringv1alpha1.BetterStackMonitorSpec{
			URL:         "https://example.com",
			MonitorType: "status",
			APITokenSecretRef: corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "api"},
				Key:                  "token",
			},
			BaseURL: "https://api.test",
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
		Data:       map[string][]byte{"token": []byte("abcd")},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(monitor).
		WithObjects(monitor.DeepCopy(), secret.DeepCopy()).
		Build()
	service := &fakeMonitorService{
		t: t,
		createFn: func(ctx context.Context, req betterstack.MonitorCreateRequest) (betterstack.Monitor, error) {
			return betterstack.Monitor{}, &betterstack.APIError{StatusCode: http.StatusInternalServerError, Message: "boom"}
		},
	}
	factory := &fakeMonitorFactory{service: service}

	r := &BetterStackMonitorReconciler{
		Client:         client,
		Scheme:         scheme,
		MonitorFactory: factory.build,
	}

	ctx := context.Background()
	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: monitor.Name, Namespace: monitor.Namespace}})
	if err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}
	if res.RequeueAfter != requeueIntervalOnError {
		t.Fatalf("expected requeue after error, got %#v", res)
	}

	updated := &monitoringv1alpha1.BetterStackMonitor{}
	if err := client.Get(ctx, types.NamespacedName{Name: monitor.Name, Namespace: monitor.Namespace}, updated); err != nil {
		t.Fatalf("failed to fetch updated monitor: %v", err)
	}
	if updated.Status.MonitorID != "" {
		t.Fatalf("monitor id should remain empty, got %q", updated.Status.MonitorID)
	}

	syncCond := findCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionSync)
	if syncCond == nil || syncCond.Status != metav1.ConditionFalse || syncCond.Reason != "SyncFailed" {
		t.Fatalf("unexpected sync condition: %+v", syncCond)
	}
}

func TestReconcileHandlesDeletion(t *testing.T) {
	scheme := newTestScheme(t)

	deletionTime := metav1.NewTime(time.Now())
	monitor := &monitoringv1alpha1.BetterStackMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "example",
			Namespace:         "default",
			Finalizers:        []string{monitoringv1alpha1.BetterStackMonitorFinalizer},
			DeletionTimestamp: &deletionTime,
		},
		Spec: monitoringv1alpha1.BetterStackMonitorSpec{
			BaseURL: "https://api.test",
			APITokenSecretRef: corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "api"},
				Key:                  "token",
			},
		},
		Status: monitoringv1alpha1.BetterStackMonitorStatus{MonitorID: "remote-123"},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
		Data:       map[string][]byte{"token": []byte("abcd")},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(monitor).
		WithObjects(monitor.DeepCopy(), secret.DeepCopy()).
		Build()
	deleted := false
	service := &fakeMonitorService{
		t: t,
		deleteFn: func(ctx context.Context, id string) error {
			if id != "remote-123" {
				t.Fatalf("unexpected delete id %s", id)
			}
			deleted = true
			return nil
		},
	}
	factory := &fakeMonitorFactory{service: service}

	r := &BetterStackMonitorReconciler{
		Client:         client,
		Scheme:         scheme,
		MonitorFactory: factory.build,
	}

	ctx := context.Background()
	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: monitor.Name, Namespace: monitor.Namespace}})
	if err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}
	if res.Requeue || res.RequeueAfter != 0 {
		t.Fatalf("expected no requeue, got %#v", res)
	}
	if !deleted {
		t.Fatalf("expected delete request to be issued")
	}

	updated := &monitoringv1alpha1.BetterStackMonitor{}
	err = client.Get(ctx, types.NamespacedName{Name: monitor.Name, Namespace: monitor.Namespace}, updated)
	if err == nil {
		if controllerutil.ContainsFinalizer(updated, monitoringv1alpha1.BetterStackMonitorFinalizer) {
			t.Fatalf("expected finalizer to be removed")
		}
	} else if !apierrors.IsNotFound(err) {
		t.Fatalf("failed to fetch updated monitor: %v", err)
	}
}

func TestReconcileHandlesDeletionMissingCredentials(t *testing.T) {
	scheme := newTestScheme(t)

	deletionTime := metav1.NewTime(time.Now())
	monitor := &monitoringv1alpha1.BetterStackMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "example",
			Namespace:         "default",
			Finalizers:        []string{monitoringv1alpha1.BetterStackMonitorFinalizer},
			DeletionTimestamp: &deletionTime,
		},
		Spec: monitoringv1alpha1.BetterStackMonitorSpec{
			APITokenSecretRef: corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "missing"},
				Key:                  "token",
			},
		},
		Status: monitoringv1alpha1.BetterStackMonitorStatus{MonitorID: "remote-123"},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(monitor).
		WithObjects(monitor.DeepCopy()).
		Build()
	factory := &fakeMonitorFactory{t: t, service: &fakeMonitorService{t: t}}

	r := &BetterStackMonitorReconciler{
		Client:         client,
		Scheme:         scheme,
		MonitorFactory: factory.build,
	}

	ctx := context.Background()
	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: monitor.Name, Namespace: monitor.Namespace}})
	if err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}
	if res.Requeue || res.RequeueAfter != 0 {
		t.Fatalf("expected no requeue, got %#v", res)
	}

	updated := &monitoringv1alpha1.BetterStackMonitor{}
	err = client.Get(ctx, types.NamespacedName{Name: monitor.Name, Namespace: monitor.Namespace}, updated)
	if err == nil {
		if controllerutil.ContainsFinalizer(updated, monitoringv1alpha1.BetterStackMonitorFinalizer) {
			t.Fatalf("expected finalizer to be removed")
		}
	} else if !apierrors.IsNotFound(err) {
		t.Fatalf("failed to fetch updated monitor: %v", err)
	}
	if factory.calls != 0 {
		t.Fatalf("expected monitor factory not to be called, got %d", factory.calls)
	}
}

func TestReconcileHandlesDeletionRemoteNotFound(t *testing.T) {
	scheme := newTestScheme(t)

	deletionTime := metav1.NewTime(time.Now())
	monitor := &monitoringv1alpha1.BetterStackMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "example",
			Namespace:         "default",
			Finalizers:        []string{monitoringv1alpha1.BetterStackMonitorFinalizer},
			DeletionTimestamp: &deletionTime,
		},
		Spec: monitoringv1alpha1.BetterStackMonitorSpec{
			APITokenSecretRef: corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "api"},
				Key:                  "token",
			},
			BaseURL: "https://api.test",
		},
		Status: monitoringv1alpha1.BetterStackMonitorStatus{MonitorID: "remote-123"},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
		Data:       map[string][]byte{"token": []byte("abcd")},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(monitor).
		WithObjects(monitor.DeepCopy(), secret.DeepCopy()).
		Build()
	service := &fakeMonitorService{
		t: t,
		deleteFn: func(ctx context.Context, id string) error {
			return &betterstack.APIError{StatusCode: http.StatusNotFound}
		},
	}
	factory := &fakeMonitorFactory{service: service}

	r := &BetterStackMonitorReconciler{
		Client:         client,
		Scheme:         scheme,
		MonitorFactory: factory.build,
	}

	ctx := context.Background()
	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: monitor.Name, Namespace: monitor.Namespace}})
	if err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}
	if res.Requeue || res.RequeueAfter != 0 {
		t.Fatalf("expected no requeue, got %#v", res)
	}

	updated := &monitoringv1alpha1.BetterStackMonitor{}
	err = client.Get(ctx, types.NamespacedName{Name: monitor.Name, Namespace: monitor.Namespace}, updated)
	if err == nil {
		if controllerutil.ContainsFinalizer(updated, monitoringv1alpha1.BetterStackMonitorFinalizer) {
			t.Fatalf("expected finalizer to be removed")
		}
	} else if !apierrors.IsNotFound(err) {
		t.Fatalf("failed to fetch updated monitor: %v", err)
	}
}

func TestFetchAPITokenValidation(t *testing.T) {
	scheme := newTestScheme(t)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "creds", Namespace: "default"},
		Data:       map[string][]byte{"token": []byte("abcd")},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(secret.DeepCopy()).
		Build()

	r := &BetterStackMonitorReconciler{Client: client, Scheme: scheme}
	ctx := context.Background()

	if _, err := r.fetchAPIToken(ctx, "default", corev1.SecretKeySelector{}); err == nil {
		t.Fatalf("expected error when name is empty")
	}

	if _, err := r.fetchAPIToken(ctx, "default", corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "missing"}, Key: "token"}); err == nil {
		t.Fatalf("expected error for missing secret")
	}

	if _, err := r.fetchAPIToken(ctx, "default", corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "creds"}, Key: "missing"}); err == nil {
		t.Fatalf("expected error for missing key")
	}

	emptySecret := secret.DeepCopy()
	emptySecret.Data = map[string][]byte{"token": nil}
	if err := client.Update(ctx, emptySecret); err != nil {
		t.Fatalf("failed to update secret: %v", err)
	}
	if _, err := r.fetchAPIToken(ctx, "default", corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "creds"}, Key: "token"}); err == nil {
		t.Fatalf("expected error for empty token")
	}

	// restore token to verify happy path
	if err := client.Update(ctx, secret.DeepCopy()); err != nil {
		t.Fatalf("failed to restore secret: %v", err)
	}
	token, err := r.fetchAPIToken(ctx, "default", corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "creds"}, Key: "token"})
	if err != nil {
		t.Fatalf("unexpected error fetching token: %v", err)
	}
	if token != "abcd" {
		t.Fatalf("expected token 'abcd', got %q", token)
	}
}

func TestReconcileReturnsErrorWhenStatusPatchFails(t *testing.T) {
	scheme := newTestScheme(t)

	monitor := &monitoringv1alpha1.BetterStackMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "example",
			Namespace:  "default",
			Generation: 5,
			Finalizers: []string{monitoringv1alpha1.BetterStackMonitorFinalizer},
		},
		Spec: monitoringv1alpha1.BetterStackMonitorSpec{
			URL:         "https://example.com",
			MonitorType: "status",
			APITokenSecretRef: corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "api"},
				Key:                  "token",
			},
			BaseURL: "https://api.test",
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
		Data:       map[string][]byte{"token": []byte("abcd")},
	}

	baseClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(monitor).
		WithObjects(monitor.DeepCopy(), secret.DeepCopy()).
		Build()

	failingClient := &failingStatusClient{Client: baseClient, failOn: 2}
	service := &fakeMonitorService{
		t: t,
		createFn: func(ctx context.Context, req betterstack.MonitorCreateRequest) (betterstack.Monitor, error) {
			return betterstack.Monitor{ID: "new-id"}, nil
		},
	}
	factory := &fakeMonitorFactory{service: service}

	r := &BetterStackMonitorReconciler{
		Client:         failingClient,
		Scheme:         scheme,
		MonitorFactory: factory.build,
	}

	ctx := context.Background()
	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: monitor.Name, Namespace: monitor.Namespace}})
	if err == nil || err.Error() != "status patch failed" {
		t.Fatalf("expected status patch error, got res=%#v err=%v", res, err)
	}
	if res.Requeue || res.RequeueAfter != 0 {
		t.Fatalf("expected zero result on error, got %#v", res)
	}

	if failingClient.calls != 2 {
		t.Fatalf("expected two status patch attempts, got %d", failingClient.calls)
	}
}

func newTestScheme(t *testing.T) *runtime.Scheme {
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

func findCondition(conditions []metav1.Condition, condType string) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == condType {
			return &conditions[i]
		}
	}
	return nil
}

type fakeMonitorFactory struct {
	t           *testing.T
	service     *fakeMonitorService
	calls       int
	lastBaseURL string
	lastToken   string
}

func (f *fakeMonitorFactory) build(baseURL, token string, _ *http.Client) betterstack.MonitorClient {
	f.calls++
	f.lastBaseURL = baseURL
	f.lastToken = token
	if f.service == nil {
		if f.t != nil {
			f.t.Fatalf("monitor service not provided")
		}
		return nil
	}
	return f.service
}

type fakeMonitorService struct {
	t *testing.T

	getFn    func(ctx context.Context, id string) (betterstack.Monitor, error)
	updateFn func(ctx context.Context, id string, req betterstack.MonitorUpdateRequest) (betterstack.Monitor, error)
	createFn func(ctx context.Context, req betterstack.MonitorCreateRequest) (betterstack.Monitor, error)
	deleteFn func(ctx context.Context, id string) error

	getCalls    int
	updateCalls int
	createCalls int
	deleteCalls int

	lastUpdateReq betterstack.MonitorUpdateRequest
	lastCreateReq betterstack.MonitorCreateRequest
}

func (s *fakeMonitorService) Get(ctx context.Context, id string) (betterstack.Monitor, error) {
	s.getCalls++
	if s.getFn != nil {
		return s.getFn(ctx, id)
	}
	return betterstack.Monitor{}, nil
}

func (s *fakeMonitorService) Update(ctx context.Context, id string, req betterstack.MonitorUpdateRequest) (betterstack.Monitor, error) {
	s.updateCalls++
	s.lastUpdateReq = req
	if s.updateFn != nil {
		return s.updateFn(ctx, id, req)
	}
	return betterstack.Monitor{}, nil
}

func (s *fakeMonitorService) Create(ctx context.Context, req betterstack.MonitorCreateRequest) (betterstack.Monitor, error) {
	s.createCalls++
	s.lastCreateReq = req
	if s.createFn != nil {
		return s.createFn(ctx, req)
	}
	return betterstack.Monitor{}, nil
}

var _ betterstack.MonitorClient = (*fakeMonitorService)(nil)

func (s *fakeMonitorService) Delete(ctx context.Context, id string) error {
	s.deleteCalls++
	if s.deleteFn != nil {
		return s.deleteFn(ctx, id)
	}
	return nil
}

type failingStatusClient struct {
	client.Client
	failOn int
	calls  int
}

func (f *failingStatusClient) Status() client.StatusWriter {
	return &failingStatusWriter{
		StatusWriter: f.Client.Status(),
		failOn:       f.failOn,
		calls:        &f.calls,
	}
}

type failingStatusWriter struct {
	client.StatusWriter
	failOn int
	calls  *int
}

func (w *failingStatusWriter) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	*w.calls++
	if *w.calls == w.failOn {
		return fmt.Errorf("status patch failed")
	}
	return w.StatusWriter.Patch(ctx, obj, patch, opts...)
}

func (w *failingStatusWriter) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	*w.calls++
	if *w.calls == w.failOn {
		return fmt.Errorf("status patch failed")
	}
	return w.StatusWriter.Update(ctx, obj, opts...)
}
