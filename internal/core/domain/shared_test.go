package domain

import "testing"

func TestValidateID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want bool
	}{
		{"valid 24-char hex", "aabbccddee112233aabbccdd", true},
		{"empty string", "", false},
		{"too short", "aabbcc", false},
		{"too long", "aabbccddee112233aabbccddd", false},
		{"exactly 23 chars", "aabbccddee112233aabbccd", false},
		{"exactly 25 chars", "aabbccddee112233aabbccdde", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateID(tt.id); got != tt.want {
				t.Errorf("ValidateID(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

func TestNewAmountFromCents(t *testing.T) {
	a := NewAmountFromCents(2999)
	if a != 2999 {
		t.Fatalf("expected 2999, got %d", a)
	}
}

func TestNewAmountFromValue(t *testing.T) {
	a := NewAmountFromValue(29)
	if a != 2900 {
		t.Fatalf("expected 2900, got %d", a)
	}
}

func TestAmount_Add(t *testing.T) {
	tests := []struct {
		name string
		a, b Amount
		want Amount
	}{
		{"positive + positive", 100, 200, 300},
		{"zero + positive", 0, 500, 500},
		{"positive + zero", 500, 0, 500},
		{"zero + zero", 0, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.Add(tt.b); got != tt.want {
				t.Errorf("(%d).Add(%d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestAmount_Multiply(t *testing.T) {
	tests := []struct {
		name string
		a    Amount
		b    int
		want Amount
	}{
		{"simple multiply", 100, 3, 300},
		{"multiply by zero", 500, 0, 0},
		{"multiply by one", 2999, 1, 2999},
		{"zero amount", 0, 10, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.Multiply(tt.b); got != tt.want {
				t.Errorf("(%d).Multiply(%d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestAmount_ToValue(t *testing.T) {
	tests := []struct {
		name string
		a    Amount
		want int
	}{
		{"exact conversion", 2900, 29},
		{"truncates remainder", 2999, 29},
		{"zero", 0, 0},
		{"less than 100", 50, 0},
		{"exactly 100", 100, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.ToValue(); got != tt.want {
				t.Errorf("(%d).ToValue() = %d, want %d", tt.a, got, tt.want)
			}
		})
	}
}
