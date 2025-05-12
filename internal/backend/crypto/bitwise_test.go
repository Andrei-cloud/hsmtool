// nolint:all // test package
package crypto

import (
	"testing"
)

func TestPerformBitwise(t *testing.T) {
	tests := []struct {
		name    string
		params  *BitwiseParams
		want    string
		wantErr bool
	}{
		{
			name: "XOR_success",
			params: &BitwiseParams{
				Operation: XOR,
				BlockA:    "0123456789ABCDEF",
				BlockB:    "FEDCBA9876543210",
			},
			want:    "FFFFFFFFFFFFFFFF",
			wantErr: false,
		},
		{
			name: "XOR_different_lengths",
			params: &BitwiseParams{
				Operation: XOR,
				BlockA:    "0123",
				BlockB:    "FEDCBA9876543210",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "XOR_invalid_hex_A",
			params: &BitwiseParams{
				Operation: XOR,
				BlockA:    "0123G",
				BlockB:    "FEDCBA9876543210",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "XOR_invalid_hex_B",
			params: &BitwiseParams{
				Operation: XOR,
				BlockA:    "0123456789ABCDEF",
				BlockB:    "FEDCG",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "AND_success",
			params: &BitwiseParams{
				Operation: AND,
				BlockA:    "0F0F0F0F",
				BlockB:    "FFFF0000",
			},
			want:    "0F0F0000",
			wantErr: false,
		},
		{
			name: "AND_different_lengths",
			params: &BitwiseParams{
				Operation: AND,
				BlockA:    "0F0F",
				BlockB:    "FFFF0000",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "OR_success",
			params: &BitwiseParams{
				Operation: OR,
				BlockA:    "0F0F0000",
				BlockB:    "0000F0F0",
			},
			want:    "0F0FF0F0",
			wantErr: false,
		},
		{
			name: "OR_different_lengths",
			params: &BitwiseParams{
				Operation: OR,
				BlockA:    "0F0F00",
				BlockB:    "0000F0F0",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "NOT_success",
			params: &BitwiseParams{
				Operation: NOT,
				BlockA:    "0123456789ABCDEF",
			},
			want:    "FEDCBA9876543210",
			wantErr: false,
		},
		{
			name: "NOT_invalid_hex_A",
			params: &BitwiseParams{
				Operation: NOT,
				BlockA:    "0123G",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Unsupported_operation",
			params: &BitwiseParams{
				Operation: "SUB",
				BlockA:    "0123",
				BlockB:    "4567",
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PerformBitwise(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("PerformBitwise() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("PerformBitwise() = %v, want %v", got, tt.want)
			}
		})
	}
}
