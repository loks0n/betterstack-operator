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
	"loks0n/betterstack-operator/internal/testutil/assert"
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
	listFn   func(ctx context.Context) ([]betterstack.Monitor, error)

	getCalls    int
	updateCalls int
	createCalls int
	deleteCalls int
	listCalls   int

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

func (s *fakeMonitorService) List(ctx context.Context) ([]betterstack.Monitor, error) {
	s.listCalls++
	if s.listFn != nil {
		return s.listFn(ctx)
	}
	return nil, nil
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
	assert.NoError(t, err, "reconcile")
	assert.Bool(t, "requeue", res.Requeue, false)
	assert.Equal(t, "requeueAfter", res.RequeueAfter, time.Duration(0))

	updated := &monitoringv1alpha1.BetterStackMonitor{}
	assert.NoError(t, client.Get(ctx, types.NamespacedName{Name: monitor.Name, Namespace: monitor.Namespace}, updated), "fetch updated monitor")
	assert.Bool(t, "finalizer present", controllerutil.ContainsFinalizer(updated, monitoringv1alpha1.BetterStackMonitorFinalizer), true)
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
	assert.NoError(t, err, "reconcile")
	assert.Equal(t, "requeueAfter", res.RequeueAfter, requeueIntervalOnError)

	updated := &monitoringv1alpha1.BetterStackMonitor{}
	assert.NoError(t, client.Get(ctx, types.NamespacedName{Name: monitor.Name, Namespace: monitor.Namespace}, updated), "fetch updated monitor")

	creds := controllertest.FindCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionCredentials)
	assert.NotNil(t, "credentials condition", creds)
	assert.Equal(t, "credentials status", creds.Status, metav1.ConditionFalse)
	assert.String(t, "credentials reason", creds.Reason, "TokenUnavailable")
	ready := controllertest.FindCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionReady)
	assert.NotNil(t, "ready condition", ready)
	assert.Equal(t, "ready status", ready.Status, metav1.ConditionFalse)
	assert.String(t, "ready reason", ready.Reason, "TokenUnavailable")
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
			RequestMethod: "get",
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
			assert.String(t, "get id", id, "remote-123")
			return betterstack.Monitor{}, &betterstack.APIError{StatusCode: http.StatusNotFound}
		},
		updateFn: func(ctx context.Context, id string, req betterstack.MonitorUpdateRequest) (betterstack.Monitor, error) {
			assert.String(t, "update id", id, "remote-123")
			return betterstack.Monitor{}, &betterstack.APIError{StatusCode: http.StatusNotFound}
		},
		createFn: func(ctx context.Context, req betterstack.MonitorCreateRequest) (betterstack.Monitor, error) {
			assert.NotNil(t, "request url", req.URL)
			assert.String(t, "request url", *req.URL, "https://example.com")
			assert.NotNil(t, "request type", req.MonitorType)
			assert.String(t, "request type", *req.MonitorType, "status")
			assert.NotNil(t, "request method", req.HTTPMethod)
			assert.String(t, "request method", *req.HTTPMethod, "get")
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
	assert.NoError(t, err, "reconcile")
	assert.Bool(t, "requeue", res.Requeue, false)
	assert.Equal(t, "requeueAfter", res.RequeueAfter, time.Duration(0))

	updated := &monitoringv1alpha1.BetterStackMonitor{}
	assert.NoError(t, client.Get(ctx, types.NamespacedName{Name: monitor.Name, Namespace: monitor.Namespace}, updated), "fetch updated monitor")
	assert.String(t, "monitor id", updated.Status.MonitorID, "new-id")
	assert.Equal(t, "observed generation", updated.Status.ObservedGeneration, monitor.Generation)
	ready := controllertest.FindCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionReady)
	assert.NotNil(t, "ready condition", ready)
	assert.Equal(t, "ready status", ready.Status, metav1.ConditionTrue)
	assert.String(t, "ready reason", ready.Reason, "MonitorSynced")
	assert.Int(t, "monitor factory calls", factory.monitorCalls, 1)
	assert.String(t, "last token", factory.lastMonitorToken, "abcd")
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
			assert.String(t, "update id", id, "remote-123")
			return betterstack.Monitor{}, &betterstack.APIError{StatusCode: http.StatusInternalServerError, Message: "boom"}
		},
	}
	factory := &fakeBetterStackMonitorClientFactory{monitor: service}

	r := &BetterStackMonitorReconciler{Client: client, Scheme: scheme, Clients: factory}

	ctx := context.Background()
	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: monitor.Name, Namespace: monitor.Namespace}})
	assert.NoError(t, err, "reconcile")
	assert.Equal(t, "requeueAfter", res.RequeueAfter, requeueIntervalOnError)

	updated := &monitoringv1alpha1.BetterStackMonitor{}
	assert.NoError(t, client.Get(ctx, types.NamespacedName{Name: monitor.Name, Namespace: monitor.Namespace}, updated), "fetch updated monitor")
	assert.String(t, "monitor id", updated.Status.MonitorID, "remote-123")

	syncCond := controllertest.FindCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionSync)
	assert.NotNil(t, "sync condition", syncCond)
	assert.Equal(t, "sync status", syncCond.Status, metav1.ConditionFalse)
	assert.String(t, "sync reason", syncCond.Reason, "SyncFailed")
	readyCond := controllertest.FindCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionReady)
	assert.NotNil(t, "ready condition", readyCond)
	assert.Equal(t, "ready status", readyCond.Status, metav1.ConditionFalse)
	assert.String(t, "ready reason", readyCond.Reason, "SyncFailed")
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
	assert.NoError(t, err, "reconcile")
	assert.Equal(t, "requeueAfter", res.RequeueAfter, requeueIntervalOnError)

	updated := &monitoringv1alpha1.BetterStackMonitor{}
	assert.NoError(t, client.Get(ctx, types.NamespacedName{Name: monitor.Name, Namespace: monitor.Namespace}, updated), "fetch updated monitor")
	assert.String(t, "monitor id", updated.Status.MonitorID, "")

	syncCond := controllertest.FindCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionSync)
	assert.NotNil(t, "sync condition", syncCond)
	assert.Equal(t, "sync status", syncCond.Status, metav1.ConditionFalse)
	assert.String(t, "sync reason", syncCond.Reason, "SyncFailed")
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
			assert.String(t, "delete id", id, "remote-123")
			deleted = true
			return nil
		},
	}
	factory := &fakeBetterStackMonitorClientFactory{monitor: service}

	r := &BetterStackMonitorReconciler{Client: client, Scheme: scheme, Clients: factory}

	ctx := context.Background()
	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: monitor.Name, Namespace: monitor.Namespace}})
	assert.NoError(t, err, "reconcile")
	assert.Bool(t, "requeue", res.Requeue, false)
	assert.Equal(t, "requeueAfter", res.RequeueAfter, time.Duration(0))
	assert.Bool(t, "delete issued", deleted, true)

	updated := &monitoringv1alpha1.BetterStackMonitor{}
	err = client.Get(ctx, types.NamespacedName{Name: monitor.Name, Namespace: monitor.Namespace}, updated)
	if apierrors.IsNotFound(err) {
		return
	}
	assert.NoError(t, err, "fetch updated monitor")
	assert.Bool(t, "finalizer present", controllerutil.ContainsFinalizer(updated, monitoringv1alpha1.BetterStackMonitorFinalizer), false)
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
	assert.NoError(t, err, "reconcile")
	assert.Bool(t, "requeue", res.Requeue, false)
	assert.Equal(t, "requeueAfter", res.RequeueAfter, time.Duration(0))

	updated := &monitoringv1alpha1.BetterStackMonitor{}
	err = client.Get(ctx, types.NamespacedName{Name: monitor.Name, Namespace: monitor.Namespace}, updated)
	if apierrors.IsNotFound(err) {
		assert.Int(t, "monitor factory calls", factory.monitorCalls, 0)
		return
	}
	assert.NoError(t, err, "fetch updated monitor")
	assert.Bool(t, "finalizer present", controllerutil.ContainsFinalizer(updated, monitoringv1alpha1.BetterStackMonitorFinalizer), false)
	assert.Int(t, "monitor factory calls", factory.monitorCalls, 0)
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
	assert.NoError(t, err, "reconcile")
	assert.Bool(t, "requeue", res.Requeue, false)
	assert.Equal(t, "requeueAfter", res.RequeueAfter, time.Duration(0))

	updated := &monitoringv1alpha1.BetterStackMonitor{}
	err = client.Get(ctx, types.NamespacedName{Name: monitor.Name, Namespace: monitor.Namespace}, updated)
	if apierrors.IsNotFound(err) {
		return
	}
	assert.NoError(t, err, "fetch updated monitor")
	assert.Bool(t, "finalizer present", controllerutil.ContainsFinalizer(updated, monitoringv1alpha1.BetterStackMonitorFinalizer), false)
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

	_, err := r.fetchAPIToken(ctx, "default", corev1.SecretKeySelector{})
	assert.Error(t, err, "expected error when name empty")

	_, err = r.fetchAPIToken(ctx, "default", corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "missing"}, Key: "token"})
	assert.Error(t, err, "expected error for missing secret")

	_, err = r.fetchAPIToken(ctx, "default", corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "creds"}, Key: "missing"})
	assert.Error(t, err, "expected error for missing key")

	emptySecret := secret.DeepCopy()
	emptySecret.Data = map[string][]byte{"token": nil}
	assert.NoError(t, client.Update(ctx, emptySecret), "update secret")
	_, err = r.fetchAPIToken(ctx, "default", corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "creds"}, Key: "token"})
	assert.Error(t, err, "expected error for empty token")

	// restore token to verify happy path
	assert.NoError(t, client.Update(ctx, secret.DeepCopy()), "restore secret")
	token, err := r.fetchAPIToken(ctx, "default", corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "creds"}, Key: "token"})
	assert.NoError(t, err, "fetch token")
	assert.String(t, "token", token, "abcd")
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
	assert.Error(t, err, "expected status patch failure")
	assert.String(t, "error", err.Error(), "status patch failed")
	assert.Bool(t, "requeue", res.Requeue, false)
	assert.Equal(t, "requeueAfter", res.RequeueAfter, time.Duration(0))
	assert.Int(t, "status attempts", failingClient.Calls(), 2)
}

