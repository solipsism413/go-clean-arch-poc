package seed

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	userUseCase "github.com/handiism/go-clean-arch-poc/internal/application/usecase/user"
	"github.com/handiism/go-clean-arch-poc/internal/application/validation"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/database/postgres"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/database/repository"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type userSeed struct {
	Email    string
	Password string
	Name     string
	RoleName string
}

type labelSeed struct {
	Name  string
	Color string
}

type taskSeed struct {
	ID            uuid.UUID
	Title         string
	Description   string
	Status        string
	Priority      string
	CreatorEmail  string
	AssigneeEmail string
	LabelNames    []string
	DueInDays     int
}

var (
	devUsers = []userSeed{
		{Email: "admin.seed@taskmanager.local", Password: "password123", Name: "Seed Admin", RoleName: entity.RoleAdmin},
		{Email: "manager.seed@taskmanager.local", Password: "password123", Name: "Seed Manager", RoleName: entity.RoleManager},
		{Email: "member.seed@taskmanager.local", Password: "password123", Name: "Seed Member", RoleName: entity.RoleMember},
		{Email: "viewer.seed@taskmanager.local", Password: "password123", Name: "Seed Viewer", RoleName: entity.RoleViewer},
	}

	devLabels = []labelSeed{
		{Name: "backend", Color: "#2563EB"},
		{Name: "frontend", Color: "#7C3AED"},
		{Name: "urgent", Color: "#DC2626"},
	}

	devTasks = []taskSeed{
		{
			ID:            uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			Title:         "Seed task: stabilize API boot flow",
			Description:   "Verify the local stack starts cleanly and document any environment-specific issues.",
			Status:        "IN_PROGRESS",
			Priority:      "HIGH",
			CreatorEmail:  "admin.seed@taskmanager.local",
			AssigneeEmail: "manager.seed@taskmanager.local",
			LabelNames:    []string{"backend", "urgent"},
			DueInDays:     2,
		},
		{
			ID:            uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			Title:         "Seed task: prepare dashboard UI polish",
			Description:   "Review current task board usability and capture a small set of UI refinements.",
			Status:        "TODO",
			Priority:      "MEDIUM",
			CreatorEmail:  "manager.seed@taskmanager.local",
			AssigneeEmail: "member.seed@taskmanager.local",
			LabelNames:    []string{"frontend"},
			DueInDays:     5,
		},
		{
			ID:            uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			Title:         "Seed task: audit read-only access",
			Description:   "Check the viewer role can browse data without getting write permissions.",
			Status:        "IN_REVIEW",
			Priority:      "LOW",
			CreatorEmail:  "admin.seed@taskmanager.local",
			AssigneeEmail: "viewer.seed@taskmanager.local",
			LabelNames:    []string{"backend"},
			DueInDays:     7,
		},
	}
)

