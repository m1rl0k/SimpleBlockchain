package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("Starting SimpleBlockchain demo...")
	
	// Create shared blockchain for the demo
	chain := NewBlockchain()
	
	fmt.Printf("Genesis block created with hash: %s\n\n", chain.LatestBlock().Hash)
	
	// Create nodes with wallets on different ports
	fmt.Println("=== Creating Nodes and Wallets ===")
	node1 := CreateNodeWithWallet(chain, "58001")
	if node1 == nil {
		fmt.Println("Failed to create node 1. Exiting.")
		return
	}
	
	node2 := CreateNodeWithWallet(chain, "58002")
	if node2 == nil {
		fmt.Println("Failed to create node 2. Exiting.")
		return
	}
	
	node3 := CreateNodeWithWallet(chain, "58003")
	if node3 == nil {
		fmt.Println("Failed to create node 3. Exiting.")
		return
	}
	
	fmt.Println("\n=== Node Wallets ===")
	fmt.Printf("Node 1 Wallet: %s\n", node1.Wallet.Address)
	fmt.Printf("Node 2 Wallet: %s\n", node2.Wallet.Address)
	fmt.Printf("Node 3 Wallet: %s\n", node3.Wallet.Address)
	
	// Give some time for the servers to start
	time.Sleep(500 * time.Millisecond)
	
	// Register all wallet public keys with the blockchain
	fmt.Println("\n=== Registering Wallet Public Keys ===")
	RegisterWallet(chain, node1.Wallet)
	RegisterWallet(chain, node2.Wallet)
	RegisterWallet(chain, node3.Wallet)
	
	// Connect nodes (create a simple topology)
	fmt.Println("\n=== Connecting Nodes ===")
	node1.AddPeer("localhost:58002")
	node1.AddPeer("localhost:58003")
	node2.AddPeer("localhost:58003")
	
	// Brief wait for connections
	time.Sleep(500 * time.Millisecond)
	
	// Mine initial block to get some coins
	fmt.Println("\n=== Mining Initial Block by Node 1 ===")
	initialBlock := node1.Mine()
	fmt.Printf("Initial block mined with hash %s\n", initialBlock.Hash)
	
	// Brief wait for block propagation
	time.Sleep(500 * time.Millisecond)
	
	// Check balances
	fmt.Println("\n=== Initial Balances after First Block ===")
	fmt.Printf("Node 1 Balance: %.2f\n", chain.GetBalanceForAddress(node1.Wallet.Address))
	fmt.Printf("Node 2 Balance: %.2f\n", chain.GetBalanceForAddress(node2.Wallet.Address))
	fmt.Printf("Node 3 Balance: %.2f\n", chain.GetBalanceForAddress(node3.Wallet.Address))
	
	// First transaction
	fmt.Println("\n=== Creating Transaction: Node 1 to Node 2 ===")
	amount := 5.0
	tx1, err := node1.CreateTransaction(node2.Wallet.Address, amount)
	if err != nil {
		fmt.Printf("Transaction creation failed: %v\n", err)
	} else {
		fmt.Printf("Transaction created: %s -> %s: %.2f\n", tx1.Sender, tx1.Recipient, tx1.Amount)
	}
	
	// Short wait for transaction propagation
	fmt.Println("Waiting for transaction to propagate...")
	time.Sleep(300 * time.Millisecond)
	
	// Show pending transactions before mining
	fmt.Println("\n=== Pending Transactions in Blockchain ===")
	fmt.Printf("Number of pending transactions: %d\n", len(chain.PendingTxs))
	for i, tx := range chain.PendingTxs {
		fmt.Printf("Pending Tx #%d: %s -> %s: %.2f\n", i, tx.Sender, tx.Recipient, tx.Amount)
	}
	
	// Mine a block with the pending transaction
	fmt.Println("\n=== Mining Block with Transaction by Node 3 ===")
	newBlock := node3.Mine()
	fmt.Printf("Block mined with hash %s\n", newBlock.Hash)
	
	// Brief wait for block propagation
	fmt.Println("Waiting for block to propagate...")
	time.Sleep(300 * time.Millisecond)
	
	// Check updated balances
	fmt.Println("\n=== Updated Balances after Transaction ===")
	fmt.Printf("Node 1 Balance: %.2f\n", chain.GetBalanceForAddress(node1.Wallet.Address))
	fmt.Printf("Node 2 Balance: %.2f\n", chain.GetBalanceForAddress(node2.Wallet.Address))
	fmt.Printf("Node 3 Balance: %.2f\n", chain.GetBalanceForAddress(node3.Wallet.Address))
	
	// Create another transaction
	fmt.Println("\n=== Creating Transaction: Node 2 to Node 3 ===")
	amount = 2.0
	tx2, err := node2.CreateTransaction(node3.Wallet.Address, amount)
	if err != nil {
		fmt.Printf("Transaction creation failed: %v\n", err)
	} else {
		fmt.Printf("Transaction created: %s -> %s: %.2f\n", tx2.Sender, tx2.Recipient, tx2.Amount)
	}
	
	// Short wait for transaction propagation
	fmt.Println("Waiting for transaction to propagate...")
	time.Sleep(300 * time.Millisecond)
	
	// Show pending transactions again
	fmt.Println("\n=== Pending Transactions in Blockchain ===")
	fmt.Printf("Number of pending transactions: %d\n", len(chain.PendingTxs))
	for i, tx := range chain.PendingTxs {
		fmt.Printf("Pending Tx #%d: %s -> %s: %.2f\n", i, tx.Sender, tx.Recipient, tx.Amount)
	}
	
	// Mine final block
	fmt.Println("\n=== Mining Final Block by Node 1 ===")
	finalBlock := node1.Mine()
	fmt.Printf("Final block mined with hash %s\n", finalBlock.Hash)
	
	// Brief wait for block propagation
	fmt.Println("Waiting for block to propagate...")
	time.Sleep(300 * time.Millisecond)
	
	// Check final balances
	fmt.Println("\n=== Final Balances after All Transactions ===")
	fmt.Printf("Node 1 Balance: %.2f\n", chain.GetBalanceForAddress(node1.Wallet.Address))
	fmt.Printf("Node 2 Balance: %.2f\n", chain.GetBalanceForAddress(node2.Wallet.Address))
	fmt.Printf("Node 3 Balance: %.2f\n", chain.GetBalanceForAddress(node3.Wallet.Address))
	
	// Print blockchain state
	fmt.Println("\n=== Final Blockchain State ===")
	fmt.Printf("Chain length: %d blocks\n", len(chain.Blocks))
	for i, block := range chain.Blocks {
		fmt.Printf("Block #%d: Hash=%s, Transactions=%d\n", 
			i, block.Hash, len(block.Transactions))
		
		// Print transactions in each block
		for j, tx := range block.Transactions {
			fmt.Printf("  Tx #%d: %s -> %s: %.2f\n", 
				j, tx.Sender, tx.Recipient, tx.Amount)
		}
	}
	
	fmt.Println("\nDemo completed successfully!")
}

// Helper function to create a node with wallet
func CreateNodeWithWallet(chain *Blockchain, port string) *Node {
	// Create wallet
	wallet, err := NewWallet()
	if err != nil {
		fmt.Printf("Error creating wallet: %v\n", err)
		return nil
	}
	fmt.Printf("Created wallet with address: %s\n", wallet.Address)
	
	// Create node with wallet
	node, err := NewNode(wallet.Address[:5], "localhost:"+port, nil, wallet)
	if err != nil {
		fmt.Printf("Error creating node: %v\n", err)
		return nil
	}
	
	// Assign the shared blockchain
	node.Blockchain = chain
	
	// Start listening
	go node.Listen()
	fmt.Printf("Node started listening on port %s\n", port)
	
	return node
}

// Helper function to register wallet
func RegisterWallet(chain *Blockchain, wallet *Wallet) {
	chain.RegisterWallet(wallet)
	fmt.Printf("Registered wallet %s with blockchain\n", wallet.Address)
}
