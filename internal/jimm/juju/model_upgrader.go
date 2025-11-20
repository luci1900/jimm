// Copyright 2025 Canonical.

package juju

import (
	"context"
	"fmt"

	"github.com/juju/zaputil/zapctx"
	"go.uber.org/zap"

	"github.com/canonical/jimm/v3/internal/db"
	"github.com/canonical/jimm/v3/internal/dbmodel"
	"github.com/canonical/jimm/v3/internal/errors"
	"github.com/canonical/jimm/v3/internal/servermon"
)

// TickModelUpgrades loops over models due for upgrade, checks their status
// and runs the logic for the next step in the upgrade process.
// This effectively implements progressing the state machine for each model's upgrade.
func (j *JujuManager) TickModelUpgrades(ctx context.Context) error {
	const op = "jimm.juju.TickModelUpgrades"
	zapctx.Info(ctx, string(op))
	durationObserver := servermon.DurationObserver(servermon.JimmMethodsDurationHistogram, string(op))
	defer durationObserver()

	// Obtain likely model upgrade IDs in a separate read transaction
	upgrades, err := j.Database.GetAllModelUpgrades(ctx)
	if err != nil {
		return errors.E(op, err)
	}

	for _, upgrade := range upgrades {
		err := j.Database.Transaction(func(database *db.Database) error {
			if err := database.GetModelUpgradeForUpdate(ctx, &upgrade); err != nil {
				// Ignore missing entries, they may have been concurrently deleted
				if errors.ErrorCode(err) != errors.CodeNotFound {
					return errors.E(op, err)
				}
			}

			var statusErr error
			switch upgrade.Status {
			case dbmodel.ModelUpgradeStatusPending:
				statusErr = j.handlePending(ctx, database, &upgrade)
			case dbmodel.ModelUpgradeStatusBootstrapping:
				//
			case dbmodel.ModelUpgradeStatusMigrating:
				//
			case dbmodel.ModelUpgradeStatusCompleted:
				//
			case dbmodel.ModelUpgradeStatusFailed:
				//
			default:
				return errors.E(op, fmt.Errorf("unhandled model upgrade status"))
			}

			if statusErr != nil {
				return errors.E(op, statusErr)
			}

			// Allow commit if no error
			return nil
		})
		if err != nil {
			zapctx.Error(ctx, "error progressing model upgrade", zap.Error(err))
			continue
		}
	}

	return nil
}

func (j *JujuManager) handlePending(ctx context.Context, database *db.Database, upgrade *dbmodel.ModelUpgrade) error {
	// make call to Juju controller
	// update model upgrade status
	return nil
}
