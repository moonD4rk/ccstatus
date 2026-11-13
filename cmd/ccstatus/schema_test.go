package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/schema"
)

func TestSchemaCommand(t *testing.T) {
	out, _, err := execRoot(t, "", "schema")
	require.NoError(t, err)
	want, err := schema.Generate()
	require.NoError(t, err)
	assert.Equal(t, string(want), out)
	assert.True(t, json.Valid([]byte(out)))

	_, _, err = execRoot(t, "", "schema", "extra")
	require.Error(t, err, "it takes no arguments")
}

func TestInitReferencesTheSchema(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	_, _, err := execRoot(t, "", "init")
	require.NoError(t, err)

	data, err := os.ReadFile(config.Path())
	require.NoError(t, err)
	var doc map[string]any
	require.NoError(t, json.Unmarshal(data, &doc))
	assert.Equal(t, schema.URL, doc["$schema"])
	assert.Regexp(t, `^\{\n  "\$schema": `, string(data), "the reference comes first")

	out, errOut, err := execValidate(t)
	require.NoError(t, err, errOut)
	assert.Contains(t, out, "Settings are valid")

	loaded, err := config.Load()
	require.NoError(t, err)
	loaded.Schema = ""
	assert.Equal(t, config.DefaultSettings(), loaded, "the reference is the only addition to the defaults")
}
