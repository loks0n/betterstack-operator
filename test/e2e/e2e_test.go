//go:build e2e

package e2e

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	stdruntime "runtime"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/wait"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	monitoringv1alpha1 "loks0n/betterstack-operator/api/v1alpha1"
	"loks0n/betterstack-operator/controllers"
	"loks0n/betterstack-operator/internal/testutil/assert"
	"loks0n/betterstack-operator/pkg/betterstack"
)

func TestBetterStackOperatorLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	ensureBinary(t, "kind")
	ensureBinary(t, "kubectl")

	clusterName := os.Getenv("KIND_CLUSTER_NAME")
	if clusterName == "" {
		clusterName = fmt.Sprintf("betterstack-e2e-%d", time.Now().UnixNano())
		createKindCluster(t, clusterName)
		defer deleteKindCluster(clusterName)
	}

	rootDir := projectRoot()
	loadDotEnvIfPresent(t, rootDir)

	token := strings.TrimSpace(os.Getenv("BETTERSTACK_TOKEN"))
	if token == "" {
		t.Skip("BETTERSTACK_TOKEN not set; skipping e2e test")
	}

	kubeconfigPath := fetchKubeconfig(t, clusterName)
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	assert.NoError(t, err, "build config")

	scheme := runtime.NewScheme()
	assert.NoError(t, clientgoscheme.AddToScheme(scheme), "add core scheme")
	assert.NoError(t, monitoringv1alpha1.AddToScheme(scheme), "add CR scheme")

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme})
	assert.NoError(t, err, "new client")

	installCRD(t, cfg, filepath.Join(rootDir, "config", "crd", "bases", "monitoring.betterstack.io_betterstackmonitors.yaml"))
	installCRD(t, cfg, filepath.Join(rootDir, "config", "crd", "bases", "monitoring.betterstack.io_betterstackheartbeats.yaml"))

	mgrCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                 scheme,
		Metrics:                server.Options{BindAddress: "0"},
		HealthProbeBindAddress: "0",
		LeaderElection:         false,
	})
	assert.NoError(t, err, "new manager")

	httpClient := &http.Client{Timeout: 30 * time.Second}

	monitorReconciler := &controllers.BetterStackMonitorReconciler{
		Client:     manager.GetClient(),
		Scheme:     manager.GetScheme(),
		HTTPClient: httpClient,
	}
	assert.NoError(t, monitorReconciler.SetupWithManager(manager), "setup monitor reconciler")

	heartbeatReconciler := &controllers.BetterStackHeartbeatReconciler{
		Client:     manager.GetClient(),
		Scheme:     manager.GetScheme(),
		HTTPClient: httpClient,
	}
	assert.NoError(t, heartbeatReconciler.SetupWithManager(manager), "setup heartbeat reconciler")

	go func() {
		if err := manager.Start(mgrCtx); err != nil {
			t.Errorf("manager stopped: %v", err)
		}
	}()

	namespace := "default"
	secretName := "betterstack-credentials"
	secretKey := "api-key"
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: secretName},
		Data:       map[string][]byte{secretKey: []byte(token)},
	}
	if err := k8sClient.Create(context.Background(), secret); err != nil && !errors.IsAlreadyExists(err) {
		assert.NoError(t, err, "create secret")
	}

	apiClient := betterstack.NewClient("", token, httpClient)

	t.Run("monitor", func(t *testing.T) {
		runMonitorLifecycle(t, k8sClient, apiClient, namespace, secretName, secretKey)
	})

	t.Run("heartbeat", func(t *testing.T) {
		runHeartbeatLifecycle(t, k8sClient, apiClient, namespace, secretName, secretKey)
	})
}

