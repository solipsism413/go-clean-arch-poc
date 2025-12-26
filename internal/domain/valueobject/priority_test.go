package valueobject_test

import (
	"testing"

	"github.com/handiism/go-clean-arch-poc/internal/domain/valueobject"
)

func TestPriority_IsValid(t *testing.T) {
	tests := []struct {
		priority valueobject.Priority
		valid    bool
	}{
		{valueobject.PriorityLow, true},
		{valueobject.PriorityMedium, true},
		{valueobject.PriorityHigh, true},
		{valueobject.PriorityUrgent, true},
		{valueobject.Priority("INVALID"), false},
		{valueobject.Priority(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.priority), func(t *testing.T) {
			if got := tt.priority.IsValid(); got != tt.valid {
				t.Errorf("IsValid() = %v, want %v", got, tt.valid)
			}
		})
	}
}

func TestParsePriority(t *testing.T) {
	tests := []struct {
		input   string
		want    valueobject.Priority
		wantErr bool
	}{
		{"LOW", valueobject.PriorityLow, false},
		{"MEDIUM", valueobject.PriorityMedium, false},
		{"HIGH", valueobject.PriorityHigh, false},
		{"URGENT", valueobject.PriorityUrgent, false},
		{"INVALID", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := valueobject.ParsePriority(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePriority() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParsePriority() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPriority_Order(t *testing.T) {
	tests := []struct {
		priority valueobject.Priority
		order    int
	}{
		{valueobject.PriorityLow, 1},
		{valueobject.PriorityMedium, 2},
		{valueobject.PriorityHigh, 3},
		{valueobject.PriorityUrgent, 4},
	}

	for _, tt := range tests {
		t.Run(string(tt.priority), func(t *testing.T) {
			if got := tt.priority.Order(); got != tt.order {
				t.Errorf("Order() = %v, want %v", got, tt.order)
			}
		})
	}
}

func TestPriority_Comparison(t *testing.T) {
	t.Run("IsHigherThan", func(t *testing.T) {
		if !valueobject.PriorityUrgent.IsHigherThan(valueobject.PriorityLow) {
			t.Error("URGENT should be higher than LOW")
		}
		if valueobject.PriorityLow.IsHigherThan(valueobject.PriorityUrgent) {
			t.Error("LOW should not be higher than URGENT")
		}
	})

	t.Run("IsLowerThan", func(t *testing.T) {
		if !valueobject.PriorityLow.IsLowerThan(valueobject.PriorityUrgent) {
			t.Error("LOW should be lower than URGENT")
		}
		if valueobject.PriorityUrgent.IsLowerThan(valueobject.PriorityLow) {
			t.Error("URGENT should not be lower than LOW")
		}
	})

	t.Run("Compare", func(t *testing.T) {
		if valueobject.PriorityLow.Compare(valueobject.PriorityHigh) != -1 {
			t.Error("LOW compared to HIGH should return -1")
		}
		if valueobject.PriorityMedium.Compare(valueobject.PriorityMedium) != 0 {
			t.Error("MEDIUM compared to MEDIUM should return 0")
		}
		if valueobject.PriorityUrgent.Compare(valueobject.PriorityLow) != 1 {
			t.Error("URGENT compared to LOW should return 1")
		}
	})
}

func TestAllPriorities(t *testing.T) {
	priorities := valueobject.AllPriorities()

	if len(priorities) != 4 {
		t.Errorf("AllPriorities() returned %d priorities, want 4", len(priorities))
	}

	// Verify order
	expected := []valueobject.Priority{
		valueobject.PriorityLow,
		valueobject.PriorityMedium,
		valueobject.PriorityHigh,
		valueobject.PriorityUrgent,
	}

	for i, p := range priorities {
		if p != expected[i] {
			t.Errorf("AllPriorities()[%d] = %v, want %v", i, p, expected[i])
		}
	}
}
