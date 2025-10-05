package controllers

import (
	"context"
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

type fakeBetterStackMonitorGroupClientFactory struct {
	group       betterstack.MonitorGroupClient
	calls       int
	lastBaseURL string
	lastToken   string
}

func (f *fakeBetterStackMonitorGroupClientFactory) MonitorGroup(baseURL, token string, _ *http.Client) betterstack.MonitorGroupClient {
	f.calls++
	f.lastBaseURL = baseURL
	f.lastToken = token
	if f.group == nil {
		return &fakeMonitorGroupService{}
	}
	return f.group
}

type fakeMonitorGroupService struct {
	createFn func(ctx context.Context, req betterstack.MonitorGroupCreateRequest) (betterstack.MonitorGroup, error)
	updateFn func(ctx context.Context, id string, req betterstack.MonitorGroupUpdateRequest) (betterstack.MonitorGroup, error)
	deleteFn func(ctx context.Context, id string) error
	getFn    func(ctx context.Context, id string) (betterstack.MonitorGroup, error)

	listFn    func(ctx context.Context) ([]betterstack.MonitorGroup, error)
	listMonFn func(ctx context.Context, groupID string) ([]betterstack.Monitor, error)

	createCalls  int
	updateCalls  int
	deleteCalls  int
	getCalls     int
	listCalls    int
	listMonCalls int

	lastCreateReq betterstack.MonitorGroupCreateRequest
	lastUpdateReq betterstack.MonitorGroupUpdateRequest
}

func (s *fakeMonitorGroupService) Create(ctx context.Context, req betterstack.MonitorGroupCreateRequest) (betterstack.MonitorGroup, error) {
	s.createCalls++
	s.lastCreateReq = req
	if s.createFn != nil {
		return s.createFn(ctx, req)
	}
	return betterstack.MonitorGroup{}, nil
}

func (s *fakeMonitorGroupService) Update(ctx context.Context, id string, req betterstack.MonitorGroupUpdateRequest) (betterstack.MonitorGroup, error) {
	s.updateCalls++
	s.lastUpdateReq = req
	if s.updateFn != nil {
		return s.updateFn(ctx, id, req)
	}
	return betterstack.MonitorGroup{}, nil
}

func (s *fakeMonitorGroupService) Delete(ctx context.Context, id string) error {
	s.deleteCalls++
	if s.deleteFn != nil {
		return s.deleteFn(ctx, id)
	}
	return nil
}

func (s *fakeMonitorGroupService) Get(ctx context.Context, id string) (betterstack.MonitorGroup, error) {
	s.getCalls++
	if s.getFn != nil {
		return s.getFn(ctx, id)
	}
	return betterstack.MonitorGroup{}, nil
}

func (s *fakeMonitorGroupService) List(ctx context.Context) ([]betterstack.MonitorGroup, error) {
	s.listCalls++
	if s.listFn != nil {
		return s.listFn(ctx)
	}
	return nil, nil
}

func (s *fakeMonitorGroupService) ListMonitors(ctx context.Context, groupID string) ([]betterstack.Monitor, error) {
	s.listMonCalls++
	if s.listMonFn != nil {
		return s.listMonFn(ctx, groupID)
	}
	return nil, nil
}

var _ betterstack.MonitorGroupClient = (*fakeMonitorGroupService)(nil)

func TestMonitorGroupReconcileAddsFinalizer(t *testing.T) {
	scheme := controllertest.NewScheme(t)

	group := &monitoringv1alpha1.BetterStackMonitorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example",
			Namespace: "default",
		},
		Spec: monitoringv1alpha1.BetterStackMonitorGroupSpec{
			APITokenSecretRef: corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "api"},
				Key:                  "token",
			},
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(group).
		WithObjects(group.DeepCopy()).
		Build()

	r := &BetterStackMonitorGroupReconciler{
		Client: client,
		Scheme: scheme,
	}

	ctx := context.Background()
	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: group.Name, Namespace: group.Namespace}})
	assert.NoError(t, err, "reconcile")
	assert.Equal(t, "requeueAfter", res.RequeueAfter, time.Duration(0))

	updated := &monitoringv1alpha1.BetterStackMonitorGroup{}
	assert.NoError(t, client.Get(ctx, types.NamespacedName{Name: group.Name, Namespace: group.Namespace}, updated), "fetch updated group")
	assert.Bool(t, "finalizer present", controllerutil.ContainsFinalizer(updated, monitoringv1alpha1.BetterStackMonitorGroupFinalizer), true)
}