func runMonitorLifecycle(t *testing.T, k8sClient client.Client, apiClient *betterstack.Client, namespace, secretName, secretKey string) {
	t.Helper()

	unique := time.Now().UnixNano()
	monitorName := fmt.Sprintf("e2e-monitor-%d", unique)
	monitorURL := fmt.Sprintf("https://example.com/healthz-%d", unique)

	cleanupE2EMonitors(t, apiClient, "https://example.com/healthz")

	monitor := &monitoringv1alpha1.BetterStackMonitor{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: monitorName},
		Spec: monitoringv1alpha1.BetterStackMonitorSpec{
			URL:                       monitorURL,
			Name:                      fmt.Sprintf("Initial E2E %d", unique),
			MonitorType:               "status",
			CheckFrequencyMinutes:     3,
			ExpectedStatusCodes:       []int{200},
			RequestMethod:             "head",
			FollowRedirects:           ptr.To(false),
			RememberCookies:           ptr.To(false),
			VerifySSL:                 ptr.To(true),
			Email:                     ptr.To(false),
			SMS:                       ptr.To(true),
			Call:                      ptr.To(false),
			Push:                      ptr.To(true),
			CriticalAlert:             ptr.To(true),
			TeamWaitSeconds:           120,
			DomainExpirationDays:      14,
			SSLExpirationDays:         30,
			RequestTimeoutSeconds:     10,
			ConfirmationPeriodSeconds: 90,
			IPVersion:                 "ipv4",
			MaintenanceDays:           []string{"mon", "tue"},
			MaintenanceFrom:           "01:00:00",
			MaintenanceTo:             "02:00:00",
			MaintenanceTimezone:       "UTC",
			RequestHeaders: []monitoringv1alpha1.BetterStackHeader{{
				Name:  "X-E2E",
				Value: "initial",
			}},
			APITokenSecretRef: corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
				Key:                  secretKey,
			},
		},
	}

	assert.NoError(t, k8sClient.Create(context.Background(), monitor), "create monitor")

	assert.NoError(t, waitForMonitorCondition(k8sClient, namespace, monitorName, func(obj *monitoringv1alpha1.BetterStackMonitor) bool {
		return metaConditionStatus(obj.Status.Conditions, monitoringv1alpha1.ConditionReady) == metav1.ConditionTrue
	}), "wait for monitor ready")

	monitorID, err := waitForMonitorID(k8sClient, namespace, monitorName)
	assert.NoError(t, err, "wait for monitor id")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	remoteMonitor := fetchRemoteMonitor(t, ctx, apiClient, monitorID)
	attrs := remoteMonitor.Attributes
	assert.String(t, "url", attrs.URL, monitorURL)
	assert.String(t, "monitor_type", attrs.MonitorType, "status")
	assert.String(t, "http_method", attrs.HTTPMethod, "head")
	assert.Int(t, "check_frequency", attrs.CheckFrequency, 180)
	assert.IntSlice(t, "expected_status_codes", attrs.ExpectedStatusCodes, []int{200})
	assert.Bool(t, "follow_redirects", attrs.FollowRedirects, false)
	assert.Bool(t, "remember_cookies", attrs.RememberCookies, false)
	assert.Bool(t, "verify_ssl", attrs.VerifySSL, true)
	assert.Bool(t, "email", attrs.Email, false)
	assert.Bool(t, "sms", attrs.SMS, true)
	assert.Bool(t, "call", attrs.Call, false)
	assert.Bool(t, "push", attrs.Push, true)
	assert.Bool(t, "critical_alert", attrs.CriticalAlert, true)
	assert.IntPtr(t, "team_wait", attrs.TeamWait, 120)
	assert.IntPtr(t, "domain_expiration", attrs.DomainExpiration, 14)
	assert.IntPtr(t, "ssl_expiration", attrs.SSLExpiration, 30)
	assert.Int(t, "request_timeout", attrs.RequestTimeout, 10)
	assert.Int(t, "recovery_period", attrs.RecoveryPeriod, 180)
	assert.Int(t, "confirmation_period", attrs.ConfirmationPeriod, 90)
	assert.StringPtr(t, "ip_version", attrs.IPVersion, "ipv4")
	assert.StringSlice(t, "maintenance_days", attrs.MaintenanceDays, []string{"mon", "tue"})
	assert.String(t, "maintenance_from", attrs.MaintenanceFrom, "01:00:00")
	assert.String(t, "maintenance_to", attrs.MaintenanceTo, "02:00:00")
	assert.String(t, "maintenance_timezone", attrs.MaintenanceTimezone, "UTC")
	assert.Item(t, "request_headers", attrs.RequestHeaders, "X-E2E", "initial", func(h betterstack.MonitorHeader) (string, string) {
		return h.Name, h.Value
	})

	assert.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{Name: monitorName, Namespace: namespace}, monitor), "get monitor for update")
	monitor.Spec.Name = "Updated E2E"
	monitor.Spec.Paused = true
	monitor.Spec.CheckFrequencyMinutes = 5
	monitor.Spec.ExpectedStatusCodes = []int{204, 205}
	monitor.Spec.RequestMethod = "get"
	monitor.Spec.FollowRedirects = ptr.To(true)
	monitor.Spec.RememberCookies = ptr.To(true)
	monitor.Spec.VerifySSL = ptr.To(false)
	monitor.Spec.Email = ptr.To(true)
	monitor.Spec.SMS = ptr.To(false)
	monitor.Spec.Push = ptr.To(false)
	monitor.Spec.CriticalAlert = ptr.To(false)
	monitor.Spec.TeamWaitSeconds = 60
	monitor.Spec.DomainExpirationDays = 7
	monitor.Spec.SSLExpirationDays = 14
	monitor.Spec.RequestTimeoutSeconds = 45
	monitor.Spec.RecoveryPeriodSeconds = 300
	monitor.Spec.ConfirmationPeriodSeconds = 120
	monitor.Spec.IPVersion = "ipv6"
	monitor.Spec.MaintenanceDays = []string{"wed"}
	monitor.Spec.MaintenanceFrom = "02:00:00"
	monitor.Spec.MaintenanceTo = "03:00:00"
	monitor.Spec.MaintenanceTimezone = "America/New_York"
	monitor.Spec.RequestHeaders = []monitoringv1alpha1.BetterStackHeader{{
		Name:  "X-E2E",
		Value: "updated",
	}}
	assert.NoError(t, k8sClient.Update(context.Background(), monitor), "update monitor")

	assert.NoError(t, waitForMonitorCondition(k8sClient, namespace, monitorName, func(obj *monitoringv1alpha1.BetterStackMonitor) bool {
		return obj.Status.ObservedGeneration == obj.Generation
	}), "wait for monitor update")

	ctxUpdate, cancelUpdate := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancelUpdate()
	updatedMonitor := fetchRemoteMonitor(t, ctxUpdate, apiClient, monitorID)
	updated := updatedMonitor.Attributes
	assert.String(t, "pronounceable_name", updated.PronounceableName, "Updated E2E")
	assert.Bool(t, "paused", updated.Paused, true)
	assert.String(t, "http_method", updated.HTTPMethod, "get")
	assert.Int(t, "check_frequency", updated.CheckFrequency, 300)
	assert.IntSlice(t, "expected_status_codes", updated.ExpectedStatusCodes, []int{204, 205})
	assert.Bool(t, "follow_redirects", updated.FollowRedirects, true)
	assert.Bool(t, "remember_cookies", updated.RememberCookies, true)
	assert.Bool(t, "verify_ssl", updated.VerifySSL, false)
	assert.Bool(t, "email", updated.Email, true)
	assert.Bool(t, "sms", updated.SMS, false)
	assert.Bool(t, "push", updated.Push, false)
	assert.Bool(t, "critical_alert", updated.CriticalAlert, false)
	assert.IntPtr(t, "team_wait", updated.TeamWait, 60)
	assert.IntPtr(t, "domain_expiration", updated.DomainExpiration, 7)
	assert.IntPtr(t, "ssl_expiration", updated.SSLExpiration, 14)
	assert.Int(t, "request_timeout", updated.RequestTimeout, 45)
	assert.Int(t, "recovery_period", updated.RecoveryPeriod, 300)
	assert.Int(t, "confirmation_period", updated.ConfirmationPeriod, 120)
	assert.StringPtr(t, "ip_version", updated.IPVersion, "ipv6")
	assert.StringSlice(t, "maintenance_days", updated.MaintenanceDays, []string{"wed"})
	assert.String(t, "maintenance_from", updated.MaintenanceFrom, "02:00:00")
	assert.String(t, "maintenance_to", updated.MaintenanceTo, "03:00:00")
	assert.String(t, "maintenance_timezone", updated.MaintenanceTimezone, "Eastern Time (US & Canada)")
	assert.Item(t, "request_headers", updated.RequestHeaders, "X-E2E", "updated", func(h betterstack.MonitorHeader) (string, string) {
		return h.Name, h.Value
	})

	assert.NoError(t, k8sClient.Delete(context.Background(), monitor), "delete monitor")

	err = wait.PollImmediate(2*time.Second, 90*time.Second, func() (bool, error) {
		err := k8sClient.Get(context.Background(), client.ObjectKey{Name: monitorName, Namespace: namespace}, monitor)
		if errors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
	assert.NoError(t, err, "wait for monitor deletion")

	ctxDelete, cancelDelete := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancelDelete()
	assert.Bool(t, "remote monitor exists", monitorExists(ctxDelete, apiClient, monitorID), false)
}

func runHeartbeatLifecycle(t *testing.T, k8sClient client.Client, apiClient *betterstack.Client, namespace, secretName, secretKey string) {
	t.Helper()

	unique := time.Now().UnixNano()
	heartbeatName := fmt.Sprintf("e2e-heartbeat-%d", unique)

	heartbeat := &monitoringv1alpha1.BetterStackHeartbeat{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: heartbeatName},
		Spec: monitoringv1alpha1.BetterStackHeartbeatSpec{
			Name:                fmt.Sprintf("Heartbeat %d", unique),
			PeriodSeconds:       60,
			GraceSeconds:        30,
			Call:                ptr.To(true),
			Email:               ptr.To(true),
			SMS:                 ptr.To(false),
			Push:                ptr.To(false),
			CriticalAlert:       ptr.To(true),
			TeamWaitSeconds:     45,
			MaintenanceDays:     []string{"mon"},
			MaintenanceFrom:     "05:00",
			MaintenanceTo:       "06:00",
			MaintenanceTimezone: "UTC",
			APITokenSecretRef: corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
				Key:                  secretKey,
			},
		},
	}

	cleanupE2EHeartbeats(t, apiClient, "Heartbeat ")

	assert.NoError(t, k8sClient.Create(context.Background(), heartbeat), "create heartbeat")

	if err := waitForHeartbeatCondition(k8sClient, namespace, heartbeatName, func(obj *monitoringv1alpha1.BetterStackHeartbeat) bool {
		return metaConditionStatus(obj.Status.Conditions, monitoringv1alpha1.ConditionReady) == metav1.ConditionTrue
	}); err != nil {
		current := &monitoringv1alpha1.BetterStackHeartbeat{}
		if getErr := k8sClient.Get(context.Background(), client.ObjectKey{Name: heartbeatName, Namespace: namespace}, current); getErr == nil {
			if cond := findCondition(current.Status.Conditions, monitoringv1alpha1.ConditionSync); cond != nil {
				assert.Failf(t, "heartbeat failed to become ready: %s", cond.Message)
				return
			}
		}
		assert.NoError(t, err, "wait for heartbeat ready")
	}

	heartbeatID, err := waitForHeartbeatID(k8sClient, namespace, heartbeatName)
	assert.NoError(t, err, "wait for heartbeat id")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	defer func() {
		if heartbeatID != "" {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cleanupCancel()
			_ = apiClient.Heartbeats.Delete(cleanupCtx, heartbeatID)
		}
	}()

	remoteHeartbeat := fetchRemoteHeartbeat(t, ctx, apiClient, heartbeatID)
	hattrs := remoteHeartbeat.Attributes
	assert.String(t, "name", hattrs.Name, heartbeat.Spec.Name)
	assert.Int(t, "period", hattrs.Period, 60)
	assert.Int(t, "grace", hattrs.Grace, 30)
	assert.Bool(t, "call", hattrs.Call, true)
	assert.Bool(t, "email", hattrs.Email, true)
	assert.Bool(t, "sms", hattrs.SMS, false)
	assert.Bool(t, "push", hattrs.Push, false)
	assert.Bool(t, "critical_alert", hattrs.CriticalAlert, true)
	assert.IntPtr(t, "team_wait", hattrs.TeamWait, 45)
	assert.StringSlice(t, "maintenance_days", hattrs.MaintenanceDays, []string{"mon"})
	assert.String(t, "maintenance_from", hattrs.MaintenanceFrom, "05:00:00")
	assert.String(t, "maintenance_to", hattrs.MaintenanceTo, "06:00:00")
	assert.String(t, "maintenance_timezone", hattrs.MaintenanceTimezone, "UTC")

	assert.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{Name: heartbeatName, Namespace: namespace}, heartbeat), "get heartbeat for update")
	heartbeat.Spec.Name = "Heartbeat Updated"
	heartbeat.Spec.PeriodSeconds = 90
	heartbeat.Spec.GraceSeconds = 45
	heartbeat.Spec.Call = ptr.To(false)
	heartbeat.Spec.Email = ptr.To(false)
	heartbeat.Spec.SMS = ptr.To(true)
	heartbeat.Spec.TeamWaitSeconds = 30
	heartbeat.Spec.MaintenanceDays = []string{"tue"}
	heartbeat.Spec.MaintenanceFrom = "06:00"
	heartbeat.Spec.MaintenanceTo = "07:00"
	assert.NoError(t, k8sClient.Update(context.Background(), heartbeat), "update heartbeat")

	assert.NoError(t, waitForHeartbeatCondition(k8sClient, namespace, heartbeatName, func(obj *monitoringv1alpha1.BetterStackHeartbeat) bool {
		return obj.Status.ObservedGeneration == obj.Generation
	}), "wait for heartbeat update")

	ctxUpdate, cancelUpdate := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancelUpdate()
	updatedHeartbeat := fetchRemoteHeartbeat(t, ctxUpdate, apiClient, heartbeatID)
	uattrs := updatedHeartbeat.Attributes
	assert.String(t, "updated name", uattrs.Name, "Heartbeat Updated")
	assert.Int(t, "updated period", uattrs.Period, 90)
	assert.Int(t, "updated grace", uattrs.Grace, 45)
	assert.Bool(t, "updated call", uattrs.Call, false)
	assert.Bool(t, "updated email", uattrs.Email, false)
	assert.Bool(t, "updated sms", uattrs.SMS, true)
	assert.IntPtr(t, "updated team wait", uattrs.TeamWait, 30)
	assert.StringSlice(t, "updated maintenance days", uattrs.MaintenanceDays, []string{"tue"})
	assert.String(t, "updated maintenance from", uattrs.MaintenanceFrom, "06:00:00")
	assert.String(t, "updated maintenance to", uattrs.MaintenanceTo, "07:00:00")

	assert.NoError(t, k8sClient.Delete(context.Background(), heartbeat), "delete heartbeat")

	err = wait.PollImmediate(2*time.Second, 90*time.Second, func() (bool, error) {
		err := k8sClient.Get(context.Background(), client.ObjectKey{Name: heartbeatName, Namespace: namespace}, heartbeat)
		if errors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
	assert.NoError(t, err, "wait for heartbeat deletion")

	ctxDelete, cancelDelete := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancelDelete()
	assert.Bool(t, "remote heartbeat exists", heartbeatExists(ctxDelete, apiClient, heartbeatID), false)
}

func cleanupE2EHeartbeats(t *testing.T, apiClient *betterstack.Client, prefix string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	heartbeats, err := apiClient.Heartbeats.List(ctx)
	assert.NoError(t, err, "list heartbeats")

	for _, hb := range heartbeats {
		delCtx, delCancel := context.WithTimeout(context.Background(), 30*time.Second)
		_ = apiClient.Heartbeats.Delete(delCtx, hb.ID)
		delCancel()
	}
}

func cleanupE2EMonitors(t *testing.T, apiClient *betterstack.Client, urlPrefix string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	monitors, err := apiClient.Monitors.List(ctx)
	assert.NoError(t, err, "list monitors")

	for _, monitor := range monitors {
		if !strings.HasPrefix(monitor.Attributes.URL, urlPrefix) {
			continue
		}
		delCtx, delCancel := context.WithTimeout(context.Background(), 30*time.Second)
		_ = apiClient.Monitors.Delete(delCtx, monitor.ID)
		delCancel()
	}
}

func ensureBinary(t *testing.T, name string) {
	t.Helper()
	_, err := exec.LookPath(name)
	assert.NoError(t, err, "required binary %s", name)
}

func createKindCluster(t *testing.T, clusterName string) {
	t.Helper()
	args := []string{"create", "cluster", "--name", clusterName, "--wait", "120s"}
	cmd := exec.Command("kind", args...)
	runCmd(t, cmd)
}

func deleteKindCluster(clusterName string) {
	_ = exec.Command("kind", "delete", "cluster", "--name", clusterName).Run()
}

func fetchKubeconfig(t *testing.T, clusterName string) string {
	t.Helper()
	cmd := exec.Command("kind", "get", "kubeconfig", "--name", clusterName)
	out := runCmd(t, cmd)
	kubeconfigPath := filepath.Join(os.TempDir(), fmt.Sprintf("%s-kubeconfig", clusterName))
	assert.NoError(t, os.WriteFile(kubeconfigPath, out, 0o600), "write kubeconfig")
	return kubeconfigPath
}

func installCRD(t *testing.T, cfg *rest.Config, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	assert.NoError(t, err, "read CRD")

	scheme := runtime.NewScheme()
	_ = apiextensionsv1.AddToScheme(scheme)
	decoder := serializer.NewCodecFactory(scheme).UniversalDeserializer()
	obj, _, err := decoder.Decode(data, nil, nil)
	assert.NoError(t, err, "decode CRD")

	crd, ok := obj.(*apiextensionsv1.CustomResourceDefinition)
	assert.Bool(t, "decoded CRD", ok, true)

	extClient, err := apiextensionsclientset.NewForConfig(cfg)
	assert.NoError(t, err, "build extensions client")
	if _, err := extClient.ApiextensionsV1().CustomResourceDefinitions().Create(context.Background(), crd, metav1.CreateOptions{}); err != nil {
		if !errors.IsAlreadyExists(err) {
			assert.NoError(t, err, "create CRD")
		}
	}

	err = wait.PollImmediate(2*time.Second, 60*time.Second, func() (bool, error) {
		latest, err := extClient.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), crd.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, cond := range latest.Status.Conditions {
			if cond.Type == apiextensionsv1.Established && cond.Status == apiextensionsv1.ConditionTrue {
				return true, nil
			}
		}
		return false, nil
	})
	assert.NoError(t, err, "wait for CRD established")
}

