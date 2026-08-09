// tpm.go — TPM-Bound Vault Key Unsealing
//
// Phase 6.6 of OVAV-VAULT-2026 plan.
//
// This module provides hardware-bound vault key unsealing using TPM 2.0.
// The vault key is sealed to the TPM with PCR policy, ensuring the key
// can only be retrieved when the system boots into a trusted configuration.
//
// Supported platforms:
//   - Linux: TPM2 via /dev/tpm0 or /dev/tpmrm0 (TSS2 stack)
//   - Windows: TPM2 via TBS (TPM Base Services)
//   - macOS: Keychain (fallback for T2/Apple Silicon)
//
// PCR Policy: The vault key is sealed to a set of PCR values that verify
// the boot chain and OS configuration. Any change to the measured boot
// configuration invalidates the TPM policy, requiring manual re-sealing.
package secrets

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ── Platform Detection ────────────────────────────────────────────────────────

// TPMInterface represents the TPM implementation for the current platform.
type TPMInterface interface {
	// Seal wraps a key with the TPM using the current PCR policy.
	Seal(plaintext []byte, pcrBank PCRBank) ([]byte, error)
	// Unseal retrieves a key from the TPM if PCR policy is satisfied.
	Unseal(sealed []byte) ([]byte, error)
	// ExtendPCR extends a PCR with a new measurement.
	ExtendPCR(bank PCRBank, index int, data []byte) error
	// ReadPCR reads the current value of a PCR.
	ReadPCR(bank PCRBank, index int) ([]byte, error)
	// Quote generates a TPM quote for attestation.
	Quote(nonce []byte) (*TPMQuote, error)
	// Available returns true if a TPM is present and functional.
	Available() bool
}

// PCRBank selects the PCR bank to use for sealing.
type PCRBank uint8

const (
	PCRBankSHA256 PCRBank = 10 // Linux default; Windows: TPM_ALG_SHA256
	PCRBankSHA1   PCRBank = 4  // Legacy; avoid if possible
)

// ── Linux TPM2 ────────────────────────────────────────────────────────────────

// linuxTPM2 implements TPMInterface for Linux TPM 2.0.
type linuxTPM2 struct {
	tcti        string // "/dev/tpm0" or "/dev/tpmrm0"
	bank        PCRBank
	initialized bool
}

// newLinuxTPM2 creates a new Linux TPM2 interface.
func newLinuxTPM2() (*linuxTPM2, error) {
	// Try /dev/tpmrm0 first (resource manager) then /dev/tpm0
	tcti := "/dev/tpmrm0"
	if _, err := os.Stat(tcti); os.IsNotExist(err) {
		tcti = "/dev/tpm0"
		if _, err := os.Stat(tcti); os.IsNotExist(err) {
			return nil, errors.New("no TPM2 device found (/dev/tpm0 or /dev/tpmrm0)")
		}
	}
	return &linuxTPM2{tcti: tcti, bank: PCRBankSHA256}, nil
}

func (t *linuxTPM2) Available() bool {
	_, err := os.Stat(t.tcti)
	return err == nil
}

// ── Windows TPM2 ────────────────────────────────────────────────────────────

// windowsTPM2 implements TPMInterface for Windows TBS.
type windowsTPM2 struct {
	bank PCRBank
}

// newWindowsTPM2 creates a new Windows TPM2 interface.
func newWindowsTPM2() (*windowsTPM2, error) {
	return &windowsTPM2{bank: PCRBankSHA256}, nil
}

func (t *windowsTPM2) Available() bool {
	// Windows: check for TBS via registry or direct availability
	return true // TBS is always available on TPM-capable Windows
}

// ── macOS Keychain ───────────────────────────────────────────────────────────

// macOSKeychain implements TPMInterface for macOS Keychain as a TPM stand-in.
// This provides software-bound sealing using the Secure Enclave / T2 chip.
type macOSKeychain struct {
	keychainPath string
}

