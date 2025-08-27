package db

import (
	"context"

	"github.com/canonical/jimm/v3/internal/dbmodel"
	"github.com/canonical/jimm/v3/internal/errors"
	"github.com/canonical/jimm/v3/internal/servermon"
)

// GetControllerProxy retrieves the proxy configuration for the specified controller ID.
func (d *Database) GetControllerProxy(ctx context.Context, controllerID uint) (proxy *dbmodel.ControllerProxy, err error) {
	const op = errors.Op("db.GetControllerProxy")
	if err := d.ready(); err != nil {
		return nil, errors.E(op, err)
	}

	durationObserver := servermon.DurationObserver(servermon.DBQueryDurationHistogram, string(op))
	defer durationObserver()
	defer servermon.ErrorCounter(servermon.DBQueryErrorCount, &err, string(op))

	db := d.DB.WithContext(ctx)

	proxy = &dbmodel.ControllerProxy{}
	if err := db.Where("controller_id = ?", controllerID).First(proxy).Error; err != nil {
		return nil, errors.E(op, dbError(err))
	}
	return proxy, nil
}

// AddControllerProxy adds a new proxy configuration for the specified controller ID.
func (d *Database) AddControllerProxy(ctx context.Context, proxy dbmodel.ControllerProxy) (err error) {
	const op = errors.Op("db.AddControllerProxy")
	if err := d.ready(); err != nil {
		return errors.E(op, err)
	}

	durationObserver := servermon.DurationObserver(servermon.DBQueryDurationHistogram, string(op))
	defer durationObserver()
	defer servermon.ErrorCounter(servermon.DBQueryErrorCount, &err, string(op))

	db := d.DB.WithContext(ctx)

	if err := db.Create(&proxy).Error; err != nil {
		return errors.E(op, dbError(err))
	}
	return nil
}
