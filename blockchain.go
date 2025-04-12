package main

import (
	"crypto/rsa"
	"fmt"
	"sync"
	"time"
)

const (
	DefaultDifficulty       = 4
	BlockTargetTime         = 10 * time.Second
	DifficultyAdjustmentInterval = 10
	MiningReward            = 50.0
	SystemAddress           = "SYSTEM"
)

// Blockchain is the main blockchain data structure
type Blockchain struct {
	Blocks        []*Block
	PendingTxs    []*Transaction
	Difficulty    int
	mu            sync.RWMutex
	miningReward  float64
	// Add a map to store the public keys of addresses
	PublicKeys    map[string]*rsa.PublicKey
}

// NewBlockchain creates a new blockchain with genesis block
func NewBlockchain() *Blockchain {
	blockchain := &Blockchain{
		Difficulty:   DefaultDifficulty,
		miningReward: MiningReward,
		PublicKeys:   make(map[string]*rsa.PublicKey),
	}
	
	// Create genesis block with no transactions
	genesisBlock := NewBlock(0, []*Transaction{}, "", DefaultDifficulty)
	genesisBlock.MineBlock()
	blockchain.Blocks = append(blockchain.Blocks, genesisBlock)
	
	return blockchain
}

// RegisterWallet registers a wallet's public key with the blockchain
func (bc *Blockchain) RegisterWallet(wallet *Wallet) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	bc.PublicKeys[wallet.Address] = wallet.PublicKey
}

// AddTransaction adds a new transaction to the pool
func (bc *Blockchain) AddTransaction(tx *Transaction) bool {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	// Print transaction for debugging
	fmt.Printf("Adding transaction: %s -> %s: %.2f\n", tx.Sender, tx.Recipient, tx.Amount)
	
	// Check for duplicate transaction IDs
	for _, pendingTx := range bc.PendingTxs {
		if pendingTx.ID == tx.ID {
			fmt.Printf("Transaction with ID %s already in pending pool, ignoring\n", tx.ID)
			return false
		}
	}
	
	// Validate transaction
	if !bc.ValidateTransaction(tx) {
		fmt.Printf("Transaction validation failed!\n")
		return false
	}
	
	bc.PendingTxs = append(bc.PendingTxs, tx)
	fmt.Printf("Transaction added to pending pool\n")
	return true
}

// ValidateTransaction checks if a transaction is valid
func (bc *Blockchain) ValidateTransaction(tx *Transaction) bool {
	// System transactions (mining rewards) are always valid
	if tx.Sender == SystemAddress {
		fmt.Printf("System transaction - automatically valid\n")
		return true
	}
	
	fmt.Printf("Checking balance for sender: %s\n", tx.Sender)
	// Check if sender has sufficient funds
	senderBalance := bc.GetBalanceForAddress(tx.Sender)
	if senderBalance < tx.Amount {
		fmt.Printf("Insufficient balance: %.2f < %.2f\n", senderBalance, tx.Amount)
		return false
	}
	fmt.Printf("Sender has sufficient balance: %.2f\n", senderBalance)
	
	fmt.Printf("Looking up public key for sender: %s\n", tx.Sender)
	// Get the public key for the sender
	publicKey, exists := bc.PublicKeys[tx.Sender]
	if !exists {
		fmt.Printf("Public key not found for sender %s\n", tx.Sender)
		return false
	}
	fmt.Printf("Public key found for sender\n")
	
	fmt.Printf("Starting signature verification\n")
	// Verify the transaction signature
	if !tx.IsValid(publicKey) {
		fmt.Printf("Transaction signature validation failed\n")
		return false
	}
	
	fmt.Printf("Transaction successfully validated\n")
	return true
}

// GetBalanceForAddress calculates balance for a given address
func (bc *Blockchain) GetBalanceForAddress(address string) float64 {
	balance := 0.0
	
	// Note: This method is called from ValidateTransaction which is called from AddTransaction
	// which already holds the write lock, so we can't acquire another lock here.
	// We'll assume the caller has appropriate locking in place.
	
	// Check confirmed transactions in blocks
	for _, block := range bc.Blocks {
		for _, tx := range block.Transactions {
			if tx.Sender == address {
				balance -= tx.Amount
			}
			if tx.Recipient == address {
				balance += tx.Amount
			}
		}
	}
	
	// Also check pending transactions
	for _, tx := range bc.PendingTxs {
		if tx.Sender == address {
			balance -= tx.Amount
		}
		if tx.Recipient == address {
			balance += tx.Amount
		}
	}
	
	return balance
}

