package conditions

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// New builds a metav1.Condition with optional transition time override.
func New(conditionType string, status metav1.ConditionStatus, reason, message string, transitionTime *metav1.Time) metav1.Condition {
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
