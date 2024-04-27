package crypto

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Uses argon2id to hash and salt the given clear text value
// returns the encoded string to be used for storage
func HashPass(password string) (string, error) {
	// values suggested in the OWASP cheatsheet
	params := &argon2Params{
		memory:      12 * 1024, // 12mb
		iterations:  3,
		parallelism: 1,
		saltLength:  16,
		keyLength:   32,
	}
	return genArgon2Hash(password, params)
}

// Uses argon2id to compare the given encoded string to the given password
func CompareHashToPass(hash, password string) error {
	p, salt, key, err := decodeArgon2Hash(hash)
	if err != nil {
		return err
	}
	// try to recreate the key
	otherKey := genArgon2Key(p, []byte(password), salt)
	// compare but mitigate timing attacks
	if subtle.ConstantTimeCompare(key, otherKey) != 1 {
		return errors.New("invalid key")
	}

	return nil
}

const argonHashVariantName = "argon2id"

// The parameters to be used in the secure hash generation
// Please refer to the OWASP cheatsheet to see recommended values
// https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html#argon2id
type argon2Params struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	saltLength  uint32
	keyLength   uint32
}

// Generate a secure salted hash using Argon2id
// OWASP https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html#argon2id
// Example https://github.com/alexedwards/argon2id
func genArgon2Hash(password string, p *argon2Params) (string, error) {
	salt, err := generateRandomBytes(p.saltLength)
	if err != nil {
		return "", err
	}
	key := genArgon2Key(p, []byte(password), salt)
	return encodeArgon2Hash(p, salt, key), nil
}

// shortcut to generate the key
func genArgon2Key(p *argon2Params, pass, salt []byte) []byte {
	return argon2.IDKey(pass, salt, p.iterations, p.memory, p.parallelism, p.keyLength)
}

// encode argon2 derived key into a self-containing string for storage purposes
func encodeArgon2Hash(params *argon2Params, salt, key []byte) string {
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Key := base64.RawStdEncoding.EncodeToString(key)
	return fmt.Sprintf("$%s$v=%d$m=%d,t=%d,p=%d$%s$%s", argonHashVariantName, argon2.Version, params.memory, params.iterations, params.parallelism, b64Salt, b64Key)
}

// decode argon2 hash string into the parameters ready for comparison
func decodeArgon2Hash(hash string) (params *argon2Params, salt, key []byte, err error) {
	vals := strings.Split(hash, "$")
	if len(vals) != 6 {
		return nil, nil, nil, fmt.Errorf("unexpected hash structure with %d elements but should be %d", len(vals), 6)
	}
	if vals[1] != argonHashVariantName {
		return nil, nil, nil, fmt.Errorf("unexpected hash function variant %s which should be %s", vals[1], argonHashVariantName)
	}
	var version int
	_, err = fmt.Sscanf(vals[2], "v=%d", &version)
	if err != nil {
		return nil, nil, nil, err
	}
	if version != argon2.Version {
		return nil, nil, nil, fmt.Errorf("argon2 version mismatch %d <> %d", version, argon2.Version)
	}
	params = &argon2Params{}
	_, err = fmt.Sscanf(vals[3], "m=%d,t=%d,p=%d", &params.memory, &params.iterations, &params.parallelism)
	if err != nil {
		return nil, nil, nil, err
	}
	salt, err = base64.RawStdEncoding.Strict().DecodeString(vals[4])
	if err != nil {
		return nil, nil, nil, err
	}
	params.saltLength = uint32(len(salt))
	key, err = base64.RawStdEncoding.Strict().DecodeString(vals[5])
	if err != nil {
		return nil, nil, nil, err
	}
	params.keyLength = uint32(len(key))
	return params, salt, key, nil
}
