// Copyright 2025 Canonical.

package db

import (
	"context"
	"encoding/json"

	"github.com/canonical/jimm/v3/internal/dbmodel"
	"github.com/canonical/jimm/v3/internal/errors"
	"github.com/canonical/jimm/v3/internal/servermon"
	"github.com/juju/zaputil/zapctx"
	"go.uber.org/zap"
)

const (
	// These constant are used to create the appropriate identifiers for controller proxy data.
	controllerProxySecretKind = "controllerProxy"
)

type controllerProxy struct {
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config"`
}

// GetControllerProxy retrieves the proxy configuration for the specified controller ID.
func (d *Database) GetControllerProxy(ctx context.Context, controllerName string) (_ string, _ map[string]interface{}, err error) {
	const op = errors.Op("database.GetControllerProxy")

	if err := d.ready(); err != nil {
		return "", nil, errors.E(op, err)
	}

	durationObserver := servermon.DurationObserver(servermon.DBQueryDurationHistogram, string(op))
	defer durationObserver()
	defer servermon.ErrorCounter(servermon.DBQueryErrorCount, &err, string(op))

	secret := dbmodel.NewSecret(controllerProxySecretKind, controllerName, nil)
	err = d.GetSecret(ctx, &secret)
	if err != nil {
		zapctx.Error(ctx, "failed to get secret data", zap.Error(err))
		return "", nil, errors.E(op, err)
	}
	// secretData is stored as a JSON object in the database.
	// So we need to unmarshal and extract the relevant fields.
	var secretData controllerProxy
	err = json.Unmarshal(secret.Data, &secretData)
	if err != nil {
		zapctx.Error(ctx, "failed to unmarshal secret data", zap.Error(err))
		return "", nil, errors.E(op, err)
	}
	return secretData.Type, secretData.Config, nil
}

// PutControllerProxy stores the proxy configuration for the specified controller ID.
func (d *Database) PutControllerProxy(ctx context.Context, controllerName string, proxyType string, config map[string]interface{}) (err error) {
	const op = errors.Op("database.PutControllerProxy")

	if err := d.ready(); err != nil {
		return errors.E(op, err)
	}

	durationObserver := servermon.DurationObserver(servermon.DBQueryDurationHistogram, string(op))
	defer durationObserver()
	defer servermon.ErrorCounter(servermon.DBQueryErrorCount, &err, string(op))

	secretData := controllerProxy{
		Type:   proxyType,
		Config: config,
	}
	dataJson, err := json.Marshal(secretData)
	if err != nil {
		zapctx.Error(ctx, "failed to marshal proxy secret data", zap.Error(err))
		return errors.E(op, err, "failed to marshal proxy secret data")
	}
	secret := dbmodel.NewSecret(controllerProxySecretKind, controllerName, dataJson)
	return d.UpsertSecret(ctx, &secret)
}

// DeleteControllerProxy removes the proxy configuration for the specified controller ID.
func (d *Database) DeleteControllerProxy(ctx context.Context, controllerName string) (err error) {
	const op = errors.Op("database.DeleteControllerProxy")

	if err := d.ready(); err != nil {
		return errors.E(op, err)
	}

	durationObserver := servermon.DurationObserver(servermon.DBQueryDurationHistogram, string(op))
	defer durationObserver()
	defer servermon.ErrorCounter(servermon.DBQueryErrorCount, &err, string(op))

	secret := dbmodel.NewSecret(controllerProxySecretKind, controllerName, nil)
	err = d.DeleteSecret(ctx, &secret)
	if err != nil {
		zapctx.Error(ctx, "failed to delete controller proxy", zap.Error(err))
		return errors.E(op, err, "failed to delete controller proxy")
	}
	return nil
}