func TestBuildMonitorRequest(t *testing.T) {
	spec := monitoringv1alpha1.BetterStackMonitorSpec{
		URL:                       "https://example.com",
		Name:                      "Example",
		MonitorType:               "status",
		TeamName:                  "SRE",
		CheckFrequencyMinutes:     3,
		Regions:                   []string{"us", "eu"},
		RequestMethod:             "post",
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
	assert.NoError(t, err, "marshal want")
	wanted := map[string]any{}
	assert.NoError(t, json.Unmarshal(wantedJSON, &wanted), "unmarshal want")

	gotReq := buildMonitorRequest(spec, nil)
	encoded, err := json.Marshal(gotReq)
	assert.NoError(t, err, "marshal request")
	got := map[string]any{}
	assert.NoError(t, json.Unmarshal(encoded, &got), "unmarshal request")
	assert.Int(t, "diff len", len(diffMaps(got, wanted)), 0)
}

func TestBuildMonitorRequestConvertsTimeoutForServerMonitors(t *testing.T) {
	spec := monitoringv1alpha1.BetterStackMonitorSpec{
		URL:                   "tcp://example.com",
		MonitorType:           "tcp",
		RequestTimeoutSeconds: 3,
	}

	req := buildMonitorRequest(spec, nil)
	assert.NotNil(t, "request timeout", req.RequestTimeout)
	assert.Int(t, "timeout", *req.RequestTimeout, 3000)
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
	assert.Int(t, "headers len", len(req.RequestHeaders), 1)
	assert.NotNil(t, "header id", req.RequestHeaders[0].ID)
	assert.String(t, "header id value", *req.RequestHeaders[0].ID, existingHeaderID)
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
