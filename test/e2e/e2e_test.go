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
	"reflect"
	stdruntime "runtime"
	"strings"
	"testing"
	"time"

	monitoringv1alpha1 "loks0n/betterstack-operator/api/v1alpha1"
	"loks0n/betterstack-operator/controllers"
	"loks0n/betterstack-operator/pkg/betterstack"

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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
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
			FollowRedirects:           boolPtr(false),
			RememberCookies:           boolPtr(false),
			VerifySSL:                 boolPtr(true),
			Email:                     boolPtr(false),
			SMS:                       boolPtr(true),
			Call:                      boolPtr(false),
			Push:                      boolPtr(true),
			CriticalAlert:             boolPtr(true),
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
		_ = apiClient.DeleteMonitor(ctx, monitorID)
	}()

	ctx, cancelFetch := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancelFetch()

	remoteMonitor := fetchRemoteMonitor(t, ctx, apiClient, monitorID)
	expectAttrString(t, remoteMonitor.Attributes, "url", monitorURL)
	expectAttrString(t, remoteMonitor.Attributes, "monitor_type", "status")
	expectAttrString(t, remoteMonitor.Attributes, "http_method", "head")
	expectAttrInt(t, remoteMonitor.Attributes, "check_frequency", 180)
	expectAttrIntSlice(t, remoteMonitor.Attributes, "expected_status_codes", []int{200})
	expectAttrBool(t, remoteMonitor.Attributes, "follow_redirects", false)
	expectAttrBool(t, remoteMonitor.Attributes, "remember_cookies", false)
	expectAttrBool(t, remoteMonitor.Attributes, "verify_ssl", true)
	expectAttrBool(t, remoteMonitor.Attributes, "email", false)
	expectAttrBool(t, remoteMonitor.Attributes, "sms", true)
	expectAttrBool(t, remoteMonitor.Attributes, "call", false)
	expectAttrBool(t, remoteMonitor.Attributes, "push", true)
	expectAttrBool(t, remoteMonitor.Attributes, "critical_alert", true)
	expectAttrInt(t, remoteMonitor.Attributes, "team_wait", 120)
	expectAttrInt(t, remoteMonitor.Attributes, "domain_expiration", 14)
	expectAttrInt(t, remoteMonitor.Attributes, "ssl_expiration", 30)
	expectAttrInt(t, remoteMonitor.Attributes, "request_timeout", 10)
	expectAttrInt(t, remoteMonitor.Attributes, "recovery_period", 180)
	expectAttrInt(t, remoteMonitor.Attributes, "confirmation_period", 90)
	expectAttrString(t, remoteMonitor.Attributes, "ip_version", "ipv4")
	expectAttrStringSlice(t, remoteMonitor.Attributes, "maintenance_days", []string{"mon", "tue"})
	expectAttrString(t, remoteMonitor.Attributes, "maintenance_from", "01:00:00")
	expectAttrString(t, remoteMonitor.Attributes, "maintenance_to", "02:00:00")
	expectAttrString(t, remoteMonitor.Attributes, "maintenance_timezone", "UTC")
	expectRequestHeader(t, remoteMonitor.Attributes, "X-E2E", "initial")

	// Update monitor name and pause flag.
	if err := k8sClient.Get(context.Background(), client.ObjectKeyFromObject(monitor), monitor); err != nil {
		t.Fatalf("get monitor: %v", err)
	}
	monitor.Spec.Name = "Updated E2E"
	monitor.Spec.Paused = true
	monitor.Spec.CheckFrequencyMinutes = 5
	monitor.Spec.ExpectedStatusCodes = []int{204, 205}
	monitor.Spec.RequestMethod = "GET"
	monitor.Spec.FollowRedirects = boolPtr(true)
	monitor.Spec.RememberCookies = boolPtr(true)
	monitor.Spec.VerifySSL = boolPtr(false)
	monitor.Spec.Email = boolPtr(true)
	monitor.Spec.SMS = boolPtr(false)
	monitor.Spec.Push = boolPtr(false)
	monitor.Spec.CriticalAlert = boolPtr(false)
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
	expectAttrString(t, updatedMonitor.Attributes, "pronounceable_name", "Updated E2E")
	expectAttrBool(t, updatedMonitor.Attributes, "paused", true)
	expectAttrString(t, updatedMonitor.Attributes, "http_method", "get")
	expectAttrInt(t, updatedMonitor.Attributes, "check_frequency", 300)
	expectAttrIntSlice(t, updatedMonitor.Attributes, "expected_status_codes", []int{204, 205})
	expectAttrBool(t, updatedMonitor.Attributes, "follow_redirects", true)
	expectAttrBool(t, updatedMonitor.Attributes, "remember_cookies", true)
	expectAttrBool(t, updatedMonitor.Attributes, "verify_ssl", false)
	expectAttrBool(t, updatedMonitor.Attributes, "email", true)
	expectAttrBool(t, updatedMonitor.Attributes, "sms", false)
	expectAttrBool(t, updatedMonitor.Attributes, "push", false)
	expectAttrBool(t, updatedMonitor.Attributes, "critical_alert", false)
	expectAttrInt(t, updatedMonitor.Attributes, "team_wait", 60)
	expectAttrInt(t, updatedMonitor.Attributes, "domain_expiration", 7)
	expectAttrInt(t, updatedMonitor.Attributes, "ssl_expiration", 14)
	expectAttrInt(t, updatedMonitor.Attributes, "request_timeout", 45)
	expectAttrInt(t, updatedMonitor.Attributes, "recovery_period", 300)
	expectAttrInt(t, updatedMonitor.Attributes, "confirmation_period", 120)
	expectAttrString(t, updatedMonitor.Attributes, "ip_version", "ipv6")
	expectAttrStringSlice(t, updatedMonitor.Attributes, "maintenance_days", []string{"wed"})
	expectAttrString(t, updatedMonitor.Attributes, "maintenance_from", "02:00:00")
	expectAttrString(t, updatedMonitor.Attributes, "maintenance_to", "03:00:00")
	expectAttrString(t, updatedMonitor.Attributes, "maintenance_timezone", "Eastern Time (US & Canada)")
	expectRequestHeader(t, updatedMonitor.Attributes, "X-E2E", "updated")

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
	monitor, err := client.GetMonitor(ctx, id)
	if err != nil {
		t.Fatalf("fetch remote monitor %s: %v", id, err)
	}
	return monitor
}

