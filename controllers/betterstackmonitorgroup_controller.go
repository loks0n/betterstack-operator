package controllers

import (
	"context"
	"fmt"
	"net/http"

	"k8s.io/utils/ptr"

	monitoringv1alpha1 "loks0n/betterstack-operator/api/v1alpha1"
	"loks0n/betterstack-operator/internal/controller/conditions"
	"loks0n/betterstack-operator/internal/controller/credentials"
	"loks0n/betterstack-operator/pkg/betterstack"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// BetterStackMonitorGroupClientFactory provides Better Stack API clients for reconcilers.
type BetterStackMonitorGroupClientFactory interface {
	MonitorGroup(baseURL, token string, httpClient *http.Client) betterstack.MonitorGroupClient
}

type defaultBetterStackMonitorGroupClientFactory struct{}

func (defaultBetterStackMonitorGroupClientFactory) MonitorGroup(baseURL, token string, httpClient *http.Client) betterstack.MonitorGroupClient {
	client := betterstack.NewClient(baseURL, token, httpClient)
	return client.MonitorGroups
}

// BetterStackMonitorGroupReconciler reconciles BetterStackMonitorGroup resources.
type BetterStackMonitorGroupReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	HTTPClient *http.Client
	Clients    BetterStackMonitorGroupClientFactory
}

const monitorGroupSecretIndexKey = "monitoring.betterstack.io/monitorgroup-secret"

//+kubebuilder:rbac:groups=monitoring.betterstack.io,resources=betterstackmonitorgroups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=monitoring.betterstack.io,resources=betterstackmonitorgroups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=monitoring.betterstack.io,resources=betterstackmonitorgroups/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *BetterStackMonitorGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	group := &monitoringv1alpha1.BetterStackMonitorGroup{}
	if err := r.Get(ctx, req.NamespacedName, group); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if group.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(group, monitoringv1alpha1.BetterStackMonitorGroupFinalizer) {
			controllerutil.AddFinalizer(group, monitoringv1alpha1.BetterStackMonitorGroupFinalizer)
			if err := r.Update(ctx, group); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
	} else {
		return r.handleDelete(ctx, group)
	}

	token, err := credentials.FetchAPIToken(ctx, r.Client, group.Namespace, group.Spec.APITokenSecretRef)
	if err != nil {
		logger.Error(err, "unable to fetch Better Stack API token")
		_ = r.patchStatus(ctx, group, func(status *monitoringv1alpha1.BetterStackMonitorGroupStatus) {
			now := metav1.Now()
			status.SetCondition(conditions.New(monitoringv1alpha1.ConditionCredentials, metav1.ConditionFalse, "TokenUnavailable", err.Error(), &now))
			status.SetCondition(conditions.New(monitoringv1alpha1.ConditionReady, metav1.ConditionFalse, "TokenUnavailable", "API credentials not available", &now))
		})
		return ctrl.Result{RequeueAfter: requeueIntervalOnError}, nil
	}

	_ = r.patchStatus(ctx, group, func(status *monitoringv1alpha1.BetterStackMonitorGroupStatus) {
		now := metav1.Now()
		status.SetCondition(conditions.New(monitoringv1alpha1.ConditionCredentials, metav1.ConditionTrue, "TokenResolved", fmt.Sprintf("Using secret %s/%s", group.Namespace, group.Spec.APITokenSecretRef.Name), &now))
	})

	service := r.monitorGroupService(group.Spec.BaseURL, token)
	request := buildMonitorGroupRequest(group.Spec)

	var apiGroup betterstack.MonitorGroup
	if group.Status.MonitorGroupID != "" {
		apiGroup, err = service.Update(ctx, group.Status.MonitorGroupID, betterstack.MonitorGroupUpdateRequest(request))
		if betterstack.IsNotFound(err) {
			logger.Info("remote monitor group missing, creating anew", "id", group.Status.MonitorGroupID)
			group.Status.MonitorGroupID = ""
			err = nil
		}
	}

	if err == nil && group.Status.MonitorGroupID == "" {
		apiGroup, err = service.Create(ctx, betterstack.MonitorGroupCreateRequest(request))
	}

	if err != nil {
		logger.Error(err, "unable to reconcile Better Stack monitor group")
		_ = r.patchStatus(ctx, group, func(status *monitoringv1alpha1.BetterStackMonitorGroupStatus) {
			now := metav1.Now()
			status.SetCondition(conditions.New(monitoringv1alpha1.ConditionSync, metav1.ConditionFalse, "SyncFailed", err.Error(), &now))
			status.SetCondition(conditions.New(monitoringv1alpha1.ConditionReady, metav1.ConditionFalse, "SyncFailed", "Monitor group reconciliation failed", &now))
		})
		return ctrl.Result{RequeueAfter: requeueIntervalOnError}, nil
	}

	now := metav1.Now()
	if err := r.patchStatus(ctx, group, func(status *monitoringv1alpha1.BetterStackMonitorGroupStatus) {
		status.MonitorGroupID = apiGroup.ID
		status.ObservedGeneration = group.Generation
		status.LastSyncedTime = &now
		status.SetCondition(conditions.New(monitoringv1alpha1.ConditionSync, metav1.ConditionTrue, "MonitorGroupSynced", "Monitor group synchronized with Better Stack", &now))
		status.SetCondition(conditions.New(monitoringv1alpha1.ConditionReady, metav1.ConditionTrue, "MonitorGroupSynced", "Monitor group synchronized with Better Stack", &now))
	}); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *BetterStackMonitorGroupReconciler) handleDelete(ctx context.Context, group *monitoringv1alpha1.BetterStackMonitorGroup) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(group, monitoringv1alpha1.BetterStackMonitorGroupFinalizer) {
		return ctrl.Result{}, nil
	}

	if group.Status.MonitorGroupID != "" {
		token, err := credentials.FetchAPIToken(ctx, r.Client, group.Namespace, group.Spec.APITokenSecretRef)
		if err != nil {
			logger.Info("skipping remote monitor group deletion due to missing credentials", "monitorGroupID", group.Status.MonitorGroupID, "error", err)
		} else {
			service := r.monitorGroupService(group.Spec.BaseURL, token)
			if err := service.Delete(ctx, group.Status.MonitorGroupID); err != nil && !betterstack.IsNotFound(err) {
				logger.Error(err, "unable to delete Better Stack monitor group", "monitorGroupID", group.Status.MonitorGroupID)
			}
		}
	}

	controllerutil.RemoveFinalizer(group, monitoringv1alpha1.BetterStackMonitorGroupFinalizer)
	if err := r.Update(ctx, group); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *BetterStackMonitorGroupReconciler) patchStatus(ctx context.Context, group *monitoringv1alpha1.BetterStackMonitorGroup, mutate func(*monitoringv1alpha1.BetterStackMonitorGroupStatus)) error {
	base := group.DeepCopy()
	mutate(&group.Status)
	return r.Status().Patch(ctx, group, client.MergeFrom(base))
}

