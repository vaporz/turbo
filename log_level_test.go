/*
 * Copyright © 2017 Xiao Zhang <zzxx513@gmail.com>.
 * Use of this source code is governed by an MIT-style
 * license that can be found in the LICENSE file.
 */
package turbo

import (
	"testing"

	logger "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// configWithRoutes builds a configuration turbo accepts, so that validate() can be
// asked about one setting at a time. See Config.validate.
func configWithRoutes() *Config {
	return &Config{
		configs:  map[string]string{},
		mappings: map[string][][4]string{urlServiceMaps: {{"GET", "/hello", "TestService", "SayHello"}}},
	}
}

// TestLogLevelConfig pins T27's configuration side: a level turbo does not know is
// refused when the configuration loads, rather than quietly becoming a default.
func TestLogLevelConfig(t *testing.T) {
	c := configWithRoutes()
	assert.NoError(t, c.validate(), "no level means the environment decides")
	assert.Equal(t, "", c.LogLevel())

	for _, level := range []string{"panic", "fatal", "error", "warn", "warning", "info", "debug", "trace", "  Info "} {
		c.configs[logLevel] = level
		assert.NoError(t, c.validate(), level)
		assert.NotEmpty(t, c.LogLevel(), level)
	}

	c.configs[logLevel] = "verbose"
	err := c.validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), logLevel)
	assert.Contains(t, err.Error(), "trace", "the message has to say what is accepted")
}

// TestLogLevelOverridesTheEnvironmentDefault pins the other half of T27. The level
// the environment picks is a default, and a service that wants a different one
// should not have to describe itself as "production" -- which also decides the
// format and where the log is written.
func TestLogLevelOverridesTheEnvironmentDefault(t *testing.T) {
	level, formatter, out := log.Level, log.Formatter, log.Out
	defer func() {
		log.SetLevel(level)
		log.SetFormatter(formatter)
		log.Out = out
	}()

	c := &Config{configs: map[string]string{environment: "development"}}
	initLogger(c)
	assert.Equal(t, logger.DebugLevel, log.Level, "development used to mean debug")

	c.configs[logLevel] = "warn"
	initLogger(c)
	assert.Equal(t, logger.WarnLevel, log.Level, "an explicit level wins over the environment")

	c.configs[environment] = "production"
	// production picks the file, the JSON format and info; the level still wins.
	// Its output is left alone here, which is the point of the setting.
	c.configs[turboLogPath] = t.TempDir()
	initLogger(c)
	assert.Equal(t, logger.WarnLevel, log.Level)
	assert.IsType(t, &logger.JSONFormatter{}, log.Formatter, "production still decides the format")
}
