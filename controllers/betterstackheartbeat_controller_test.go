package controllers

import (
	"context"
	"encoding/json"
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
	"loks0n/betterstack-operator/internal/testutil/controllertest"
	"loks0n/betterstack-operator/pkg/betterstack"
)

type fakeBetterStackHeartbeatClientFactory struct {
	t                    *testing.T
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
		if f.t != nil {
			f.t.Fatalf("heartbeat service not provided")
		}
		return &fakeHeartbeatService{}
	}
	return f.heartbeat
}

type fakeHeartbeatService struct {
	t *testing.T

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
	if err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}
	if res.Requeue || res.RequeueAfter != 0 {
		t.Fatalf("expected no explicit requeue, got %#v", res)
	}

	updated := &monitoringv1alpha1.BetterStackHeartbeat{}
	if err := client.Get(ctx, types.NamespacedName{Name: heartbeat.Name, Namespace: heartbeat.Namespace}, updated); err != nil {
		t.Fatalf("failed to fetch updated heartbeat: %v", err)
	}
	if !controllerutil.ContainsFinalizer(updated, monitoringv1alpha1.BetterStackHeartbeatFinalizer) {
		t.Fatalf("expected finalizer to be present")
	}
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
	if err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}
	if res.RequeueAfter != requeueIntervalOnError {
		t.Fatalf("expected requeue after %v, got %v", requeueIntervalOnError, res.RequeueAfter)
	}

	updated := &monitoringv1alpha1.BetterStackHeartbeat{}
	if err := client.Get(ctx, types.NamespacedName{Name: heartbeat.Name, Namespace: heartbeat.Namespace}, updated); err != nil {
		t.Fatalf("failed to fetch updated heartbeat: %v", err)
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
			if req.Name == nil || *req.Name != "Example" {
				t.Fatalf("unexpected name: %+v", req.Name)
			}
			if req.TeamName == nil || *req.TeamName != "SRE" {
				t.Fatalf("unexpected team name: %+v", req.TeamName)
			}
			if req.Period == nil || *req.Period != 60 {
				t.Fatalf("unexpected period: %+v", req.Period)
			}
			if req.Grace == nil || *req.Grace != 30 {
				t.Fatalf("unexpected grace: %+v", req.Grace)
			}
			if req.TeamWait == nil || *req.TeamWait != 120 {
				t.Fatalf("unexpected team wait: %+v", req.TeamWait)
			}
			if req.PolicyID == nil || *req.PolicyID != "policy-1" {
				t.Fatalf("unexpected policy id: %+v", req.PolicyID)
			}
			if req.HeartbeatGroupID == nil || *req.HeartbeatGroupID != 7 {
				t.Fatalf("unexpected heartbeat group id: %+v", req.HeartbeatGroupID)
			}
			if req.SortIndex == nil || *req.SortIndex != 12 {
				t.Fatalf("unexpected sort index: %+v", req.SortIndex)
			}
			if req.Paused == nil || *req.Paused != true {
				t.Fatalf("expected paused true, got %+v", req.Paused)
			}
			if len(req.MaintenanceDays) != 2 {
				t.Fatalf("unexpected maintenance days: %+v", req.MaintenanceDays)
			}
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
	if err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}
	if res.Requeue || res.RequeueAfter != 0 {
		t.Fatalf("expected no requeue, got %#v", res)
	}

	updated := &monitoringv1alpha1.BetterStackHeartbeat{}
	if err := client.Get(ctx, types.NamespacedName{Name: heartbeat.Name, Namespace: heartbeat.Namespace}, updated); err != nil {
		t.Fatalf("failed to fetch updated heartbeat: %v", err)
	}

	if updated.Status.HeartbeatID != "new-id" {
		t.Fatalf("expected heartbeat id new-id, got %q", updated.Status.HeartbeatID)
	}
	if updated.Status.ObservedGeneration != heartbeat.Generation {
		t.Fatalf("expected observed generation %d, got %d", heartbeat.Generation, updated.Status.ObservedGeneration)
	}
	ready := controllertest.FindCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionReady)
	if ready == nil || ready.Status != metav1.ConditionTrue || ready.Reason != "HeartbeatSynced" {
		t.Fatalf("unexpected ready condition: %+v", ready)
	}
	if factory.heartbeatCalls != 1 {
		t.Fatalf("expected heartbeat factory to be invoked once, got %d", factory.heartbeatCalls)
	}
	if factory.lastHeartbeatToken != "abcd" {
		t.Fatalf("expected token 'abcd', got %q", factory.lastHeartbeatToken)
	}
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
	if err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}
	if res.RequeueAfter != requeueIntervalOnError {
		t.Fatalf("expected requeue after error, got %#v", res)
	}

	updated := &monitoringv1alpha1.BetterStackHeartbeat{}
	if err := client.Get(ctx, types.NamespacedName{Name: heartbeat.Name, Namespace: heartbeat.Namespace}, updated); err != nil {
		t.Fatalf("failed to fetch updated heartbeat: %v", err)
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
	if err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}
	if res.RequeueAfter != requeueIntervalOnError {
		t.Fatalf("expected requeue after error, got %#v", res)
	}

	updated := &monitoringv1alpha1.BetterStackHeartbeat{}
	if err := client.Get(ctx, types.NamespacedName{Name: heartbeat.Name, Namespace: heartbeat.Namespace}, updated); err != nil {
		t.Fatalf("failed to fetch updated heartbeat: %v", err)
	}
	if updated.Status.HeartbeatID != "" {
		t.Fatalf("heartbeat id should remain empty, got %q", updated.Status.HeartbeatID)
	}

	syncCond := controllertest.FindCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionSync)
	if syncCond == nil || syncCond.Status != metav1.ConditionFalse || syncCond.Reason != "SyncFailed" {
		t.Fatalf("unexpected sync condition: %+v", syncCond)
	}
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
			if id != "remote-123" {
				t.Fatalf("unexpected delete id %s", id)
			}
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
	if err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}
	if res.Requeue || res.RequeueAfter != 0 {
		t.Fatalf("expected no requeue, got %#v", res)
	}
	if !deleted {
		t.Fatalf("expected delete request to be issued")
	}

	updated := &monitoringv1alpha1.BetterStackHeartbeat{}
	err = client.Get(ctx, types.NamespacedName{Name: heartbeat.Name, Namespace: heartbeat.Namespace}, updated)
	if err == nil {
		if controllerutil.ContainsFinalizer(updated, monitoringv1alpha1.BetterStackHeartbeatFinalizer) {
			t.Fatalf("expected finalizer to be removed")
		}
	} else if !apierrors.IsNotFound(err) {
		t.Fatalf("failed to fetch updated heartbeat: %v", err)
	}
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
	if err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}
	if res.Requeue || res.RequeueAfter != 0 {
		t.Fatalf("expected no requeue, got %#v", res)
	}

	updated := &monitoringv1alpha1.BetterStackHeartbeat{}
	err = client.Get(ctx, types.NamespacedName{Name: heartbeat.Name, Namespace: heartbeat.Namespace}, updated)
	if err == nil {
		if controllerutil.ContainsFinalizer(updated, monitoringv1alpha1.BetterStackHeartbeatFinalizer) {
			t.Fatalf("expected finalizer to be removed")
		}
	} else if !apierrors.IsNotFound(err) {
		t.Fatalf("failed to fetch updated heartbeat: %v", err)
	}
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
	if err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}
	if res.Requeue || res.RequeueAfter != 0 {
		t.Fatalf("expected no requeue, got %#v", res)
	}

	updated := &monitoringv1alpha1.BetterStackHeartbeat{}
	err = client.Get(ctx, types.NamespacedName{Name: heartbeat.Name, Namespace: heartbeat.Namespace}, updated)
	if err == nil {
		if controllerutil.ContainsFinalizer(updated, monitoringv1alpha1.BetterStackHeartbeatFinalizer) {
			t.Fatalf("expected finalizer to be removed")
		}
	} else if !apierrors.IsNotFound(err) {
		t.Fatalf("failed to fetch updated heartbeat: %v", err)
	}
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
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}

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

	if diff := diffMaps(got, expected); len(diff) > 0 {
		t.Fatalf("unexpected request diff: %v", diff)
	}
}
