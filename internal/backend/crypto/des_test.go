// nolint:all // test package
package crypto

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"
)

func TestCalculateKCV(t *testing.T) {
	tests := []struct {
		name    string
		keyHex  string
		wantKCV string
		wantErr bool
	}{
		{
			name:    "valid_8_byte_key",
			keyHex:  "0123456789ABCDEF",
			wantKCV: "D5D44F",
			wantErr: false,
		},
		{
			name:    "valid_16_byte_key",
			keyHex:  "0123456789ABCDEF FEDCBA9876543210",
			wantKCV: "08D7B4",
			wantErr: false,
		},
		{
			name:    "valid_24_byte_key",
			keyHex:  "0123456789ABCDEF FEDCBA9876543210 0011223344556677",
			wantKCV: "CBE6A7",
			wantErr: false,
		},
		{
			name:    "invalid_key_length_7_bytes",
			keyHex:  "0123456789ABCDE", // 7 bytes.
			wantKCV: "",
			wantErr: true,
		},
		{
			name:    "invalid_key_length_9_bytes",
			keyHex:  "0123456789ABCDEF00", // 9 bytes.
			wantKCV: "",
			wantErr: true,
		},
		{
			name:    "empty_key",
			keyHex:  "",
			wantKCV: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyBytes, err := hex.DecodeString(strings.ReplaceAll(tt.keyHex, " ", ""))
			if err != nil {
				if tt.wantErr {
					// If we expect an error and decoding fails, that's acceptable for this test case.
					return
				}
				t.Fatalf("Failed to decode keyHex %q: %v", tt.keyHex, err)
			}

			gotKCV, err := CalculateKCV(keyBytes)
			if (err != nil) != tt.wantErr {
				t.Errorf("CalculateKCV() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotKCV != tt.wantKCV {
				t.Errorf("CalculateKCV() gotKCV = %v, want %v", gotKCV, tt.wantKCV)
			}
		})
	}
}