func (r *BetterStackMonitorGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	ctx := context.Background()
	if err := mgr.GetFieldIndexer().IndexField(ctx, &monitoringv1alpha1.BetterStackMonitorGroup{}, monitorGroupSecretIndexKey, func(obj client.Object) []string {
		group, ok := obj.(*monitoringv1alpha1.BetterStackMonitorGroup)
		if !ok {
			return nil
		}
		secretName := group.Spec.APITokenSecretRef.Name
		if secretName == "" {
			return nil
		}
		return []string{secretIndexValue(group.Namespace, secretName)}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&monitoringv1alpha1.BetterStackMonitorGroup{}).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(r.requestsForSecret)).
		Complete(r)
}

func (r *BetterStackMonitorGroupReconciler) monitorGroupService(baseURL, token string) betterstack.MonitorGroupClient {
	factory := r.Clients
	if factory == nil {
		factory = defaultBetterStackMonitorGroupClientFactory{}
	}
	return factory.MonitorGroup(baseURL, token, r.HTTPClient)
}

func (r *BetterStackMonitorGroupReconciler) requestsForSecret(ctx context.Context, obj client.Object) []reconcile.Request {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return nil
	}
	if secret.Namespace == "" || secret.Name == "" {
		return nil
	}

	secretKey := secretIndexValue(secret.Namespace, secret.Name)
	list := &monitoringv1alpha1.BetterStackMonitorGroupList{}
	if err := r.List(ctx, list, client.InNamespace(secret.Namespace), client.MatchingFields{monitorGroupSecretIndexKey: secretKey}); err != nil {
		log.FromContext(ctx).Error(err, "unable to list monitor groups for secret", "secret", secretKey)
		return nil
	}

	requests := make([]reconcile.Request, 0, len(list.Items))
	for _, group := range list.Items {
		requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: group.Namespace, Name: group.Name}})
	}
	return requests
}

func buildMonitorGroupRequest(spec monitoringv1alpha1.BetterStackMonitorGroupSpec) betterstack.MonitorGroupRequest {
	req := betterstack.MonitorGroupRequest{}

	if spec.Name != "" {
		req.Name = ptr.To(spec.Name)
	}
	if spec.TeamName != "" {
		req.TeamName = ptr.To(spec.TeamName)
	}
	if spec.SortIndex != nil {
		req.SortIndex = spec.SortIndex
	}
	if spec.Paused != nil {
		req.Paused = spec.Paused
	}

	return req
}
