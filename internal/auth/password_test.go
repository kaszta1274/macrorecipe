package auth

import "testing"

func TestHashAndCheck(t *testing.T) {
	passwordHash, err := HashPassword("password123")
	if err != nil {
		t.Fatal(err)
	}

	match, err := VerifyPasswordHash("password123", passwordHash)
	if err != nil {
		t.Fatal(err)
	}
	if !match {
		t.Fatal("VerifyPasswordHash() rejected the password used to create the hash")
	}

	match, err = VerifyPasswordHash("wrongpassword", passwordHash)
	if err != nil {
		t.Fatal(err)
	}
	if match {
		t.Fatal("VerifyPasswordHash() accepted the wrong password")
	}
}
