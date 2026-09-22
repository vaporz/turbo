/*
 * Copyright © 2017 Xiao Zhang <zzxx513@gmail.com>.
 * Use of this source code is governed by an MIT-style
 * license that can be found in the LICENSE file.
 */
package turbo

import (
	"bytes"
	"sort"
	"strings"
	"testing"

	logger "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestSortKeysFirstOrdersWithoutLosing(t *testing.T) {
	keys := []string{"user_id", "amount", "device_code", "order_id"}
	SortKeysFirst("level", "msg", "user_id")(keys)
	assert.Equal(t, []string{"user_id", "amount", "device_code", "order_id"}, keys)

	// the named fields lead in the order they were named, the rest stay alphabetical
	keys = []string{"user_id", "amount", "level", "device_code", "msg"}
	SortKeysFirst("level", "msg")(keys)
	assert.Equal(t, []string{"level", "msg", "amount", "device_code", "user_id"}, keys)

	// a name that is not there is simply never matched
	keys = []string{"b", "a"}
	SortKeysFirst("level")(keys)
	assert.Equal(t, []string{"a", "b"}, keys)
}

// TestSortKeysFirstKeepsEveryFieldInTheLogLine is T13 itself: the same
// TextFormatter run that loses fields with a hand written sorter keeps all of
// them with SortKeysFirst.
func TestSortKeysFirstKeepsEveryFieldInTheLogLine(t *testing.T) {
	line := func(sortKeys func([]string)) string {
		var out bytes.Buffer
		l := logger.New()
		l.Out = &out
		l.Formatter = &logger.TextFormatter{DisableColors: true, DisableTimestamp: true, SortingFunc: sortKeys}
		l.WithFields(logger.Fields{
			"user_id": "u1", "device_code": "A3", "amount": 100, "order_id": "o9",
		}).Info("paid")
		return out.String()
	}

	// a sorter that rewrites names instead of reordering them
	broken := line(func(keys []string) {
		sort.Strings(keys)
		for i := 2; i < len(keys); i++ {
			keys[i] = ""
		}
	})
	assert.Contains(t, broken, `="<nil>"`)   // the fields became nils
	assert.NotContains(t, broken, "user_id") // and the names are gone

	good := line(SortKeysFirst("level", "msg", "user_id", "device_code", "amount", "order_id"))
	for _, name := range []string{"user_id=u1", "device_code=A3", "amount=100", "order_id=o9"} {
		assert.Contains(t, good, name)
	}
}

func TestCheckSortingFunc(t *testing.T) {
	// nil means "let logrus sort", which is safe
	assert.NoError(t, CheckSortingFunc(nil, "b", "a"))

	assert.NoError(t, CheckSortingFunc(SortKeysFirst("a"), "b", "a", "c"))
	assert.NoError(t, CheckSortingFunc(func(keys []string) { sort.Strings(keys) }, "b", "a", "c"))

	// rewriting names, truncating, and duplicating are all caught
	assert.Error(t, CheckSortingFunc(func(keys []string) { keys[0] = "level" }, "amount", "device_code"))
	assert.Error(t, CheckSortingFunc(func(keys []string) { keys[1] = "" }, "amount", "device_code"))
	assert.Error(t, CheckSortingFunc(func(keys []string) { keys[0] = keys[1] }, "amount", "device_code"))

	err := CheckSortingFunc(func(keys []string) { keys[0] = "" }, "amount", "device_code")
	assert.Contains(t, err.Error(), "not a permutation")
	assert.Contains(t, err.Error(), "SortKeysFirst")
	assert.True(t, strings.Contains(err.Error(), "amount"))
}
