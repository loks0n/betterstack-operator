package controllers

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"strconv"
	"strings"
	"time"

	monitoringv1alpha1 "loks0n/betterstack-operator/api/v1alpha1"
	"loks0n/betterstack-operator/pkg/betterstack"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	requeueIntervalOnError = time.Minute
)

// BetterStackMonitorReconciler reconciles BetterStackMonitor resources.
type BetterStackMonitorReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	HTTPClient     *http.Client
	Clients BetterStackClientFactory
}

// BetterStackClientFactory provides Better Stack API clients for reconcilers.
type BetterStackClientFactory interface {
	Monitor(baseURL, token string, httpClient *http.Client) betterstack.MonitorClient
	Heartbeat(baseURL, token string, httpClient *http.Client) betterstack.HeartbeatClient
}

type defaultBetterStackClientFactory struct{}

func (defaultBetterStackClientFactory) Monitor(baseURL, token string, httpClient *http.Client) betterstack.MonitorClient {
	client := betterstack.NewClient(baseURL, token, httpClient)
	return client.Monitors
}

func (defaultBetterStackClientFactory) Heartbeat(baseURL, token string, httpClient *http.Client) betterstack.HeartbeatClient {
	client := betterstack.NewClient(baseURL, token, httpClient)
	return client.Heartbeats
}

//+kubebuilder:rbac:groups=monitoring.betterstack.io,resources=betterstackmonitors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=monitoring.betterstack.io,resources=betterstackmonitors/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=monitoring.betterstack.io,resources=betterstackmonitors/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *BetterStackMonitorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	monitor := &monitoringv1alpha1.BetterStackMonitor{}
	if err := r.Get(ctx, req.NamespacedName, monitor); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if monitor.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(monitor, monitoringv1alpha1.BetterStackMonitorFinalizer) {
			controllerutil.AddFinalizer(monitor, monitoringv1alpha1.BetterStackMonitorFinalizer)
			if err := r.Update(ctx, monitor); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
	} else {
		return r.handleDelete(ctx, monitor)
	}

	token, err := r.fetchAPIToken(ctx, monitor.Namespace, monitor.Spec.APITokenSecretRef)
	if err != nil {
		logger.Error(err, "unable to fetch Better Stack API token")
		_ = r.patchStatus(ctx, monitor, func(status *monitoringv1alpha1.BetterStackMonitorStatus) {
			now := metav1.Now()
			status.SetCondition(newCondition(monitoringv1alpha1.ConditionCredentials, metav1.ConditionFalse, "TokenUnavailable", err.Error(), &now))
			status.SetCondition(newCondition(monitoringv1alpha1.ConditionReady, metav1.ConditionFalse, "TokenUnavailable", "API credentials not available", &now))
		})
		return ctrl.Result{RequeueAfter: requeueIntervalOnError}, nil
	}

	_ = r.patchStatus(ctx, monitor, func(status *monitoringv1alpha1.BetterStackMonitorStatus) {
		now := metav1.Now()
		status.SetCondition(newCondition(monitoringv1alpha1.ConditionCredentials, metav1.ConditionTrue, "TokenResolved", fmt.Sprintf("Using secret %s/%s", monitor.Namespace, monitor.Spec.APITokenSecretRef.Name), &now))
	})

	monitorAPI := r.monitorService(monitor.Spec.BaseURL, token)

	var existingMonitor *betterstack.Monitor
	if monitor.Status.MonitorID != "" {
		existing, getErr := monitorAPI.Get(ctx, monitor.Status.MonitorID)
		if getErr != nil && !betterstack.IsNotFound(getErr) {
			logger.Error(getErr, "unable to fetch existing Better Stack monitor", "id", monitor.Status.MonitorID)
		} else if getErr == nil {
			existingMonitor = &existing
		}
	}
	request := buildMonitorRequest(monitor.Spec, existingMonitor)

	var apiMonitor betterstack.Monitor
	if monitor.Status.MonitorID != "" {
		apiMonitor, err = monitorAPI.Update(ctx, monitor.Status.MonitorID, request)
		if betterstack.IsNotFound(err) {
			logger.Info("remote monitor missing, creating anew", "id", monitor.Status.MonitorID)
			monitor.Status.MonitorID = ""
			err = nil
		}
	}

	if err == nil && monitor.Status.MonitorID == "" {
		apiMonitor, err = monitorAPI.Create(ctx, request)
	}

	if err != nil {
		logger.Error(err, "unable to reconcile Better Stack monitor")
		_ = r.patchStatus(ctx, monitor, func(status *monitoringv1alpha1.BetterStackMonitorStatus) {
			now := metav1.Now()
			status.SetCondition(newCondition(monitoringv1alpha1.ConditionSync, metav1.ConditionFalse, "SyncFailed", err.Error(), &now))
			status.SetCondition(newCondition(monitoringv1alpha1.ConditionReady, metav1.ConditionFalse, "SyncFailed", "Monitor reconciliation failed", &now))
		})
		return ctrl.Result{RequeueAfter: requeueIntervalOnError}, nil
	}

	now := metav1.Now()
	updateErr := r.patchStatus(ctx, monitor, func(status *monitoringv1alpha1.BetterStackMonitorStatus) {
		status.MonitorID = apiMonitor.ID
		status.ObservedGeneration = monitor.Generation
		status.LastSyncedTime = &now
		status.SetCondition(newCondition(monitoringv1alpha1.ConditionSync, metav1.ConditionTrue, "MonitorSynced", "Monitor synchronized with Better Stack", &now))
		status.SetCondition(newCondition(monitoringv1alpha1.ConditionReady, metav1.ConditionTrue, "MonitorSynced", "Monitor synchronized with Better Stack", &now))
	})
	if updateErr != nil {
		return ctrl.Result{}, updateErr
	}

	return ctrl.Result{}, nil
}