func waitForMonitorCondition(k8sClient client.Client, namespace, name string, predicate func(*monitoringv1alpha1.BetterStackMonitor) bool) error {
	err := wait.PollImmediate(2*time.Second, 2*time.Minute, func() (bool, error) {
		obj := &monitoringv1alpha1.BetterStackMonitor{}
		if err := k8sClient.Get(context.Background(), client.ObjectKey{Name: name, Namespace: namespace}, obj); err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return predicate(obj), nil
	})
	return err
}

func waitForMonitorID(k8sClient client.Client, namespace, name string) (string, error) {
	var id string
	err := wait.PollImmediate(2*time.Second, 2*time.Minute, func() (bool, error) {
		obj := &monitoringv1alpha1.BetterStackMonitor{}
		if err := k8sClient.Get(context.Background(), client.ObjectKey{Name: name, Namespace: namespace}, obj); err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		if obj.Status.MonitorID != "" {
			id = obj.Status.MonitorID
			return true, nil
		}
		return false, nil
	})
	return id, err
}

func fetchRemoteMonitor(t *testing.T, ctx context.Context, client *betterstack.Client, id string) betterstack.Monitor {
	t.Helper()
	monitor, err := client.Monitors.Get(ctx, id)
	assert.NoError(t, err, "fetch remote monitor %s", id)
	return monitor
}

