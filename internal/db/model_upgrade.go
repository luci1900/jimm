// Copyright 2025 Canonical.

package db

import (
	"context"

	"github.com/canonical/jimm/v3/internal/dbmodel"
	"github.com/canonical/jimm/v3/internal/errors"
	"github.com/canonical/jimm/v3/internal/servermon"
	"gorm.io/gorm/clause"
)

func (d *Database) GetAllModelUpgrades(ctx context.Context) (_ []dbmodel.ModelUpgrade, err error) {
	const op = "db.GetAllModelUpgrades"

	if erre := d.ready(); erre != nil {
		return nil, errors.E(op, err)
	}

	durationObserver := servermon.DurationObserver(servermon.DBQueryDurationHistogram, string(op))
	defer durationObserver()
	defer servermon.ErrorCounter(servermon.DBQueryErrorCount, &err, string(op))

	var upgrades []dbmodel.ModelUpgrade
	db := d.DB.WithContext(ctx)
	err = db.Find(&upgrades).Error
	if err != nil {
		return nil, errors.E(op, dbError(err))
	}
	return upgrades, nil
}

func (d *Database) GetModelUpgradeForUpdate(ctx context.Context, upgrade *dbmodel.ModelUpgrade) (err error) {
	const op = "db.GetModelUpgradeForUpdate"

	durationObserver := servermon.DurationObserver(servermon.DBQueryDurationHistogram, string(op))
	defer durationObserver()
	defer servermon.ErrorCounter(servermon.DBQueryErrorCount, &err, string(op))

	if err := d.ready(); err != nil {
		return errors.E(err)
	}

	db := d.DB.WithContext(ctx)

	err = db.Where("id = ?", upgrade.ID).Clauses(clause.Locking{Strength: "UPDATE"}).First(upgrade).Error
	if err != nil {
		err = dbError(err)
		if errors.ErrorCode(err) == errors.CodeNotFound {
			return errors.E(op, errors.CodeNotFound, "model upgrade not found")
		}
		return errors.E(op, err)
	}
	return nil
}
