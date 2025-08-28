package juju_test

import (
	"testing"

	"github.com/canonical/jimm/v3/internal/jimm/juju"
	qt "github.com/frankban/quicktest"
	"github.com/juju/juju/caas/kubernetes/provider/proxy"
	"github.com/juju/juju/proxy/factory"
)

func TestControllerProxy(t *testing.T) {
	c := qt.New(t)
	rawConfig := map[string]interface{}{
		"api-host":              "https://127.0.0.1:443",
		"ca-cert":               "cadata====",
		"namespace":             "test",
		"remote-port":           "8123",
		"service":               "test",
		"service-account-token": "token====",
	}
	controllerProxy := juju.ControllerProxy{
		Type:   proxy.ProxierTypeKey,
		Config: rawConfig,
	}
	defaultFactory, err := factory.NewDefaultFactory()
	c.Assert(err, qt.IsNil)
	proxier, err := controllerProxy.ToProxier(defaultFactory)
	if err != nil {
		t.Fatalf("ToProxier() unexpected error: %v", err)
	}
	rawConfigFromProxier, err := proxier.RawConfig()
	c.Assert(err, qt.IsNil)
	c.Assert(rawConfigFromProxier, qt.DeepEquals, rawConfig)
}

func TestControllerProxy_Invalid_ZeroValues(t *testing.T) {
	c := qt.New(t)
	rawConfig := map[string]interface{}{
		"another-field": "",
	}

	controllerProxy := juju.ControllerProxy{
		Type:   proxy.ProxierTypeKey,
		Config: rawConfig,
	}
	defaultFactory, err := factory.NewDefaultFactory()
	c.Assert(err, qt.IsNil)
	proxier, err := controllerProxy.ToProxier(defaultFactory)
	c.Assert(err, qt.IsNil)
	rawConfigFromProxier, err := proxier.RawConfig()
	c.Assert(err, qt.IsNil)
	c.Assert(rawConfigFromProxier, qt.DeepEquals, map[string]interface{}{
		"api-host":              "",
		"ca-cert":               "",
		"namespace":             "",
		"remote-port":           "",
		"service":               "",
		"service-account-token": "",
	})
}

func TestControllerProxy_Invalid_Type(t *testing.T) {
	c := qt.New(t)
	rawConfig := map[string]interface{}{
		"api-host": "https://localhost",
	}
	controllerProxy := juju.ControllerProxy{
		Type:   "invalid-type",
		Config: rawConfig,
	}
	defaultFactory, err := factory.NewDefaultFactory()
	c.Assert(err, qt.IsNil)
	_, err = controllerProxy.ToProxier(defaultFactory)
	c.Assert(err, qt.Not(qt.IsNil))
	c.Assert(err.Error(), qt.Equals, "proxy register for key invalid-type not found")
}
