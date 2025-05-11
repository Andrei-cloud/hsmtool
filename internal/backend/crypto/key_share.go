// Package crypto provides cryptographic utilities for the application.
package crypto

// Import dependencies.

// Helper functions that provide core cryptographic functionality.

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
)

// Constants for key handling.
const (
	// Key lengths in bits.
	KeyLength64  = 64  // Single length DES key (8 bytes).
	KeyLength128 = 128 // Double length DES key (16 bytes).
	KeyLength192 = 192 // Triple length DES key (24 bytes).
	KeyLength256 = 256 // 32-byte key.

	// Number of bytes in key check value.
	KCVLength = 3
)

// Common errors.
var (
	ErrInvalidKeyLength      = errors.New("invalid key length")
	ErrInvalidHexString      = errors.New("invalid hex string")
	ErrInvalidKeyFormat      = errors.New("invalid key format")
	ErrInvalidComponentCount = errors.New("invalid component count")
)

// GenerateKey generates a random cryptographic key of the specified length in bits.
// Returns the key as a hex string and its KCV, or an error if the length is invalid.
// If enforceOddParity is true, each byte in the key will have odd parity.
func GenerateKey(lengthBits int, enforceOddParity bool) (string, string, error) {
	// Validate key length.
	if lengthBits != KeyLength64 &&
		lengthBits != KeyLength128 &&
		lengthBits != KeyLength192 &&
		lengthBits != KeyLength256 {
		return "", "", ErrInvalidKeyLength
	}

	// Convert bits to bytes.
	lengthBytes := lengthBits / 8
	keyBytes := make([]byte, lengthBytes)

	// Generate random key material.
	if _, err := rand.Read(keyBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate random key: %w", err)
	}

	// Adjust parity if requested.
	if enforceOddParity {
		adjustParity(keyBytes)
	}

	// Calculate KCV.
	kcv, err := CalculateKCV(keyBytes)
	if err != nil {
		return "", "", fmt.Errorf("failed to calculate KCV: %w", err)
	}

	// Convert key to hex string.
	keyHex := hex.EncodeToString(keyBytes)

	// Clean up key material from memory.
	defer cleanBytes(keyBytes)

	return keyHex, kcv, nil
}

// SplitKey splits a key into the specified number of XOR components.
// The key must be provided as a hex string.
// Returns the components as hex strings and the KCV of the original key.
func SplitKey(keyHex string, numComponents int) ([]string, string, error) {
	// Validate number of components.
	if numComponents < 2 {
		return nil, "", ErrInvalidComponentCount
	}

	// Validate hex string format.
	if err := validateHexString(keyHex, 0); err != nil {
		return nil, "", err
	}

	// Decode key hex string.
	keyBytes, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, "", ErrInvalidHexString
	}
	defer cleanBytes(keyBytes)

	// Create components.
	componentLists := make([][]byte, numComponents)
	for i := 0; i < numComponents; i++ {
		componentLists[i] = make([]byte, len(keyBytes))
	}

	// Generate random components.
	for i := 0; i < numComponents-1; i++ {
		if _, err := rand.Read(componentLists[i]); err != nil {
			cleanComponentLists(componentLists)
			return nil, "", fmt.Errorf("failed to generate component: %w", err)
		}
	}

	// Calculate final component using XOR.
	copy(componentLists[numComponents-1], keyBytes)
	for i := 0; i < numComponents-1; i++ {
		xorBytes(componentLists[numComponents-1], componentLists[i])
	}

	// Calculate KCV of original key.
	kcv, err := CalculateKCV(keyBytes)
	if err != nil {
		cleanComponentLists(componentLists)
		return nil, "", err
	}
	// Convert components to hex.
	components := make([]string, numComponents)
	for i := 0; i < numComponents; i++ {
		components[i] = hex.EncodeToString(componentLists[i])
	}

	cleanComponentLists(componentLists)

	return components, kcv, nil
}

