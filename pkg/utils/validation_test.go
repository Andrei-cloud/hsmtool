// nolint:all // test package
package utils

import (
	"strconv"
	"testing"
)

func TestValidateHex(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid_hex_lowercase", "abcdef0123456789", false},
		{"valid_hex_uppercase", "ABCDEF0123456789", false},
		{"valid_hex_mixedcase", "AbCdEf0123456789", false},
		{"invalid_hex_char", "abcdef012345678g", true},
		{"empty_string", "", true}, // Assuming empty is invalid, adjust if not.
		{"odd_length", "abcde", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHex(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHex(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateKeyLength(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedLength int
		wantErr        bool
	}{
		{"valid_single_des", "0123456789ABCDEF", 8, false},
		{"valid_double_des", "0123456789ABCDEF0123456789ABCDEF", 16, false},
		{"valid_triple_des", "0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF", 24, false},
		{"invalid_length_too_short_for_expected", "0123456789ABCDEF", 16, true},
		{"invalid_length_too_long_for_expected", "0123456789ABCDEF0123456789ABCDEF", 8, true},
		{"invalid_hex_char", "0123456789ABCDEG", 8, true},
		{"empty_string", "", 8, true},
		{"valid_empty_string_for_zero_length", "", 0, false},
		{"invalid_non_empty_string_for_zero_length", "00", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKeyLength(tt.input, tt.expectedLength)
			if (err != nil) != tt.wantErr {
				t.Errorf(
					"ValidateKeyLength(%q, %d) error = %v, wantErr %v",
					tt.input,
					tt.expectedLength,
					err,
					tt.wantErr,
				)
			}
		})
	}
}

func TestValidateHexFixedLength(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		byteLength int
		wantErr    bool
	}{
		{"valid_8_bytes", "0123456789ABCDEF", 8, false},
		{"valid_16_bytes", "0123456789ABCDEF0123456789ABCDEF", 16, false},
		{"invalid_length_too_short", "0123456789ABCDE", 8, true},
		{"invalid_length_too_long", "0123456789ABCDEF00", 8, true},
		{"invalid_hex_char", "0123456789ABCDEG", 8, true},
		{"empty_string", "", 8, true},
		{
			"zero_byte_length_valid_empty",
			"",
			0,
			false,
		}, // Assuming empty string is valid for 0 length.
		{"zero_byte_length_invalid_not_empty", "00", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHexFixedLength(tt.input, tt.byteLength)
			if (err != nil) != tt.wantErr {
				t.Errorf(
					"ValidateHexFixedLength(%q, %d) error = %v, wantErr %v",
					tt.input,
					tt.byteLength,
					err,
					tt.wantErr,
				)
			}
		})
	}
}

func TestDecodeHex(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []byte
		wantErr bool
	}{
		{"valid_hex", "0123AF", []byte{0x01, 0x23, 0xAF}, false},
		{"valid_hex_with_spaces", "01 23 AF", []byte{0x01, 0x23, 0xAF}, false},
		{"invalid_hex_char", "0123AG", nil, true},
		{"odd_length_after_space_removal", "01 2 3AF", nil, true},
		{"empty_string", "", []byte{}, false},
		{"spaces_only", "   ", []byte{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeHex(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeHex(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if string(got) != string(tt.want) { // Compare as strings for easier diffing.
				t.Errorf("DecodeHex(%q) = %x, want %x", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateKeyName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid_name", "MyTestKey123", false},
		{"valid_name_with_hyphen_underscore", "My-Test_Key_123", false},
		{"invalid_name_empty", "", true},
		{
			"invalid_name_too_long",
			"aVeryLongKeyNameThatExceedsTheMaximumAllowedLengthOfFiftyCharacters",
			true,
		},
		{"invalid_name_special_chars", "MyKey!@#", true},
		{
			"valid_name_max_length",
			"AbcdefghijAbcdefghijAbcdefghijAbcdefghijAbcdefgh",
			false,
		}, // 50 chars.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKeyName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateKeyName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePort(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid_port_min", "1", false},
		{"valid_port_max", "65535", false},
		{"valid_port_common", "8080", false},
		{"invalid_port_zero", "0", true},
		{"invalid_port_negative", "-100", true},
		{"invalid_port_too_high", "65536", true},
		{"invalid_port_non_numeric", "abc", true},
		{"empty_string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			portInt := -1 // Default to an invalid port for error cases if conversion fails.
			var convErr error
			if tt.input != "" { // Avoid error on empty string for strconv.Atoi.
				portInt, convErr = strconv.Atoi(tt.input)
			}

			err := ValidatePort(portInt)

			// Consider conversion error as part of the validation failure for non-numeric inputs.
			actualWantErr := tt.wantErr
			if convErr != nil &&
				tt.input != "" { // If conversion failed and input was not empty, we expect an error.
				actualWantErr = true
			} else if tt.input == "" { // Empty string input for port is an error.
				actualWantErr = true
			}

			if (err != nil) != actualWantErr {
				t.Errorf(
					"ValidatePort(%q converted to %d) error = %v, wantErr %v (convErr: %v)",
					tt.input,
					portInt,
					err,
					actualWantErr,
					convErr,
				)
			}
		})
	}
}

func TestValidateIPAddress(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid_ipv4", "192.168.1.1", false},
		{"valid_ipv4_loopback", "127.0.0.1", false},
		{"valid_ipv6", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", false},
		{"valid_ipv6_compressed", "::1", false},
		{"invalid_ip_partial", "192.168.1", true},
		{"invalid_ip_out_of_range", "256.100.50.1", true},
		{"invalid_ip_non_numeric", "192.168.1.a", true},
		{"empty_string", "", true},
		{"hostname", "localhost", true}, // ValidateIPAddress should only validate IPs.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIPAddress(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIPAddress(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateNumericInput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid_integer", "12345", false},
		{"valid_zero", "0", false},
		{"invalid_negative", "-100", true}, // Assuming non-negative, adjust if negative is allowed.
		{"invalid_decimal", "123.45", true},
		{"invalid_non_numeric", "abc", true},
		{"empty_string", "", true},
		{"valid_large_number", "12345678901234567890", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNumericInput(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf(
					"ValidateNumericInput(%q) error = %v, wantErr %v",
					tt.input,
					err,
					tt.wantErr,
				)
			}
		})
	}
}
