/*
 * Copyright © 2017 Xiao Zhang <zzxx513@gmail.com>.
 * Use of this source code is governed by an MIT-style
 * license that can be found in the LICENSE file.
 */
package turbo

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	logger "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// reloadServable is a Servable whose server field a test drives directly. The
// concrete servers need a running rpc client, which a rebuild test does not.
type reloadServable struct {
	*Server
}

func (s reloadServable) Service(string) interface{} { return nil }

// TestARebuildReadsTheConfigurationUnderTheLock pins T23: a rebuild reads the
// configuration and the registered components, and the watcher replaces the
// configuration while requests are served. Without the lock the two overlap and
// -race reports it -- and a change landing in the middle of a rebuild could even
// be read half way, giving a routing table built from two configurations. Run
// this test with -race.
func TestARebuildReadsTheConfigurationUnderTheLock(t *testing.T) {
	// The rebuild has to succeed, so the configuration names a component that is
	// registered. Its output is not interesting here: the reads it makes are.
	level := log.Level
	log.SetLevel(logger.PanicLevel)
	defer log.SetLevel(level)

	config := func() *Config {
		return &Config{
			File: "service.yaml",
			mappings: map[string][][4]string{
				urlServiceMaps: {{"GET", "/hello", "TestService", "SayHello"}},
				interceptors:   {{"GET", "/hello", "TestInterceptor", ""}},
			},
		}
	}
	first, second := config(), config()
	s := reloadServable{&Server{
		Config:     first,
		Components: &Components{registeredComponents: map[string]interface{}{"TestInterceptor": &auditStub{}}},
	}}

	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			// what the watcher does when the configuration changes
			s.componentsLock.Lock()
			if s.Config == first {
				s.Config = second
			} else {
				s.Config = first
			}
			s.componentsLock.Unlock()
		}
	}()

	for i := 0; i < 200; i++ {
		// what the reload path does
		_ = reloadComponents(s, first)
	}

	close(stop)
	wg.Wait()
}

// T22: a configuration file is commonly written in place, which means it is
// truncated before the new content arrives -- and the watcher reacts to the
// truncation. The two tests below pin the two halves of the fix: the wait that
// keeps a reload from reading a half written file, and the validation that
// refuses the empty configuration a reader would otherwise get.

// TestWaitForStableFileWaitsForTheWriterToFinish pins the wait itself: a writer
// that truncates the file and then writes it in pieces must not be read half way
// through, so the wait returns only once the content has stopped changing.
func TestWaitForStableFileWaitsForTheWriterToFinish(t *testing.T) {
	path := filepath.Join(t.TempDir(), "service.yaml")
	first := "urlmapping:\n  - GET /hello TestService SayHello\n"
	complete := first + "  - GET /eat MinionsService Eat\n"

	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		// truncate first, the way cp, ">" and an editor saving in place do
		_ = os.WriteFile(path, nil, 0644)
		time.Sleep(40 * time.Millisecond)
		_ = os.WriteFile(path, []byte(first), 0644)
		time.Sleep(80 * time.Millisecond)
		_ = os.WriteFile(path, []byte(complete), 0644)
	}()

	start := time.Now()
	waitForStableFile(path)
	elapsed := time.Since(start)
	<-writerDone

	content, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, complete, string(content),
		"the reload must not read a file that is still being written")
	assert.Less(t, elapsed, fileSettleWait,
		"a writer that keeps making progress must not be given the whole timeout")
}

// TestWaitForStableFileReturnsAtOnceWithoutAFile pins the other end: a file that
// is not there (deleted, or replaced by a rename that has not happened yet) is
// not waited for, because waiting cannot make it appear. The loader reports it.
func TestWaitForStableFileReturnsAtOnceWithoutAFile(t *testing.T) {
	start := time.Now()
	waitForStableFile(filepath.Join(t.TempDir(), "absent.yaml"))
	assert.Less(t, time.Since(start), fileSettleQuiet)
}

// TestWaitForStableFileGivesAQuietFileOneQuietPeriod pins the cost of the wait:
// a file nobody is writing is read after one quiet period, not after the
// timeout, so a reload stays quick.
func TestWaitForStableFileGivesAQuietFileOneQuietPeriod(t *testing.T) {
	path := filepath.Join(t.TempDir(), "service.yaml")
	assert.NoError(t, os.WriteFile(path, []byte("urlmapping:\n  - GET /hello TestService SayHello\n"), 0644))

	start := time.Now()
	waitForStableFile(path)
	elapsed := time.Since(start)

	assert.GreaterOrEqual(t, elapsed, fileSettleQuiet)
	assert.Less(t, elapsed, fileSettleWait)
}

// TestConfigWithNoRouteIsRefused pins the validation. An empty file is valid
// YAML, so it used to load as a configuration with no route at all: every
// request answered 404 while the process still looked healthy.
func TestConfigWithNoRouteIsRefused(t *testing.T) {
	path := filepath.Join(t.TempDir(), "service.yaml")
	assert.NoError(t, os.WriteFile(path, []byte("config:\n  environment: development\n"), 0644))

	c := &Config{
		Viper:    *viper.New(),
		File:     path,
		mappings: make(map[string][][4]string),
	}

	err := c.loadServiceConfigErr()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "urlmapping")
}
