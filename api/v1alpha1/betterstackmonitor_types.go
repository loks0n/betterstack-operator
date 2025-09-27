package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	BetterStackMonitorFinalizer = "betterstack.monitoring.loks0n/finalizer"

	ConditionReady       = "Ready"
	ConditionCredentials = "CredentialsAvailable"
	ConditionSync        = "Synced"
)

// BetterStackMonitorSpec defines the desired state of a Better Stack monitor.
type BetterStackMonitorSpec struct {
	// URL is the endpoint Better Stack should monitor.
	URL string `json:"url"`

	// Name is the human readable display name for the monitor.
	Name string `json:"name,omitempty"`

	// MonitorType controls the Better Stack monitor type (status, expected_status_code, keyword, keyword_absence, ping, tcp, udp, smtp, pop, imap, dns, playwright).
	MonitorType string `json:"monitorType,omitempty"`

	// TeamName assigns the monitor to a specific Better Stack team (needed when using a global token).
	TeamName string `json:"teamName,omitempty"`

	// CheckFrequencyMinutes controls how often Better Stack checks the monitor.
	// Accepted values depend on your plan; Better Stack currently allows 0.5–30 minute intervals.
	CheckFrequencyMinutes int `json:"checkFrequencyMinutes,omitempty"`

	// Regions specifies the Better Stack regions to probe from.
	Regions []string `json:"regions,omitempty"`

	// RequestMethod overrides the HTTP method used during the check (for example GET or POST).
	RequestMethod string `json:"requestMethod,omitempty"`

	// ExpectedStatusCode sets a single expected HTTP status code treated as success.
	ExpectedStatusCode int `json:"expectedStatusCode,omitempty"`

	// ExpectedStatusCodes allows specifying multiple acceptable HTTP status codes.
	ExpectedStatusCodes []int `json:"expectedStatusCodes,omitempty"`

	// RequiredKeyword must be present/absent depending on the monitor type.
	RequiredKeyword string `json:"requiredKeyword,omitempty"`

	// Paused marks the monitor as paused in Better Stack.
	Paused bool `json:"paused,omitempty"`

	// Contact preference overrides.
	Email           *bool `json:"email,omitempty"`
	SMS             *bool `json:"sms,omitempty"`
	Call            *bool `json:"call,omitempty"`
	Push            *bool `json:"push,omitempty"`
	CriticalAlert   *bool `json:"criticalAlert,omitempty"`
	FollowRedirects *bool `json:"followRedirects,omitempty"`
	VerifySSL       *bool `json:"verifySSL,omitempty"`
	RememberCookies *bool `json:"rememberCookies,omitempty"`

	PolicyID             string `json:"policyID,omitempty"`
	ExpirationPolicyID   string `json:"expirationPolicyID,omitempty"`
	MonitorGroupID       string `json:"monitorGroupID,omitempty"`
	TeamWaitSeconds      int    `json:"teamWaitSeconds,omitempty"`
	DomainExpirationDays int    `json:"domainExpirationDays,omitempty"`
	SSLExpirationDays    int    `json:"sslExpirationDays,omitempty"`

	// Port is kept as an integer for CRD ergonomics and converted to the
	// string form expected by the Better Stack API (e.g. "443" or "25,465").
	Port int `json:"port,omitempty"`
	// RequestTimeoutSeconds is expressed in seconds for all monitor types. When
	// Better Stack expects millisecond values (ping, tcp, udp, smtp, pop, imap,
	// dns) the controller converts this value automatically.
	RequestTimeoutSeconds     int    `json:"requestTimeoutSeconds,omitempty"`
	RecoveryPeriodSeconds     int    `json:"recoveryPeriodSeconds,omitempty"`
	ConfirmationPeriodSeconds int    `json:"confirmationPeriodSeconds,omitempty"`
	IPVersion                 string `json:"ipVersion,omitempty"`

	MaintenanceDays     []string `json:"maintenanceDays,omitempty"`
	MaintenanceFrom     string   `json:"maintenanceFrom,omitempty"`
	MaintenanceTo       string   `json:"maintenanceTo,omitempty"`
	MaintenanceTimezone string   `json:"maintenanceTimezone,omitempty"`

	RequestHeaders       []BetterStackHeader `json:"requestHeaders,omitempty"`
	RequestBody          string              `json:"requestBody,omitempty"`
	AuthUsername         string              `json:"authUsername,omitempty"`
	AuthPassword         string              `json:"authPassword,omitempty"`
	EnvironmentVariables map[string]string   `json:"environmentVariables,omitempty"`
	PlaywrightScript     string              `json:"playwrightScript,omitempty"`
	ScenarioName         string              `json:"scenarioName,omitempty"`

	// AdditionalAttributes are raw Better Stack API attributes merged into the payload.
	AdditionalAttributes map[string]string `json:"additionalAttributes,omitempty"`

	// Better Stack API base URL. Defaults to https://uptime.betterstack.com/api/v2 when omitted.
	BaseURL string `json:"baseURL,omitempty"`

	// APITokenSecretRef references the secret containing the Better Stack API token.
	APITokenSecretRef corev1.SecretKeySelector `json:"apiTokenSecretRef"`
}