func TestMonitorGroupReconcileMissingCredentials(t *testing.T) {
	scheme := controllertest.NewScheme(t)

	group := &monitoringv1alpha1.BetterStackMonitorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "example",
			Namespace:  "default",
			Generation: 5,
			Finalizers: []string{monitoringv1alpha1.BetterStackMonitorGroupFinalizer},
		},
		Spec: monitoringv1alpha1.BetterStackMonitorGroupSpec{
			APITokenSecretRef: corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "missing"},
				Key:                  "token",
			},
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(group).
		WithObjects(group.DeepCopy()).
		Build()

	r := &BetterStackMonitorGroupReconciler{
		Client: client,
		Scheme: scheme,
	}

	ctx := context.Background()
	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: group.Name, Namespace: group.Namespace}})
	assert.NoError(t, err, "reconcile")
	assert.Equal(t, "requeueAfter", res.RequeueAfter, requeueIntervalOnError)

	updated := &monitoringv1alpha1.BetterStackMonitorGroup{}
	assert.NoError(t, client.Get(ctx, types.NamespacedName{Name: group.Name, Namespace: group.Namespace}, updated), "fetch updated group")

	creds := controllertest.FindCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionCredentials)
	assert.NotNil(t, "credentials condition", creds)
	assert.Equal(t, "credentials status", creds.Status, metav1.ConditionFalse)
	assert.String(t, "credentials reason", creds.Reason, "TokenUnavailable")

	ready := controllertest.FindCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionReady)
	assert.NotNil(t, "ready condition", ready)
	assert.Equal(t, "ready status", ready.Status, metav1.ConditionFalse)
	assert.String(t, "ready reason", ready.Reason, "TokenUnavailable")
}

func TestMonitorGroupReconcileCreatesGroup(t *testing.T) {
	scheme := controllertest.NewScheme(t)

	group := &monitoringv1alpha1.BetterStackMonitorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "example",
			Namespace:  "default",
			Generation: 2,
			Finalizers: []string{monitoringv1alpha1.BetterStackMonitorGroupFinalizer},
		},
		Spec: monitoringv1alpha1.BetterStackMonitorGroupSpec{
			Name:     "Backend services",
			TeamName: "Team A",
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

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(group).
		WithObjects(group.DeepCopy(), secret.DeepCopy()).
		Build()

	service := &fakeMonitorGroupService{
		createFn: func(ctx context.Context, req betterstack.MonitorGroupCreateRequest) (betterstack.MonitorGroup, error) {
			assert.NotNil(t, "request name", req.Name)
			assert.String(t, "request name", *req.Name, "Backend services")
			assert.NotNil(t, "request team", req.TeamName)
			assert.String(t, "request team", *req.TeamName, "Team A")
			return betterstack.MonitorGroup{ID: "group-123"}, nil
		},
	}

	factory := &fakeBetterStackMonitorGroupClientFactory{group: service}

	r := &BetterStackMonitorGroupReconciler{
		Client:  client,
		Scheme:  scheme,
		Clients: factory,
	}

	ctx := context.Background()
	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: group.Name, Namespace: group.Namespace}})
	assert.NoError(t, err, "reconcile")
	assert.Equal(t, "requeueAfter", res.RequeueAfter, time.Duration(0))

	updated := &monitoringv1alpha1.BetterStackMonitorGroup{}
	assert.NoError(t, client.Get(ctx, types.NamespacedName{Name: group.Name, Namespace: group.Namespace}, updated), "fetch updated group")
	assert.String(t, "group id", updated.Status.MonitorGroupID, "group-123")
	assert.Equal(t, "observed generation", updated.Status.ObservedGeneration, int64(2))
	assert.NotNil(t, "last synced", updated.Status.LastSyncedTime)

	syncCond := controllertest.FindCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionSync)
	assert.NotNil(t, "sync condition", syncCond)
	assert.Equal(t, "sync status", syncCond.Status, metav1.ConditionTrue)
	assert.String(t, "sync reason", syncCond.Reason, "MonitorGroupSynced")
}

