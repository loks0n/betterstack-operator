package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	BetterStackHeartbeatFinalizer = "betterstack.monitoring.loks0n/heartbeat-finalizer"
)

// BetterStackHeartbeatSpec defines the desired state of a Better Stack heartbeat.
type BetterStackHeartbeatSpec struct {
	// Name is the human readable display name for the heartbeat.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// PeriodSeconds controls how often the monitored system must report in before Better Stack flags the heartbeat as missing.
	// +kubebuilder:validation:Minimum=1
	PeriodSeconds int `json:"periodSeconds"`

	// GraceSeconds adds a tolerance window after the period expires before alerting.
	// +kubebuilder:validation:Minimum=0
	GraceSeconds int `json:"graceSeconds,omitempty"`

	// TeamName assigns the heartbeat to a specific Better Stack team (needed when using a global token).
	TeamName string `json:"teamName,omitempty"`

	// Contact preference overrides.
	Call          *bool `json:"call,omitempty"`
	SMS           *bool `json:"sms,omitempty"`
	Email         *bool `json:"email,omitempty"`
	Push          *bool `json:"push,omitempty"`
	CriticalAlert *bool `json:"criticalAlert,omitempty"`

	// TeamWaitSeconds delays escalation to the next team.
	// +kubebuilder:validation:Minimum=0
	TeamWaitSeconds int `json:"teamWaitSeconds,omitempty"`

	// HeartbeatGroupID associates the heartbeat with an existing group.
	// +kubebuilder:validation:Minimum=0
	HeartbeatGroupID *int `json:"heartbeatGroupID,omitempty"`

	// SortIndex controls ordering inside Better Stack dashboards.
	// +kubebuilder:validation:Minimum=0
	SortIndex *int `json:"sortIndex,omitempty"`

	// Paused marks the heartbeat as paused in Better Stack.
	Paused *bool `json:"paused,omitempty"`

	// Maintenance windows.
	// +kubebuilder:validation:Items={type=string,enum={mon,tue,wed,thu,fri,sat,sun}}
	MaintenanceDays     []string `json:"maintenanceDays,omitempty"`
	MaintenanceFrom     string   `json:"maintenanceFrom,omitempty"`
	MaintenanceTo       string   `json:"maintenanceTo,omitempty"`
	MaintenanceTimezone string   `json:"maintenanceTimezone,omitempty"`

	// PolicyID controls the alerting policy Better Stack applies.
	PolicyID *string `json:"policyID,omitempty"`

	// Better Stack API base URL. Defaults to https://uptime.betterstack.com/api/v2 when omitted.
	// +kubebuilder:validation:Format=uri
	BaseURL string `json:"baseURL,omitempty"`

	// APITokenSecretRef references the secret containing the Better Stack API token.
	// +kubebuilder:validation:Required
	APITokenSecretRef corev1.SecretKeySelector `json:"apiTokenSecretRef"`
}

// BetterStackHeartbeatStatus represents the observed state of the heartbeat.
type BetterStackHeartbeatStatus struct {
	// HeartbeatID is the identifier assigned by Better Stack.
	HeartbeatID string `json:"heartbeatID,omitempty"`

	// ObservedGeneration reflects the spec generation the controller last processed.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions capture the readiness state of the heartbeat.
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastSyncedTime records when the operator last reconciled successfully.
	LastSyncedTime *metav1.Time `json:"lastSyncedTime,omitempty"`
}

// SetCondition updates a condition on the status, creating or replacing it.
func (s *BetterStackHeartbeatStatus) SetCondition(cond metav1.Condition) {
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

// +kubebuilder:object:root=true
// +kubebuilder:resource:categories=betterstack,scope=Namespaced
// +kubebuilder:subresource:status

// BetterStackHeartbeat is the Schema for the betterstackheartbeats API.
type BetterStackHeartbeat struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BetterStackHeartbeatSpec   `json:"spec,omitempty"`
	Status BetterStackHeartbeatStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BetterStackHeartbeatList contains a list of BetterStackHeartbeat.
type BetterStackHeartbeatList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BetterStackHeartbeat `json:"items"`
}

// DeepCopyInto copies the receiver into out.
func (in *BetterStackHeartbeatSpec) DeepCopyInto(out *BetterStackHeartbeatSpec) {
	*out = *in
	if in.MaintenanceDays != nil {
		out.MaintenanceDays = make([]string, len(in.MaintenanceDays))
		copy(out.MaintenanceDays, in.MaintenanceDays)
	}
}

// DeepCopy creates a new copy of the receiver.
func (in *BetterStackHeartbeatSpec) DeepCopy() *BetterStackHeartbeatSpec {
	if in == nil {
		return nil
	}
	out := new(BetterStackHeartbeatSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies the receiver into out.
func (in *BetterStackHeartbeatStatus) DeepCopyInto(out *BetterStackHeartbeatStatus) {
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
func (in *BetterStackHeartbeatStatus) DeepCopy() *BetterStackHeartbeatStatus {
	if in == nil {
		return nil
	}
	out := new(BetterStackHeartbeatStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies the receiver into out.
func (in *BetterStackHeartbeat) DeepCopyInto(out *BetterStackHeartbeat) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy creates a new copy of the receiver.
func (in *BetterStackHeartbeat) DeepCopy() *BetterStackHeartbeat {
	if in == nil {
		return nil
	}
	out := new(BetterStackHeartbeat)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject satisfies the runtime.Object interface.
func (in *BetterStackHeartbeat) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto copies the receiver into out.
func (in *BetterStackHeartbeatList) DeepCopyInto(out *BetterStackHeartbeatList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		out.Items = make([]BetterStackHeartbeat, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
}

// DeepCopy creates a new copy of the receiver.
func (in *BetterStackHeartbeatList) DeepCopy() *BetterStackHeartbeatList {
	if in == nil {
		return nil
	}
	out := new(BetterStackHeartbeatList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject satisfies the runtime.Object interface.
func (in *BetterStackHeartbeatList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
