package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
	"time"

	"k8s.io/utils/ptr"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	monitoringv1alpha1 "loks0n/betterstack-operator/api/v1alpha1"
	"loks0n/betterstack-operator/internal/testutil/controllertest"
	"loks0n/betterstack-operator/pkg/betterstack"
)

type fakeBetterStackMonitorClientFactory struct {
	t                  *testing.T
	monitor            betterstack.MonitorClient
	monitorCalls       int
	lastMonitorBaseURL string
	lastMonitorToken   string
}

func (f *fakeBetterStackMonitorClientFactory) Monitor(baseURL, token string, _ *http.Client) betterstack.MonitorClient {
	f.monitorCalls++
	f.lastMonitorBaseURL = baseURL
	f.lastMonitorToken = token
	if f.monitor == nil {
		if f.t != nil {
			f.t.Fatalf("monitor service not provided")
		}
		return &fakeMonitorService{}
	}
	return f.monitor
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

func (s *fakeMonitorService) Delete(ctx context.Context, id string) error {
	s.deleteCalls++
	if s.deleteFn != nil {
		return s.deleteFn(ctx, id)
	}
	return nil
}

var _ betterstack.MonitorClient = (*fakeMonitorService)(nil)

func TestReconcileAddsFinalizer(t *testing.T) {
	scheme := controllertest.NewScheme(t)

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
	scheme := controllertest.NewScheme(t)

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

	creds := controllertest.FindCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionCredentials)
	if creds == nil || creds.Status != metav1.ConditionFalse || creds.Reason != "TokenUnavailable" {
		t.Fatalf("unexpected credentials condition: %+v", creds)
	}
	ready := controllertest.FindCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionReady)
	if ready == nil || ready.Status != metav1.ConditionFalse || ready.Reason != "TokenUnavailable" {
		t.Fatalf("unexpected ready condition: %+v", ready)
	}
}

func TestReconcileCreatesMonitorWhenRemoteMissing(t *testing.T) {
	scheme := controllertest.NewScheme(t)

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
	factory := &fakeBetterStackMonitorClientFactory{monitor: service}

	r := &BetterStackMonitorReconciler{
		Client:  client,
		Scheme:  scheme,
		Clients: factory,
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
	ready := controllertest.FindCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionReady)
	if ready == nil || ready.Status != metav1.ConditionTrue || ready.Reason != "MonitorSynced" {
		t.Fatalf("unexpected ready condition: %+v", ready)
	}
	if factory.monitorCalls != 1 {
		t.Fatalf("expected monitor factory to be invoked once, got %d", factory.monitorCalls)
	}
	if factory.lastMonitorToken != "abcd" {
		t.Fatalf("expected token 'abcd', got %q", factory.lastMonitorToken)
	}
}

func TestReconcileHandlesUpdateError(t *testing.T) {
	scheme := controllertest.NewScheme(t)

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
	factory := &fakeBetterStackMonitorClientFactory{monitor: service}

	r := &BetterStackMonitorReconciler{Client: client, Scheme: scheme, Clients: factory}

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

	syncCond := controllertest.FindCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionSync)
	if syncCond == nil || syncCond.Status != metav1.ConditionFalse || syncCond.Reason != "SyncFailed" {
		t.Fatalf("unexpected sync condition: %+v", syncCond)
	}
	readyCond := controllertest.FindCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionReady)
	if readyCond == nil || readyCond.Status != metav1.ConditionFalse || readyCond.Reason != "SyncFailed" {
		t.Fatalf("unexpected ready condition: %+v", readyCond)
	}
}

func TestReconcileHandlesCreateError(t *testing.T) {
	scheme := controllertest.NewScheme(t)

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
	factory := &fakeBetterStackMonitorClientFactory{monitor: service}

	r := &BetterStackMonitorReconciler{Client: client, Scheme: scheme, Clients: factory}

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

	syncCond := controllertest.FindCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionSync)
	if syncCond == nil || syncCond.Status != metav1.ConditionFalse || syncCond.Reason != "SyncFailed" {
		t.Fatalf("unexpected sync condition: %+v", syncCond)
	}
}