func TestMonitorGroupReconcileUpdatesGroup(t *testing.T) {
	scheme := controllertest.NewScheme(t)

	paused := true
	sortIndex := 5

	group := &monitoringv1alpha1.BetterStackMonitorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "example",
			Namespace:  "default",
			Generation: 4,
			Finalizers: []string{monitoringv1alpha1.BetterStackMonitorGroupFinalizer},
		},
		Spec: monitoringv1alpha1.BetterStackMonitorGroupSpec{
			Name:      "Backend",
			TeamName:  "Team B",
			SortIndex: ptr.To(sortIndex),
			Paused:    ptr.To(paused),
			APITokenSecretRef: corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "api"},
				Key:                  "token",
			},
		},
		Status: monitoringv1alpha1.BetterStackMonitorGroupStatus{
			MonitorGroupID: "group-123",
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
		Data:       map[string][]byte{"token": []byte("abcd")},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(group).
		WithObjects(group.DeepCopy(), secret.DeepCopy()).
		Build()

	service := &fakeMonitorGroupService{
		updateFn: func(ctx context.Context, id string, req betterstack.MonitorGroupUpdateRequest) (betterstack.MonitorGroup, error) {
			assert.String(t, "update id", id, "group-123")
			assert.NotNil(t, "update name", req.Name)
			assert.String(t, "update name", *req.Name, "Backend")
			assert.NotNil(t, "update team", req.TeamName)
			assert.String(t, "update team", *req.TeamName, "Team B")
			assert.NotNil(t, "update sort index", req.SortIndex)
			assert.Equal(t, "update sort index", *req.SortIndex, sortIndex)
			assert.NotNil(t, "update paused", req.Paused)
			assert.Equal(t, "update paused", *req.Paused, paused)
			return betterstack.MonitorGroup{ID: "group-123"}, nil
		},
	}

	factory := &fakeBetterStackMonitorGroupClientFactory{group: service}

	r := &BetterStackMonitorGroupReconciler{
		Client:  client,
		Scheme:  scheme,
		Clients: factory,
	}

	ctx := context.Background()
	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: group.Name, Namespace: group.Namespace}})
	assert.NoError(t, err, "reconcile")
	assert.Equal(t, "requeueAfter", res.RequeueAfter, time.Duration(0))

	updated := &monitoringv1alpha1.BetterStackMonitorGroup{}
	assert.NoError(t, client.Get(ctx, types.NamespacedName{Name: group.Name, Namespace: group.Namespace}, updated), "fetch updated group")
	assert.String(t, "group id", updated.Status.MonitorGroupID, "group-123")
	assert.Equal(t, "observed generation", updated.Status.ObservedGeneration, int64(4))
}

func TestMonitorGroupReconcileUpdateMissingCreatesGroup(t *testing.T) {
	scheme := controllertest.NewScheme(t)

	group := &monitoringv1alpha1.BetterStackMonitorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "example",
			Namespace:  "default",
			Generation: 3,
			Finalizers: []string{monitoringv1alpha1.BetterStackMonitorGroupFinalizer},
		},
		Spec: monitoringv1alpha1.BetterStackMonitorGroupSpec{
			Name: "Backend",
			APITokenSecretRef: corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "api"},
				Key:                  "token",
			},
		},
		Status: monitoringv1alpha1.BetterStackMonitorGroupStatus{
			MonitorGroupID: "group-123",
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
		Data:       map[string][]byte{"token": []byte("abcd")},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(group).
		WithObjects(group.DeepCopy(), secret.DeepCopy()).
		Build()

	service := &fakeMonitorGroupService{
		updateFn: func(ctx context.Context, id string, req betterstack.MonitorGroupUpdateRequest) (betterstack.MonitorGroup, error) {
			return betterstack.MonitorGroup{}, &betterstack.APIError{StatusCode: http.StatusNotFound}
		},
		createFn: func(ctx context.Context, req betterstack.MonitorGroupCreateRequest) (betterstack.MonitorGroup, error) {
			return betterstack.MonitorGroup{ID: "new-group"}, nil
		},
	}

	factory := &fakeBetterStackMonitorGroupClientFactory{group: service}

	r := &BetterStackMonitorGroupReconciler{
		Client:  client,
		Scheme:  scheme,
		Clients: factory,
	}

	ctx := context.Background()
	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: group.Name, Namespace: group.Namespace}})
	assert.NoError(t, err, "reconcile")
	assert.Equal(t, "requeueAfter", res.RequeueAfter, time.Duration(0))

	updated := &monitoringv1alpha1.BetterStackMonitorGroup{}
	assert.NoError(t, client.Get(ctx, types.NamespacedName{Name: group.Name, Namespace: group.Namespace}, updated), "fetch updated group")
	assert.String(t, "group id", updated.Status.MonitorGroupID, "new-group")
}

