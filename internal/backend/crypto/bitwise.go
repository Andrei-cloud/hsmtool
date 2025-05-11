package crypto

import (
	"encoding/hex"
	"fmt"
)

// BitwiseOperation represents a bitwise operation type.
type BitwiseOperation string

const (
	// XOR operation.
	XOR BitwiseOperation = "XOR"
	// AND operation.
	AND BitwiseOperation = "AND"
	// OR operation.
	OR BitwiseOperation = "OR"
	// NOT operation.
	NOT BitwiseOperation = "NOT"
)

// BitwiseParams holds parameters for bitwise operations.
type BitwiseParams struct {
	Operation BitwiseOperation
	BlockA    string // Hex string input.
	BlockB    string // Hex string input (optional for NOT).
}

// PerformBitwise executes the specified bitwise operation.
func PerformBitwise(params *BitwiseParams) (string, error) {
	if params == nil {
		return "", fmt.Errorf("params cannot be nil")
	}

	// Decode hex input A.
	a, err := hex.DecodeString(params.BlockA)
	if err != nil {
		return "", fmt.Errorf("invalid hex in block A: %v", err)
	}

	// For NOT operation, we don't need block B.
	if params.Operation == NOT {
		result := make([]byte, len(a))
		for i := range a {
			result[i] = ^a[i]
		}
		return hex.EncodeToString(result), nil
	}

	// Decode hex input B for other operations.
	b, err := hex.DecodeString(params.BlockB)
	if err != nil {
		return "", fmt.Errorf("invalid hex in block B: %v", err)
	}

	// Validate input lengths match.
	if len(a) != len(b) {
		return "", fmt.Errorf("input blocks must be same length")
	}

	result := make([]byte, len(a))

	// Perform the operation.
	switch params.Operation {
	case XOR:
		for i := range a {
			result[i] = a[i] ^ b[i]
		}
	case AND:
		for i := range a {
			result[i] = a[i] & b[i]
		}
	case OR:
		for i := range a {
			result[i] = a[i] | b[i]
		}
	default:
		return "", fmt.Errorf("unsupported operation: %s", params.Operation)
	}

	return hex.EncodeToString(result), nil
}