func TestReconcileHandlesDeletion(t *testing.T) {
	scheme := controllertest.NewScheme(t)

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
	factory := &fakeBetterStackMonitorClientFactory{monitor: service}

	r := &BetterStackMonitorReconciler{Client: client, Scheme: scheme, Clients: factory}

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
	scheme := controllertest.NewScheme(t)

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
	factory := &fakeBetterStackMonitorClientFactory{t: t, monitor: &fakeMonitorService{t: t}}

	r := &BetterStackMonitorReconciler{Client: client, Scheme: scheme, Clients: factory}

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
	if factory.monitorCalls != 0 {
		t.Fatalf("expected monitor factory not to be called, got %d", factory.monitorCalls)
	}
}

func TestReconcileHandlesDeletionRemoteNotFound(t *testing.T) {
	scheme := controllertest.NewScheme(t)

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
	factory := &fakeBetterStackMonitorClientFactory{monitor: service}

	r := &BetterStackMonitorReconciler{Client: client, Scheme: scheme, Clients: factory}

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
	scheme := controllertest.NewScheme(t)

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
	scheme := controllertest.NewScheme(t)

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

	failingClient := &controllertest.FailingStatusClient{Client: baseClient, FailOn: 2}
	service := &fakeMonitorService{
		t: t,
		createFn: func(ctx context.Context, req betterstack.MonitorCreateRequest) (betterstack.Monitor, error) {
			return betterstack.Monitor{ID: "new-id"}, nil
		},
	}
	factory := &fakeBetterStackMonitorClientFactory{monitor: service}

	r := &BetterStackMonitorReconciler{Client: failingClient, Scheme: scheme, Clients: factory}

	ctx := context.Background()
	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: monitor.Name, Namespace: monitor.Namespace}})
	if err == nil || err.Error() != "status patch failed" {
		t.Fatalf("expected status patch error, got res=%#v err=%v", res, err)
	}
	if res.Requeue || res.RequeueAfter != 0 {
		t.Fatalf("expected zero result on error, got %#v", res)
	}

	if failingClient.Calls() != 2 {
		t.Fatalf("expected two status patch attempts, got %d", failingClient.Calls())
	}
}

func TestBuildMonitorRequest(t *testing.T) {
	spec := monitoringv1alpha1.BetterStackMonitorSpec{
		URL:                       "https://example.com",
		Name:                      "Example",
		MonitorType:               "status",
		TeamName:                  "SRE",
		CheckFrequencyMinutes:     3,
		Regions:                   []string{"us", "eu"},
		RequestMethod:             "POST",
		ExpectedStatusCodes:       []int{201, 202},
		RequiredKeyword:           "healthy",
		Paused:                    true,
		Email:                     ptr.To(false),
		SMS:                       ptr.To(true),
		Call:                      ptr.To(false),
		Push:                      ptr.To(true),
		CriticalAlert:             ptr.To(true),
		FollowRedirects:           ptr.To(true),
		VerifySSL:                 ptr.To(false),
		RememberCookies:           ptr.To(true),
		PolicyID:                  "policy-1",
		ExpirationPolicyID:        "exp-1",
		MonitorGroupID:            "group-1",
		TeamWaitSeconds:           120,
		DomainExpirationDays:      14,
		SSLExpirationDays:         30,
		Port:                      443,
		RequestTimeoutSeconds:     30,
		RecoveryPeriodSeconds:     300,
		ConfirmationPeriodSeconds: 60,
		IPVersion:                 "ipv6",
		MaintenanceDays:           []string{"mon", "tue"},
		MaintenanceFrom:           "01:00:00",
		MaintenanceTo:             "02:00:00",
		MaintenanceTimezone:       "UTC",
		RequestHeaders: []monitoringv1alpha1.BetterStackHeader{{
			Name:  "Content-Type",
			Value: "application/json",
		}},
		RequestBody:          "{}",
		AuthUsername:         "user",
		AuthPassword:         "pass",
		EnvironmentVariables: map[string]string{"TOKEN": "value"},
		PlaywrightScript:     "console.log('ok')",
		ScenarioName:         "Scenario",
		AdditionalAttributes: map[string]string{"custom": "value"},
	}

	want := map[string]any{
		"url":                   spec.URL,
		"pronounceable_name":    spec.Name,
		"monitor_type":          spec.MonitorType,
		"team_name":             spec.TeamName,
		"check_frequency":       spec.CheckFrequencyMinutes * 60,
		"regions":               spec.Regions,
		"http_method":           "post",
		"expected_status_codes": spec.ExpectedStatusCodes,
		"required_keyword":      spec.RequiredKeyword,
		"paused":                true,
		"email":                 false,
		"sms":                   true,
		"call":                  false,
		"push":                  true,
		"critical_alert":        true,
		"follow_redirects":      true,
		"verify_ssl":            false,
		"remember_cookies":      true,
		"policy_id":             spec.PolicyID,
		"expiration_policy_id":  spec.ExpirationPolicyID,
		"monitor_group_id":      spec.MonitorGroupID,
		"team_wait":             spec.TeamWaitSeconds,
		"domain_expiration":     spec.DomainExpirationDays,
		"ssl_expiration":        spec.SSLExpirationDays,
		"port":                  "443",
		"request_timeout":       spec.RequestTimeoutSeconds,
		"recovery_period":       spec.RecoveryPeriodSeconds,
		"confirmation_period":   spec.ConfirmationPeriodSeconds,
		"ip_version":            spec.IPVersion,
		"maintenance_days":      spec.MaintenanceDays,
		"maintenance_from":      spec.MaintenanceFrom,
		"maintenance_to":        spec.MaintenanceTo,
		"maintenance_timezone":  spec.MaintenanceTimezone,
		"request_headers":       []map[string]string{{"name": "Content-Type", "value": "application/json"}},
		"request_body":          spec.RequestBody,
		"auth_username":         spec.AuthUsername,
		"auth_password":         spec.AuthPassword,
		"environment_variables": spec.EnvironmentVariables,
		"playwright_script":     spec.PlaywrightScript,
		"scenario_name":         spec.ScenarioName,
		"custom":                "value",
	}
	wantedJSON, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal want: %v", err)
	}
	wanted := map[string]any{}
	if err := json.Unmarshal(wantedJSON, &wanted); err != nil {
		t.Fatalf("unmarshal want: %v", err)
	}

	gotReq := buildMonitorRequest(spec, nil)
	encoded, err := json.Marshal(gotReq)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	got := map[string]any{}
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	if diff := diffMaps(got, wanted); len(diff) > 0 {
		t.Fatalf("unexpected attributes map: diff=%v", diff)
	}
}