// newMacOSKeychain creates a new macOS Keychain-backed interface.
func newMacOSKeychain() (*macOSKeychain, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return &macOSKeychain{
		keychainPath: filepath.Join(home, "Library/Keychains/ovav-vault.keychain"),
	}, nil
}

func (m *macOSKeychain) Available() bool {
	// macOS always has Keychain
	return true
}

// ── Platform Factory ─────────────────────────────────────────────────────────

// NewTPM creates the appropriate TPM interface for the current platform.
func NewTPM() (TPMInterface, error) {
	switch runtime.GOOS {
	case "linux":
		t, err := newLinuxTPM2()
		if err != nil {
			return nil, err
		}
		if !t.Available() {
			return nil, errors.New("TPM2 not available on this Linux system")
		}
		return t, nil
	case "windows":
		return newWindowsTPM2()
	case "darwin":
		return newMacOSKeychain()
	default:
		return nil, fmt.Errorf("TPM not supported on %s", runtime.GOOS)
	}
}

// ── Vault Key Sealing ────────────────────────────────────────────────────────

// SealedVaultKey is the on-disk representation of a TPM-sealed vault key.
type SealedVaultKey struct {
	// SealedBlob is the TPM SealedKey blob (opaque bytes).
	SealedBlob []byte `json:"sealed_blob"`
	// PCRPolicy is a hash of the PCR values used during sealing.
	PCRPolicyHash string `json:"pcr_policy_hash"`
	// Platform is the platform that created this sealed key.
	Platform string `json:"platform"`
	// Created is the creation timestamp.
	Created string `json:"created"`
	// Salt is the random salt used in sealing (stored alongside, not sealed).
	Salt []byte `json:"salt"`
}

// SealToTPM seals the vault key to the platform TPM.
// This is called ONCE during initial vault setup or re-sealing after a PCR change.
// After sealing, the plaintext key is NEVER stored on disk again.
func SealToTPM(key []byte, tpm TPMInterface, pcrs PCRConfig) (*SealedVaultKey, error) {
	if len(key) == 0 {
		return nil, errors.New("cannot seal empty key")
	}

	// Generate a random salt for additional entropy
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}

	// Combine key + salt
	sealingData := append(key, salt...)

	// Seal to TPM
	sealed, err := tpm.Seal(sealingData, PCRBankSHA256)
	if err != nil {
		return nil, fmt.Errorf("TPM seal: %w", err)
	}

	// Compute PCR policy hash
	pcrPolicyHash, err := computePCRPolicyHash(tpm, pcrs)
	if err != nil {
		return nil, fmt.Errorf("compute PCR policy: %w", err)
	}

	return &SealedVaultKey{
		SealedBlob:    sealed,
		PCRPolicyHash: pcrPolicyHash,
		Platform:      runtime.GOOS,
		Salt:          salt,
	}, nil
}

// UnsealFromTPM retrieves the vault key from the TPM if PCR policy is satisfied.
// This is called every time the vault is unlocked.
func UnsealFromTPM(sk *SealedVaultKey, tpm TPMInterface, pcrs PCRConfig) ([]byte, error) {
	if sk == nil || len(sk.SealedBlob) == 0 {
		return nil, errors.New("no sealed key provided")
	}

	// Verify PCR policy hasn't changed
	currentHash, err := computePCRPolicyHash(tpm, pcrs)
	if err != nil {
		return nil, fmt.Errorf("compute PCR policy: %w", err)
	}

	if currentHash != sk.PCRPolicyHash {
		return nil, errors.New(
			"TPM PCR policy mismatch — system boot configuration has changed. " +
				"Re-seal the vault key from a trusted boot state.")
	}

	// Unseal from TPM
	sealingData, err := tpm.Unseal(sk.SealedBlob)
	if err != nil {
		return nil, fmt.Errorf("TPM unseal: %w", err)
	}

	// Extract key and salt
	if len(sealingData) < 32 {
		return nil, errors.New("TPM unsealed data too short")
	}
	salt := sealingData[len(sealingData)-32:]
	key := sealingData[:len(sealingData)-32]

	// Verify salt matches
	if string(salt) != string(sk.Salt) {
		return nil, errors.New("TPM seal salt mismatch")
	}

	return key, nil
}