// SeedDevelopmentData seeds deterministic development data without changing migrations.
func SeedDevelopmentData(ctx context.Context, db *postgres.Database, logger *slog.Logger) error {
	if db == nil {
		return fmt.Errorf("database is required")
	}

	tm := postgres.NewTransactionManager(db.Pool)
	userService := userUseCase.NewUserUseCase(
		repository.NewUserRepository(db.Pool),
		repository.NewRoleRepository(db.Pool),
		nil,
		nil,
		tm,
		validation.NewValidator(),
		logger,
	)

	if err := userService.SeedSystemRoles(ctx); err != nil {
		return fmt.Errorf("seed system roles: %w", err)
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin seed transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	queries := sqlc.New(tx)
	userIDs := make(map[string]uuid.UUID, len(devUsers))

	for _, seedUser := range devUsers {
		userID, err := upsertUser(ctx, queries, seedUser)
		if err != nil {
			return err
		}

		role, err := queries.GetRoleByName(ctx, seedUser.RoleName)
		if err != nil {
			return fmt.Errorf("load role %q: %w", seedUser.RoleName, err)
		}

		if err := queries.AssignRoleToUser(ctx, sqlc.AssignRoleToUserParams{UserID: userID, RoleID: role.ID}); err != nil {
			return fmt.Errorf("assign role %q to %q: %w", seedUser.RoleName, seedUser.Email, err)
		}

		userIDs[seedUser.Email] = userID
	}

	labelIDs, err := upsertLabels(ctx, queries)
	if err != nil {
		return err
	}

	for _, seedTask := range devTasks {
		creatorID, ok := userIDs[seedTask.CreatorEmail]
		if !ok {
			return fmt.Errorf("missing creator %q", seedTask.CreatorEmail)
		}

		var assigneeID *uuid.UUID
		if seedTask.AssigneeEmail != "" {
			id, ok := userIDs[seedTask.AssigneeEmail]
			if !ok {
				return fmt.Errorf("missing assignee %q", seedTask.AssigneeEmail)
			}
			assigneeID = &id
		}

		if err := upsertTask(ctx, tx, seedTask, creatorID, assigneeID); err != nil {
			return err
		}

		for _, labelName := range seedTask.LabelNames {
			labelID, ok := labelIDs[labelName]
			if !ok {
				return fmt.Errorf("missing label %q", labelName)
			}

			if err := queries.AddLabelToTask(ctx, sqlc.AddLabelToTaskParams{TaskID: seedTask.ID, LabelID: labelID}); err != nil {
				return fmt.Errorf("attach label %q to task %q: %w", labelName, seedTask.Title, err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit seed transaction: %w", err)
	}

	logger.Info("development seed applied",
		"users", len(devUsers),
		"labels", len(devLabels),
		"tasks", len(devTasks),
	)

	return nil
}

func upsertUser(ctx context.Context, queries *sqlc.Queries, seedUser userSeed) (uuid.UUID, error) {
	existingUser, lookupErr := queries.GetUserByEmail(ctx, seedUser.Email)
	if lookupErr != nil && !errors.Is(lookupErr, pgx.ErrNoRows) {
		return uuid.Nil, fmt.Errorf("load user %q: %w", seedUser.Email, lookupErr)
	}

	user, err := entity.NewUser(seedUser.Email, seedUser.Password, seedUser.Name)
	if err != nil {
		return uuid.Nil, fmt.Errorf("build user %q: %w", seedUser.Email, err)
	}

	if errors.Is(lookupErr, pgx.ErrNoRows) {
		if _, createErr := queries.CreateUser(ctx, sqlc.CreateUserParams{
			ID:           user.ID,
			Email:        user.Email,
			PasswordHash: user.PasswordHash,
			Name:         user.Name,
			CreatedAt:    user.CreatedAt,
			UpdatedAt:    user.UpdatedAt,
		}); createErr != nil {
			return uuid.Nil, fmt.Errorf("create user %q: %w", seedUser.Email, createErr)
		}

		return user.ID, nil
	}

	if _, updateErr := queries.UpdateUser(ctx, sqlc.UpdateUserParams{
		ID:           existingUser.ID,
		Email:        seedUser.Email,
		PasswordHash: user.PasswordHash,
		Name:         seedUser.Name,
	}); updateErr != nil {
		return uuid.Nil, fmt.Errorf("update user %q: %w", seedUser.Email, updateErr)
	}

	return existingUser.ID, nil
}

func upsertLabels(ctx context.Context, queries *sqlc.Queries) (map[string]uuid.UUID, error) {
	rows, err := queries.ListLabels(ctx)
	if err != nil {
		return nil, fmt.Errorf("list labels: %w", err)
	}

	labelIDs := make(map[string]uuid.UUID, len(devLabels))
	existingByName := make(map[string]sqlc.Label, len(rows))
	for _, row := range rows {
		existingByName[row.Name] = row
	}

	for _, seedLabel := range devLabels {
		if existingLabel, ok := existingByName[seedLabel.Name]; ok {
			if _, err := queries.UpdateLabel(ctx, sqlc.UpdateLabelParams{
				ID:    existingLabel.ID,
				Name:  seedLabel.Name,
				Color: seedLabel.Color,
			}); err != nil {
				return nil, fmt.Errorf("update label %q: %w", seedLabel.Name, err)
			}

			labelIDs[seedLabel.Name] = existingLabel.ID
			continue
		}

		label, err := entity.NewLabel(seedLabel.Name, seedLabel.Color)
		if err != nil {
			return nil, fmt.Errorf("build label %q: %w", seedLabel.Name, err)
		}

		createdLabel, err := queries.CreateLabel(ctx, sqlc.CreateLabelParams{
			ID:        label.ID,
			Name:      label.Name,
			Color:     label.Color,
			CreatedAt: label.CreatedAt,
			UpdatedAt: label.UpdatedAt,
		})
		if err != nil {
			return nil, fmt.Errorf("create label %q: %w", seedLabel.Name, err)
		}

		labelIDs[seedLabel.Name] = createdLabel.ID
	}

	return labelIDs, nil
}

func upsertTask(ctx context.Context, tx pgx.Tx, seedTask taskSeed, creatorID uuid.UUID, assigneeID *uuid.UUID) error {
	description := seedTask.Description
	dueDate := time.Now().UTC().AddDate(0, 0, seedTask.DueInDays)

	var assignee pgtype.UUID
	if assigneeID != nil {
		assignee = pgtype.UUID{Bytes: *assigneeID, Valid: true}
	}

	_, err := tx.Exec(ctx, `
		INSERT INTO tasks (id, title, description, status, priority, due_date, assignee_id, creator_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			status = EXCLUDED.status,
			priority = EXCLUDED.priority,
			due_date = EXCLUDED.due_date,
			assignee_id = EXCLUDED.assignee_id,
			creator_id = EXCLUDED.creator_id,
			updated_at = NOW()
	`, seedTask.ID, seedTask.Title, description, seedTask.Status, seedTask.Priority, dueDate, assignee, creatorID)
	if err != nil {
		return fmt.Errorf("upsert task %q: %w", seedTask.Title, err)
	}

	return nil
}
