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

func TestBetterStackMonitorLifecycle(t *testing.T) {
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
	if err != nil {
		t.Fatalf("build config: %v", err)
	}

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("add core scheme: %v", err)
	}
	if err := monitoringv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add cr scheme: %v", err)
	}

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	installCRD(t, cfg, rootDir)

	mgrCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: "0",
		},
		HealthProbeBindAddress: "0",
		LeaderElection:         false,
	})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	reconciler := &controllers.BetterStackMonitorReconciler{
		Client:     manager.GetClient(),
		Scheme:     manager.GetScheme(),
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
	if err := reconciler.SetupWithManager(manager); err != nil {
		t.Fatalf("setup reconciler: %v", err)
	}

	go func() {
		if err := manager.Start(mgrCtx); err != nil {
			t.Errorf("manager stopped: %v", err)
		}
	}()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "betterstack-credentials"},
		Data:       map[string][]byte{"api-key": []byte(token)},
	}
	if err := k8sClient.Create(context.Background(), secret); err != nil && !errors.IsAlreadyExists(err) {
		t.Fatalf("create secret: %v", err)
	}

	uniqueSuffix := time.Now().UnixNano()
	monitorURL := fmt.Sprintf("https://example.com/healthz-%d", uniqueSuffix)

	monitor := &monitoringv1alpha1.BetterStackMonitor{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "e2e-monitor"},
		Spec: monitoringv1alpha1.BetterStackMonitorSpec{
			URL:                       monitorURL,
			Name:                      "Initial E2E",
			MonitorType:               "status",
			CheckFrequencyMinutes:     3,
			ExpectedStatusCodes:       []int{200},
			RequestMethod:             "HEAD",
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
				LocalObjectReference: corev1.LocalObjectReference{Name: secret.Name},
				Key:                  "api-key",
			},
		},
	}

	if err := k8sClient.Create(context.Background(), monitor); err != nil {
		t.Fatalf("create monitor: %v", err)
	}

	waitForCondition(t, k8sClient, monitor.Namespace, monitor.Name, func(obj *monitoringv1alpha1.BetterStackMonitor) bool {
		return metaConditionStatus(obj.Status.Conditions, monitoringv1alpha1.ConditionReady) == metav1.ConditionTrue
	})

	monitorID := waitForMonitorID(t, k8sClient, monitor.Namespace, monitor.Name)
	apiClient := betterstack.NewClient("", token, reconciler.HTTPClient)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = apiClient.Monitors.Delete(ctx, monitorID)
	}()

	ctx, cancelFetch := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancelFetch()

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

	// Update monitor name and pause flag.
	if err := k8sClient.Get(context.Background(), client.ObjectKeyFromObject(monitor), monitor); err != nil {
		t.Fatalf("get monitor: %v", err)
	}
	monitor.Spec.Name = "Updated E2E"
	monitor.Spec.Paused = true
	monitor.Spec.CheckFrequencyMinutes = 5
	monitor.Spec.ExpectedStatusCodes = []int{204, 205}
	monitor.Spec.RequestMethod = "GET"
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
	if err := k8sClient.Update(context.Background(), monitor); err != nil {
		t.Fatalf("update monitor: %v", err)
	}

	waitForCondition(t, k8sClient, monitor.Namespace, monitor.Name, func(obj *monitoringv1alpha1.BetterStackMonitor) bool {
		return obj.Status.ObservedGeneration == obj.Generation
	})

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

	if err := k8sClient.Delete(context.Background(), monitor); err != nil {
		t.Fatalf("delete monitor: %v", err)
	}

	err = wait.PollImmediate(2*time.Second, 90*time.Second, func() (bool, error) {
		err := k8sClient.Get(context.Background(), client.ObjectKey{Name: monitor.Name, Namespace: monitor.Namespace}, monitor)
		if errors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
	if err != nil {
		t.Fatalf("waiting for monitor delete: %v", err)
	}

	ctxDelete, cancelDelete := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancelDelete()
	if exists := monitorExists(ctxDelete, apiClient, monitorID); exists {
		t.Fatalf("remote monitor %s still exists after deletion", monitorID)
	}
}

func ensureBinary(t *testing.T, name string) {
	t.Helper()
	if _, err := exec.LookPath(name); err != nil {
		t.Fatalf("required binary %q not found in PATH", name)
	}
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
	if err := os.WriteFile(kubeconfigPath, out, 0600); err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}
	return kubeconfigPath
}

func installCRD(t *testing.T, cfg *rest.Config, root string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "config", "crd", "bases", "monitoring.betterstack.io_betterstackmonitors.yaml"))
	if err != nil {
		t.Fatalf("read CRD: %v", err)
	}
	scheme := runtime.NewScheme()
	_ = apiextensionsv1.AddToScheme(scheme)
	decoder := serializer.NewCodecFactory(scheme).UniversalDeserializer()
	obj, _, err := decoder.Decode(data, nil, nil)
	if err != nil {
		t.Fatalf("decode CRD: %v", err)
	}
	crd, ok := obj.(*apiextensionsv1.CustomResourceDefinition)
	if !ok {
		t.Fatalf("decoded object is not a CRD")
	}
	extClient, err := apiextensionsclientset.NewForConfig(cfg)
	if err != nil {
		t.Fatalf("ext client: %v", err)
	}
	if _, err := extClient.ApiextensionsV1().CustomResourceDefinitions().Create(context.Background(), crd, metav1.CreateOptions{}); err != nil {
		if !errors.IsAlreadyExists(err) {
			t.Fatalf("create CRD: %v", err)
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
	if err != nil {
		t.Fatalf("waiting for CRD established: %v", err)
	}
}

func waitForCondition(t *testing.T, k8sClient client.Client, namespace, name string, predicate func(*monitoringv1alpha1.BetterStackMonitor) bool) {
	t.Helper()
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
	if err != nil {
		t.Fatalf("condition not met for %s/%s: %v", namespace, name, err)
	}
}

func waitForMonitorID(t *testing.T, k8sClient client.Client, namespace, name string) string {
	t.Helper()
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
	if err != nil {
		t.Fatalf("monitor ID not set for %s/%s: %v", namespace, name, err)
	}
	return id
}

func fetchRemoteMonitor(t *testing.T, ctx context.Context, client *betterstack.Client, id string) betterstack.Monitor {
	t.Helper()
	monitor, err := client.Monitors.Get(ctx, id)
	if err != nil {
		t.Fatalf("fetch remote monitor %s: %v", id, err)
	}
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

func runCmd(t *testing.T, cmd *exec.Cmd) []byte {
	t.Helper()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("command %s failed: %v\nstdout: %s\nstderr: %s", strings.Join(cmd.Args, " "), err, stdout.String(), stderr.String())
	}
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
		t.Fatalf("open .env: %v", err)
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
	if err := scanner.Err(); err != nil {
		t.Fatalf("read .env: %v", err)
	}
}
