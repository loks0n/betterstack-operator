package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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

type fakeBetterStackHeartbeatClientFactory struct {
	heartbeat            betterstack.HeartbeatClient
	heartbeatCalls       int
	lastHeartbeatBaseURL string
	lastHeartbeatToken   string
}

func (f *fakeBetterStackHeartbeatClientFactory) Heartbeat(baseURL, token string, _ *http.Client) betterstack.HeartbeatClient {
	f.heartbeatCalls++
	f.lastHeartbeatBaseURL = baseURL
	f.lastHeartbeatToken = token
	if f.heartbeat == nil {
		return &fakeHeartbeatService{}
	}
	return f.heartbeat
}

type fakeHeartbeatService struct {
	getFn    func(ctx context.Context, id string) (betterstack.Heartbeat, error)
	updateFn func(ctx context.Context, id string, req betterstack.HeartbeatUpdateRequest) (betterstack.Heartbeat, error)
	createFn func(ctx context.Context, req betterstack.HeartbeatCreateRequest) (betterstack.Heartbeat, error)
	deleteFn func(ctx context.Context, id string) error

	getCalls    int
	updateCalls int
	createCalls int
	deleteCalls int

	lastUpdateReq betterstack.HeartbeatUpdateRequest
	lastCreateReq betterstack.HeartbeatCreateRequest
}

func (s *fakeHeartbeatService) Get(ctx context.Context, id string) (betterstack.Heartbeat, error) {
	s.getCalls++
	if s.getFn != nil {
		return s.getFn(ctx, id)
	}
	return betterstack.Heartbeat{}, nil
}

func (s *fakeHeartbeatService) Update(ctx context.Context, id string, req betterstack.HeartbeatUpdateRequest) (betterstack.Heartbeat, error) {
	s.updateCalls++
	s.lastUpdateReq = req
	if s.updateFn != nil {
		return s.updateFn(ctx, id, req)
	}
	return betterstack.Heartbeat{}, nil
}

func (s *fakeHeartbeatService) Create(ctx context.Context, req betterstack.HeartbeatCreateRequest) (betterstack.Heartbeat, error) {
	s.createCalls++
	s.lastCreateReq = req
	if s.createFn != nil {
		return s.createFn(ctx, req)
	}
	return betterstack.Heartbeat{}, nil
}

func (s *fakeHeartbeatService) Delete(ctx context.Context, id string) error {
	s.deleteCalls++
	if s.deleteFn != nil {
		return s.deleteFn(ctx, id)
	}
	return nil
}

var _ betterstack.HeartbeatClient = (*fakeHeartbeatService)(nil)

func TestHeartbeatReconcileAddsFinalizer(t *testing.T) {
	scheme := controllertest.NewScheme(t)

	heartbeat := &monitoringv1alpha1.BetterStackHeartbeat{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example",
			Namespace: "default",
		},
		Spec: monitoringv1alpha1.BetterStackHeartbeatSpec{
			Name:          "Example",
			PeriodSeconds: 60,
			APITokenSecretRef: corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "api"},
				Key:                  "token",
			},
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(heartbeat).
		WithObjects(heartbeat.DeepCopy()).
		Build()

	r := &BetterStackHeartbeatReconciler{Client: client, Scheme: scheme}

	ctx := context.Background()
	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: heartbeat.Name, Namespace: heartbeat.Namespace}})
	assert.NoError(t, err, "reconcile")
	assert.Equal(t, "requeueAfter", res.RequeueAfter, time.Duration(0))

	updated := &monitoringv1alpha1.BetterStackHeartbeat{}
	assert.NoError(t, client.Get(ctx, types.NamespacedName{Name: heartbeat.Name, Namespace: heartbeat.Namespace}, updated), "fetch updated heartbeat")
	assert.Bool(t, "finalizer present", controllerutil.ContainsFinalizer(updated, monitoringv1alpha1.BetterStackHeartbeatFinalizer), true)
}