func TestMonitorGroupReconcileHandlesUpdateError(t *testing.T) {
	scheme := controllertest.NewScheme(t)

	group := &monitoringv1alpha1.BetterStackMonitorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "example",
			Namespace:  "default",
			Generation: 3,
			Finalizers: []string{monitoringv1alpha1.BetterStackMonitorGroupFinalizer},
		},
		Spec: monitoringv1alpha1.BetterStackMonitorGroupSpec{
			Name: "Backend",
			APITokenSecretRef: corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "api"},
				Key:                  "token",
			},
		},
		Status: monitoringv1alpha1.BetterStackMonitorGroupStatus{
			MonitorGroupID: "group-123",
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
		Data:       map[string][]byte{"token": []byte("abcd")},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(group).
		WithObjects(group.DeepCopy(), secret.DeepCopy()).
		Build()

	service := &fakeMonitorGroupService{
		updateFn: func(ctx context.Context, id string, req betterstack.MonitorGroupUpdateRequest) (betterstack.MonitorGroup, error) {
			return betterstack.MonitorGroup{}, fmt.Errorf("api failure")
		},
	}
	factory := &fakeBetterStackMonitorGroupClientFactory{group: service}

	r := &BetterStackMonitorGroupReconciler{Client: client, Scheme: scheme, Clients: factory}

	ctx := context.Background()
	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: group.Name, Namespace: group.Namespace}})
	assert.NoError(t, err, "reconcile")
	assert.Equal(t, "requeueAfter", res.RequeueAfter, requeueIntervalOnError)

	updated := &monitoringv1alpha1.BetterStackMonitorGroup{}
	assert.NoError(t, client.Get(ctx, types.NamespacedName{Name: group.Name, Namespace: group.Namespace}, updated), "fetch updated group")

	syncCond := controllertest.FindCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionSync)
	assert.NotNil(t, "sync condition", syncCond)
	assert.Equal(t, "sync status", syncCond.Status, metav1.ConditionFalse)
	assert.String(t, "sync reason", syncCond.Reason, "SyncFailed")

	ready := controllertest.FindCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionReady)
	assert.NotNil(t, "ready condition", ready)
	assert.Equal(t, "ready status", ready.Status, metav1.ConditionFalse)
	assert.String(t, "ready reason", ready.Reason, "SyncFailed")
}

func TestMonitorGroupReconcileHandlesCreateError(t *testing.T) {
	scheme := controllertest.NewScheme(t)

	group := &monitoringv1alpha1.BetterStackMonitorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "example",
			Namespace:  "default",
			Generation: 3,
			Finalizers: []string{monitoringv1alpha1.BetterStackMonitorGroupFinalizer},
		},
		Spec: monitoringv1alpha1.BetterStackMonitorGroupSpec{
			Name: "Backend",
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

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(group).
		WithObjects(group.DeepCopy(), secret.DeepCopy()).
		Build()

	service := &fakeMonitorGroupService{
		createFn: func(ctx context.Context, req betterstack.MonitorGroupCreateRequest) (betterstack.MonitorGroup, error) {
			return betterstack.MonitorGroup{}, fmt.Errorf("create failed")
		},
	}
	factory := &fakeBetterStackMonitorGroupClientFactory{group: service}

	r := &BetterStackMonitorGroupReconciler{Client: client, Scheme: scheme, Clients: factory}

	ctx := context.Background()
	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: group.Name, Namespace: group.Namespace}})
	assert.NoError(t, err, "reconcile")
	assert.Equal(t, "requeueAfter", res.RequeueAfter, requeueIntervalOnError)

	updated := &monitoringv1alpha1.BetterStackMonitorGroup{}
	assert.NoError(t, client.Get(ctx, types.NamespacedName{Name: group.Name, Namespace: group.Namespace}, updated), "fetch updated group")

	syncCond := controllertest.FindCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionSync)
	assert.NotNil(t, "sync condition", syncCond)
	assert.Equal(t, "sync status", syncCond.Status, metav1.ConditionFalse)
	assert.String(t, "sync reason", syncCond.Reason, "SyncFailed")

	ready := controllertest.FindCondition(updated.Status.Conditions, monitoringv1alpha1.ConditionReady)
	assert.NotNil(t, "ready condition", ready)
	assert.Equal(t, "ready status", ready.Status, metav1.ConditionFalse)
	assert.String(t, "ready reason", ready.Reason, "SyncFailed")
}

