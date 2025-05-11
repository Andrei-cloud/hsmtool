package crypto

import (
	"crypto/cipher"
	"crypto/des"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// ECB is Electronic Codebook mode.
// CBC is Cipher Block Chaining mode.
const (
	ECB CipherMode = iota
	CBC
)

// NoPadding means no padding will be applied.
// ISO97971 is ISO 9797-1 padding method 1 (pad with zeros).
// ISO97972 is ISO 9797-1 padding method 2 (append 0x80 followed by zeros).
const (
	NoPadding PaddingMode = iota
	ISO97971
	ISO97972
)

// CipherMode specifies the block cipher mode of operation.
type CipherMode int

// PaddingMode specifies the padding method.
type PaddingMode int

// DESParams holds parameters for DES operation.
type DESParams struct {
	Data    []byte
	Key     []byte
	IV      []byte // iv for CBC mode.
	Mode    CipherMode
	Padding PaddingMode
	Encrypt bool
}

// CalculateKCV calculates Key Check Value (first 3 bytes of encrypting zeros).
// CalculateKCV calculates a KCV (key check value) for a given key and returns it as a hex string.
// Returns error if key validation or calculation fails.
func CalculateKCV(key []byte) (string, error) {
	if len(key) != 8 && len(key) != 16 && len(key) != 24 {
		return "", errors.New("invalid key length: must be 8, 16, or 24 bytes")
	}

	// Create a block of zeros.
	zeros := make([]byte, 8)

	// Encrypt zeros with the key.
	params := &DESParams{
		Data:    zeros,
		Key:     key,
		Mode:    ECB,
		Padding: NoPadding,
		Encrypt: true,
	}

	result, err := ProcessDES(params)
	if err != nil {
		return "", fmt.Errorf("failed to calculate KCV: %v", err)
	}

	// Return first 3 bytes as uppercase hex.

	return strings.ToUpper(hex.EncodeToString(result[:3])), nil
}

// ProcessDES performs DES encryption/decryption according to parameters.
func ProcessDES(params *DESParams) ([]byte, error) {
	if params == nil {
		return nil, errors.New("params cannot be nil")
	}

	// Validate key length.
	keyLen := len(params.Key)
	if keyLen != 8 && keyLen != 16 && keyLen != 24 {
		return nil, errors.New("invalid key length: must be 8, 16, or 24 bytes")
	}

	// Create block cipher.
	var block cipher.Block
	var err error

	switch keyLen {
	case 8:
		block, err = des.NewCipher(params.Key)
	case 16:
		// For double length key, use K1,K2,K1 mode
		tripleKey := make([]byte, 24)
		copy(tripleKey[:16], params.Key)     // Copy K1,K2
		copy(tripleKey[16:], params.Key[:8]) // Copy K1 again
		block, err = des.NewTripleDESCipher(tripleKey)
	case 24:
		block, err = des.NewTripleDESCipher(params.Key)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Apply padding if needed.
	paddedData, err := pad(params.Data, block.BlockSize(), params.Padding)
	if err != nil {
		return nil, fmt.Errorf("padding error: %w", err)
	}

	// Process data according to mode.
	result := make([]byte, len(paddedData))
	switch params.Mode {
	case ECB:
		processECB(block, paddedData, result, params.Encrypt)

	case CBC:
		// Validate iv length.
		if len(params.IV) != block.BlockSize() {
			return nil, fmt.Errorf("invalid iv length: must be %d bytes", block.BlockSize())
		}
		if params.Encrypt {
			encrypter := cipher.NewCBCEncrypter(block, params.IV)
			encrypter.CryptBlocks(result, paddedData)
		} else {
			decrypter := cipher.NewCBCDecrypter(block, params.IV)
			decrypter.CryptBlocks(result, paddedData)
		}
	default:

		return nil, errors.New("unsupported mode")
	}

	return result, nil
}

// pad adds padding according to the specified mode.
func pad(data []byte, blockSize int, mode PaddingMode) ([]byte, error) {
	if mode == NoPadding {
		if len(data)%blockSize != 0 {
			return nil, errors.New(
				"data length must be multiple of block size when using no padding",
			)
		}

		return data, nil
	}

	padLen := blockSize - (len(data) % blockSize)
	padded := make([]byte, len(data)+padLen)
	copy(padded, data)

	switch mode {
	case ISO97971:
		// Pad with zeros.
		return padded, nil
	case ISO97972:
		// Append 0x80 followed by zeros.
		padded[len(data)] = 0x80
		return padded, nil
	default:
		return nil, errors.New("unsupported padding mode")
	}
}

// processECB performs ECB mode encryption/decryption.
func processECB(block cipher.Block, in, out []byte, encrypt bool) {
	blockSize := block.BlockSize()
	for i := 0; i < len(in); i += blockSize {
		if encrypt {
			block.Encrypt(out[i:], in[i:i+blockSize])
		} else {
			block.Decrypt(out[i:], in[i:i+blockSize])
		}
	}
}