func monitorExists(ctx context.Context, client *betterstack.Client, id string) bool {
	_, err := client.GetMonitor(ctx, id)
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

func boolPtr(v bool) *bool {
	return &v
}

func expectAttrString(t *testing.T, attrs map[string]any, key, expected string) {
	t.Helper()
	v, ok := attrs[key]
	if !ok {
		t.Fatalf("attribute %q missing", key)
	}
	if v == nil {
		t.Fatalf("attribute %q is nil", key)
	}
	str, ok := v.(string)
	if !ok {
		t.Fatalf("attribute %q not string: %T", key, v)
	}
	if str != expected {
		t.Fatalf("attribute %q mismatch: got %q want %q", key, str, expected)
	}
}

func expectAttrBool(t *testing.T, attrs map[string]any, key string, expected bool) {
	t.Helper()
	v, ok := attrs[key]
	if !ok {
		t.Fatalf("attribute %q missing", key)
	}
	b, ok := v.(bool)
	if !ok {
		t.Fatalf("attribute %q not bool: %T", key, v)
	}
	if b != expected {
		t.Fatalf("attribute %q mismatch: got %v want %v", key, b, expected)
	}
}

func expectAttrInt(t *testing.T, attrs map[string]any, key string, expected int) {
	t.Helper()
	v, ok := attrs[key]
	if !ok {
		t.Fatalf("attribute %q missing", key)
	}
	var got int
	switch n := v.(type) {
	case float64:
		got = int(n)
	case int:
		got = n
	case nil:
		t.Fatalf("attribute %q is nil", key)
	default:
		t.Fatalf("attribute %q not numeric: %T", key, v)
	}
	if got != expected {
		t.Fatalf("attribute %q mismatch: got %d want %d", key, got, expected)
	}
}

func expectAttrStringSlice(t *testing.T, attrs map[string]any, key string, expected []string) {
	t.Helper()
	v, ok := attrs[key]
	if !ok {
		t.Fatalf("attribute %q missing", key)
	}
	arr, ok := v.([]any)
	if !ok {
		t.Fatalf("attribute %q not array: %T", key, v)
	}
	got := make([]string, 0, len(arr))
	for _, item := range arr {
		s, ok := item.(string)
		if !ok {
			t.Fatalf("attribute %q contains non-string: %T", key, item)
		}
		got = append(got, s)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("attribute %q mismatch: got %v want %v", key, got, expected)
	}
}

func expectAttrIntSlice(t *testing.T, attrs map[string]any, key string, expected []int) {
	t.Helper()
	v, ok := attrs[key]
	if !ok {
		t.Fatalf("attribute %q missing", key)
	}
	arr, ok := v.([]any)
	if !ok {
		t.Fatalf("attribute %q not array: %T", key, v)
	}
	got := make([]int, 0, len(arr))
	for _, item := range arr {
		switch n := item.(type) {
		case float64:
			got = append(got, int(n))
		case int:
			got = append(got, n)
		default:
			t.Fatalf("attribute %q contains non-number: %T", key, item)
		}
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("attribute %q mismatch: got %v want %v", key, got, expected)
	}
}

func expectRequestHeader(t *testing.T, attrs map[string]any, name, value string) {
	t.Helper()
	v, ok := attrs["request_headers"]
	if !ok {
		t.Fatalf("attribute request_headers missing")
	}
	arr, ok := v.([]any)
	if !ok {
		t.Fatalf("request_headers not array: %T", v)
	}
	for _, item := range arr {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if obj["name"] == name && obj["value"] == value {
			return
		}
	}
	t.Fatalf("request header %q=%q not found in %v", name, value, arr)
}
