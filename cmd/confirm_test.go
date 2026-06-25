package cmd

import (
	"strings"
	"testing"
)

func TestConfirmDestructiveFrom(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		isTTY   bool
		yes     bool
		wantErr bool
	}{
		{name: "assume yes skips prompt", input: "", isTTY: false, yes: true, wantErr: false},
		{name: "no tty without yes refuses", input: "", isTTY: false, yes: false, wantErr: true},
		{name: "tty answers yes", input: "y\n", isTTY: true, yes: false, wantErr: false},
		{name: "tty answers full yes", input: "yes\n", isTTY: true, yes: false, wantErr: false},
		{name: "tty answers no", input: "n\n", isTTY: true, yes: false, wantErr: true},
		{name: "tty empty answer aborts", input: "\n", isTTY: true, yes: false, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := confirmDestructiveFrom("delete device x", strings.NewReader(tc.input), tc.isTTY, tc.yes)
			if (err != nil) != tc.wantErr {
				t.Fatalf("confirmDestructiveFrom err = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}
