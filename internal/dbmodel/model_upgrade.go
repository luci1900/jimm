// Copyright 2025 Canonical.

package dbmodel

import (
	"database/sql"
	"time"
)

const (
	ModelUpgradeStatusPending       = "pending"
	ModelUpgradeStatusBootstrapping = "bootstrapping"
	ModelUpgradeStatusMigrating     = "migrating"
	ModelUpgradeStatusCompleted     = "completed"
	ModelUpgradeStatusFailed        = "failed"
)

type ModelUpgrade struct {
	// Note this doesn't use the standard gorm.Model to avoid soft-deletes.
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	// ModelUUID is the UUID of the incoming model.
	ModelUUID sql.NullString

	// Status should be a proper typed string enum
	Status string

	// other fields like source/target controller, etc.
}
