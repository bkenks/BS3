package cryptoutil

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/bkenks/bs3/internal/constants"
	"golang.org/x/crypto/argon2"
)

// --- Additional Information ---
// Helpful Reminders of Functionality

// ~~~ Mental Model ~~~
// AES(key) → engine
// GCM(engine) → secure transmission wrapper
// nonce → unique session identifier
// Seal() → encrypt + attach tamper-proof seal
// Open() → verify seal + decrypt

// --- END "Additional Information" ---

type Envelope struct {
	EncryptedDEK  []byte
	EncryptedData []byte
}

// =====================================================
// Salting
// =====================================================
// salting functions

func GenerateSalt() ([]byte, error) {
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, err
	}
	return salt, nil
}

func LoadSalt() ([]byte, error) {
	salt, err := os.ReadFile(constants.SaltPath)
	if err != nil {
		return nil, err
	}

	return salt, nil
}

func CreateSalt() ([]byte, error) {
	salt, err := GenerateSalt()
	if err != nil {
		return nil, err
	}

	// Write to file with secure permissions
	if err := os.WriteFile(constants.SaltPath, salt, 0600); err != nil {
		return nil, fmt.Errorf("failed to write salt file: %w", err)
	}

	return salt, nil
}

// =====================================================
// END "Salting"
// =====================================================
////////////////////////////////////////////////////////////////////////////////

// =====================================================
// Hashing
// =====================================================
// functions used to hash values

const (
	ArgonIterations = uint32(3)
	ArgonMemory     = uint32(64 * 1024)
	ArgonThreads    = uint8(2)
	ArgonKeyLength  = uint32(32)
)

// Hashing Function for tokens (API tokens are only used currently)
// Type: HMAC
// - used for high entropy secrets to hash with masterKey as the password to verify
func GenerateToken(masterKey []byte, length int) ([]byte, []byte, error) {
	// Generate new token
	newRawToken := make([]byte, length)
	_, err := rand.Read(newRawToken)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate token bytes: %v", err)
	}

	// Hash Token to return
	newHashToken, err := HashToken(
		masterKey,
		[]byte(newRawToken),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hash token: %v", err)
	}
	return newHashToken, newRawToken, nil
}

func HashToken(masterKey, token []byte) ([]byte, error) {
	// No salt needed as token is randomly generated bits
	// so rainbow table attack can not occur

	mac := hmac.New(sha256.New, masterKey)
	mac.Write(token)
	return mac.Sum(nil), nil
}

// Hashing Function for converting master passphrase (human-friendly) to
// hash which is used to encrypt all other secrets and hmac for tokens
// Type: Argon2
// - used for low entryopy secrets to slow down brute force attacks
// - hashed with salt to prevent rainbow table attacks
func DeriveMasterKey(passphrase string, salt []byte) ([]byte, error) {
	key := argon2.IDKey(
		[]byte(passphrase),
		salt,
		ArgonIterations,
		ArgonMemory,
		ArgonThreads,
		ArgonKeyLength,
	)

	return key, nil
}

// HashMasterKey returns a SHA-256 hash of the master key
func HashMasterKey(masterKey []byte) []byte {
	hash := sha256.Sum256(masterKey)
	return hash[:]
}

// Hashing function for basic auth passwords (human-readable)
// Type: Argon2
// - used for low entropy secrets
// - differs from DeriveMasterKey only by encoding the output to a string for storage whereas
// DeriveMasterKey does not store anything to the disk, just generate the key to use
func HashPassword(plaintext string, salt []byte) (string, error) {
	hash := argon2.IDKey(
		[]byte(plaintext), // password
		salt,              // salt
		ArgonIterations,   // iterations
		ArgonMemory,       // memory in KB (64MB)
		ArgonThreads,      // threads
		ArgonKeyLength,    // key length (32 bytes = 256 bits)
	)

	// Encode as base64 and store with parameters: time$memory$threads$salt$hash
	encoded := fmt.Sprintf("%d$%d$%d$%s$%s",
		ArgonIterations,
		ArgonMemory,
		ArgonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)

	return encoded, nil
}

