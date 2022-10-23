package config

import (
	"path"
	"testing"

	"src.goblgobl.com/tests/assert"
)

func Test_Config_InvalidPath(t *testing.T) {
	_, err := Configure("invalid.json")
	assert.Equal(t, err.Error(), "code: 103001 - open invalid.json: no such file or directory")
}

func Test_Config_InvalidJson(t *testing.T) {
	_, err := Configure(testConfigPath("invalid_config.json"))
	assert.Equal(t, err.Error(), "code: 103002 - expected colon after object key")
}

func Test_Config_Defaults(t *testing.T) {
	config, err := Configure(testConfigPath("minimal_config.json"))
	assert.Nil(t, err)

	assert.Equal(t, config.TOTP.SecretLength, 16)
}

func testConfigPath(file string) string {
	return path.Join("../tests/data/", file)
}