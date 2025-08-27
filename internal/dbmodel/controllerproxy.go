// Copyright 2025 Canonical.

package dbmodel

import (
	"time"

	"github.com/juju/errors"
	"github.com/juju/juju/proxy"
)

// ControllerProxy represents a proxy configuration for accessing a controller.
type ControllerProxy struct {

	// Note that we do not use gorm.Model to avoid the use of soft-deletes.
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	ControllerId uint

	Type   string
	Config Map
}

// ProxyFactory defines an interface for creating proxier instances from a type key and configuration map.
type ProxyFactory interface {
	ProxierFromConfig(typeKey string, config map[string]interface{}) (proxy.Proxier, error)
}

// ToProxier attempts to convert the stored proxy configuration into a working
// proxier object using the provided proxy factory.
func (c ControllerProxy) ToProxier(proxyFactory ProxyFactory) (proxy.Proxier, error) {
	// The factory uses the Config map to decode settings into the specific config struct
	// for each proxier type. This process has two key effects:
	// - Any keys in Config that do not exist in the config struct are ignored.
	// - Any fields in the config struct that are missing from Config are set to their zero value.
	// As a result, the proxier receives a config struct without validation of its values.
	// If required config values are missing or incorrect, the proxy may fail to start when `.Start()` is called.
	proxier, err := proxyFactory.ProxierFromConfig(c.Type, c.Config)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return proxier, nil
}