// =====================================================
// END "Hashing"
// =====================================================

// =====================================================
// Verification
// =====================================================
// functions for verifying hashes

func VerifyToken(masterKey, token, storedHash []byte) (bool, error) {
	computed, err := HashToken(masterKey, token)
	if err != nil {
		return false, fmt.Errorf("could not verify token: %v", err)
	}
	return subtle.ConstantTimeCompare(computed, storedHash) == 1, nil
}

// VerifyMasterKey compares an input master key with the stored hash
func VerifyMasterKey(inputKey, storedHash []byte) (bool, error) {
	inputHash := HashMasterKey(inputKey)
	return subtle.ConstantTimeCompare(inputHash, storedHash) == 1, nil
}

// VerifyPassword checks a password against a stored hash
func VerifyPassword(password, encodedHash string) (bool, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 5 {
		return false, fmt.Errorf("hash does not have correct amount of parts")
	}

	// Parse parameters
	var time, memory uint32
	var threads uint8
	fmt.Sscanf(parts[0], "%d", &time)
	fmt.Sscanf(parts[1], "%d", &memory)
	fmt.Sscanf(parts[2], "%d", &threads)

	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false, fmt.Errorf("failed to decode password salt")
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("failed to decode password hash")
	}

	// Compute key with same parameters
	computed := argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(len(hash)))

	return subtle.ConstantTimeCompare(hash, computed) == 1, nil
}

// =====================================================
// END "Verification"
// =====================================================

func GenerateDEK() ([]byte, error) {
	dek := make([]byte, 32)
	_, err := rand.Read(dek)
	if err != nil {
		return nil, fmt.Errorf("failed to generate DEK: %w", err)
	}
	return dek, nil
}

// End "Hashing"
////////////////////////////////////////////////////////////////////////////////

// =====================================================
// Encryption
// =====================================================
// functions for encrypting and decrypting data

func Encrypt(key []byte, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create new cipher on encrypt: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM on encrypt: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	_, err = rand.Read(nonce)
	if err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	return append(nonce, ciphertext...), nil
}

// Key is the encryption/decryption password, data is the secret we want to decrypt
func Decrypt(key []byte, data []byte) ([]byte, error) {

	// Create new block (16 byte grid)
	// Create round keys
	// Prepare state machine
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create new cipher on decrypt: %w", err)
	}

	//
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM on decrypt: %w", err)
	}

	// Predetermined by GCM (typically always 12 bytes for standard GCM)
	nonceSize := gcm.NonceSize()

	// Data stores nonce + ciphertext + auth tag appended together,
	// so if data is smaller than the nonce, clearly something is wrong
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	// Says "give me a slice from index 0 up to (but not including) nonceSize"
	// i.e. Extract nonce from data (data = nonce + ciphertext + auth tag)
	nonce := data[:nonceSize]

	// Says "give me everything from the index nonceSize to the end"
	ciphertext := data[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)

	return plaintext, nil
}

// =====================================================
// END "Encryption"
// =====================================================

// =====================================================
// Secret Handlers
// =====================================================
// functions that handle the end-to-end process of
// protecting/unprotecting secrets

func ProtectSecret(masterKey []byte, plaintext []byte) (*Envelope, error) {
	dek, err := GenerateDEK()
	if err != nil {
		return nil, err
	}

	encryptedData, err := Encrypt(dek, plaintext)
	if err != nil {
		return nil, err
	}

	encryptedDEK, err := Encrypt(masterKey, dek)
	if err != nil {
		return nil, err
	}

	envelope := &Envelope{
		EncryptedData: encryptedData,
		EncryptedDEK:  encryptedDEK,
	}

	return envelope, nil
}

func UnprotectSecret(masterKey []byte, env *Envelope) ([]byte, error) {

	dek, err := Decrypt(masterKey, env.EncryptedDEK)
	if err != nil {
		return nil, err
	}

	plaintext, err := Decrypt(dek, env.EncryptedData)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// =====================================================
// END "Secret Handlers"
// =====================================================
