package configuration_test

import (
	"testing"

	"github.com/ivarprudnikov/secretshare/internal/configuration"
)

func TestIfSetsProdEnv(t *testing.T) {

	defaultConfig := configuration.NewConfigReader()
	if !defaultConfig.IsProd() {
		t.Fatal("Should be set to prod by default")
	}

	t.Setenv("SERVER_ENV", "anything")
	explicitConfig := configuration.NewConfigReader()
	if !explicitConfig.IsProd() {
		t.Fatal("Should be set to prod because env var does not match anything else")
	}

	t.Setenv("SERVER_ENV", "test")
	testConfig := configuration.NewConfigReader()
	if testConfig.IsProd() {
		t.Fatal("Should be set to non prod environment")
	}
}

func TestTestKeysAreSet(t *testing.T) {
	t.Setenv("SERVER_ENV", "test")
	testConfig := configuration.NewConfigReader()
	if testConfig.IsProd() {
		t.Fatal("Should be set to non prod environment")
	}
	if testConfig.GetSalt() != "12345678123456781234567812345678" {
		t.Fatal("Unexpected test key value")
	}
	if testConfig.GetCookieAuth() != "12345678123456781234567812345678" {
		t.Fatal("Unexpected test key value")
	}
	if testConfig.GetCookieEnc() != "12345678123456781234567812345678" {
		t.Fatal("Unexpected test key value")
	}
}

func TestIncorrectSaltKeyPanic(t *testing.T) {
	defaultConfig := configuration.NewConfigReader()
	defer func() {
		rec := recover()
		if rec == nil {
			t.Fatal("did not panic")
		}
		err := rec.(string)
		if err != "DB_SALT_KEY must be 32 characters in length" {
			t.Fatalf("Wrong panic message: %s", err)
		}
	}()
	defaultConfig.GetSalt()
}

func TestIncorrectAuthKeyPanic(t *testing.T) {
	defaultConfig := configuration.NewConfigReader()
	defer func() {
		rec := recover()
		if rec == nil {
			t.Fatal("did not panic")
		}
		err := rec.(string)
		if err != "COOK_AUTH_KEY must be 32 characters in length" {
			t.Fatalf("Wrong panic message: %s", err)
		}
	}()
	defaultConfig.GetCookieAuth()
}

func TestIncorrectEncKeyPanic(t *testing.T) {
	defaultConfig := configuration.NewConfigReader()
	defer func() {
		rec := recover()
		if rec == nil {
			t.Fatal("did not panic")
		}
		err := rec.(string)
		if err != "COOK_ENC_KEY must be 32 characters in length" {
			t.Fatalf("Wrong panic message: %s", err)
		}
	}()
	defaultConfig.GetCookieEnc()
}