// MinePendingTransactions mines a new block with pending transactions
func (bc *Blockchain) MinePendingTransactions(minerAddress string) *Block {
	bc.mu.Lock()
	
	fmt.Printf("Mining block with %d pending transactions\n", len(bc.PendingTxs))
	
	// Create mining reward transaction
	rewardTx, _ := NewTransaction(SystemAddress, minerAddress, bc.miningReward, nil)
	
	// Copy pending transactions to mine
	txsToMine := make([]*Transaction, len(bc.PendingTxs))
	copy(txsToMine, bc.PendingTxs)
	
	// Add reward transaction
	txsToMine = append(txsToMine, rewardTx)
	
	// Reset pending transactions
	bc.PendingTxs = []*Transaction{}
	
	// Create new block
	prevBlock := bc.LatestBlock()
	newBlock := NewBlock(prevBlock.Index+1, txsToMine, prevBlock.Hash, bc.Difficulty)
	
	bc.mu.Unlock()
	
	// Mine the block
	newBlock.MineBlock()
	
	// Add to blockchain
	bc.mu.Lock()
	bc.Blocks = append(bc.Blocks, newBlock)
	bc.mu.Unlock()
	
	// Adjust difficulty if needed
	bc.AdjustDifficulty()
	
	return newBlock
}

// AddBlockToChain validates and adds a block to the chain
func (bc *Blockchain) AddBlockToChain(newBlock *Block) bool {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	fmt.Printf("Attempting to add block #%d with hash %s\n", newBlock.Index, newBlock.Hash)
	
	// Check if we already have this block hash to prevent duplicates
	for _, block := range bc.Blocks {
		if block.Hash == newBlock.Hash {
			fmt.Printf("Block with hash %s already exists in chain, ignoring\n", newBlock.Hash)
			return false
		}
	}
	
	// Find existing block at this index if any
	var existingBlockAtIndex *Block
	for _, block := range bc.Blocks {
		if block.Index == newBlock.Index {
			existingBlockAtIndex = block
			break
		}
	}
	
	// If there's an existing block at this index, we have a potential fork
	if existingBlockAtIndex != nil {
		fmt.Printf("Block with index %d already exists, handling potential fork\n", newBlock.Index)
		
		// In a production system, we'd implement proper fork resolution
		// For now, we'll use a simple rule: keep the block with the lowest hash value
		// (indicating more work was done to find it)
		if newBlock.Hash < existingBlockAtIndex.Hash {
			fmt.Printf("New block has lower hash value, replacing existing block\n")
			
			// Replace the block at this index
			for i, block := range bc.Blocks {
				if block.Index == newBlock.Index {
					bc.Blocks[i] = newBlock
					// Truncate the chain after this point to avoid inconsistencies
					bc.Blocks = bc.Blocks[:i+1]
					break
				}
			}
			
			return true
		} else {
			fmt.Printf("Existing block has lower hash value, keeping it\n")
			return false
		}
	}
	
	latestBlock := bc.LatestBlock()
	
	// Validate the block connects to our chain
	if newBlock.Index != latestBlock.Index+1 {
		fmt.Printf("Block index mismatch: expected %d, got %d\n", latestBlock.Index+1, newBlock.Index)
		return false
	}
	
	if newBlock.PrevHash != latestBlock.Hash {
		fmt.Printf("Block previous hash mismatch: expected %s, got %s\n", latestBlock.Hash, newBlock.PrevHash)
		return false
	}
	
	// Verify the hash matches the calculated hash
	calculatedHash := newBlock.CalculateHash()
	if newBlock.Hash != calculatedHash {
		fmt.Printf("Block hash is invalid, expected %s, got %s\n", calculatedHash, newBlock.Hash)
		return false
	}
	
	// Verify transactions
	fmt.Printf("Validating %d transactions in block\n", len(newBlock.Transactions))
	for _, tx := range newBlock.Transactions {
		if !bc.ValidateTransaction(tx) {
			fmt.Printf("Transaction validation failed: %s -> %s: %.2f\n", 
				tx.Sender, tx.Recipient, tx.Amount)
			return false
		} else {
			fmt.Printf("Transaction valid: %s -> %s: %.2f\n", tx.Sender, tx.Recipient, tx.Amount)
		}
	}
	
	// Add block to chain
	bc.Blocks = append(bc.Blocks, newBlock)
	fmt.Printf("Block #%d successfully added to the chain\n", newBlock.Index)
	
	// Move any transactions in this block from pending pool
	bc.removePendingTransactionsInBlock(newBlock)
	
	// Adjust difficulty if needed
	bc.AdjustDifficulty()
	
	return true
}

