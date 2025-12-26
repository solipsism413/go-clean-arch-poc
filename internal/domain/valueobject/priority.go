package valueobject

// Priority represents the priority level of a task.
type Priority string

// Priority constants.
const (
	PriorityLow    Priority = "LOW"
	PriorityMedium Priority = "MEDIUM"
	PriorityHigh   Priority = "HIGH"
	PriorityUrgent Priority = "URGENT"
)

// validPriorities contains all valid priority values.
var validPriorities = map[Priority]bool{
	PriorityLow:    true,
	PriorityMedium: true,
	PriorityHigh:   true,
	PriorityUrgent: true,
}

// priorityOrder defines the ordering of priorities for comparison.
var priorityOrder = map[Priority]int{
	PriorityLow:    1,
	PriorityMedium: 2,
	PriorityHigh:   3,
	PriorityUrgent: 4,
}

// IsValid returns true if the priority is a valid Priority.
func (p Priority) IsValid() bool {
	return validPriorities[p]
}

// String returns the string representation of the priority.
func (p Priority) String() string {
	return string(p)
}

// ParsePriority parses a string into a Priority.
// Returns an error if the string is not a valid priority.
func ParsePriority(s string) (Priority, error) {
	priority := Priority(s)
	if !priority.IsValid() {
		return "", ErrInvalidPriority
	}
	return priority, nil
}

// AllPriorities returns all valid priorities in order from low to high.
func AllPriorities() []Priority {
	return []Priority{
		PriorityLow,
		PriorityMedium,
		PriorityHigh,
		PriorityUrgent,
	}
}

// Order returns the numeric order of the priority (1-4).
func (p Priority) Order() int {
	return priorityOrder[p]
}

// IsHigherThan returns true if this priority is higher than other.
func (p Priority) IsHigherThan(other Priority) bool {
	return p.Order() > other.Order()
}

// IsLowerThan returns true if this priority is lower than other.
func (p Priority) IsLowerThan(other Priority) bool {
	return p.Order() < other.Order()
}

// Compare returns -1 if p < other, 0 if p == other, 1 if p > other.
func (p Priority) Compare(other Priority) int {
	if p.Order() < other.Order() {
		return -1
	}
	if p.Order() > other.Order() {
		return 1
	}
	return 0
}
