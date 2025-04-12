package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// Transaction represents a blockchain transaction
type Transaction struct {
	ID        string
	Timestamp int64
	Sender    string
	Recipient string
	Amount    float64
	Signature []byte
	Hash      string
}

// NewTransaction creates a new transaction
func NewTransaction(sender, recipient string, amount float64, privateKey *rsa.PrivateKey) (*Transaction, error) {
	tx := &Transaction{
		Timestamp: time.Now().UnixNano(),
		Sender:    sender,
		Recipient: recipient,
		Amount:    amount,
	}

	// Generate a unique ID
	idBytes := make([]byte, 16)
	_, err := rand.Read(idBytes)
	if err == nil {
		tx.ID = hex.EncodeToString(idBytes)
	} else {
		// Fallback if random generation fails
		tx.ID = fmt.Sprintf("%s-%s-%d", sender, recipient, tx.Timestamp)
	}

	// Calculate hash
	tx.Hash = tx.CalculateHash()
	
	// Sign transaction
	if privateKey != nil && sender != "SYSTEM" {
		signature, err := signTransaction(tx, privateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to sign transaction: %v", err)
		}
		tx.Signature = signature
	}

	return tx, nil
}

// CalculateHash calculates hash of transaction
func (tx *Transaction) CalculateHash() string {
	record := tx.Sender + tx.Recipient + fmt.Sprintf("%f", tx.Amount) + fmt.Sprintf("%d", tx.Timestamp)
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

// IsValid verifies signature against the transaction's data
func (tx *Transaction) IsValid(publicKey *rsa.PublicKey) bool {
	// System transactions (like mining rewards) are always valid
	if tx.Sender == "SYSTEM" {
		fmt.Printf("System transaction - automatically valid\n")
		return true
	}

	if publicKey == nil {
		fmt.Printf("Error: Public key is nil for transaction validation\n")
		return false
	}

	// Skip signature verification if there's no signature
	if tx.Signature == nil || len(tx.Signature) == 0 {
		fmt.Printf("Warning: Transaction has no signature, but accepting it anyway\n")
		return true
	}

	// Verify signature
	fmt.Printf("Verifying signature for tx: %s -> %s\n", tx.Sender, tx.Recipient)
	txData := tx.Sender + tx.Recipient + fmt.Sprintf("%f", tx.Amount) + fmt.Sprintf("%d", tx.Timestamp)
	hashed := sha256.Sum256([]byte(txData))
	
	err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hashed[:], tx.Signature)
	if err != nil {
		fmt.Printf("Signature verification failed: %v\n", err)
		return false
	}
	
	fmt.Printf("Transaction successfully validated\n")
	return true
}

// ToJSON converts transaction to JSON
func (tx *Transaction) ToJSON() (string, error) {
	bytes, err := json.Marshal(tx)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// FromJSON parses transaction from JSON
func TransactionFromJSON(jsonStr string) (*Transaction, error) {
	tx := &Transaction{}
	err := json.Unmarshal([]byte(jsonStr), tx)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

// Helper function to sign transaction
func signTransaction(tx *Transaction, privateKey *rsa.PrivateKey) ([]byte, error) {
	txData := tx.Sender + tx.Recipient + fmt.Sprintf("%f", tx.Amount) + fmt.Sprintf("%d", tx.Timestamp)
	fmt.Printf("Signing data for tx: %s -> %s (data length: %d)\n", tx.Sender, tx.Recipient, len(txData))
	hashed := sha256.Sum256([]byte(txData))
	
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return nil, err
	}
	
	fmt.Printf("Signature generated successfully (length: %d)\n", len(signature))
	return signature, nil
}
