package configuration

import (
	"fmt"
	"os"
)

const keyEnvironment = "SERVER_ENV"
const keySalt = "DB_SALT_KEY"
const keyCookieAuth = "COOK_AUTH_KEY"
const keyCookieEnc = "COOK_ENC_KEY"
const tableUsers = "AZTABLE_USERS"
const tableMessages = "AZTABLE_MESSAGES"
const envTest = "test"
const testKey = "12345678123456781234567812345678"
const requiredKeyLen = 32

type ConfigReader struct {
	isProd bool
}

// Returns the config reader which will read values from the environment
// By default acts if in prodcution unless the SERVER_ENV var is set to something else than "production"
func NewConfigReader() *ConfigReader {
	isProd := true
	if val, ok := os.LookupEnv(keyEnvironment); ok && val == envTest {
		isProd = false
	}
	return &ConfigReader{isProd: isProd}
}

func (c *ConfigReader) IsValid() (bool, []string) {
	invalidVars := []string{}
	for _, k := range []string{keySalt, keyCookieAuth, keyCookieEnc} {
		if len(c.getKey(k, false)) != requiredKeyLen {
			invalidVars = append(invalidVars, k)
		}
	}
	return len(invalidVars) == 0, invalidVars
}

func (c *ConfigReader) IsProd() bool {
	return c.isProd
}

func (c *ConfigReader) GetSalt() string {
	return c.getKey(keySalt, true)
}

func (c *ConfigReader) GetCookieAuth() string {
	return c.getKey(keyCookieAuth, true)
}

func (c *ConfigReader) GetCookieEnc() string {
	return c.getKey(keyCookieEnc, true)
}

func (c *ConfigReader) GetUsersTableName() string {
	return c.getKey(tableUsers, true)
}

func (c *ConfigReader) GetMessagesTableName() string {
	return c.getKey(tableMessages, true)
}

// Production environment expects the value to be set in the
// environmental variable. If not set the application will fail to start.
func (c *ConfigReader) getKey(name string, assert bool) string {
	var k string
	if !c.isProd {
		k = testKey
	} else {
		k = os.Getenv(name)
	}
	if assert {
		assertKeyLength(name, k)
	}
	return k
}

func assertKeyLength(name, val string) {
	if len(val) != requiredKeyLen {
		panic(fmt.Sprintf("%s must be %d characters in length", name, requiredKeyLen))
	}
}
