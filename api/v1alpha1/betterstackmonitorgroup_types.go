package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// BetterStackMonitorGroupSpec defines the desired state of a Better Stack monitor group.
type BetterStackMonitorGroupSpec struct {
	// Name is the human readable display name for the monitor group.
	Name string `json:"name,omitempty"`

	// TeamName assigns the group to a specific Better Stack team (needed when using a global token).
	TeamName string `json:"teamName,omitempty"`

	// SortIndex controls ordering of monitor groups within Better Stack dashboards.
	// +kubebuilder:validation:Minimum=0
	SortIndex *int `json:"sortIndex,omitempty"`

	// Paused marks the monitor group as paused in Better Stack.
	Paused *bool `json:"paused,omitempty"`

	// Better Stack API base URL. Defaults to https://uptime.betterstack.com/api/v2 when omitted.
	// +kubebuilder:validation:Format=uri
	BaseURL string `json:"baseURL,omitempty"`

	// APITokenSecretRef references the secret containing the Better Stack API token.
	// +kubebuilder:validation:Required
	APITokenSecretRef corev1.SecretKeySelector `json:"apiTokenSecretRef"`
}

// BetterStackMonitorGroupStatus represents the observed state of the monitor group.
type BetterStackMonitorGroupStatus struct {
	// MonitorGroupID is the identifier assigned by Better Stack.
	MonitorGroupID string `json:"monitorGroupID,omitempty"`

	// ObservedGeneration reflects the spec generation the controller last processed.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions capture the readiness state of the monitor group.
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastSyncedTime records when the operator last reconciled successfully.
	LastSyncedTime *metav1.Time `json:"lastSyncedTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:categories=betterstack,scope=Namespaced
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Name",type=string,JSONPath=".spec.name"
// +kubebuilder:printcolumn:name="ID",type=string,JSONPath=".status.monitorGroupID"
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
type BetterStackMonitorGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   BetterStackMonitorGroupSpec   `json:"spec"`
	Status BetterStackMonitorGroupStatus `json:"status"`
}

// +kubebuilder:object:root=true

type BetterStackMonitorGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []BetterStackMonitorGroup `json:"items"`
}

func (in *BetterStackMonitorGroupSpec) DeepCopyInto(out *BetterStackMonitorGroupSpec) {
	*out = *in
	if in.SortIndex != nil {
		out.SortIndex = new(int)
		*out.SortIndex = *in.SortIndex
	}
	if in.Paused != nil {
		out.Paused = new(bool)
		*out.Paused = *in.Paused
	}
}

func (in *BetterStackMonitorGroupSpec) DeepCopy() *BetterStackMonitorGroupSpec {
	if in == nil {
		return nil
	}
	out := new(BetterStackMonitorGroupSpec)
	in.DeepCopyInto(out)
	return out
}

func (in *BetterStackMonitorGroupStatus) DeepCopyInto(out *BetterStackMonitorGroupStatus) {
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

func (in *BetterStackMonitorGroupStatus) DeepCopy() *BetterStackMonitorGroupStatus {
	if in == nil {
		return nil
	}
	out := new(BetterStackMonitorGroupStatus)
	in.DeepCopyInto(out)
	return out
}

func (in *BetterStackMonitorGroup) DeepCopyInto(out *BetterStackMonitorGroup) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

func (in *BetterStackMonitorGroup) DeepCopy() *BetterStackMonitorGroup {
	if in == nil {
		return nil
	}
	out := new(BetterStackMonitorGroup)
	in.DeepCopyInto(out)
	return out
}

func (in *BetterStackMonitorGroup) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *BetterStackMonitorGroupList) DeepCopyInto(out *BetterStackMonitorGroupList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		out.Items = make([]BetterStackMonitorGroup, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
}

func (in *BetterStackMonitorGroupList) DeepCopy() *BetterStackMonitorGroupList {
	if in == nil {
		return nil
	}
	out := new(BetterStackMonitorGroupList)
	in.DeepCopyInto(out)
	return out
}

func (in *BetterStackMonitorGroupList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (s *BetterStackMonitorGroupStatus) SetCondition(cond metav1.Condition) {
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
