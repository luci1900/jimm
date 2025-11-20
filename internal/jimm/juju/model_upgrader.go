// Copyright 2025 Canonical.

package juju

import (
	"context"

	"github.com/juju/zaputil/zapctx"
	"go.uber.org/zap"

	"github.com/canonical/jimm/v3/internal/db"
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

	upgrades, err := j.Database.GetAllModelUpgrades(ctx)
	if err != nil {
		return errors.E(op, err)
	}

	// loop over all models upgrades registered
	for _, upgrade := range upgrades {
		err := j.Database.Transaction(func(database *db.Database) error {
			if err := database.GetModelUpgradeForUpdate(ctx, &upgrade); err != nil {
				return errors.E(op, err)
			}

			// switch over status
			// call appropriate state transition method
			// update status, commit

			// any calls made during state transition shouldn't block the transaction
			// that is holding the lock

			return nil
		})
		if err != nil {
			zapctx.Error(ctx, "error progressing model upgrade", zap.Error(err))
			continue
		}
	}

	return nil
}
