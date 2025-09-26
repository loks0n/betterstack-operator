package controllers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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
	Scheme     *runtime.Scheme
	HTTPClient *http.Client
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
			return ctrl.Result{Requeue: true}, nil
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

	attr := buildMonitorAttributes(monitor.Spec)
	client := betterstack.NewClient(monitor.Spec.BaseURL, token, r.HTTPClient)

	var apiMonitor betterstack.Monitor
	if monitor.Status.MonitorID != "" {
		apiMonitor, err = client.UpdateMonitor(ctx, monitor.Status.MonitorID, attr)
		if betterstack.IsNotFound(err) {
			logger.Info("remote monitor missing, creating anew", "id", monitor.Status.MonitorID)
			monitor.Status.MonitorID = ""
			err = nil
		}
	}

	if err == nil && monitor.Status.MonitorID == "" {
		apiMonitor, err = client.CreateMonitor(ctx, attr)
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
			client := betterstack.NewClient(monitor.Spec.BaseURL, token, r.HTTPClient)
			if err := client.DeleteMonitor(ctx, monitor.Status.MonitorID); err != nil && !betterstack.IsNotFound(err) {
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

func buildMonitorAttributes(spec monitoringv1alpha1.BetterStackMonitorSpec) map[string]any {
	attrs := map[string]any{
		"url": spec.URL,
	}

	if spec.Name != "" {
		attrs["pronounceable_name"] = spec.Name
	}
	if spec.MonitorType != "" {
		attrs["monitor_type"] = spec.MonitorType
	}
	if spec.TeamName != "" {
		attrs["team_name"] = spec.TeamName
	}
	if spec.CheckFrequencyMinutes > 0 {
		attrs["check_frequency"] = spec.CheckFrequencyMinutes * 60
	}
	if len(spec.Regions) > 0 {
		attrs["regions"] = spec.Regions
	}
	if spec.RequestMethod != "" {
		attrs["http_method"] = strings.ToLower(spec.RequestMethod)
	}
	if len(spec.ExpectedStatusCodes) > 0 {
		attrs["expected_status_codes"] = spec.ExpectedStatusCodes
	} else if spec.ExpectedStatusCode > 0 {
		attrs["expected_status_codes"] = []int{spec.ExpectedStatusCode}
	}
	if spec.RequiredKeyword != "" {
		attrs["required_keyword"] = spec.RequiredKeyword
	}
	attrs["paused"] = spec.Paused

	if spec.Email != nil {
		attrs["email"] = *spec.Email
	}
	if spec.SMS != nil {
		attrs["sms"] = *spec.SMS
	}
	if spec.Call != nil {
		attrs["call"] = *spec.Call
	}
	if spec.Push != nil {
		attrs["push"] = *spec.Push
	}
	if spec.CriticalAlert != nil {
		attrs["critical_alert"] = *spec.CriticalAlert
	}
	if spec.FollowRedirects != nil {
		attrs["follow_redirects"] = *spec.FollowRedirects
	}
	if spec.VerifySSL != nil {
		attrs["verify_ssl"] = *spec.VerifySSL
	}
	if spec.RememberCookies != nil {
		attrs["remember_cookies"] = *spec.RememberCookies
	}

	if spec.PolicyID != "" {
		attrs["policy_id"] = spec.PolicyID
	}
	if spec.ExpirationPolicyID != "" {
		attrs["expiration_policy_id"] = spec.ExpirationPolicyID
	}
	if spec.MonitorGroupID != "" {
		attrs["monitor_group_id"] = spec.MonitorGroupID
	}
	if spec.TeamWaitSeconds > 0 {
		attrs["team_wait"] = spec.TeamWaitSeconds
	}
	if spec.DomainExpirationDays > 0 {
		attrs["domain_expiration"] = spec.DomainExpirationDays
	}
	if spec.SSLExpirationDays > 0 {
		attrs["ssl_expiration"] = spec.SSLExpirationDays
	}
	if spec.Port > 0 {
		attrs["port"] = spec.Port
	}
	if spec.RequestTimeoutSeconds > 0 {
		attrs["request_timeout"] = spec.RequestTimeoutSeconds
	}
	if spec.RecoveryPeriodSeconds > 0 {
		attrs["recovery_period"] = spec.RecoveryPeriodSeconds
	}
	if spec.ConfirmationPeriodSeconds > 0 {
		attrs["confirmation_period"] = spec.ConfirmationPeriodSeconds
	}
	if spec.IPVersion != "" {
		attrs["ip_version"] = spec.IPVersion
	}
	if len(spec.MaintenanceDays) > 0 {
		attrs["maintenance_days"] = spec.MaintenanceDays
	}
	if spec.MaintenanceFrom != "" {
		attrs["maintenance_from"] = spec.MaintenanceFrom
	}
	if spec.MaintenanceTo != "" {
		attrs["maintenance_to"] = spec.MaintenanceTo
	}
	if spec.MaintenanceTimezone != "" {
		attrs["maintenance_timezone"] = spec.MaintenanceTimezone
	}
	if len(spec.RequestHeaders) > 0 {
		headers := make([]map[string]string, 0, len(spec.RequestHeaders))
		for _, h := range spec.RequestHeaders {
			headers = append(headers, map[string]string{"name": h.Name, "value": h.Value})
		}
		attrs["request_headers"] = headers
	}
	if spec.RequestBody != "" {
		attrs["request_body"] = spec.RequestBody
	}
	if spec.AuthUsername != "" {
		attrs["auth_username"] = spec.AuthUsername
	}
	if spec.AuthPassword != "" {
		attrs["auth_password"] = spec.AuthPassword
	}
	if len(spec.EnvironmentVariables) > 0 {
		attrs["environment_variables"] = spec.EnvironmentVariables
	}
	if spec.PlaywrightScript != "" {
		attrs["playwright_script"] = spec.PlaywrightScript
	}
	if spec.ScenarioName != "" {
		attrs["scenario_name"] = spec.ScenarioName
	}

	for k, v := range spec.AdditionalAttributes {
		attrs[k] = v
	}
	return attrs
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
