package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
)

// Wallet handles cryptographic keys and addresses for a user
type Wallet struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
	Address    string
}

// NewWallet creates a new wallet with generated keys
func NewWallet() (*Wallet, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %v", err)
	}

	publicKey := &privateKey.PublicKey
	
	// Generate wallet address from public key
	address := generateAddressFromPublicKey(publicKey)
	
	return &Wallet{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Address:    address,
	}, nil
}

// generateAddressFromPublicKey creates a unique address from a public key
func generateAddressFromPublicKey(publicKey *rsa.PublicKey) string {
	publicKeyBytes := x509.MarshalPKCS1PublicKey(publicKey)
	address := fmt.Sprintf("%x", publicKeyBytes[:20]) // Use first 20 bytes
	return address
}

// CreateTransaction creates a new signed transaction from this wallet
func (w *Wallet) CreateTransaction(recipient string, amount float64) (*Transaction, error) {
	tx, err := NewTransaction(w.Address, recipient, amount, w.PrivateKey)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

// SaveToFile saves wallet keys to a file
func (w *Wallet) SaveToFile(filename string) error {
	// Marshal private key to PEM format
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(w.PrivateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})
	
	// Write private key to file
	err := ioutil.WriteFile(filename, privateKeyPEM, 0600)
	if err != nil {
		return fmt.Errorf("failed to write private key to file: %v", err)
	}
	
	return nil
}

// LoadFromFile loads wallet from key file
func LoadWalletFromFile(filename string) (*Wallet, error) {
	// Read private key from file
	privateKeyPEM, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key from file: %v", err)
	}
	
	// Decode PEM block
	block, _ := pem.Decode(privateKeyPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	
	// Parse private key
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}
	
	// Create wallet
	publicKey := &privateKey.PublicKey
	address := generateAddressFromPublicKey(publicKey)
	
	return &Wallet{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Address:    address,
	}, nil
}

// GetBalance calculates wallet balance from blockchain
func (w *Wallet) GetBalance(blockchain *Blockchain) float64 {
	balance := 0.0
	
	for _, block := range blockchain.Blocks {
		for _, tx := range block.Transactions {
			if tx.Sender == w.Address {
				balance -= tx.Amount
			}
			if tx.Recipient == w.Address {
				balance += tx.Amount
			}
		}
	}
	
	return balance
}
