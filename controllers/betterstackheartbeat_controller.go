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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// BetterStackHeartbeatClientFactory provides Better Stack API clients for reconcilers.
type BetterStackHeartbeatClientFactory interface {
	Heartbeat(baseURL, token string, httpClient *http.Client) betterstack.HeartbeatClient
}

type defaultBetterStackHeartbeatClientFactory struct{}

func (defaultBetterStackHeartbeatClientFactory) Heartbeat(baseURL, token string, httpClient *http.Client) betterstack.HeartbeatClient {
	client := betterstack.NewClient(baseURL, token, httpClient)
	return client.Heartbeats
}

// BetterStackHeartbeatReconciler reconciles BetterStackHeartbeat resources.
type BetterStackHeartbeatReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	HTTPClient *http.Client
	Clients    BetterStackHeartbeatClientFactory
}

//+kubebuilder:rbac:groups=monitoring.betterstack.io,resources=betterstackheartbeats,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=monitoring.betterstack.io,resources=betterstackheartbeats/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=monitoring.betterstack.io,resources=betterstackheartbeats/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *BetterStackHeartbeatReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	heartbeat := &monitoringv1alpha1.BetterStackHeartbeat{}
	if err := r.Get(ctx, req.NamespacedName, heartbeat); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if heartbeat.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(heartbeat, monitoringv1alpha1.BetterStackHeartbeatFinalizer) {
			controllerutil.AddFinalizer(heartbeat, monitoringv1alpha1.BetterStackHeartbeatFinalizer)
			if err := r.Update(ctx, heartbeat); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
	} else {
		return r.handleDelete(ctx, heartbeat)
	}

	token, err := credentials.FetchAPIToken(ctx, r.Client, heartbeat.Namespace, heartbeat.Spec.APITokenSecretRef)
	if err != nil {
		logger.Error(err, "unable to fetch Better Stack API token")
		_ = r.patchStatus(ctx, heartbeat, func(status *monitoringv1alpha1.BetterStackHeartbeatStatus) {
			now := metav1.Now()
			status.SetCondition(conditions.New(monitoringv1alpha1.ConditionCredentials, metav1.ConditionFalse, "TokenUnavailable", err.Error(), &now))
			status.SetCondition(conditions.New(monitoringv1alpha1.ConditionReady, metav1.ConditionFalse, "TokenUnavailable", "API credentials not available", &now))
		})
		return ctrl.Result{RequeueAfter: requeueIntervalOnError}, nil
	}

	_ = r.patchStatus(ctx, heartbeat, func(status *monitoringv1alpha1.BetterStackHeartbeatStatus) {
		now := metav1.Now()
		status.SetCondition(conditions.New(monitoringv1alpha1.ConditionCredentials, metav1.ConditionTrue, "TokenResolved", fmt.Sprintf("Using secret %s/%s", heartbeat.Namespace, heartbeat.Spec.APITokenSecretRef.Name), &now))
	})

	service := r.heartbeatService(heartbeat.Spec.BaseURL, token)
	request := buildHeartbeatRequest(heartbeat.Spec)

	var apiHeartbeat betterstack.Heartbeat
	if heartbeat.Status.HeartbeatID != "" {
		apiHeartbeat, err = service.Update(ctx, heartbeat.Status.HeartbeatID, betterstack.HeartbeatUpdateRequest(request))
		if betterstack.IsNotFound(err) {
			logger.Info("remote heartbeat missing, creating anew", "id", heartbeat.Status.HeartbeatID)
			heartbeat.Status.HeartbeatID = ""
			err = nil
		}
	}

	if err == nil && heartbeat.Status.HeartbeatID == "" {
		apiHeartbeat, err = service.Create(ctx, request)
	}

	if err != nil {
		logger.Error(err, "unable to reconcile Better Stack heartbeat")
		_ = r.patchStatus(ctx, heartbeat, func(status *monitoringv1alpha1.BetterStackHeartbeatStatus) {
			now := metav1.Now()
			status.SetCondition(conditions.New(monitoringv1alpha1.ConditionSync, metav1.ConditionFalse, "SyncFailed", err.Error(), &now))
			status.SetCondition(conditions.New(monitoringv1alpha1.ConditionReady, metav1.ConditionFalse, "SyncFailed", "Heartbeat reconciliation failed", &now))
		})
		return ctrl.Result{RequeueAfter: requeueIntervalOnError}, nil
	}

	now := metav1.Now()
	updateErr := r.patchStatus(ctx, heartbeat, func(status *monitoringv1alpha1.BetterStackHeartbeatStatus) {
		status.HeartbeatID = apiHeartbeat.ID
		status.ObservedGeneration = heartbeat.Generation
		status.LastSyncedTime = &now
		status.SetCondition(conditions.New(monitoringv1alpha1.ConditionSync, metav1.ConditionTrue, "HeartbeatSynced", "Heartbeat synchronized with Better Stack", &now))
		status.SetCondition(conditions.New(monitoringv1alpha1.ConditionReady, metav1.ConditionTrue, "HeartbeatSynced", "Heartbeat synchronized with Better Stack", &now))
	})
	if updateErr != nil {
		return ctrl.Result{}, updateErr
	}

	return ctrl.Result{}, nil
}