func TestHeartbeatReconcileHandlesMissingCredentials(t *testing.T) {
	scheme := controllertest.NewScheme(t)

	heartbeat := &monitoringv1alpha1.BetterStackHeartbeat{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "example",
			Namespace:  "default",
			Generation: 4,
			Finalizers: []string{monitoringv1alpha1.BetterStackHeartbeatFinalizer},
		},
		Spec: monitoringv1alpha1.BetterStackHeartbeatSpec{
			Name:          "Example",
			PeriodSeconds: 60,
			APITokenSecretRef: corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "api"},
				Key:                  "token",
			},
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(heartbeat).
		WithObjects(heartbeat.DeepCopy()).
		Build()

	r := &BetterStackHeartbeatReconciler{Client: client, Scheme: scheme}

	ctx := context.Background()
	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: heartbeat.Name, Namespace: heartbeat.Namespace}})
	assert.NoError(t, err, "reconcile")
	assert.Equal(t, "requeueAfter", res.RequeueAfter, requeueIntervalOnError)

	updated := &monitoringv1alpha1.BetterStackHeartbeat{}
	assert.NoError(t, client.Get(ctx, types.NamespacedName{Name: heartbeat.Name, Namespace: heartbeat.Namespace}, updated), "fetch updated heartbeat")

	creds := controllertest.FindCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionCredentials)
	assert.NotNil(t, "credentials condition", creds)
	assert.Equal(t, "credentials status", creds.Status, metav1.ConditionFalse)
	assert.String(t, "credentials reason", creds.Reason, "TokenUnavailable")
	ready := controllertest.FindCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionReady)
	assert.NotNil(t, "ready condition", ready)
	assert.Equal(t, "ready status", ready.Status, metav1.ConditionFalse)
	assert.String(t, "ready reason", ready.Reason, "TokenUnavailable")
}

