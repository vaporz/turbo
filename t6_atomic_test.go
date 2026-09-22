/*
 * Copyright © 2017 Xiao Zhang <zzxx513@gmail.com>.
 * Use of this source code is governed by an MIT-style
 * license that can be found in the LICENSE file.
 */
package turbo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestWriteFileWithTemplateKeepsTheOldFileWhenTheTemplateFails pins T6. The
// generator used to truncate its target and then write, so a failure halfway
// through left a half written file where a working one used to be -- and the
// next build failed somewhere else entirely.
func TestWriteFileWithTemplateKeepsTheOldFileWhenTheTemplateFails(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "generated.go")
	assert.NoError(t, os.WriteFile(target, []byte("// previous output\n"), 0644))

	// a template that parses but fails while executing
	type broken struct{ A *struct{ B string } }
	assert.Panics(t, func() { writeFileWithTemplate(target, broken{}, "{{.A.B}}\n") })

	kept, err := os.ReadFile(target)
	assert.NoError(t, err)
	assert.Equal(t, "// previous output\n", string(kept))

	// and the temporary file it wrote to is gone
	entries, err := os.ReadDir(dir)
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "generated.go", entries[0].Name())
}

func TestWriteFileWithTemplateReplacesTheFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "generated.go")

	writeFileWithTemplate(target, map[string]string{"Name": "world"}, "hello {{.Name}}\n")

	content, err := os.ReadFile(target)
	assert.NoError(t, err)
	assert.Equal(t, "hello world\n", string(content))

	entries, err := os.ReadDir(dir)
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
}