// CombineComponents combines multiple key components to reconstruct the original key.
// Components must be provided as hex strings.
// Returns the reconstructed key as a hex string.
func CombineComponents(components []string) (string, error) {
	// Validate input.
	if len(components) < 2 {
		return "", ErrInvalidComponentCount
	}

	// Validate format of all components.
	for _, comp := range components {
		if err := validateHexString(comp, 0); err != nil {
			return "", err
		}
	}

	// Decode first component to get length.
	firstComponent, err := hex.DecodeString(components[0])
	if err != nil {
		return "", ErrInvalidHexString
	}
	defer cleanBytes(firstComponent)

	keyLength := len(firstComponent)
	resultBytes := make([]byte, keyLength)
	copy(resultBytes, firstComponent)

	// Combine remaining components using XOR.
	for i := 1; i < len(components); i++ {
		componentBytes, err := hex.DecodeString(components[i])
		if err != nil {
			cleanBytes(resultBytes)
			return "", ErrInvalidHexString
		}

		if len(componentBytes) != keyLength {
			cleanBytes(resultBytes)
			cleanBytes(componentBytes)
			return "", ErrInvalidKeyLength
		}

		xorBytes(resultBytes, componentBytes)
		cleanBytes(componentBytes)
	}
	// Convert result to hex string.
	resultHex := hex.EncodeToString(resultBytes)
	cleanBytes(resultBytes)

	return resultHex, nil
}

// ValidateComponentConsistency checks if the components will XOR back to the original key.
// Returns true if the components are consistent, false otherwise.
func ValidateComponentConsistency(original string, components []string) bool {
	// Decode original key.
	origBytes, err := hex.DecodeString(original)
	if err != nil {
		return false
	}
	defer cleanBytes(origBytes)

	// Combine components.
	recombined, err := CombineComponents(components)
	if err != nil {
		return false
	}

	recombinedBytes, err := hex.DecodeString(recombined)
	if err != nil {
		return false
	}
	defer cleanBytes(recombinedBytes)

	// Compare - must match exactly.
	if len(origBytes) != len(recombinedBytes) {
		return false
	}
	for i := 0; i < len(origBytes); i++ {
		if origBytes[i] != recombinedBytes[i] {
			return false
		}
	}

	return true
}

// CalculateKCV has been moved to des.go.

// Helper functions.

// validateHexString checks if a string is a valid hex string
// that represents a byte array of a specific length (or any length if lengthBytes is 0).
func validateHexString(hexStr string, lengthBytes int) error {
	if len(hexStr)%2 != 0 {
		return ErrInvalidHexString
	}

	if lengthBytes > 0 && len(hexStr)/2 != lengthBytes {
		return ErrInvalidKeyLength
	}

	_, err := hex.DecodeString(hexStr)
	if err != nil {
		return ErrInvalidHexString
	}

	return nil
}

// xorBytes performs in-place XOR of two byte slices: dst ^= src.
func xorBytes(dst, src []byte) {
	for i := 0; i < len(dst); i++ {
		dst[i] ^= src[i]
	}
}

// cleanBytes overwrites a byte slice with zeros.
func cleanBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// cleanComponentLists cleans up component byte slices.
func cleanComponentLists(components [][]byte) {
	for i := range components {
		cleanBytes(components[i])
	}
}

// adjustParity sets odd parity for each byte in a DES key.
// DES keys use odd parity - each byte should have an odd number of 1 bits.
// This function examines bits 1-7 of each byte and sets bit 0 (LSB) accordingly.
func adjustParity(key []byte) {
	for i := 0; i < len(key); i++ {
		// Count 1 bits in bits 1-7.
		b := key[i] >> 1
		count := 0
		for b > 0 {
			count += int(b & 1)
			b >>= 1
		}
		// Set LSB to make total number of 1s odd.
		if count%2 == 0 {
			key[i] |= 1 // Set LSB to 1.
		} else {
			key[i] &= 0xFE // Clear LSB.
		}
	}
}

// ValidateKeyParity checks if all bytes in a DES key have odd parity.
func ValidateKeyParity(key []byte) bool {
	for i := 0; i < len(key); i++ {
		// Count all bits (including LSB).
		b := key[i]
		count := 0
		for b > 0 {
			count += int(b & 1)
			b >>= 1
		}
		if count%2 == 0 {
			return false
		}
	}

	return true
}
