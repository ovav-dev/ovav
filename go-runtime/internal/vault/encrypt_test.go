package vault

import (
	"bytes"
	"testing"
)

func TestEncryptDecryptRoundtrip(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	plaintext := []byte("OVAV vault test data — confidential agent sources")
	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Ciphertext should be longer than plaintext (nonce + tag overhead)
	if len(ciphertext) <= len(plaintext) {
		t.Errorf("ciphertext len %d <= plaintext len %d", len(ciphertext), len(plaintext))
	}

	decrypted, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Errorf("roundtrip failed: got %q, want %q", decrypted, plaintext)
	}
}

func TestDecryptWrongKey(t *testing.T) {
	key1, _ := GenerateKey()
	key2, _ := GenerateKey()

	plaintext := []byte("secret data")
	ciphertext, _ := Encrypt(plaintext, key1)

	_, err := Decrypt(ciphertext, key2)
	if err == nil {
		t.Error("expected decryption with wrong key to fail")
	}
}

func TestDecryptCorruptedData(t *testing.T) {
	key, _ := GenerateKey()
	plaintext := []byte("secret data")
	ciphertext, _ := Encrypt(plaintext, key)

	// Corrupt the ciphertext
	corrupted := make([]byte, len(ciphertext))
	copy(corrupted, ciphertext)
	corrupted[len(corrupted)-1] ^= 0xFF

	_, err := Decrypt(corrupted, key)
	if err == nil {
		t.Error("expected decryption of corrupted data to fail")
	}
}

func TestEncryptInvalidKeySize(t *testing.T) {
	_, err := Encrypt([]byte("data"), []byte("short"))
	if err == nil {
		t.Error("expected error for short key")
	}
}

func TestGenerateKeyUniqueness(t *testing.T) {
	k1, _ := GenerateKey()
	k2, _ := GenerateKey()
	if bytes.Equal(k1, k2) {
		t.Error("two generated keys should be different")
	}
}

func TestEncryptDecryptEmptyData(t *testing.T) {
	key, _ := GenerateKey()
	ciphertext, err := Encrypt([]byte{}, key)
	if err != nil {
		t.Fatalf("Encrypt empty: %v", err)
	}
	decrypted, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decrypt empty: %v", err)
	}
	if len(decrypted) != 0 {
		t.Errorf("expected empty, got %q", decrypted)
	}
}

func TestEncryptDecryptLargeData(t *testing.T) {
	key, _ := GenerateKey()
	large := bytes.Repeat([]byte("OVAV"), 10000) // 40KB
	ciphertext, err := Encrypt(large, key)
	if err != nil {
		t.Fatalf("Encrypt large: %v", err)
	}
	decrypted, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decrypt large: %v", err)
	}
	if !bytes.Equal(large, decrypted) {
		t.Error("large data roundtrip failed")
	}
}