func TestMonitorGroupReconcileStatusPatchFailure(t *testing.T) {
	scheme := controllertest.NewScheme(t)

	group := &monitoringv1alpha1.BetterStackMonitorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "example",
			Namespace:  "default",
			Generation: 2,
			Finalizers: []string{monitoringv1alpha1.BetterStackMonitorGroupFinalizer},
		},
		Spec: monitoringv1alpha1.BetterStackMonitorGroupSpec{
			Name: "Backend",
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

	baseClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(group).
		WithObjects(group.DeepCopy(), secret.DeepCopy()).
		Build()

	failingClient := &controllertest.FailingStatusClient{Client: baseClient, FailOn: 2}
	service := &fakeMonitorGroupService{
		createFn: func(ctx context.Context, req betterstack.MonitorGroupCreateRequest) (betterstack.MonitorGroup, error) {
			return betterstack.MonitorGroup{ID: "group-123"}, nil
		},
	}
	factory := &fakeBetterStackMonitorGroupClientFactory{group: service}

	r := &BetterStackMonitorGroupReconciler{Client: failingClient, Scheme: scheme, Clients: factory}

	ctx := context.Background()
	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: group.Name, Namespace: group.Namespace}})
	assert.Error(t, err, "expected status patch failure")
	assert.String(t, "error", err.Error(), "status patch failed")
	assert.Equal(t, "requeueAfter", res.RequeueAfter, time.Duration(0))
	assert.Int(t, "status attempts", failingClient.Calls(), 2)
}

func TestMonitorGroupHandleDelete(t *testing.T) {
	scheme := controllertest.NewScheme(t)

	group := &monitoringv1alpha1.BetterStackMonitorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "example",
			Namespace:         "default",
			Finalizers:        []string{monitoringv1alpha1.BetterStackMonitorGroupFinalizer},
			DeletionTimestamp: &metav1.Time{Time: metav1.Now().Add(-time.Minute)},
		},
		Spec: monitoringv1alpha1.BetterStackMonitorGroupSpec{
			APITokenSecretRef: corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "api"},
				Key:                  "token",
			},
		},
		Status: monitoringv1alpha1.BetterStackMonitorGroupStatus{
			MonitorGroupID: "group-123",
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
		Data:       map[string][]byte{"token": []byte("abcd")},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(group).
		WithObjects(group.DeepCopy(), secret.DeepCopy()).
		Build()

	deleted := false
	service := &fakeMonitorGroupService{
		deleteFn: func(ctx context.Context, id string) error {
			deleted = true
			assert.String(t, "delete id", id, "group-123")
			return nil
		},
	}

	factory := &fakeBetterStackMonitorGroupClientFactory{group: service}

	r := &BetterStackMonitorGroupReconciler{
		Client:  client,
		Scheme:  scheme,
		Clients: factory,
	}

	ctx := context.Background()
	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: group.Name, Namespace: group.Namespace}})
	assert.NoError(t, err, "reconcile")
	assert.Equal(t, "requeueAfter", res.RequeueAfter, time.Duration(0))
	assert.Bool(t, "deleted", deleted, true)

	updated := &monitoringv1alpha1.BetterStackMonitorGroup{}
	err = client.Get(ctx, types.NamespacedName{Name: group.Name, Namespace: group.Namespace}, updated)
	assert.Error(t, err, "fetch updated group should be not found")
	assert.Bool(t, "not found", apierrors.IsNotFound(err), true)
}

