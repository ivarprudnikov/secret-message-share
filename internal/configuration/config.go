package configuration

import (
	"fmt"
	"os"
)

const keyEnvironment = "SERVER_ENV"
const keySalt = "DB_SALT_KEY"
const keyCookieAuth = "COOK_AUTH_KEY"
const keyCookieEnc = "COOK_ENC_KEY"
const envTest = "test"
const testKey = "12345678123456781234567812345678"

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

func (c *ConfigReader) IsProd() bool {
	return c.isProd
}

func (c *ConfigReader) GetSalt() string {
	return c.getKey(keySalt)
}

func (c *ConfigReader) GetCookieAuth() string {
	return c.getKey(keyCookieAuth)
}

func (c *ConfigReader) GetCookieEnc() string {
	return c.getKey(keyCookieEnc)
}

func (c *ConfigReader) getKey(name string) string {
	var k string
	if !c.isProd {
		k = testKey
	} else {
		k = os.Getenv(name)
	}
	checkKeyLength(name, k)
	return k
}

func checkKeyLength(name, val string) {
	requiredLen := 32
	if len(val) != requiredLen {
		panic(fmt.Sprintf("%s must be %d characters in length", name, requiredLen))
	}
}