func (r *BetterStackHeartbeatReconciler) handleDelete(ctx context.Context, heartbeat *monitoringv1alpha1.BetterStackHeartbeat) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(heartbeat, monitoringv1alpha1.BetterStackHeartbeatFinalizer) {
		return ctrl.Result{}, nil
	}

	if heartbeat.Status.HeartbeatID != "" {
		token, err := credentials.FetchAPIToken(ctx, r.Client, heartbeat.Namespace, heartbeat.Spec.APITokenSecretRef)
		if err != nil {
			logger.Info("skipping remote heartbeat deletion due to missing credentials", "heartbeatID", heartbeat.Status.HeartbeatID, "error", err)
		} else {
			service := r.heartbeatService(heartbeat.Spec.BaseURL, token)
			if err := service.Delete(ctx, heartbeat.Status.HeartbeatID); err != nil && !betterstack.IsNotFound(err) {
				logger.Error(err, "unable to delete Better Stack heartbeat", "heartbeatID", heartbeat.Status.HeartbeatID)
			}
		}
	}

	controllerutil.RemoveFinalizer(heartbeat, monitoringv1alpha1.BetterStackHeartbeatFinalizer)
	if err := r.Update(ctx, heartbeat); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *BetterStackHeartbeatReconciler) patchStatus(ctx context.Context, heartbeat *monitoringv1alpha1.BetterStackHeartbeat, mutate func(*monitoringv1alpha1.BetterStackHeartbeatStatus)) error {
	base := heartbeat.DeepCopy()
	mutate(&heartbeat.Status)
	return r.Status().Patch(ctx, heartbeat, client.MergeFrom(base))
}

func buildHeartbeatRequest(spec monitoringv1alpha1.BetterStackHeartbeatSpec) betterstack.HeartbeatCreateRequest {
	req := betterstack.HeartbeatCreateRequest{}

	if spec.TeamName != "" {
		req.TeamName = ptr.To(spec.TeamName)
	}
	if spec.Name != "" {
		req.Name = ptr.To(spec.Name)
	}
	if spec.PeriodSeconds > 0 {
		req.Period = ptr.To(spec.PeriodSeconds)
	}
	if spec.GraceSeconds > 0 {
		req.Grace = ptr.To(spec.GraceSeconds)
	}
	if spec.Call != nil {
		req.Call = spec.Call
	}
	if spec.SMS != nil {
		req.SMS = spec.SMS
	}
	if spec.Email != nil {
		req.Email = spec.Email
	}
	if spec.Push != nil {
		req.Push = spec.Push
	}
	if spec.CriticalAlert != nil {
		req.CriticalAlert = spec.CriticalAlert
	}
	if spec.TeamWaitSeconds > 0 {
		req.TeamWait = ptr.To(spec.TeamWaitSeconds)
	}
	if spec.HeartbeatGroupID != nil {
		req.HeartbeatGroupID = spec.HeartbeatGroupID
	}
	if spec.SortIndex != nil {
		req.SortIndex = spec.SortIndex
	}
	if spec.Paused != nil {
		req.Paused = spec.Paused
	}
	if len(spec.MaintenanceDays) > 0 {
		req.MaintenanceDays = append([]string(nil), spec.MaintenanceDays...)
	}
	if spec.MaintenanceFrom != "" {
		req.MaintenanceFrom = ptr.To(spec.MaintenanceFrom)
	}
	if spec.MaintenanceTo != "" {
		req.MaintenanceTo = ptr.To(spec.MaintenanceTo)
	}
	if spec.MaintenanceTimezone != "" {
		req.MaintenanceTimezone = ptr.To(spec.MaintenanceTimezone)
	}
	if spec.PolicyID != nil {
		req.PolicyID = spec.PolicyID
	}

	return req
}

func (r *BetterStackHeartbeatReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&monitoringv1alpha1.BetterStackHeartbeat{}).
		Complete(r)
}

func (r *BetterStackHeartbeatReconciler) heartbeatService(baseURL, token string) betterstack.HeartbeatClient {
	factory := r.Clients
	if factory == nil {
		factory = defaultBetterStackHeartbeatClientFactory{}
	}
	return factory.Heartbeat(baseURL, token, r.HTTPClient)
}
