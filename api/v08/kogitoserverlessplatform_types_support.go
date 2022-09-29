package v08

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewKogitoServerlessPlatformList --
func NewKogitoServerlessPlatformList() KogitoServerlessPlatformList {
	return KogitoServerlessPlatformList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupVersion.String(),
			Kind:       KogitoServerlessPlatformKind,
		},
	}
}

// NewKogitoServerlessPlatform --
func NewKogitoServerlessPlatform(namespace string, name string) KogitoServerlessPlatform {
	return KogitoServerlessPlatform{
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupVersion.String(),
			Kind:       KogitoServerlessPlatformKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
}

// ResyncStatusFullConfig copies the spec configuration into the status field.
func (in *KogitoServerlessPlatform) ResyncStatusFullConfig() {
	cl := in.Spec.DeepCopy()
	in.Status.KogitoServerlessPlatformSpec = *cl
}

// GetCondition returns the condition with the provided type.
func (in *KogitoServerlessPlatformStatus) GetCondition(condType PlatformConditionType) *PlatformCondition {
	for i := range in.Conditions {
		c := in.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

// SetErrorCondition --
func (in *KogitoServerlessPlatformStatus) SetErrorCondition(condType PlatformConditionType, reason string, err error) {
	in.SetConditions(PlatformCondition{
		Type:               condType,
		Status:             corev1.ConditionFalse,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            err.Error(),
	})
}

// SetConditions updates the resource to include the provided conditions.
//
// If a condition that we are about to add already exists and has the same status and
// reason then we are not going to update.
func (in *KogitoServerlessPlatformStatus) SetConditions(conditions ...PlatformCondition) {
	for _, condition := range conditions {
		if condition.LastUpdateTime.IsZero() {
			condition.LastUpdateTime = metav1.Now()
		}
		if condition.LastTransitionTime.IsZero() {
			condition.LastTransitionTime = metav1.Now()
		}

		currentCond := in.GetCondition(condition.Type)

		if currentCond != nil && currentCond.Status == condition.Status && currentCond.Reason == condition.Reason {
			return
		}
		// Do not update lastTransitionTime if the status of the condition doesn't change.
		if currentCond != nil && currentCond.Status == condition.Status {
			condition.LastTransitionTime = currentCond.LastTransitionTime
		}

		in.RemoveCondition(condition.Type)
		in.Conditions = append(in.Conditions, condition)
	}
}

// RemoveCondition removes the resource condition with the provided type.
func (in *KogitoServerlessPlatformStatus) RemoveCondition(condType PlatformConditionType) {
	newConditions := in.Conditions[:0]
	for _, c := range in.Conditions {
		if c.Type != condType {
			newConditions = append(newConditions, c)
		}
	}

	in.Conditions = newConditions
}

const (
	// ServiceTypeUser service user type label marker
	ServiceTypeUser = "user"
)

// +kubebuilder:object:generate=false
// ResourceCondition is a common type for all conditions
type ResourceCondition interface {
	GetType() string
	GetStatus() corev1.ConditionStatus
	GetLastUpdateTime() metav1.Time
	GetLastTransitionTime() metav1.Time
	GetReason() string
	GetMessage() string
}

var _ ResourceCondition = PlatformCondition{}

// GetConditions --
func (in *KogitoServerlessPlatformStatus) GetConditions() []ResourceCondition {
	res := make([]ResourceCondition, 0, len(in.Conditions))
	for _, c := range in.Conditions {
		res = append(res, c)
	}
	return res
}

// GetType --
func (c PlatformCondition) GetType() string {
	return string(c.Type)
}

// GetStatus --
func (c PlatformCondition) GetStatus() corev1.ConditionStatus {
	return c.Status
}

// GetLastUpdateTime --
func (c PlatformCondition) GetLastUpdateTime() metav1.Time {
	return c.LastUpdateTime
}

// GetLastTransitionTime --
func (c PlatformCondition) GetLastTransitionTime() metav1.Time {
	return c.LastTransitionTime
}

// GetReason --
func (c PlatformCondition) GetReason() string {
	return c.Reason
}

// GetMessage --
func (c PlatformCondition) GetMessage() string {
	return c.Message
}