func TestMonitorGroupHandleDeleteMissingCredentials(t *testing.T) {
	scheme := controllertest.NewScheme(t)

	deletionTime := metav1.NewTime(time.Now())
	group := &monitoringv1alpha1.BetterStackMonitorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "example",
			Namespace:         "default",
			Finalizers:        []string{monitoringv1alpha1.BetterStackMonitorGroupFinalizer},
			DeletionTimestamp: &deletionTime,
		},
		Spec: monitoringv1alpha1.BetterStackMonitorGroupSpec{
			APITokenSecretRef: corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "missing"},
				Key:                  "token",
			},
		},
		Status: monitoringv1alpha1.BetterStackMonitorGroupStatus{
			MonitorGroupID: "group-123",
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(group).
		WithObjects(group.DeepCopy()).
		Build()

	factory := &fakeBetterStackMonitorGroupClientFactory{group: &fakeMonitorGroupService{}}

	r := &BetterStackMonitorGroupReconciler{Client: client, Scheme: scheme, Clients: factory}

	ctx := context.Background()
	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: group.Name, Namespace: group.Namespace}})
	assert.NoError(t, err, "reconcile")
	assert.Equal(t, "requeueAfter", res.RequeueAfter, time.Duration(0))

	updated := &monitoringv1alpha1.BetterStackMonitorGroup{}
	err = client.Get(ctx, types.NamespacedName{Name: group.Name, Namespace: group.Namespace}, updated)
	if apierrors.IsNotFound(err) {
		assert.Int(t, "factory calls", factory.calls, 0)
		return
	}
	assert.NoError(t, err, "fetch updated group")
	assert.Bool(t, "finalizer present", controllerutil.ContainsFinalizer(updated, monitoringv1alpha1.BetterStackMonitorGroupFinalizer), false)
	assert.Int(t, "factory calls", factory.calls, 0)
}

func TestMonitorGroupHandleDeleteRemoteNotFound(t *testing.T) {
	scheme := controllertest.NewScheme(t)

	deletionTime := metav1.NewTime(time.Now())
	group := &monitoringv1alpha1.BetterStackMonitorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "example",
			Namespace:         "default",
			Finalizers:        []string{monitoringv1alpha1.BetterStackMonitorGroupFinalizer},
			DeletionTimestamp: &deletionTime,
		},
		Spec: monitoringv1alpha1.BetterStackMonitorGroupSpec{
			APITokenSecretRef: corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "api"},
				Key:                  "token",
			},
		},
		Status: monitoringv1alpha1.BetterStackMonitorGroupStatus{
			MonitorGroupID: "group-123",
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
		Data:       map[string][]byte{"token": []byte("abcd")},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(group).
		WithObjects(group.DeepCopy(), secret.DeepCopy()).
		Build()

	service := &fakeMonitorGroupService{
		deleteFn: func(ctx context.Context, id string) error {
			return &betterstack.APIError{StatusCode: http.StatusNotFound}
		},
	}
	factory := &fakeBetterStackMonitorGroupClientFactory{group: service}

	r := &BetterStackMonitorGroupReconciler{Client: client, Scheme: scheme, Clients: factory}

	ctx := context.Background()
	res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: group.Name, Namespace: group.Namespace}})
	assert.NoError(t, err, "reconcile")
	assert.Equal(t, "requeueAfter", res.RequeueAfter, time.Duration(0))

	updated := &monitoringv1alpha1.BetterStackMonitorGroup{}
	err = client.Get(ctx, types.NamespacedName{Name: group.Name, Namespace: group.Namespace}, updated)
	if apierrors.IsNotFound(err) {
		return
	}
	assert.NoError(t, err, "fetch updated group")
	assert.Bool(t, "finalizer present", controllerutil.ContainsFinalizer(updated, monitoringv1alpha1.BetterStackMonitorGroupFinalizer), false)
}

func TestBuildMonitorGroupRequest(t *testing.T) {
	paused := true
	sortIndex := 7
	spec := monitoringv1alpha1.BetterStackMonitorGroupSpec{
		Name:      "Backend",
		TeamName:  "Team A",
		SortIndex: ptr.To(sortIndex),
		Paused:    ptr.To(paused),
	}

	req := buildMonitorGroupRequest(spec)
	assert.NotNil(t, "name", req.Name)
	assert.String(t, "name", *req.Name, "Backend")
	assert.NotNil(t, "team", req.TeamName)
	assert.String(t, "team", *req.TeamName, "Team A")
	assert.NotNil(t, "sort index", req.SortIndex)
	assert.Equal(t, "sort index", *req.SortIndex, sortIndex)
	assert.NotNil(t, "paused", req.Paused)
	assert.Bool(t, "paused", *req.Paused, true)

	emptyReq := buildMonitorGroupRequest(monitoringv1alpha1.BetterStackMonitorGroupSpec{})
	assert.Nil(t, "empty name", emptyReq.Name)
	assert.Nil(t, "empty team", emptyReq.TeamName)
	assert.Nil(t, "empty sort", emptyReq.SortIndex)
	assert.Nil(t, "empty paused", emptyReq.Paused)
}
