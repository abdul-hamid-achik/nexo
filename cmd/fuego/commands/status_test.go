package commands

import (
	"testing"
)

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected string
	}{
		{"zero", 0, "0"},
		{"small number", 123, "123"},
		{"under thousand", 999, "999"},
		{"exactly thousand", 1000, "1.0K"},
		{"thousands", 1500, "1.5K"},
		{"thousands round", 2000, "2.0K"},
		{"ten thousands", 15000, "15.0K"},
		{"hundred thousands", 150000, "150.0K"},
		{"under million", 999999, "1000.0K"},
		{"exactly million", 1000000, "1.0M"},
		{"millions", 1500000, "1.5M"},
		{"millions round", 2000000, "2.0M"},
		{"ten millions", 15000000, "15.0M"},
		{"hundred millions", 150000000, "150.0M"},
		{"billion", 1000000000, "1000.0M"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatNumber(tt.input)
			if result != tt.expected {
				t.Errorf("formatNumber(%d) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestStatusCmd_NotNil(t *testing.T) {
	if statusCmd == nil {
		t.Error("statusCmd should not be nil")
	}
}

func TestStatusCmd_Use(t *testing.T) {
	if statusCmd.Use != "status <app>" {
		t.Errorf("statusCmd.Use = %q, want 'status <app>'", statusCmd.Use)
	}
}

func TestStatusCmd_Short(t *testing.T) {
	if statusCmd.Short == "" {
		t.Error("statusCmd.Short should not be empty")
	}
}

func TestStatusCmd_RequiresArgs(t *testing.T) {
	// StatusCmd requires exactly 1 arg
	args := statusCmd.Args
	if args == nil {
		t.Error("statusCmd.Args should not be nil")
	}

	// Test with no args - should fail
	err := args(statusCmd, []string{})
	if err == nil {
		t.Error("Expected error with no args")
	}

	// Test with 1 arg - should pass
	err = args(statusCmd, []string{"myapp"})
	if err != nil {
		t.Errorf("Expected no error with 1 arg, got: %v", err)
	}

	// Test with 2 args - should fail
	err = args(statusCmd, []string{"myapp", "extra"})
	if err == nil {
		t.Error("Expected error with 2 args")
	}
}