func TestHeartbeatReconcileCreatesHeartbeatWhenRemoteMissing(t *testing.T) {
	scheme := controllertest.NewScheme(t)

	group := 7
	sort := 12
	paused := ptr.To(true)
	policy := "policy-1"

	heartbeat := &monitoringv1alpha1.BetterStackHeartbeat{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "example",
			Namespace:  "default",
			Generation: 5,
			Finalizers: []string{monitoringv1alpha1.BetterStackHeartbeatFinalizer},
		},
		Spec: monitoringv1alpha1.BetterStackHeartbeatSpec{
			Name:                "Example",
			TeamName:            "SRE",
			PeriodSeconds:       60,
			GraceSeconds:        30,
			Call:                ptr.To(true),
			SMS:                 ptr.To(false),
			Email:               ptr.To(true),
			Push:                ptr.To(true),
			CriticalAlert:       ptr.To(true),
			TeamWaitSeconds:     120,
			HeartbeatGroupID:    &group,
			SortIndex:           &sort,
			Paused:              paused,
			MaintenanceDays:     []string{"mon", "tue"},
			MaintenanceFrom:     "01:00",
			MaintenanceTo:       "02:00",
			MaintenanceTimezone: "UTC",
			PolicyID:            &policy,
			BaseURL:             "https://api.test",
			APITokenSecretRef: corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "api"},
				Key:                  "token",
			},
		},
		Status: monitoringv1alpha1.BetterStackHeartbeatStatus{
			HeartbeatID: "remote-123",
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
		Data:       map[string][]byte{"token": []byte("abcd")},
	}

	service := &fakeHeartbeatService{
		updateFn: func(ctx context.Context, id string, req betterstack.HeartbeatUpdateRequest) (betterstack.Heartbeat, error) {
			return betterstack.Heartbeat{}, &betterstack.APIError{StatusCode: http.StatusNotFound}
		},
		createFn: func(ctx context.Context, req betterstack.HeartbeatCreateRequest) (betterstack.Heartbeat, error) {
			assert.NotNil(t, "request name", req.Name)
			assert.String(t, "request name", *req.Name, "Example")
			assert.NotNil(t, "request team", req.TeamName)
			assert.String(t, "request team", *req.TeamName, "SRE")
			assert.NotNil(t, "request period", req.Period)
			assert.Int(t, "request period", *req.Period, 60)
			assert.NotNil(t, "request grace", req.Grace)
			assert.Int(t, "request grace", *req.Grace, 30)
			assert.NotNil(t, "request team wait", req.TeamWait)
			assert.Int(t, "request team wait", *req.TeamWait, 120)
			assert.NotNil(t, "request policy", req.PolicyID)
			assert.String(t, "request policy", *req.PolicyID, "policy-1")
			assert.NotNil(t, "request heartbeat group", req.HeartbeatGroupID)
			assert.Int(t, "request heartbeat group", *req.HeartbeatGroupID, 7)
			assert.NotNil(t, "request sort index", req.SortIndex)
			assert.Int(t, "request sort index", *req.SortIndex, 12)
			assert.NotNil(t, "request paused", req.Paused)
			assert.Bool(t, "request paused", *req.Paused, true)
			assert.Int(t, "request maintenance days len", len(req.MaintenanceDays), 2)
			return betterstack.Heartbeat{ID: "new-id"}, nil
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(heartbeat).
		WithObjects(heartbeat.DeepCopy(), secret.DeepCopy()).
		Build()

	factory := &fakeBetterStackHeartbeatClientFactory{heartbeat: service}

	r := &BetterStackHeartbeatReconciler{Client: client, Scheme: scheme, Clients: factory}

	ctx := context.Background()
	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: heartbeat.Name, Namespace: heartbeat.Namespace}})
	assert.NoError(t, err, "reconcile")
	assert.Equal(t, "requeueAfter", res.RequeueAfter, time.Duration(0))

	updated := &monitoringv1alpha1.BetterStackHeartbeat{}
	assert.NoError(t, client.Get(ctx, types.NamespacedName{Name: heartbeat.Name, Namespace: heartbeat.Namespace}, updated), "fetch updated heartbeat")
	assert.String(t, "heartbeat id", updated.Status.HeartbeatID, "new-id")
	assert.Equal(t, "observed generation", updated.Status.ObservedGeneration, heartbeat.Generation)
	ready := controllertest.FindCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionReady)
	assert.NotNil(t, "ready condition", ready)
	assert.Equal(t, "ready status", ready.Status, metav1.ConditionTrue)
	assert.String(t, "ready reason", ready.Reason, "HeartbeatSynced")
	assert.Int(t, "heartbeat factory calls", factory.heartbeatCalls, 1)
	assert.String(t, "last token", factory.lastHeartbeatToken, "abcd")
}

func TestHeartbeatReconcileHandlesUpdateError(t *testing.T) {
	scheme := controllertest.NewScheme(t)

	heartbeat := &monitoringv1alpha1.BetterStackHeartbeat{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "example",
			Namespace:  "default",
			Generation: 2,
			Finalizers: []string{monitoringv1alpha1.BetterStackHeartbeatFinalizer},
		},
		Spec: monitoringv1alpha1.BetterStackHeartbeatSpec{
			Name:          "Example",
			PeriodSeconds: 60,
			BaseURL:       "https://api.test",
			APITokenSecretRef: corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "api"},
				Key:                  "token",
			},
		},
		Status: monitoringv1alpha1.BetterStackHeartbeatStatus{HeartbeatID: "remote-123"},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
		Data:       map[string][]byte{"token": []byte("abcd")},
	}

	service := &fakeHeartbeatService{
		updateFn: func(ctx context.Context, id string, req betterstack.HeartbeatUpdateRequest) (betterstack.Heartbeat, error) {
			return betterstack.Heartbeat{}, &betterstack.APIError{StatusCode: http.StatusInternalServerError, Message: "boom"}
		},
	}
	factory := &fakeBetterStackHeartbeatClientFactory{heartbeat: service}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(heartbeat).
		WithObjects(heartbeat.DeepCopy(), secret.DeepCopy()).
		Build()

	r := &BetterStackHeartbeatReconciler{Client: client, Scheme: scheme, Clients: factory}

	ctx := context.Background()
	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: heartbeat.Name, Namespace: heartbeat.Namespace}})
	assert.NoError(t, err, "reconcile")
	assert.Equal(t, "requeueAfter", res.RequeueAfter, requeueIntervalOnError)

	updated := &monitoringv1alpha1.BetterStackHeartbeat{}
	assert.NoError(t, client.Get(ctx, types.NamespacedName{Name: heartbeat.Name, Namespace: heartbeat.Namespace}, updated), "fetch updated heartbeat")

	syncCond := controllertest.FindCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionSync)
	assert.NotNil(t, "sync condition", syncCond)
	assert.Equal(t, "sync status", syncCond.Status, metav1.ConditionFalse)
	assert.String(t, "sync reason", syncCond.Reason, "SyncFailed")
	readyCond := controllertest.FindCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionReady)
	assert.NotNil(t, "ready condition", readyCond)
	assert.Equal(t, "ready status", readyCond.Status, metav1.ConditionFalse)
	assert.String(t, "ready reason", readyCond.Reason, "SyncFailed")
}

