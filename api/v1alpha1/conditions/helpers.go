package conditions

import (
	"time"

	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func SetStatusCondition(conditions *[]Condition, newCondition Condition) (changed bool) {
	if conditions == nil {
		return false
	}
	existingCondition := FindStatusCondition(*conditions, newCondition.Type)
	if existingCondition == nil {
		if newCondition.LastTransitionTime.IsZero() {
			newCondition.LastTransitionTime = meta.NewTime(time.Now())
		}
		*conditions = append(*conditions, newCondition)
		return true
	}

	if existingCondition.Status != newCondition.Status {
		existingCondition.Status = newCondition.Status
		if !newCondition.LastTransitionTime.IsZero() {
			existingCondition.LastTransitionTime = newCondition.LastTransitionTime
		} else {
			existingCondition.LastTransitionTime = meta.NewTime(time.Now())
		}
		changed = true
	}

	if existingCondition.Reason != newCondition.Reason {
		existingCondition.Reason = newCondition.Reason
		changed = true
	}
	if existingCondition.ObservedGeneration != newCondition.ObservedGeneration {
		existingCondition.ObservedGeneration = newCondition.ObservedGeneration
		changed = true
	}

	return changed
}

func RemoveStatusCondition(conditions *[]Condition, conditionType ConditionType) (removed bool) {
	if conditions == nil || len(*conditions) == 0 {
		return false
	}
	newConditions := make([]Condition, 0, len(*conditions)-1)
	for _, condition := range *conditions {
		if condition.Type != conditionType {
			newConditions = append(newConditions, condition)
		}
	}

	removed = len(*conditions) != len(newConditions)
	*conditions = newConditions

	return removed
}

// FindStatusCondition finds the conditionType in conditions.
func FindStatusCondition(conditions []Condition, conditionType ConditionType) *Condition {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}

	return nil
}

// IsStatusConditionPresentAndEqual returns true when conditionType is present and equal to status.
func IsStatusConditionPresentAndEqual(conditions []Condition, conditionType ConditionType, status ConditionStatus) bool {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return condition.Status == status
		}
	}
	return false
}

// IsAnyConditionWithStatus returns true when any condition in the set matches the status
func IsAnyConditionWithStatus(conditions []Condition, status ConditionStatus) bool {
	for _, condition := range conditions {
		if condition.Status == status {
			return true
		}
	}
	return false
}

// AreAllConditionsWithStatus returns true when all conditions in the set matches the status
// Returns false as soon as one doesn't match, which means not all conditions needs to be evaluated
// to return false
func AreAllConditionsWithStatus(conditions []Condition, status ConditionStatus) bool {
	completed := true
	for _, condition := range conditions {
		completed = condition.Status == status
		if !completed {
			break
		}
	}
	return completed
}