func TestBuildMonitorRequestConvertsTimeoutForServerMonitors(t *testing.T) {
	spec := monitoringv1alpha1.BetterStackMonitorSpec{
		URL:                   "tcp://example.com",
		MonitorType:           "tcp",
		RequestTimeoutSeconds: 3,
	}

	req := buildMonitorRequest(spec, nil)
	if req.RequestTimeout == nil {
		t.Fatalf("request timeout missing")
	}
	if got, want := *req.RequestTimeout, 3000; got != want {
		t.Fatalf("timeout not converted, got %d want %d", got, want)
	}
}

func TestBuildMonitorRequestAssignsHeaderIDsWhenPresent(t *testing.T) {
	existingHeaderID := "hdr-123"
	existing := &betterstack.Monitor{
		Attributes: betterstack.MonitorAttributes{
			RequestHeaders: []betterstack.MonitorHeader{{
				ID:    existingHeaderID,
				Name:  "X-Test",
				Value: "old",
			}},
		},
	}

	spec := monitoringv1alpha1.BetterStackMonitorSpec{
		URL: "https://example.com",
		RequestHeaders: []monitoringv1alpha1.BetterStackHeader{{
			Name:  "X-Test",
			Value: "new",
		}},
	}

	req := buildMonitorRequest(spec, existing)
	if len(req.RequestHeaders) != 1 {
		t.Fatalf("expected 1 header, got %d", len(req.RequestHeaders))
	}
	if req.RequestHeaders[0].ID == nil || *req.RequestHeaders[0].ID != existingHeaderID {
		t.Fatalf("expected header id %s, got %v", existingHeaderID, req.RequestHeaders[0].ID)
	}
}

func diffMaps(got, want map[string]any) map[string][2]any {
	diff := make(map[string][2]any)
	keys := make(map[string]struct{})
	for k := range got {
		keys[k] = struct{}{}
	}
	for k := range want {
		keys[k] = struct{}{}
	}
	for k := range keys {
		gv, gok := got[k]
		wv, wok := want[k]
		if !gok || !wok || !reflect.DeepEqual(gv, wv) {
			diff[k] = [2]any{gv, wv}
		}
	}
	return diff
}