func TestHeartbeatReconcileHandlesCreateError(t *testing.T) {
	scheme := controllertest.NewScheme(t)

	heartbeat := &monitoringv1alpha1.BetterStackHeartbeat{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "example",
			Namespace:  "default",
			Generation: 2,
			Finalizers: []string{monitoringv1alpha1.BetterStackHeartbeatFinalizer},
		},
		Spec: monitoringv1alpha1.BetterStackHeartbeatSpec{
			Name:          "Example",
			PeriodSeconds: 60,
			BaseURL:       "https://api.test",
			APITokenSecretRef: corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "api"},
				Key:                  "token",
			},
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
		Data:       map[string][]byte{"token": []byte("abcd")},
	}

	service := &fakeHeartbeatService{
		createFn: func(ctx context.Context, req betterstack.HeartbeatCreateRequest) (betterstack.Heartbeat, error) {
			return betterstack.Heartbeat{}, &betterstack.APIError{StatusCode: http.StatusInternalServerError, Message: "boom"}
		},
	}
	factory := &fakeBetterStackHeartbeatClientFactory{heartbeat: service}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(heartbeat).
		WithObjects(heartbeat.DeepCopy(), secret.DeepCopy()).
		Build()

	r := &BetterStackHeartbeatReconciler{Client: client, Scheme: scheme, Clients: factory}

	ctx := context.Background()
	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: heartbeat.Name, Namespace: heartbeat.Namespace}})
	assert.NoError(t, err, "reconcile")
	assert.Equal(t, "requeueAfter", res.RequeueAfter, requeueIntervalOnError)

	updated := &monitoringv1alpha1.BetterStackHeartbeat{}
	assert.NoError(t, client.Get(ctx, types.NamespacedName{Name: heartbeat.Name, Namespace: heartbeat.Namespace}, updated), "fetch updated heartbeat")
	assert.String(t, "heartbeat id", updated.Status.HeartbeatID, "")

	syncCond := controllertest.FindCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionSync)
	assert.NotNil(t, "sync condition", syncCond)
	assert.Equal(t, "sync status", syncCond.Status, metav1.ConditionFalse)
	assert.String(t, "sync reason", syncCond.Reason, "SyncFailed")
}