func monitorExists(ctx context.Context, client *betterstack.Client, id string) bool {
	_, err := client.Monitors.Get(ctx, id)
	if err == nil {
		return true
	}
	if betterstack.IsNotFound(err) {
		return false
	}
	return true
}

func waitForHeartbeatCondition(k8sClient client.Client, namespace, name string, predicate func(*monitoringv1alpha1.BetterStackHeartbeat) bool) error {
	err := wait.PollImmediate(2*time.Second, 2*time.Minute, func() (bool, error) {
		obj := &monitoringv1alpha1.BetterStackHeartbeat{}
		if err := k8sClient.Get(context.Background(), client.ObjectKey{Name: name, Namespace: namespace}, obj); err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return predicate(obj), nil
	})
	return err
}

func waitForHeartbeatID(k8sClient client.Client, namespace, name string) (string, error) {
	var id string
	err := wait.PollImmediate(2*time.Second, 2*time.Minute, func() (bool, error) {
		obj := &monitoringv1alpha1.BetterStackHeartbeat{}
		if err := k8sClient.Get(context.Background(), client.ObjectKey{Name: name, Namespace: namespace}, obj); err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		if obj.Status.HeartbeatID != "" {
			id = obj.Status.HeartbeatID
			return true, nil
		}
		return false, nil
	})
	return id, err
}

