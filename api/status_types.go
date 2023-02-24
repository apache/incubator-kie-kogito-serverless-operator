// Copyright 2023 Red Hat, Inc. and/or its affiliates
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"fmt"
	"reflect"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Based on https://github.com/knative/pkg/blob/980a33719a10024f45e44320316c5bd35cef18d6/apis/condition_set.go

// Status ...
// +kubebuilder:object:generate=true
type Status struct {
	// The latest available observations of a resource's current state.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions Conditions `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
	// The generation observed by the deployment controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

type ConditionAccessor interface {
	GetConditions() Conditions
	SetConditions(c Conditions)
	GetCondition(t ConditionType) *Condition
	GetTopLevelConditionType() ConditionType
	IsReady() bool
	GetTopLevelCondition() *Condition
}

func (s *Status) GetConditions() Conditions {
	return s.Conditions
}

func (s *Status) SetConditions(c Conditions) {
	s.Conditions = c
}

// GetCondition finds and returns the Condition that matches the ConditionType
// previously set on Conditions.
func (s *Status) GetCondition(t ConditionType) *Condition {
	for _, c := range s.Conditions {
		if c.Type == t {
			return &c
		}
	}
	return nil
}

type ConditionManager interface {
	ClearCondition(t ConditionType) error
	MarkTrue(t ConditionType)
	MarkTrueWithReason(t ConditionType, reason, messageFormat string, messageA ...interface{})
	MarkUnknown(t ConditionType, reason, messageFormat string, messageA ...interface{})
	MarkFalse(t ConditionType, reason, messageFormat string, messageA ...interface{})
	InitializeConditions()
}

var _ ConditionManager = &conditionManager{}

type conditionManager struct {
	accessor   ConditionAccessor
	ready      ConditionType
	dependents []ConditionType
}

func NewConditionManager(accessor ConditionAccessor, ready ConditionType, dependents ...ConditionType) ConditionManager {
	return &conditionManager{
		accessor:   accessor,
		ready:      ready,
		dependents: dependents,
	}
}

// setCondition sets or updates the Condition on Conditions for Condition.Type.
// If there is an update, Conditions are stored back sorted.
func (s *conditionManager) setCondition(cond Condition) {
	if s.accessor == nil {
		return
	}
	t := cond.Type
	var conditions Conditions
	for _, c := range s.accessor.GetConditions() {
		if c.Type != t {
			conditions = append(conditions, c)
		} else {
			// If we'd only update the LastTransitionTime, then return.
			cond.LastUpdateTime = c.LastUpdateTime
			if reflect.DeepEqual(cond, c) {
				return
			}
		}
	}
	cond.LastUpdateTime = metav1.NewTime(time.Now())
	conditions = append(conditions, cond)
	// Sorted for convenience of the consumer, i.e. kubectl.
	sort.Slice(conditions, func(i, j int) bool { return conditions[i].Type < conditions[j].Type })
	s.accessor.SetConditions(conditions)
}

func (s *conditionManager) isTerminal(t ConditionType) bool {
	for _, cond := range s.dependents {
		if cond == t {
			return true
		}
	}
	return t == s.ready
}

// ClearCondition removes the non-terminal condition that matches the ConditionType
// Not implemented for terminal conditions
func (s *conditionManager) ClearCondition(t ConditionType) error {
	var conditions Conditions

	if s.accessor == nil {
		return nil
	}
	// Terminal conditions are not handled as they can't be nil
	if s.isTerminal(t) {
		return fmt.Errorf("clearing terminal conditions not implemented")
	}
	cond := s.accessor.GetCondition(t)
	if cond == nil {
		return nil
	}
	for _, c := range s.accessor.GetConditions() {
		if c.Type != t {
			conditions = append(conditions, c)
		}
	}

	// Sorted for convenience of the consumer, i.e. kubectl.
	sort.Slice(conditions, func(i, j int) bool { return conditions[i].Type < conditions[j].Type })
	s.accessor.SetConditions(conditions)

	return nil
}

// MarkTrue sets the status of t to true
func (s *conditionManager) MarkTrue(t ConditionType) {
	// Set the specified condition.
	s.setCondition(Condition{
		Type:   t,
		Status: corev1.ConditionTrue,
	})
	s.recomputeReadiness(t)
}

// MarkTrueWithReason sets the status of t to true with the reason
func (s *conditionManager) MarkTrueWithReason(t ConditionType, reason, messageFormat string, messageA ...interface{}) {
	// set the specified condition
	s.setCondition(Condition{
		Type:    t,
		Status:  corev1.ConditionTrue,
		Reason:  reason,
		Message: fmt.Sprintf(messageFormat, messageA...),
	})
	s.recomputeReadiness(t)
}

// recomputeReadiness marks the ready condition to true if all other dependents are also true.
func (s *conditionManager) recomputeReadiness(t ConditionType) {
	if c := s.findUnreadyDependent(); c != nil {
		// Propagate unhappy dependent to happy condition.
		s.setCondition(Condition{
			Type:    s.ready,
			Status:  c.Status,
			Reason:  c.Reason,
			Message: c.Message,
		})
	} else if t != s.ready {
		// Set the happy condition to true.
		s.setCondition(Condition{
			Type:   s.ready,
			Status: corev1.ConditionTrue,
		})
	}
}

func (s *conditionManager) findUnreadyDependent() *Condition {
	// Do not modify the accessors condition order.
	conditions := s.accessor.GetConditions().DeepCopy()

	// Filter based on terminal status.
	n := 0
	for _, c := range conditions {
		if c.Type != s.ready {
			conditions[n] = c
			n++
		}
	}
	conditions = conditions[:n]

	// Sort set conditions by time.
	sort.Slice(conditions, func(i, j int) bool {
		return conditions[i].LastUpdateTime.Time.After(conditions[j].LastUpdateTime.Time)
	})

	// First check the conditions with Status == False.
	for _, c := range conditions {
		// False conditions trump Unknown.
		if c.IsFalse() {
			return &c
		}
	}
	// Second check for conditions with Status == Unknown.
	for _, c := range conditions {
		if c.IsUnknown() {
			return &c
		}
	}

	// If something was not initialized.
	if len(s.dependents) > len(conditions) {
		return &Condition{
			Status: corev1.ConditionUnknown,
		}
	}

	// All dependents are fine.
	return nil
}

// MarkUnknown sets the status of t to Unknown and also sets the ready condition
// to Unknown if no other dependent condition is in an error state.
func (s *conditionManager) MarkUnknown(t ConditionType, reason, messageFormat string, messageA ...interface{}) {
	// set the specified condition
	s.setCondition(Condition{
		Type:    t,
		Status:  corev1.ConditionUnknown,
		Reason:  reason,
		Message: fmt.Sprintf(messageFormat, messageA...),
	})

	// check the dependents.
	isDependent := false
	for _, cond := range s.dependents {
		c := s.accessor.GetCondition(cond)
		// Failed conditions trump Unknown conditions
		if c.IsFalse() {
			// Double check that the ready condition is also false.
			ready := s.accessor.GetCondition(s.ready)
			if !ready.IsFalse() {
				s.MarkFalse(s.ready, reason, messageFormat, messageA...)
			}
			return
		}
		if cond == t {
			isDependent = true
		}
	}

	if isDependent {
		// set the ready condition, if it is one of our dependent subconditions.
		s.setCondition(Condition{
			Type:    s.ready,
			Status:  corev1.ConditionUnknown,
			Reason:  reason,
			Message: fmt.Sprintf(messageFormat, messageA...),
		})
	}
}

// MarkFalse sets the status of t and the ready condition to False.
func (s *conditionManager) MarkFalse(t ConditionType, reason, messageFormat string, messageA ...interface{}) {
	types := []ConditionType{t}
	for _, cond := range s.dependents {
		if cond == t {
			types = append(types, s.ready)
		}
	}

	for _, t := range types {
		s.setCondition(Condition{
			Type:    t,
			Status:  corev1.ConditionFalse,
			Reason:  reason,
			Message: fmt.Sprintf(messageFormat, messageA...),
		})
	}
}

// InitializeConditions updates all Conditions in the ConditionSet to Unknown
// if not set.
func (s *conditionManager) InitializeConditions() {
	ready := s.accessor.GetCondition(s.ready)
	if ready == nil {
		ready = &Condition{
			Type:   s.ready,
			Status: corev1.ConditionUnknown,
		}
		s.setCondition(*ready)
	}
	// If the ready state is true, it implies that all of the terminal
	// subconditions must be true, so initialize any unset conditions to
	// true if our ready condition is true, otherwise unknown.
	status := corev1.ConditionUnknown
	if ready.Status == corev1.ConditionTrue {
		status = corev1.ConditionTrue
	}
	for _, t := range s.dependents {
		s.initializeTerminalCondition(t, status)
	}
}

// initializeTerminalCondition initializes a Condition to the given status if unset.
func (s *conditionManager) initializeTerminalCondition(t ConditionType, status corev1.ConditionStatus) *Condition {
	if c := s.accessor.GetCondition(t); c != nil {
		return c
	}
	c := Condition{
		Type:   t,
		Status: status,
	}
	s.setCondition(c)
	return &c
}
