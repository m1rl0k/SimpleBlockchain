package main

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"
)

// Block represents a block in the blockchain
type Block struct {
	Index        int
	Timestamp    int64
	Transactions []*Transaction
	MerkleRoot   string
	Hash         string
	PrevHash     string
	Nonce        int
	Difficulty   int
}

// NewBlock creates a new block and initializes it
func NewBlock(index int, transactions []*Transaction, prevHash string, difficulty int) *Block {
	block := &Block{
		Index:        index,
		Timestamp:    time.Now().UnixNano(),
		Transactions: transactions,
		PrevHash:     prevHash,
		Difficulty:   difficulty,
	}

	block.MerkleRoot = CalculateMerkleRoot(transactions)
	block.Hash = block.CalculateHash()
	return block
}

// CalculateHash returns the hash of the block
func (b *Block) CalculateHash() string {
	record := strconv.Itoa(b.Index) + 
		strconv.FormatInt(b.Timestamp, 10) + 
		b.MerkleRoot + 
		b.PrevHash + 
		strconv.Itoa(b.Nonce) +
		strconv.Itoa(b.Difficulty)
	
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

// MineBlock mines a new block
func (b *Block) MineBlock() {
	target := strings.Repeat("0", b.Difficulty)
	for {
		if strings.HasPrefix(b.Hash, target) {
			return
		}
		b.Nonce++
		b.Hash = b.CalculateHash()
	}
}

// CalculateMerkleRoot creates a Merkle tree from transactions and returns the root
func CalculateMerkleRoot(transactions []*Transaction) string {
	if len(transactions) == 0 {
		return ""
	}
	
	var hashes []string
	
	for _, tx := range transactions {
		hashes = append(hashes, tx.Hash)
	}
	
	// If odd number of transactions, duplicate the last one
	if len(hashes)%2 != 0 {
		hashes = append(hashes, hashes[len(hashes)-1])
	}
	
	// Process pairs of hashes until we reach the root
	for len(hashes) > 1 {
		var newHashes []string
		
		for i := 0; i < len(hashes); i += 2 {
			hash1 := hashes[i]
			hash2 := hashes[i+1]
			
			// Concatenate and hash
			combined := hash1 + hash2
			h := sha256.New()
			h.Write([]byte(combined))
			newHash := hex.EncodeToString(h.Sum(nil))
			
			newHashes = append(newHashes, newHash)
		}
		
		hashes = newHashes
		
		// If odd number after reduction, duplicate the last one
		if len(hashes)%2 != 0 && len(hashes) > 1 {
			hashes = append(hashes, hashes[len(hashes)-1])
		}
	}
	
	return hashes[0]
}