func fetchRemoteHeartbeat(t *testing.T, ctx context.Context, client *betterstack.Client, id string) betterstack.Heartbeat {
	t.Helper()
	hb, err := client.Heartbeats.Get(ctx, id)
	assert.NoError(t, err, "fetch remote heartbeat %s", id)
	return hb
}

func heartbeatExists(ctx context.Context, client *betterstack.Client, id string) bool {
	_, err := client.Heartbeats.Get(ctx, id)
	if err == nil {
		return true
	}
	if betterstack.IsNotFound(err) {
		return false
	}
	return true
}

func runCmd(t *testing.T, cmd *exec.Cmd) []byte {
	t.Helper()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	assert.NoError(t, err, "command %s failed\nstdout: %s\nstderr: %s", strings.Join(cmd.Args, " "), stdout.String(), stderr.String())
	return stdout.Bytes()
}

func metaConditionStatus(conditions []metav1.Condition, conditionType string) metav1.ConditionStatus {
	for _, cond := range conditions {
		if cond.Type == conditionType {
			return cond.Status
		}
	}
	return metav1.ConditionUnknown
}

func findCondition(conditions []metav1.Condition, condType string) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == condType {
			return &conditions[i]
		}
	}
	return nil
}

func projectRoot() string {
	_, filename, _, ok := stdruntime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}

func loadDotEnvIfPresent(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, ".env")
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		assert.NoError(t, err, "open .env")
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, "\"'")
		if key != "" && value != "" {
			_ = os.Setenv(key, value)
		}
	}
	assert.NoError(t, scanner.Err(), "read .env")
}