func TestHeartbeatReconcileHandlesQuotaExceeded(t *testing.T) {
	scheme := controllertest.NewScheme(t)

	heartbeat := &monitoringv1alpha1.BetterStackHeartbeat{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "example",
			Namespace:  "default",
			Generation: 4,
			Finalizers: []string{monitoringv1alpha1.BetterStackHeartbeatFinalizer},
		},
		Spec: monitoringv1alpha1.BetterStackHeartbeatSpec{
			Name:          "Example",
			PeriodSeconds: 60,
			BaseURL:       "https://api.test",
			APITokenSecretRef: corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "api"},
				Key:                  "token",
			},
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
		Data:       map[string][]byte{"token": []byte("abcd")},
	}

	service := &fakeHeartbeatService{
		createFn: func(ctx context.Context, req betterstack.HeartbeatCreateRequest) (betterstack.Heartbeat, error) {
			return betterstack.Heartbeat{}, &betterstack.APIError{StatusCode: http.StatusForbidden, Message: "Heartbeat quota reached. Please upgrade your account."}
		},
	}
	factory := &fakeBetterStackHeartbeatClientFactory{heartbeat: service}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(heartbeat).
		WithObjects(heartbeat.DeepCopy(), secret.DeepCopy()).
		Build()

	r := &BetterStackHeartbeatReconciler{Client: client, Scheme: scheme, Clients: factory}

	ctx := context.Background()
	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: heartbeat.Name, Namespace: heartbeat.Namespace}})
	assert.NoError(t, err, "reconcile")
	assert.Equal(t, "requeueAfter", res.RequeueAfter, requeueIntervalOnError)

	updated := &monitoringv1alpha1.BetterStackHeartbeat{}
	assert.NoError(t, client.Get(ctx, types.NamespacedName{Name: heartbeat.Name, Namespace: heartbeat.Namespace}, updated), "fetch updated heartbeat")
	syncCond := controllertest.FindCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionSync)
	assert.NotNil(t, "sync condition", syncCond)
	assert.Equal(t, "sync status", syncCond.Status, metav1.ConditionFalse)
	assert.String(t, "sync reason", syncCond.Reason, ReasonHeartbeatQuotaExceeded)
	assert.String(t, "sync message", syncCond.Message, "Better Stack heartbeat quota reached")
	readyCond := controllertest.FindCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionReady)
	assert.NotNil(t, "ready condition", readyCond)
	assert.Equal(t, "ready status", readyCond.Status, metav1.ConditionFalse)
	assert.String(t, "ready reason", readyCond.Reason, ReasonHeartbeatQuotaExceeded)
	assert.String(t, "ready message", readyCond.Message, "Better Stack heartbeat quota reached")
}

func TestHeartbeatReconcileHandlesDeletion(t *testing.T) {
	scheme := controllertest.NewScheme(t)

	deletionTime := metav1.NewTime(time.Now())
	heartbeat := &monitoringv1alpha1.BetterStackHeartbeat{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "example",
			Namespace:         "default",
			Finalizers:        []string{monitoringv1alpha1.BetterStackHeartbeatFinalizer},
			DeletionTimestamp: &deletionTime,
		},
		Spec: monitoringv1alpha1.BetterStackHeartbeatSpec{
			BaseURL: "https://api.test",
			APITokenSecretRef: corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "api"},
				Key:                  "token",
			},
		},
		Status: monitoringv1alpha1.BetterStackHeartbeatStatus{HeartbeatID: "remote-123"},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
		Data:       map[string][]byte{"token": []byte("abcd")},
	}

	deleted := false
	service := &fakeHeartbeatService{
		deleteFn: func(ctx context.Context, id string) error {
			deleted = true
			assert.String(t, "delete id", id, "remote-123")
			return nil
		},
	}
	factory := &fakeBetterStackHeartbeatClientFactory{heartbeat: service}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(heartbeat).
		WithObjects(heartbeat.DeepCopy(), secret.DeepCopy()).
		Build()

	r := &BetterStackHeartbeatReconciler{Client: client, Scheme: scheme, Clients: factory}

	ctx := context.Background()
	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: heartbeat.Name, Namespace: heartbeat.Namespace}})
	assert.NoError(t, err, "reconcile")
	assert.Equal(t, "requeueAfter", res.RequeueAfter, time.Duration(0))
	assert.Bool(t, "delete issued", deleted, true)

	updated := &monitoringv1alpha1.BetterStackHeartbeat{}
	err = client.Get(ctx, types.NamespacedName{Name: heartbeat.Name, Namespace: heartbeat.Namespace}, updated)
	if apierrors.IsNotFound(err) {
		return
	}
	assert.NoError(t, err, "fetch updated heartbeat")
	assert.Bool(t, "finalizer present", controllerutil.ContainsFinalizer(updated, monitoringv1alpha1.BetterStackHeartbeatFinalizer), false)
}

