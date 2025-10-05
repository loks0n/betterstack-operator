package controllers

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"strconv"
	"strings"

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

// BetterStackClientFactory provides Better Stack API clients for reconcilers.
type BetterStackMonitorClientFactory interface {
	Monitor(baseURL, token string, httpClient *http.Client) betterstack.MonitorClient
}

type defaultBetterStackMonitorClientFactory struct{}

func (defaultBetterStackMonitorClientFactory) Monitor(baseURL, token string, httpClient *http.Client) betterstack.MonitorClient {
	client := betterstack.NewClient(baseURL, token, httpClient)
	return client.Monitors
}

// BetterStackMonitorReconciler reconciles BetterStackMonitor resources.
type BetterStackMonitorReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	HTTPClient *http.Client
	Clients    BetterStackMonitorClientFactory
}

const (
	monitorSecretIndexKey      = "monitoring.betterstack.io/monitor-secret"
	ReasonMonitorQuotaExceeded = "MonitorQuotaExceeded"
)

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

	token, err := credentials.FetchAPIToken(ctx, r.Client, monitor.Namespace, monitor.Spec.APITokenSecretRef)
	if err != nil {
		logger.Error(err, "unable to fetch Better Stack API token")
		_ = r.patchStatus(ctx, monitor, func(status *monitoringv1alpha1.BetterStackMonitorStatus) {
			now := metav1.Now()
			status.SetCondition(conditions.New(monitoringv1alpha1.ConditionCredentials, metav1.ConditionFalse, "TokenUnavailable", err.Error(), &now))
			status.SetCondition(conditions.New(monitoringv1alpha1.ConditionReady, metav1.ConditionFalse, "TokenUnavailable", "API credentials not available", &now))
		})
		return ctrl.Result{RequeueAfter: requeueIntervalOnError}, nil
	}

	_ = r.patchStatus(ctx, monitor, func(status *monitoringv1alpha1.BetterStackMonitorStatus) {
		now := metav1.Now()
		status.SetCondition(conditions.New(monitoringv1alpha1.ConditionCredentials, metav1.ConditionTrue, "TokenResolved", fmt.Sprintf("Using secret %s/%s", monitor.Namespace, monitor.Spec.APITokenSecretRef.Name), &now))
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
		syncReason := "SyncFailed"
		syncMessage := err.Error()
		readyMessage := "Monitor reconciliation failed"
		if isMonitorQuotaExceeded(err) {
			syncReason = ReasonMonitorQuotaExceeded
			syncMessage = "Better Stack monitor quota reached"
			readyMessage = "Better Stack monitor quota reached"
		}
		_ = r.patchStatus(ctx, monitor, func(status *monitoringv1alpha1.BetterStackMonitorStatus) {
			now := metav1.Now()
			status.SetCondition(conditions.New(monitoringv1alpha1.ConditionSync, metav1.ConditionFalse, syncReason, syncMessage, &now))
			status.SetCondition(conditions.New(monitoringv1alpha1.ConditionReady, metav1.ConditionFalse, syncReason, readyMessage, &now))
		})
		return ctrl.Result{RequeueAfter: requeueIntervalOnError}, nil
	}

	now := metav1.Now()
	updateErr := r.patchStatus(ctx, monitor, func(status *monitoringv1alpha1.BetterStackMonitorStatus) {
		status.MonitorID = apiMonitor.ID
		status.ObservedGeneration = monitor.Generation
		status.LastSyncedTime = &now
		status.SetCondition(conditions.New(monitoringv1alpha1.ConditionSync, metav1.ConditionTrue, "MonitorSynced", "Monitor synchronized with Better Stack", &now))
		status.SetCondition(conditions.New(monitoringv1alpha1.ConditionReady, metav1.ConditionTrue, "MonitorSynced", "Monitor synchronized with Better Stack", &now))
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
		token, err := credentials.FetchAPIToken(ctx, r.Client, monitor.Namespace, monitor.Spec.APITokenSecretRef)
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

func (r *BetterStackMonitorReconciler) patchStatus(ctx context.Context, monitor *monitoringv1alpha1.BetterStackMonitor, mutate func(*monitoringv1alpha1.BetterStackMonitorStatus)) error {
	base := monitor.DeepCopy()
	mutate(&monitor.Status)
	return r.Status().Patch(ctx, monitor, client.MergeFrom(base))
}

func buildMonitorRequest(spec monitoringv1alpha1.BetterStackMonitorSpec, existing *betterstack.Monitor) betterstack.MonitorCreateRequest {
	req := betterstack.MonitorCreateRequest{}

	if spec.URL != "" {
		req.URL = ptr.To(spec.URL)
	}
	if spec.Name != "" {
		req.PronounceableName = ptr.To(spec.Name)
	}
	if spec.MonitorType != "" {
		req.MonitorType = ptr.To(spec.MonitorType)
	}
	if spec.TeamName != "" {
		req.TeamName = ptr.To(spec.TeamName)
	}
	if spec.CheckFrequencyMinutes > 0 {
		frequency := spec.CheckFrequencyMinutes * 60
		req.CheckFrequency = ptr.To(frequency)
	}
	if len(spec.Regions) > 0 {
		req.Regions = append([]string(nil), spec.Regions...)
	}
	if spec.RequestMethod != "" {
		method := strings.ToLower(spec.RequestMethod)
		req.HTTPMethod = ptr.To(method)
	}
	if len(spec.ExpectedStatusCodes) > 0 {
		req.ExpectedStatusCodes = append([]int(nil), spec.ExpectedStatusCodes...)
	} else if spec.ExpectedStatusCode > 0 {
		req.ExpectedStatusCodes = []int{spec.ExpectedStatusCode}
	}
	if spec.RequiredKeyword != "" {
		req.RequiredKeyword = ptr.To(spec.RequiredKeyword)
	}
	req.Paused = ptr.To(spec.Paused)

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
		req.PolicyID = ptr.To(spec.PolicyID)
	}
	if spec.ExpirationPolicyID != "" {
		req.ExpirationPolicyID = ptr.To(spec.ExpirationPolicyID)
	}
	if spec.MonitorGroupID != "" {
		req.MonitorGroupID = ptr.To(spec.MonitorGroupID)
	}
	if spec.TeamWaitSeconds > 0 {
		req.TeamWait = ptr.To(spec.TeamWaitSeconds)
	}
	if spec.DomainExpirationDays > 0 {
		req.DomainExpiration = ptr.To(spec.DomainExpirationDays)
	}
	if spec.SSLExpirationDays > 0 {
		req.SSLExpiration = ptr.To(spec.SSLExpirationDays)
	}
	if spec.Port > 0 {
		port := strconv.Itoa(spec.Port)
		req.Port = ptr.To(port)
	}
	if spec.RequestTimeoutSeconds > 0 {
		timeout := spec.RequestTimeoutSeconds
		switch strings.ToLower(spec.MonitorType) {
		case "ping", "tcp", "udp", "smtp", "pop", "imap", "dns":
			timeout = timeout * 1000
		}
		req.RequestTimeout = ptr.To(timeout)
	}
	if spec.RecoveryPeriodSeconds > 0 {
		req.RecoveryPeriod = ptr.To(spec.RecoveryPeriodSeconds)
	}
	if spec.ConfirmationPeriodSeconds > 0 {
		req.ConfirmationPeriod = ptr.To(spec.ConfirmationPeriodSeconds)
	}
	if spec.IPVersion != "" {
		req.IPVersion = ptr.To(spec.IPVersion)
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
		req.RequestBody = ptr.To(spec.RequestBody)
	}
	if spec.AuthUsername != "" {
		req.AuthUsername = ptr.To(spec.AuthUsername)
	}
	if spec.AuthPassword != "" {
		req.AuthPassword = ptr.To(spec.AuthPassword)
	}
	if len(spec.EnvironmentVariables) > 0 {
		req.EnvironmentVariables = make(map[string]string, len(spec.EnvironmentVariables))
		maps.Copy(req.EnvironmentVariables, spec.EnvironmentVariables)
	}
	if spec.PlaywrightScript != "" {
		req.PlaywrightScript = ptr.To(spec.PlaywrightScript)
	}
	if spec.ScenarioName != "" {
		req.ScenarioName = ptr.To(spec.ScenarioName)
	}
	if len(spec.AdditionalAttributes) > 0 {
		req.AdditionalAttributes = make(map[string]any, len(spec.AdditionalAttributes))
		for k, v := range spec.AdditionalAttributes {
			req.AdditionalAttributes[k] = v
		}
	}

	return req
}

func (r *BetterStackMonitorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	ctx := context.Background()
	if err := mgr.GetFieldIndexer().IndexField(ctx, &monitoringv1alpha1.BetterStackMonitor{}, monitorSecretIndexKey, func(obj client.Object) []string {
		monitor, ok := obj.(*monitoringv1alpha1.BetterStackMonitor)
		if !ok {
			return nil
		}
		secretName := monitor.Spec.APITokenSecretRef.Name
		if secretName == "" {
			return nil
		}
		return []string{secretIndexValue(monitor.Namespace, secretName)}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&monitoringv1alpha1.BetterStackMonitor{}).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(r.requestsForSecret)).
		Complete(r)
}

func (r *BetterStackMonitorReconciler) monitorService(baseURL, token string) betterstack.MonitorClient {
	factory := r.Clients
	if factory == nil {
		factory = defaultBetterStackMonitorClientFactory{}
	}
	return factory.Monitor(baseURL, token, r.HTTPClient)
}

func (r *BetterStackMonitorReconciler) requestsForSecret(ctx context.Context, obj client.Object) []reconcile.Request {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return nil
	}
	if secret.Namespace == "" || secret.Name == "" {
		return nil
	}

	secretKey := secretIndexValue(secret.Namespace, secret.Name)
	list := &monitoringv1alpha1.BetterStackMonitorList{}
	if err := r.List(ctx, list, client.InNamespace(secret.Namespace), client.MatchingFields{monitorSecretIndexKey: secretKey}); err != nil {
		log.FromContext(ctx).Error(err, "unable to list monitors for secret", "secret", secretKey)
		return nil
	}

	requests := make([]reconcile.Request, 0, len(list.Items))
	for _, monitor := range list.Items {
		requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: monitor.Namespace, Name: monitor.Name}})
	}
	return requests
}

func isMonitorQuotaExceeded(err error) bool {
	var apiErr *betterstack.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	if apiErr.StatusCode != http.StatusForbidden {
		return false
	}
	return strings.Contains(strings.ToLower(apiErr.Message), "quota")
}