func (r *BetterStackMonitorReconciler) handleDelete(ctx context.Context, monitor *monitoringv1alpha1.BetterStackMonitor) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(monitor, monitoringv1alpha1.BetterStackMonitorFinalizer) {
		return ctrl.Result{}, nil
	}

	if monitor.Status.MonitorID != "" {
		token, err := r.fetchAPIToken(ctx, monitor.Namespace, monitor.Spec.APITokenSecretRef)
		if err != nil {
			logger.Info("skipping remote monitor deletion due to missing credentials", "monitorID", monitor.Status.MonitorID, "error", err)
		} else {
			service := r.monitorService(monitor.Spec.BaseURL, token)
			if err := service.Delete(ctx, monitor.Status.MonitorID); err != nil && !betterstack.IsNotFound(err) {
				logger.Error(err, "unable to delete Better Stack monitor", "monitorID", monitor.Status.MonitorID)
			}
		}
	}

	controllerutil.RemoveFinalizer(monitor, monitoringv1alpha1.BetterStackMonitorFinalizer)
	if err := r.Update(ctx, monitor); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *BetterStackMonitorReconciler) fetchAPIToken(ctx context.Context, namespace string, selector corev1.SecretKeySelector) (string, error) {
	if selector.Name == "" {
		return "", errors.New("apiTokenSecretRef.name must be specified")
	}

	key := types.NamespacedName{Name: selector.Name, Namespace: namespace}
	secret := &corev1.Secret{}
	if err := r.Get(ctx, key, secret); err != nil {
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

func (r *BetterStackMonitorReconciler) patchStatus(ctx context.Context, monitor *monitoringv1alpha1.BetterStackMonitor, mutate func(*monitoringv1alpha1.BetterStackMonitorStatus)) error {
	base := monitor.DeepCopy()
	mutate(&monitor.Status)
	return r.Status().Patch(ctx, monitor, client.MergeFrom(base))
}

func buildMonitorRequest(spec monitoringv1alpha1.BetterStackMonitorSpec, existing *betterstack.Monitor) betterstack.MonitorCreateRequest {
	req := betterstack.MonitorCreateRequest{}

	if spec.URL != "" {
		req.URL = stringPtr(spec.URL)
	}
	if spec.Name != "" {
		req.PronounceableName = stringPtr(spec.Name)
	}
	if spec.MonitorType != "" {
		req.MonitorType = stringPtr(spec.MonitorType)
	}
	if spec.TeamName != "" {
		req.TeamName = stringPtr(spec.TeamName)
	}
	if spec.CheckFrequencyMinutes > 0 {
		frequency := spec.CheckFrequencyMinutes * 60
		req.CheckFrequency = intPtr(frequency)
	}
	if len(spec.Regions) > 0 {
		req.Regions = append([]string(nil), spec.Regions...)
	}
	if spec.RequestMethod != "" {
		method := strings.ToLower(spec.RequestMethod)
		req.HTTPMethod = stringPtr(method)
	}
	if len(spec.ExpectedStatusCodes) > 0 {
		req.ExpectedStatusCodes = append([]int(nil), spec.ExpectedStatusCodes...)
	} else if spec.ExpectedStatusCode > 0 {
		req.ExpectedStatusCodes = []int{spec.ExpectedStatusCode}
	}
	if spec.RequiredKeyword != "" {
		req.RequiredKeyword = stringPtr(spec.RequiredKeyword)
	}
	req.Paused = boolPtrValue(spec.Paused)

	if spec.Email != nil {
		req.Email = spec.Email
	}
	if spec.SMS != nil {
		req.SMS = spec.SMS
	}
	if spec.Call != nil {
		req.Call = spec.Call
	}
	if spec.Push != nil {
		req.Push = spec.Push
	}
	if spec.CriticalAlert != nil {
		req.CriticalAlert = spec.CriticalAlert
	}
	if spec.FollowRedirects != nil {
		req.FollowRedirects = spec.FollowRedirects
	}
	if spec.VerifySSL != nil {
		req.VerifySSL = spec.VerifySSL
	}
	if spec.RememberCookies != nil {
		req.RememberCookies = spec.RememberCookies
	}

	if spec.PolicyID != "" {
		req.PolicyID = stringPtr(spec.PolicyID)
	}
	if spec.ExpirationPolicyID != "" {
		req.ExpirationPolicyID = stringPtr(spec.ExpirationPolicyID)
	}
	if spec.MonitorGroupID != "" {
		req.MonitorGroupID = stringPtr(spec.MonitorGroupID)
	}
	if spec.TeamWaitSeconds > 0 {
		req.TeamWait = intPtr(spec.TeamWaitSeconds)
	}
	if spec.DomainExpirationDays > 0 {
		req.DomainExpiration = intPtr(spec.DomainExpirationDays)
	}
	if spec.SSLExpirationDays > 0 {
		req.SSLExpiration = intPtr(spec.SSLExpirationDays)
	}
	if spec.Port > 0 {
		// Better Stack expects ports as strings (e.g. "25,465"); convert from the
		// integer we expose on the CRD for friendlier YAML.
		port := strconv.Itoa(spec.Port)
		req.Port = stringPtr(port)
	}
	if spec.RequestTimeoutSeconds > 0 {
		timeout := spec.RequestTimeoutSeconds
		switch strings.ToLower(spec.MonitorType) {
		case "ping", "tcp", "udp", "smtp", "pop", "imap", "dns":
			timeout = timeout * 1000
		}
		req.RequestTimeout = intPtr(timeout)
	}
	if spec.RecoveryPeriodSeconds > 0 {
		req.RecoveryPeriod = intPtr(spec.RecoveryPeriodSeconds)
	}
	if spec.ConfirmationPeriodSeconds > 0 {
		req.ConfirmationPeriod = intPtr(spec.ConfirmationPeriodSeconds)
	}
	if spec.IPVersion != "" {
		req.IPVersion = stringPtr(spec.IPVersion)
	}
	if len(spec.MaintenanceDays) > 0 {
		req.MaintenanceDays = append([]string(nil), spec.MaintenanceDays...)
	}
	if spec.MaintenanceFrom != "" {
		req.MaintenanceFrom = stringPtr(spec.MaintenanceFrom)
	}
	if spec.MaintenanceTo != "" {
		req.MaintenanceTo = stringPtr(spec.MaintenanceTo)
	}
	if spec.MaintenanceTimezone != "" {
		req.MaintenanceTimezone = stringPtr(spec.MaintenanceTimezone)
	}
	if len(spec.RequestHeaders) > 0 {
		existingHeaders := map[string][]betterstack.MonitorHeader{}
		if existing != nil {
			for _, hdr := range existing.Attributes.RequestHeaders {
				key := strings.ToLower(hdr.Name)
				existingHeaders[key] = append(existingHeaders[key], hdr)
			}
		}

		req.RequestHeaders = make([]betterstack.MonitorRequestHeader, 0, len(spec.RequestHeaders))
		for _, h := range spec.RequestHeaders {
			header := betterstack.MonitorRequestHeader{Name: h.Name, Value: h.Value}
			key := strings.ToLower(h.Name)
			if list := existingHeaders[key]; len(list) > 0 {
				hdr := list[0]
				if len(list) > 1 {
					existingHeaders[key] = list[1:]
				} else {
					delete(existingHeaders, key)
				}
				id := hdr.ID
				header.ID = &id
			}
			req.RequestHeaders = append(req.RequestHeaders, header)
		}
	}
	if spec.RequestBody != "" {
		req.RequestBody = stringPtr(spec.RequestBody)
	}
	if spec.AuthUsername != "" {
		req.AuthUsername = stringPtr(spec.AuthUsername)
	}
	if spec.AuthPassword != "" {
		req.AuthPassword = stringPtr(spec.AuthPassword)
	}
	if len(spec.EnvironmentVariables) > 0 {
		req.EnvironmentVariables = make(map[string]string, len(spec.EnvironmentVariables))
		maps.Copy(req.EnvironmentVariables, spec.EnvironmentVariables)
	}
	if spec.PlaywrightScript != "" {
		req.PlaywrightScript = stringPtr(spec.PlaywrightScript)
	}
	if spec.ScenarioName != "" {
		req.ScenarioName = stringPtr(spec.ScenarioName)
	}
	if len(spec.AdditionalAttributes) > 0 {
		req.AdditionalAttributes = make(map[string]any, len(spec.AdditionalAttributes))
		for k, v := range spec.AdditionalAttributes {
			req.AdditionalAttributes[k] = v
		}
	}

	return req
}

func stringPtr(v string) *string {
	return &v
}

func intPtr(v int) *int {
	return &v
}

func boolPtrValue(v bool) *bool {
	return &v
}

func newCondition(conditionType string, status metav1.ConditionStatus, reason, message string, transitionTime *metav1.Time) metav1.Condition {
	cond := metav1.Condition{
		Type:    conditionType,
		Status:  status,
		Reason:  reason,
		Message: message,
	}
	if transitionTime != nil {
		cond.LastTransitionTime = *transitionTime
	} else {
		cond.LastTransitionTime = metav1.Now()
	}
	return cond
}

func (r *BetterStackMonitorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&monitoringv1alpha1.BetterStackMonitor{}).
		Complete(r)
}

func (r *BetterStackMonitorReconciler) monitorService(baseURL, token string) betterstack.MonitorClient {
	factory := r.Clients
	if factory == nil {
		factory = defaultBetterStackClientFactory{}
	}
	return factory.Monitor(baseURL, token, r.HTTPClient)
}
