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

func Test_Config_SingleTenancy_NoTOTP(t *testing.T) {
	_, err := Configure(testConfigPath("minimal_config.json"))
	assert.Equal(t, err.Error(), "code: 104004 - totp.issuer must be set")
}

func Test_Config_TOTP(t *testing.T) {
	config, err := Configure(testConfigPath("totp_config.json"))
	assert.Nil(t, err)
	assert.Equal(t, config.TOTP.Max, 0)
	assert.Equal(t, config.TOTP.SetupTTL, 300)
	assert.Equal(t, config.TOTP.SecretLength, 16)
	assert.Equal(t, config.TOTP.Issuer, "gobl.test")

}

func testConfigPath(file string) string {
	return path.Join("../tests/data/", file)
}
