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

func Test_Config_TOTP_Minimal(t *testing.T) {
	config, err := Configure(testConfigPath("totp_min_config.json"))
	assert.Nil(t, err)
	assert.Equal(t, config.TOTP.Max, 0)
	assert.Equal(t, config.TOTP.SetupTTL, 300)
	assert.Equal(t, config.TOTP.SecretLength, 16)
	assert.Equal(t, config.TOTP.Issuer, "gobl.test")
}

func Test_Config_TOTP(t *testing.T) {
	config, err := Configure(testConfigPath("totp_config.json"))
	assert.Nil(t, err)
	assert.Equal(t, config.TOTP.Max, 92)
	assert.Equal(t, config.TOTP.SetupTTL, 112)
	assert.Equal(t, config.TOTP.SecretLength, 17)
	assert.Equal(t, config.TOTP.Issuer, "gobl1.test")
}

func Test_Config_DefaultTicket(t *testing.T) {
	config, err := Configure(testConfigPath("totp_config.json"))
	assert.Nil(t, err)
	assert.Equal(t, config.Ticket.Max, 0)
	assert.Equal(t, config.Ticket.MaxPayloadLength, 0)
}

func Test_Config_Ticket(t *testing.T) {
	config, err := Configure(testConfigPath("ticket_config.json"))
	assert.Nil(t, err)
	assert.Equal(t, config.Ticket.Max, 76)
	assert.Equal(t, config.Ticket.MaxPayloadLength, 877)
}

func testConfigPath(file string) string {
	return path.Join("../tests/data/", file)
}