func TestHeartbeatReconcileHandlesDeletionMissingCredentials(t *testing.T) {
	scheme := controllertest.NewScheme(t)

	deletionTime := metav1.NewTime(time.Now())
	heartbeat := &monitoringv1alpha1.BetterStackHeartbeat{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "example",
			Namespace:         "default",
			Finalizers:        []string{monitoringv1alpha1.BetterStackHeartbeatFinalizer},
			DeletionTimestamp: &deletionTime,
		},
		Spec: monitoringv1alpha1.BetterStackHeartbeatSpec{
			APITokenSecretRef: corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "missing"},
				Key:                  "token",
			},
		},
		Status: monitoringv1alpha1.BetterStackHeartbeatStatus{HeartbeatID: "remote-123"},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(heartbeat).
		WithObjects(heartbeat.DeepCopy()).
		Build()

	r := &BetterStackHeartbeatReconciler{Client: client, Scheme: scheme, Clients: &fakeBetterStackHeartbeatClientFactory{}}

	ctx := context.Background()
	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: heartbeat.Name, Namespace: heartbeat.Namespace}})
	assert.NoError(t, err, "reconcile")
	assert.Equal(t, "requeueAfter", res.RequeueAfter, time.Duration(0))

	updated := &monitoringv1alpha1.BetterStackHeartbeat{}
	err = client.Get(ctx, types.NamespacedName{Name: heartbeat.Name, Namespace: heartbeat.Namespace}, updated)
	if apierrors.IsNotFound(err) {
		return
	}
	assert.NoError(t, err, "fetch updated heartbeat")
	assert.Bool(t, "finalizer present", controllerutil.ContainsFinalizer(updated, monitoringv1alpha1.BetterStackHeartbeatFinalizer), false)
}

func TestHeartbeatReconcileHandlesDeletionRemoteNotFound(t *testing.T) {
	scheme := controllertest.NewScheme(t)

	deletionTime := metav1.NewTime(time.Now())
	heartbeat := &monitoringv1alpha1.BetterStackHeartbeat{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "example",
			Namespace:         "default",
			Finalizers:        []string{monitoringv1alpha1.BetterStackHeartbeatFinalizer},
			DeletionTimestamp: &deletionTime,
		},
		Spec: monitoringv1alpha1.BetterStackHeartbeatSpec{
			BaseURL: "https://api.test",
			APITokenSecretRef: corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "api"},
				Key:                  "token",
			},
		},
		Status: monitoringv1alpha1.BetterStackHeartbeatStatus{HeartbeatID: "remote-123"},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
		Data:       map[string][]byte{"token": []byte("abcd")},
	}

	service := &fakeHeartbeatService{
		deleteFn: func(ctx context.Context, id string) error {
			return &betterstack.APIError{StatusCode: http.StatusNotFound}
		},
	}
	factory := &fakeBetterStackHeartbeatClientFactory{heartbeat: service}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(heartbeat).
		WithObjects(heartbeat.DeepCopy(), secret.DeepCopy()).
		Build()

	r := &BetterStackHeartbeatReconciler{Client: client, Scheme: scheme, Clients: factory}

	ctx := context.Background()
	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: heartbeat.Name, Namespace: heartbeat.Namespace}})
	assert.NoError(t, err, "reconcile")
	assert.Equal(t, "requeueAfter", res.RequeueAfter, time.Duration(0))

	updated := &monitoringv1alpha1.BetterStackHeartbeat{}
	err = client.Get(ctx, types.NamespacedName{Name: heartbeat.Name, Namespace: heartbeat.Namespace}, updated)
	if apierrors.IsNotFound(err) {
		return
	}
	assert.NoError(t, err, "fetch updated heartbeat")
	assert.Bool(t, "finalizer present", controllerutil.ContainsFinalizer(updated, monitoringv1alpha1.BetterStackHeartbeatFinalizer), false)
}