// removePendingTransactionsInBlock removes transactions from pending pool if they're in a block
func (bc *Blockchain) removePendingTransactionsInBlock(block *Block) {
	// Create map of transaction IDs in the block
	txIds := make(map[string]bool)
	for _, tx := range block.Transactions {
		txIds[tx.ID] = true
	}
	
	// Filter pending transactions
	filteredTxs := []*Transaction{}
	for _, pendingTx := range bc.PendingTxs {
		if !txIds[pendingTx.ID] {
			filteredTxs = append(filteredTxs, pendingTx)
		}
	}
	
	bc.PendingTxs = filteredTxs
}

// AdjustDifficulty dynamically adjusts the mining difficulty
func (bc *Blockchain) AdjustDifficulty() {
	if len(bc.Blocks) % DifficultyAdjustmentInterval != 0 {
		return
	}
	
	latestBlock := bc.LatestBlock()
	prevAdjustmentBlock := bc.Blocks[len(bc.Blocks)-DifficultyAdjustmentInterval]
	
	// Calculate time taken for last DifficultyAdjustmentInterval blocks
	timeExpected := BlockTargetTime * time.Duration(DifficultyAdjustmentInterval)
	timeTaken := time.Duration(latestBlock.Timestamp - prevAdjustmentBlock.Timestamp) * time.Nanosecond
	
	// Adjust difficulty
	if timeTaken < timeExpected/2 {
		bc.Difficulty++
	} else if timeTaken > timeExpected*2 {
		if bc.Difficulty > 1 {
			bc.Difficulty--
		}
	}
}

// IsValid validates the entire blockchain
func (bc *Blockchain) IsValid() bool {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	
	// Validate each block
	for i := 1; i < len(bc.Blocks); i++ {
		currentBlock := bc.Blocks[i]
		prevBlock := bc.Blocks[i-1]
		
		// Check hash
		if currentBlock.Hash != currentBlock.CalculateHash() {
			return false
		}
		
		// Check link to previous block
		if currentBlock.PrevHash != prevBlock.Hash {
			return false
		}
		
		// Check block index
		if currentBlock.Index != prevBlock.Index+1 {
			return false
		}
	}
	
	return true
}

// ReplaceChain replaces the chain if the provided one is longer and valid
func (bc *Blockchain) ReplaceChain(newChain *Blockchain) bool {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	// Check if new chain is valid
	if !newChain.IsValid() {
		return false
	}
	
	// Check if new chain is longer
	if len(newChain.Blocks) <= len(bc.Blocks) {
		return false
	}
	
	// Replace chain
	bc.Blocks = newChain.Blocks
	bc.Difficulty = newChain.Difficulty
	
	return true
}

// LatestBlock returns the latest block in the chain
func (bc *Blockchain) LatestBlock() *Block {
	return bc.Blocks[len(bc.Blocks)-1]
}

// Print prints the blockchain
func (bc *Blockchain) Print() {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	
	for _, block := range bc.Blocks {
		fmt.Printf("Block #%d\n", block.Index)
		fmt.Printf("  Timestamp: %d\n", block.Timestamp)
		fmt.Printf("  Hash: %s\n", block.Hash)
		fmt.Printf("  PrevHash: %s\n", block.PrevHash)
		fmt.Printf("  Difficulty: %d\n", block.Difficulty)
		fmt.Printf("  Nonce: %d\n", block.Nonce)
		fmt.Printf("  MerkleRoot: %s\n", block.MerkleRoot)
		fmt.Printf("  Transactions: %d\n", len(block.Transactions))
		
		for j, tx := range block.Transactions {
			fmt.Printf("    Tx #%d: %s -> %s: %.2f\n", j, tx.Sender, tx.Recipient, tx.Amount)
		}
		fmt.Println()
	}
}

// GetBlockByHash returns a block by its hash
func (bc *Blockchain) GetBlockByHash(hash string) *Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	
	for _, block := range bc.Blocks {
		if block.Hash == hash {
			return block
		}
	}
	
	return nil
}

// GetBlockByIndex returns a block by its index
func (bc *Blockchain) GetBlockByIndex(index int) *Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	
	for _, block := range bc.Blocks {
		if block.Index == index {
			return block
		}
	}
	
	return nil
}
