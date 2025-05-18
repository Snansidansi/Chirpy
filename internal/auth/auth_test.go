package auth

import (
	"fmt"
	"testing"
)

func TestHashPassword(t *testing.T) {
	passwords := []string{
		"a",
		"hello world",
		"password",
		"12345",
		"@%§2.",
	}

	for i, password := range passwords {
		t.Run(fmt.Sprintf("Test %v", i), func(t *testing.T) {
			hash, err := HashPassword(password)
			if err != nil {
				t.Errorf("got unexpected error: %v", err)
				return
			}

			if hash == password {
				t.Errorf("password did not get hashed: %v", password)
			}
		})
	}
}

func TestCheckPasswordHash(t *testing.T) {
	tests := []struct {
		password string
		wantErr  bool
	}{
		{
			password: "abcde",
			wantErr:  true,
		},
		{
			password: "abcde",
			wantErr:  false,
		},
		{
			password: "Hell0 World!@",
			wantErr:  true,
		},
		{
			password: "Hell0 World!@",
			wantErr:  false,
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("Test %v", i), func(t *testing.T) {
			hash := "invalid"
			if !tt.wantErr {
				realHash, _ := HashPassword(tt.password)
				hash = realHash
			}

			gotErr := CheckPasswordHash(hash, tt.password)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("CheckPasswordHash() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("CheckPasswordHash() succeeded unexpectedly")
			}
		})
	}
}