// BetterStackHeader represents an HTTP header definition for a monitor.
type BetterStackHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// DeepCopyInto copies the receiver into the provided out struct.
func (in *BetterStackMonitorSpec) DeepCopyInto(out *BetterStackMonitorSpec) {
	*out = *in
	if in.Regions != nil {
		out.Regions = make([]string, len(in.Regions))
		copy(out.Regions, in.Regions)
	}
	if in.ExpectedStatusCodes != nil {
		out.ExpectedStatusCodes = make([]int, len(in.ExpectedStatusCodes))
		copy(out.ExpectedStatusCodes, in.ExpectedStatusCodes)
	}
	if in.MaintenanceDays != nil {
		out.MaintenanceDays = make([]string, len(in.MaintenanceDays))
		copy(out.MaintenanceDays, in.MaintenanceDays)
	}
	if in.RequestHeaders != nil {
		out.RequestHeaders = make([]BetterStackHeader, len(in.RequestHeaders))
		copy(out.RequestHeaders, in.RequestHeaders)
	}
	if in.AdditionalAttributes != nil {
		out.AdditionalAttributes = make(map[string]string, len(in.AdditionalAttributes))
		for key, val := range in.AdditionalAttributes {
			out.AdditionalAttributes[key] = val
		}
	}
	if in.EnvironmentVariables != nil {
		out.EnvironmentVariables = make(map[string]string, len(in.EnvironmentVariables))
		for key, val := range in.EnvironmentVariables {
			out.EnvironmentVariables[key] = val
		}
	}
}

// DeepCopy creates a new copy of the receiver.
func (in *BetterStackMonitorSpec) DeepCopy() *BetterStackMonitorSpec {
	if in == nil {
		return nil
	}
	out := new(BetterStackMonitorSpec)
	in.DeepCopyInto(out)
	return out
}

// BetterStackMonitorStatus represents the observed state of the monitor.
type BetterStackMonitorStatus struct {
	// MonitorID is the identifier assigned by Better Stack.
	MonitorID string `json:"monitorID,omitempty"`

	// ObservedGeneration reflects the spec generation the controller last processed.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions capture the readiness state of the monitor.
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastSyncedTime records when the operator last reconciled successfully.
	LastSyncedTime *metav1.Time `json:"lastSyncedTime,omitempty"`
}

// DeepCopyInto copies the receiver into the provided out struct.
func (in *BetterStackMonitorStatus) DeepCopyInto(out *BetterStackMonitorStatus) {
	*out = *in
	if in.Conditions != nil {
		out.Conditions = make([]metav1.Condition, len(in.Conditions))
		for i := range in.Conditions {
			in.Conditions[i].DeepCopyInto(&out.Conditions[i])
		}
	}
	if in.LastSyncedTime != nil {
		out.LastSyncedTime = in.LastSyncedTime.DeepCopy()
	}
}

// DeepCopy creates a new copy of the receiver.
func (in *BetterStackMonitorStatus) DeepCopy() *BetterStackMonitorStatus {
	if in == nil {
		return nil
	}
	out := new(BetterStackMonitorStatus)
	in.DeepCopyInto(out)
	return out
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// BetterStackMonitor is the Schema for the betterstackmonitors API.
// +kubebuilder:resource:categories=betterstack,scope=Namespaced
// +kubebuilder:printcolumn:name="URL",type=string,JSONPath=".spec.url"
// +kubebuilder:printcolumn:name="ID",type=string,JSONPath=".status.monitorID"
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
type BetterStackMonitor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BetterStackMonitorSpec   `json:"spec,omitempty"`
	Status BetterStackMonitorStatus `json:"status,omitempty"`
}

// DeepCopyInto copies the receiver into the provided out struct.
func (in *BetterStackMonitor) DeepCopyInto(out *BetterStackMonitor) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy creates a new copy of the receiver.
func (in *BetterStackMonitor) DeepCopy() *BetterStackMonitor {
	if in == nil {
		return nil
	}
	out := new(BetterStackMonitor)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject satisfies the runtime.Object interface.
func (in *BetterStackMonitor) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// +kubebuilder:object:root=true

// BetterStackMonitorList contains a list of BetterStackMonitor.
type BetterStackMonitorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BetterStackMonitor `json:"items"`
}

// DeepCopyInto copies the receiver into the provided out struct.
func (in *BetterStackMonitorList) DeepCopyInto(out *BetterStackMonitorList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		out.Items = make([]BetterStackMonitor, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
}

// DeepCopy creates a new copy of the receiver.
func (in *BetterStackMonitorList) DeepCopy() *BetterStackMonitorList {
	if in == nil {
		return nil
	}
	out := new(BetterStackMonitorList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject satisfies the runtime.Object interface.
func (in *BetterStackMonitorList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// SetCondition updates a condition on the status, creating or replacing it.
func (s *BetterStackMonitorStatus) SetCondition(cond metav1.Condition) {
	var conditions []metav1.Condition
	replaced := false
	for _, existing := range s.Conditions {
		if existing.Type == cond.Type {
			conditions = append(conditions, cond)
			replaced = true
			continue
		}
		conditions = append(conditions, existing)
	}
	if !replaced {
		conditions = append(conditions, cond)
	}
	s.Conditions = conditions
}
