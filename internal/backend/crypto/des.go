package crypto

import (
	"crypto/cipher"
	"crypto/des"
	"encoding/hex"
	"fmt"
)

// CipherMode specifies the block cipher mode of operation.
type CipherMode int

const (
	// ECB is Electronic Codebook mode.
	ECB CipherMode = iota
	// CBC is Cipher Block Chaining mode.
	CBC
	// CFB is Cipher Feedback mode.
	CFB
)

// PaddingMode specifies the padding method.
type PaddingMode int

const (
	// NoPadding means no padding will be applied.
	NoPadding PaddingMode = iota
	// ISO97971 is ISO 9797-1 padding method 1 (pad with zeros).
	ISO97971
	// ISO97972 is ISO 9797-1 padding method 2 (append 0x80 followed by zeros).
	ISO97972
)

// DESParams holds parameters for DES operation.
type DESParams struct {
	Data    []byte
	Key     []byte
	Mode    CipherMode
	Padding PaddingMode
	Encrypt bool
}

// CalculateKCV calculates Key Check Value (first 3 bytes of encrypting zeros).
func CalculateKCV(key []byte) (string, error) {
	if len(key) != 8 && len(key) != 16 && len(key) != 24 {
		return "", fmt.Errorf("invalid key length: must be 8, 16, or 24 bytes")
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

	// Return first 3 bytes as hex.
	return hex.EncodeToString(result[:3]), nil
}

// ProcessDES performs DES encryption/decryption according to parameters.
func ProcessDES(params *DESParams) ([]byte, error) {
	if params == nil {
		return nil, fmt.Errorf("params cannot be nil")
	}

	// Validate key length.
	if len(params.Key) != 8 && len(params.Key) != 16 && len(params.Key) != 24 {
		return nil, fmt.Errorf("invalid key length: must be 8, 16, or 24 bytes")
	}

	// Create block cipher.
	var block cipher.Block
	var err error

	if len(params.Key) == 8 {
		block, err = des.NewCipher(params.Key)
	} else {
		block, err = des.NewTripleDESCipher(params.Key)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %v", err)
	}

	// Apply padding if needed.
	paddedData, err := pad(params.Data, block.BlockSize(), params.Padding)
	if err != nil {
		return nil, fmt.Errorf("padding error: %v", err)
	}

	// Process data according to mode.
	result := make([]byte, len(paddedData))
	switch params.Mode {
	case ECB:
		processECB(block, paddedData, result, params.Encrypt)
	case CBC:
		// TODO: Implement CBC mode.
		return nil, fmt.Errorf("CBC mode not implemented")
	case CFB:
		// TODO: Implement CFB mode.
		return nil, fmt.Errorf("CFB mode not implemented")
	default:
		return nil, fmt.Errorf("unsupported mode")
	}

	return result, nil
}

// pad adds padding according to the specified mode.
func pad(data []byte, blockSize int, mode PaddingMode) ([]byte, error) {
	if mode == NoPadding {
		if len(data)%blockSize != 0 {
			return nil, fmt.Errorf(
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
		return nil, fmt.Errorf("unsupported padding mode")
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
