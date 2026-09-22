/*
 * Copyright © 2017 Xiao Zhang <zzxx513@gmail.com>.
 * Use of this source code is governed by an MIT-style
 * license that can be found in the LICENSE file.
 */
package turbo

import (
	"fmt"
	"sort"
	"strings"
)

// Log field order is presentation, and logrus makes it easy to get wrong.
//
// Only TextFormatter has a SortingFunc: JSONFormatter marshals a map, so
// encoding/json decides the order there and no field can go missing. But
// TextFormatter writes exactly the names a SortingFunc leaves in the slice it is
// handed, so a function that *rewrites* names instead of reordering them deletes
// the fields it wrote over -- and what disappears is whatever somebody added to
// debug a problem:
//
//	// what the function receives
//	amount device_code order_id user_id
//	// keys[0] = "level"; keys[1] = "msg"   -- a rewrite
//	level msg order_id user_id amount         // device_code is gone
//	// keys[i] = "" for i >= 2              -- a truncation
//	amount device_code ="<nil>" ="<nil>"      // fields silently became nil
//
// SortKeysFirst gives the ordering a service actually wants -- name the
// interesting fields, keep the rest alphabetical -- and it can only reorder: the
// result is always a permutation of the input, so nothing can be lost or renamed.
func SortKeysFirst(first ...string) func([]string) {
	rank := make(map[string]int, len(first))
	for i, name := range first {
		rank[name] = i
	}
	return func(keys []string) {
		sort.SliceStable(keys, func(i, j int) bool {
			rankI, iIsFirst := rank[keys[i]]
			rankJ, jIsFirst := rank[keys[j]]
			switch {
			case iIsFirst && jIsFirst:
				return rankI < rankJ
			case iIsFirst:
				return true
			case jIsFirst:
				return false
			}
			return keys[i] < keys[j]
		})
	}
}

// CheckSortingFunc reports whether a sorting function only reorders the names it
// is given. It belongs in a service's own tests:
//
//	if err := turbo.CheckSortingFunc(mySorter, "level", "msg", "user_id", "amount"); err != nil {
//		t.Fatal(err)
//	}
//
// It is a test time check rather than a wrapper, because a wrapper cannot repair
// the damage: logrus reads the slice it passed in, not one the function happened
// to build, so a name the function drops is gone before anything could add it
// back. A lost log field is also exactly the kind of mistake that surfaces only
// when somebody needs the field, which is the worst possible moment to find out.
func CheckSortingFunc(sortKeys func([]string), keys ...string) error {
	if sortKeys == nil {
		return nil // logrus sorts the names alphabetically on its own
	}
	before := append([]string(nil), keys...)
	sort.Strings(before)

	after := append([]string(nil), keys...)
	sortKeys(after)
	sort.Strings(after)

	if strings.Join(before, "\x00") == strings.Join(after, "\x00") {
		return nil
	}
	return fmt.Errorf("turbo: this SortingFunc is not a permutation of the names it was given: "+
		"it had %v and left %v. logrus writes exactly the names it gets back, so a name that "+
		"disappears takes its field with it; use turbo.SortKeysFirst to order fields instead",
		keys, after)
}