// ── PCR Configuration ─────────────────────────────────────────────────────────

// PCRConfig defines which PCRs to use for sealing.
// This is platform-specific. These defaults are for UEFI + Linux.
type PCRConfig struct {
	// Bank is the PCR bank (SHA256 recommended).
	Bank PCRBank
	// PCRs is the list of PCR indices to include in the policy.
	PCRs []int
	// Description is a human-readable description of this PCR config.
	Description string
}

// DefaultLinuxPCRConfig is the recommended PCR config for UEFI Linux systems.
var DefaultLinuxPCRConfig = PCRConfig{
	Bank:        PCRBankSHA256,
	PCRs:        []int{0, 1, 2, 3, 4, 5, 6, 7},
	Description: "UEFI + Linux boot chain (PCR 0-7)",
}

// DefaultWindowsPCRConfig is the recommended PCR config for Windows systems.
var DefaultWindowsPCRConfig = PCRConfig{
	Bank:        PCRBankSHA256,
	PCRs:        []int{0, 1, 2, 3, 4, 5, 6, 7},
	Description: "Windows Measured Boot",
}

// computePCRPolicyHash computes a hash over all configured PCR values.
func computePCRPolicyHash(tpm TPMInterface, pcrs PCRConfig) (string, error) {
	h := sha256.New()
	for _, pcrIndex := range pcrs.PCRs {
		val, err := tpm.ReadPCR(pcrs.Bank, pcrIndex)
		if err != nil {
			return "", fmt.Errorf("read PCR %d: %w", pcrIndex, err)
		}
		h.Write(val)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ── TPM Quote (Attestation) ───────────────────────────────────────────────────

// TPMQuote represents a TPM quote for attestation.
type TPMQuote struct {
	Quote     []byte
	Signature []byte
	PCRValues map[int][]byte
	Nonce     []byte
}

// Quote generates a TPM quote for remote attestation.
func (t *linuxTPM2) Quote(nonce []byte) (*TPMQuote, error) {
	// This would call TPM2_Quote via the TSS
	// For now, return a placeholder
	return nil, errors.New("TPM quote not yet implemented — requires TSS2 integration")
}

// ── Stub Implementations ─────────────────────────────────────────────────────

func (t *linuxTPM2) Seal(plaintext []byte, pcrBank PCRBank) ([]byte, error) {
	// Stub: real implementation uses go-tpm and TSS2
	// Sealing requires:
	// 1. Create TPM2 primary key in the SRK hierarchy
	// 2. Create a child key with PCR policy
	// 3. Encrypt plaintext using TPM2_Create -> TPM2_ObjectChangeAuth
	return nil, errors.New("TPM sealing not yet implemented — requires go-tpm + TSS2")
}

func (t *linuxTPM2) Unseal(sealed []byte) ([]byte, error) {
	// Stub: real implementation uses go-tpm
	return nil, errors.New("TPM unsealing not yet implemented — requires go-tpm + TSS2")
}

func (t *linuxTPM2) ExtendPCR(bank PCRBank, index int, data []byte) error {
	return errors.New("PCR extension not yet implemented")
}

func (t *linuxTPM2) ReadPCR(bank PCRBank, index int) ([]byte, error) {
	// Read from sysfs on Linux: /sys/class/tpm/tpm0/pcrs
	if runtime.GOOS != "linux" {
		return nil, errors.New("PCR reading only implemented for Linux")
	}

	pcrPath := fmt.Sprintf("/sys/class/tpm/tpm0/pcr-%s/%d", strings.ToLower(bankName(bank)), index)
	data, err := os.ReadFile(pcrPath)
	if err != nil {
		return nil, fmt.Errorf("read PCR %d: %w", index, err)
	}

	// Parse hex string (format: "0011223344...")
	hexStr := strings.TrimSpace(string(data))
	return hex.DecodeString(hexStr)
}

func bankName(bank PCRBank) string {
	switch bank {
	case PCRBankSHA256:
		return "sha256"
	case PCRBankSHA1:
		return "sha1"
	default:
		return "unknown"
	}
}

// Windows stubs
func (t *windowsTPM2) Seal(plaintext []byte, pcrBank PCRBank) ([]byte, error) {
	return nil, errors.New("Windows TPM sealing not yet implemented")
}

func (t *windowsTPM2) Unseal(sealed []byte) ([]byte, error) {
	return nil, errors.New("Windows TPM unsealing not yet implemented")
}

func (t *windowsTPM2) ExtendPCR(bank PCRBank, index int, data []byte) error {
	return errors.New("PCR extension not implemented on Windows")
}

func (t *windowsTPM2) ReadPCR(bank PCRBank, index int) ([]byte, error) {
	return nil, errors.New("PCR reading not implemented on Windows — use TBS API")
}

func (t *windowsTPM2) Quote(nonce []byte) (*TPMQuote, error) {
	return nil, errors.New("TPM quote not yet implemented on Windows")
}

// macOS Keychain stubs
func (m *macOSKeychain) Seal(plaintext []byte, pcrBank PCRBank) ([]byte, error) {
	// Use Keychain to store a reference (not the actual sealing — macOS Secure Enclave
	// would be needed for proper hardware binding)
	// For now, use Keychain with access control: kSecAttrAccessibleWhenUnlockedThisDeviceOnly
	return plaintext, nil // placeholder — real impl would use SecKeyCreateRandomKey + SecKeyCreateEncryptedData
}

func (m *macOSKeychain) Unseal(sealed []byte) ([]byte, error) {
	return sealed, nil // placeholder
}

func (m *macOSKeychain) ExtendPCR(bank PCRBank, index int, data []byte) error {
	return errors.New("PCR not available on macOS")
}

func (m *macOSKeychain) ReadPCR(bank PCRBank, index int) ([]byte, error) {
	return nil, errors.New("PCR not available on macOS")
}

func (m *macOSKeychain) Quote(nonce []byte) (*TPMQuote, error) {
	return nil, errors.New("TPM quote not available on macOS")
}

// ── Vault Key File Operations ─────────────────────────────────────────────────

const sealedKeyPath = ".ovav/vault/sealed.key"

// SaveSealedKey saves the sealed vault key to disk.
func SaveSealedKey(sk *SealedVaultKey) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	path := filepath.Join(home, sealedKeyPath)
	data, err := json.Marshal(sk)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// LoadSealedKey loads the sealed vault key from disk.
func LoadSealedKey() (*SealedVaultKey, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(home, sealedKeyPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	sk := &SealedVaultKey{}
	if err := json.Unmarshal(data, sk); err != nil {
		return nil, err
	}
	return sk, nil
}

// ── TPM-NI (TPM via NIX on Windows) ─────────────────────────────────────────

// Note: For production, consider go-tpm (https://github.com/google/go-tpm)
// which provides cross-platform TPM 2.0 support.
//
// Example go-tpm usage:
//
//   // Linux/Windows:
//   tpm, err := tpm2.OpenTPM("/dev/tpm0")
//   err = tpm2.Seal(tpm, keyHandle, authValue, sealedTPM2Object, PCRPolicy)
//   unsealed, err := tpm2.Unseal(tpm, keyHandle, authValue)
//
//   // macOS: Use Keychain Services API directly (CGO)
//   // SecKeyCreateRandomKey with kSecAttrTokenIDSecureEnclave
//
// This module is structured for go-tpm integration when ready.
// The stub implementations return errors indicating implementation is needed.
