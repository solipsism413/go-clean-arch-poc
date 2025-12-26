package entity

import (
	"time"

	"github.com/google/uuid"
)

// Label represents a label that can be attached to tasks.
// Labels help organize and categorize tasks.
type Label struct {
	// ID is the unique identifier for the label.
	ID uuid.UUID

	// Name is the display name of the label.
	Name string

	// Color is the hex color code for the label (e.g., "#FF5733").
	Color string

	// CreatedAt is the timestamp when the label was created.
	CreatedAt time.Time

	// UpdatedAt is the timestamp when the label was last updated.
	UpdatedAt time.Time
}

// NewLabel creates a new Label with the given parameters.
func NewLabel(name, color string) (*Label, error) {
	if name == "" {
		return nil, ErrEmptyLabelName
	}
	if color == "" {
		return nil, ErrEmptyLabelColor
	}

	now := time.Now().UTC()
	return &Label{
		ID:        uuid.New(),
		Name:      name,
		Color:     color,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// UpdateName updates the label name.
func (l *Label) UpdateName(name string) error {
	if name == "" {
		return ErrEmptyLabelName
	}
	l.Name = name
	l.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateColor updates the label color.
func (l *Label) UpdateColor(color string) error {
	if color == "" {
		return ErrEmptyLabelColor
	}
	l.Color = color
	l.UpdatedAt = time.Now().UTC()
	return nil
}