func TestHeartbeatReconcileReturnsErrorWhenStatusPatchFails(t *testing.T) {
	scheme := controllertest.NewScheme(t)

	heartbeat := &monitoringv1alpha1.BetterStackHeartbeat{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "example",
			Namespace:  "default",
			Generation: 3,
			Finalizers: []string{monitoringv1alpha1.BetterStackHeartbeatFinalizer},
		},
		Spec: monitoringv1alpha1.BetterStackHeartbeatSpec{
			Name:          "Example",
			PeriodSeconds: 60,
			BaseURL:       "https://api.test",
			APITokenSecretRef: corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "api"},
				Key:                  "token",
			},
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
		Data:       map[string][]byte{"token": []byte("abcd")},
	}

	service := &fakeHeartbeatService{
		createFn: func(ctx context.Context, req betterstack.HeartbeatCreateRequest) (betterstack.Heartbeat, error) {
			return betterstack.Heartbeat{ID: "new-id"}, nil
		},
	}
	factory := &fakeBetterStackHeartbeatClientFactory{heartbeat: service}

	baseClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(heartbeat).
		WithObjects(heartbeat.DeepCopy(), secret.DeepCopy()).
		Build()

	failingClient := &controllertest.FailingStatusClient{Client: baseClient, FailOn: 2}

	r := &BetterStackHeartbeatReconciler{Client: failingClient, Scheme: scheme, Clients: factory}

	ctx := context.Background()
	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: heartbeat.Name, Namespace: heartbeat.Namespace}})
	assert.Error(t, err, "expected status patch failure")
	assert.String(t, "error", err.Error(), "status patch failed")
	assert.Equal(t, "requeueAfter", res.RequeueAfter, time.Duration(0))
	assert.Int(t, "status attempts", failingClient.Calls(), 2)
}

func TestBuildHeartbeatRequest(t *testing.T) {
	group := 2
	sort := 99
	paused := ptr.To(true)
	policy := "policy-1"
	spec := monitoringv1alpha1.BetterStackHeartbeatSpec{
		TeamName:            "SRE",
		Name:                "Example",
		PeriodSeconds:       90,
		GraceSeconds:        45,
		Call:                ptr.To(true),
		SMS:                 ptr.To(false),
		Email:               ptr.To(true),
		Push:                ptr.To(true),
		CriticalAlert:       ptr.To(false),
		TeamWaitSeconds:     30,
		HeartbeatGroupID:    &group,
		SortIndex:           &sort,
		Paused:              paused,
		MaintenanceDays:     []string{"sat", "sun"},
		MaintenanceFrom:     "03:00",
		MaintenanceTo:       "04:00",
		MaintenanceTimezone: "UTC",
		PolicyID:            &policy,
	}

	req := buildHeartbeatRequest(spec)

	encoded, err := json.Marshal(req)
	assert.NoError(t, err, "marshal request")

	var got map[string]any
	assert.NoError(t, json.Unmarshal(encoded, &got), "unmarshal request")

	expected := map[string]any{
		"team_name":            "SRE",
		"name":                 "Example",
		"period":               float64(90),
		"grace":                float64(45),
		"call":                 true,
		"sms":                  false,
		"email":                true,
		"push":                 true,
		"critical_alert":       false,
		"team_wait":            float64(30),
		"heartbeat_group_id":   float64(2),
		"sort_index":           float64(99),
		"paused":               true,
		"maintenance_days":     []any{"sat", "sun"},
		"maintenance_from":     "03:00",
		"maintenance_to":       "04:00",
		"maintenance_timezone": "UTC",
		"policy_id":            "policy-1",
	}

	diff := diffMaps(got, expected)
	assert.String(t, "diff", fmt.Sprint(diff), "map[]")
}
