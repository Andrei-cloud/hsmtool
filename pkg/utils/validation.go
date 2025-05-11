package utils

import (
	"encoding/hex"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
)

var (
	// alphanumericRegex validates alphanumeric strings.
	alphanumericRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

	// hexRegex validates hex strings.
	hexRegex = regexp.MustCompile(`^[0-9A-Fa-f]+$`)
)

// ValidateHex checks if a string is valid hexadecimal.
func ValidateHex(input string) error {
	// Remove spaces and validate.
	clean := strings.ReplaceAll(input, " ", "")

	if !hexRegex.MatchString(clean) {
		return fmt.Errorf("invalid hex string")
	}

	if len(clean)%2 != 0 {
		return fmt.Errorf("hex string length must be even")
	}

	return nil
}

// ValidateKeyLength checks if a hex string has valid key length.
func ValidateKeyLength(hexKey string, expectedLength int) error {
	if err := ValidateHex(hexKey); err != nil {
		return err
	}

	clean := strings.ReplaceAll(hexKey, " ", "")
	actualLen := len(clean) / 2 // Convert hex string length to byte length.

	if actualLen != expectedLength {
		return fmt.Errorf(
			"invalid key length: got %d bytes, want %d bytes",
			actualLen,
			expectedLength,
		)
	}

	return nil
}

// ValidateHexFixedLength checks if a hex string has a specific byte length.
func ValidateHexFixedLength(input string, byteLength int) error {
	if err := ValidateHex(input); err != nil {
		return err
	}

	clean := strings.ReplaceAll(input, " ", "")
	if len(clean) != byteLength*2 {
		return fmt.Errorf("invalid length: got %d bytes, want %d bytes", len(clean)/2, byteLength)
	}

	return nil
}

// DecodeHex decodes a hex string, handling spaces.
func DecodeHex(input string) ([]byte, error) {
	// Remove spaces for decoding.
	clean := strings.ReplaceAll(input, " ", "")

	return hex.DecodeString(clean)
}

// ValidateKeyName checks if a key name is valid.
func ValidateKeyName(name string) error {
	if name == "" {
		return fmt.Errorf("key name cannot be empty")
	}

	if !alphanumericRegex.MatchString(name) {
		return fmt.Errorf("key name must be alphanumeric")
	}

	return nil
}

// ValidatePort checks if a port number is valid.
func ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid port number")
	}

	return nil
}

// ValidateIPAddress checks if an IP address is valid.
func ValidateIPAddress(ip string) error {
	if ip == "" {
		return fmt.Errorf("IP address cannot be empty")
	}

	if net.ParseIP(ip) == nil {
		return fmt.Errorf("invalid IP address")
	}

	return nil
}

// ValidateNumericInput validates a string as a numeric value.
func ValidateNumericInput(input string) error {
	if input == "" {
		return fmt.Errorf("numeric input cannot be empty")
	}

	_, err := strconv.Atoi(input)
	if err != nil {
		return fmt.Errorf("invalid numeric input")
	}

	return nil
}