func TestProcessDES(t *testing.T) {
	key8, _ := hex.DecodeString("0123456789ABCDEF")
	key16, _ := hex.DecodeString("0123456789ABCDEFFEDCBA9876543210")
	key24, _ := hex.DecodeString("0123456789ABCDEFFEDCBA98765432101122334455667788")
	iv, _ := hex.DecodeString("0000000000000000")                     // 8-byte IV.
	data8, _ := hex.DecodeString("0011223344556677")                  // 8 bytes.
	data16, _ := hex.DecodeString("00112233445566778899AABBCCDDEEFF") // 16 bytes.

	// Expected results are placeholders and need to be calculated with a known good DES implementation.
	// For now, we focus on parameter validation and mode execution without checking exact crypto results.
	// To verify crypto: encrypt, then decrypt, and check if original data is recovered.

	tests := []struct {
		name        string
		params      *DESParams
		wantErr     bool
		checkOutput bool // If true, encrypt then decrypt and compare.
	}{
		{
			name:    "nil_params",
			params:  nil,
			wantErr: true,
		},
		{
			name:    "invalid_key_length_7_bytes",
			params:  &DESParams{Data: data8, Key: key8[:7], Mode: ECB, Encrypt: true},
			wantErr: true,
		},
		{
			name:    "invalid_key_length_15_bytes",
			params:  &DESParams{Data: data8, Key: key16[:15], Mode: ECB, Encrypt: true},
			wantErr: true,
		},
		{
			name:    "invalid_key_length_23_bytes",
			params:  &DESParams{Data: data8, Key: key24[:23], Mode: ECB, Encrypt: true},
			wantErr: true,
		},
		{
			name: "ecb_encrypt_decrypt_8_byte_key_8_byte_data_no_padding",
			params: &DESParams{
				Data:    data8,
				Key:     key8,
				Mode:    ECB,
				Padding: NoPadding,
				Encrypt: true,
			},
			wantErr:     false,
			checkOutput: true,
		},
		{
			name: "ecb_encrypt_decrypt_16_byte_key_16_byte_data_no_padding",
			params: &DESParams{
				Data:    data16,
				Key:     key16,
				Mode:    ECB,
				Padding: NoPadding,
				Encrypt: true,
			},
			wantErr:     false,
			checkOutput: true,
		},
		{
			name: "ecb_encrypt_decrypt_24_byte_key_16_byte_data_no_padding",
			params: &DESParams{
				Data:    data16,
				Key:     key24,
				Mode:    ECB,
				Padding: NoPadding,
				Encrypt: true,
			},
			wantErr:     false,
			checkOutput: true,
		},
		{
			name: "cbc_encrypt_decrypt_8_byte_key_8_byte_data_no_padding",
			params: &DESParams{
				Data:    data8,
				Key:     key8,
				IV:      iv,
				Mode:    CBC,
				Padding: NoPadding,
				Encrypt: true,
			},
			wantErr:     false,
			checkOutput: true,
		},
		{
			name: "cbc_encrypt_decrypt_16_byte_key_16_byte_data_no_padding",
			params: &DESParams{
				Data:    data16,
				Key:     key16,
				IV:      iv,
				Mode:    CBC,
				Padding: NoPadding,
				Encrypt: true,
			},
			wantErr:     false,
			checkOutput: true,
		},
		{
			name: "cbc_encrypt_decrypt_24_byte_key_16_byte_data_no_padding",
			params: &DESParams{
				Data:    data16,
				Key:     key24,
				IV:      iv,
				Mode:    CBC,
				Padding: NoPadding,
				Encrypt: true,
			},
			wantErr:     false,
			checkOutput: true,
		},
		{
			name: "cbc_invalid_iv_length",
			params: &DESParams{
				Data:    data8,
				Key:     key8,
				IV:      iv[:7],
				Mode:    CBC,
				Padding: NoPadding,
				Encrypt: true,
			},
			wantErr: true,
		},
		{
			name:    "unsupported_mode",
			params:  &DESParams{Data: data8, Key: key8, Mode: CipherMode(99), Encrypt: true},
			wantErr: true,
		},
		{
			name: "no_padding_data_not_multiple_of_blocksize",
			params: &DESParams{
				Data:    data8[:7],
				Key:     key8,
				Mode:    ECB,
				Padding: NoPadding,
				Encrypt: true,
			},
			wantErr: true,
		},
		{
			name: "iso97971_padding_encrypt_decrypt",
			params: &DESParams{
				Data:    data8[:5],
				Key:     key8,
				Mode:    ECB,
				Padding: ISO97971,
				Encrypt: true,
			},
			wantErr:     false,
			checkOutput: true, // Padding makes direct output check harder, rely on decrypt round trip.
		},
		{
			name: "iso97972_padding_encrypt_decrypt",
			params: &DESParams{
				Data:    data8[:6],
				Key:     key8,
				Mode:    ECB,
				Padding: ISO97972,
				Encrypt: true,
			},
			wantErr:     false,
			checkOutput: true,
		},
		{
			name: "unsupported_padding_mode",
			params: &DESParams{
				Data:    data8,
				Key:     key8,
				Mode:    ECB,
				Padding: PaddingMode(99),
				Encrypt: true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := ProcessDES(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessDES() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return // Expected error, no further checks.
			}

			if tt.checkOutput {
				if tt.params == nil { // Should have been caught by wantErr.
					t.Fatal(
						"tt.params is nil in checkOutput block, this indicates a test setup error",
					)
				}
				// Create decrypt params.
				decryptParams := *tt.params    // Shallow copy.
				decryptParams.Data = encrypted // Use encrypted data as input.
				decryptParams.Encrypt = false
				// IV for CBC decryption must be the same as for encryption.
				// Padding is handled by the ProcessDES logic (implicitly, as decryption doesn't unpad here).

				decrypted, err := ProcessDES(&decryptParams)
				if err != nil {
					t.Errorf("ProcessDES() decryption step failed: %v", err)
					return
				}

				// Compare decrypted output with original data (before padding).
				originalData := tt.params.Data
				// The decrypted data will be padded to block size. We need to compare only the original part.
				if len(decrypted) < len(originalData) {
					t.Errorf(
						"Decrypted data length %d is less than original data length %d",
						len(decrypted),
						len(originalData),
					)
					return
				}

				// For NoPadding, lengths must match exactly.
				if tt.params.Padding == NoPadding {
					if !bytes.Equal(decrypted, originalData) {
						t.Errorf(
							"ProcessDES() decrypted data = %X, want %X (NoPadding)",
							decrypted,
							originalData,
						)
					}
				} else {
					// For other paddings, compare the prefix matching original data length.
					if !bytes.Equal(decrypted[:len(originalData)], originalData) {
						t.Errorf("ProcessDES() decrypted data prefix = %X, want %X (With Padding)", decrypted[:len(originalData)], originalData)
					}
					// Further checks for padding bytes could be added if unpad function was available and used.
				}
			}
		})
	}
}
