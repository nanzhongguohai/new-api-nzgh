package controller

import (
	"encoding/json"
	"testing"
)

func TestOIDCCodeUserID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   interface{}
		want    int
		wantErr bool
	}{
		{
			name:  "int from memory store",
			input: 123,
			want:  123,
		},
		{
			name:  "float64 from redis json",
			input: float64(456),
			want:  456,
		},
		{
			name:  "string value",
			input: "789",
			want:  789,
		},
		{
			name:  "json number",
			input: json.Number("321"),
			want:  321,
		},
		{
			name:    "fractional float64",
			input:   12.5,
			wantErr: true,
		},
		{
			name:    "invalid string",
			input:   "abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := oidcCodeUserID(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil with value %d", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %d, got %d", tt.want, got)
			}
		})
	}
}
