package main

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"
)

// MessageType defines the type of P2P message
type MessageType string

const (
	MessageTypeNewBlock     MessageType = "NEW_BLOCK"
	MessageTypeNewTx        MessageType = "NEW_TX"
	MessageTypeRequestChain MessageType = "REQUEST_CHAIN"
	MessageTypeRequestBlock MessageType = "REQUEST_BLOCK"
	MessageTypeResponseChain MessageType = "RESPONSE_CHAIN"
	MessageTypeResponseBlock MessageType = "RESPONSE_BLOCK"
)

// P2PMessage represents a message in the P2P network
type P2PMessage struct {
	Type    MessageType `json:"type"`
	Data    string      `json:"data"`
	NodeID  string      `json:"nodeId"`
}

// Node represents a node in the blockchain network
type Node struct {
	ID          string
	Address     string
	Blockchain  *Blockchain
	Wallet      *Wallet
	Neighbors   []string
	server      net.Listener
	mu          sync.RWMutex
	knownBlocks map[string]bool
	knownTxs    map[string]bool
}

// NewNode creates a new node
func NewNode(id string, address string, neighbors []string, wallet *Wallet) (*Node, error) {
	node := &Node{
		ID:          id,
		Address:     address,
		Blockchain:  NewBlockchain(),
		Wallet:      wallet,
		Neighbors:   neighbors,
		knownBlocks: make(map[string]bool),
		knownTxs:    make(map[string]bool),
	}

	return node, nil
}

// StartServer starts the node's server
func (n *Node) StartServer() error {
	go n.Listen()
	fmt.Printf("Node %s started server on %s\n", n.ID, n.Address)
	return nil
}

// Listen listens for incoming connections
func (n *Node) Listen() {
	// Create a listener if not already created
	if n.server == nil {
		var err error
		n.server, err = net.Listen("tcp", n.Address)
		if err != nil {
			fmt.Printf("Error starting listener on %s: %v\n", n.Address, err)
			return
		}
		fmt.Printf("Node %s started listening on %s\n", n.ID, n.Address)
	}

	for {
		conn, err := n.server.Accept()
		if err != nil {
			fmt.Printf("Error accepting connection: %v\n", err)
			continue
		}

		go n.handleConnection(conn)
	}
}

// handleConnection handles an incoming connection
func (n *Node) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	
	buffer := make([]byte, 4096*4) // Larger buffer
	numBytes, err := conn.Read(buffer)
	if err != nil {
		fmt.Printf("Error reading from connection: %v\n", err)
		return
	}

	// Parse message
	var message P2PMessage
	err = json.Unmarshal(buffer[:numBytes], &message)
	if err != nil {
		fmt.Printf("Error parsing message: %v\n", err)
		return
	}

	fmt.Printf("Node %s received message of type %s from %s\n", n.ID, message.Type, message.NodeID)

	// Handle message based on type
	switch message.Type {
	case MessageTypeNewBlock:
		n.handleNewBlockMessage(message)
	case MessageTypeNewTx:
		n.handleNewTransactionMessage(message)
	case MessageTypeRequestChain:
		n.handleRequestChainMessage(conn, message)
	case MessageTypeRequestBlock:
		n.handleRequestBlockMessage(conn, message)
	case MessageTypeResponseChain:
		n.handleResponseChainMessage(message)
	case MessageTypeResponseBlock:
		n.handleResponseBlockMessage(message)
	default:
		fmt.Printf("Unknown message type: %s\n", message.Type)
	}
}

// handleNewBlockMessage handles a new block message
func (n *Node) handleNewBlockMessage(message P2PMessage) {
	// Parse block from message
	var block Block
	err := json.Unmarshal([]byte(message.Data), &block)
	if err != nil {
		fmt.Printf("Error parsing block: %v\n", err)
		return
	}

	// Double-check if we already know this block to prevent duplicates
	n.mu.Lock()
	if n.knownBlocks[block.Hash] {
		fmt.Printf("Node %s already knows block with hash %s, ignoring\n", n.ID, block.Hash)
		n.mu.Unlock()
		return
	}
	n.knownBlocks[block.Hash] = true
	n.mu.Unlock()

	fmt.Printf("Node %s received new block with hash %s\n", n.ID, block.Hash)

	// Attempt to add the block to our chain
	success := n.Blockchain.AddBlockToChain(&block)
	if success {
		fmt.Printf("Node %s added block with index %d and hash %s\n", n.ID, block.Index, block.Hash)
		
		// Forward the block to other nodes (but not back to sender)
		n.forwardBlock(&block, message.NodeID)
	} else {
		fmt.Printf("Node %s failed to add block with hash %s\n", n.ID, block.Hash)
	}
}

// handleNewTransactionMessage handles a new transaction message
func (n *Node) handleNewTransactionMessage(message P2PMessage) {
	// Parse transaction from message
	var tx Transaction
	err := json.Unmarshal([]byte(message.Data), &tx)
	if err != nil {
		fmt.Printf("Error parsing transaction: %v\n", err)
		return
	}

	// Double-check if we already know this transaction to prevent duplicates
	n.mu.Lock()
	if n.knownTxs[tx.ID] {
		fmt.Printf("Node %s already knows transaction with ID %s, ignoring\n", n.ID, tx.ID)
		n.mu.Unlock()
		return
	}
	n.knownTxs[tx.ID] = true
	n.mu.Unlock()

	fmt.Printf("Node %s received new transaction: %s -> %s: %.2f\n", 
		n.ID, tx.Sender, tx.Recipient, tx.Amount)
	
	// Add to pending transactions
	if n.Blockchain.AddTransaction(&tx) {
		fmt.Printf("Node %s added transaction to pending pool\n", n.ID)
		
		// Forward the transaction to other nodes (but not back to sender)
		n.forwardTransaction(&tx, message.NodeID)
	} else {
		fmt.Printf("Node %s failed to add transaction\n", n.ID)
	}
}

// forwardBlock forwards a block to neighbors except the original sender
func (n *Node) forwardBlock(block *Block, senderID string) {
	// Serialize block
	blockJSON, err := json.Marshal(block)
	if err != nil {
		fmt.Printf("Error serializing block: %v\n", err)
		return
	}
	
	// Create message
	message := P2PMessage{
		Type:   MessageTypeNewBlock,
		Data:   string(blockJSON),
		NodeID: n.ID,
	}
	
	// Forward to all neighbors except the sender
	n.mu.RLock()
	for _, neighbor := range n.Neighbors {
		// Skip the sender to avoid loops
		if neighbor == senderID {
			continue
		}
		go n.sendToNeighbor(neighbor, message)
	}
	n.mu.RUnlock()
}

// forwardTransaction forwards a transaction to neighbors except the original sender
func (n *Node) forwardTransaction(tx *Transaction, senderID string) {
	// Serialize transaction
	txJSON, err := json.Marshal(tx)
	if err != nil {
		fmt.Printf("Error serializing transaction: %v\n", err)
		return
	}
	
	// Create message
	message := P2PMessage{
		Type:   MessageTypeNewTx,
		Data:   string(txJSON),
		NodeID: n.ID,
	}
	
	// Forward to all neighbors except the sender
	n.mu.RLock()
	for _, neighbor := range n.Neighbors {
		// Skip the sender to avoid loops
		if neighbor == senderID {
			continue
		}
		go n.sendToNeighbor(neighbor, message)
	}
	n.mu.RUnlock()
}

// handleRequestChainMessage handles a request for the blockchain
func (n *Node) handleRequestChainMessage(conn net.Conn, message P2PMessage) {
	// Serialize blockchain blocks
	n.Blockchain.mu.RLock()
	blocks := n.Blockchain.Blocks
	n.Blockchain.mu.RUnlock()
	
	blocksJSON, err := json.Marshal(blocks)
	if err != nil {
		fmt.Printf("Error serializing blockchain: %v\n", err)
		return
	}
	
	// Create response message
	response := P2PMessage{
		Type:   MessageTypeResponseChain,
		Data:   string(blocksJSON),
		NodeID: n.ID,
	}
	
	// Send response
	responseJSON, err := json.Marshal(response)
	if err != nil {
		fmt.Printf("Error serializing response: %v\n", err)
		return
	}
	
	_, err = conn.Write(responseJSON)
	if err != nil {
		fmt.Printf("Error sending response: %v\n", err)
	}
}

// handleRequestBlockMessage handles a request for a specific block
func (n *Node) handleRequestBlockMessage(conn net.Conn, message P2PMessage) {
	// Get block hash from message
	blockHash := message.Data
	
	// Find block
	block := n.Blockchain.GetBlockByHash(blockHash)
	if block == nil {
		fmt.Printf("Block with hash %s not found\n", blockHash)
		return
	}
	
	// Serialize block
	blockJSON, err := json.Marshal(block)
	if err != nil {
		fmt.Printf("Error serializing block: %v\n", err)
		return
	}
	
	// Create response message
	response := P2PMessage{
		Type:   MessageTypeResponseBlock,
		Data:   string(blockJSON),
		NodeID: n.ID,
	}
	
	// Send response
	responseJSON, err := json.Marshal(response)
	if err != nil {
		fmt.Printf("Error serializing response: %v\n", err)
		return
	}
	
	_, err = conn.Write(responseJSON)
	if err != nil {
		fmt.Printf("Error sending response: %v\n", err)
	}
}

// handleResponseChainMessage handles a response with a blockchain
func (n *Node) handleResponseChainMessage(message P2PMessage) {
	// Parse blocks from message
	var blocks []*Block
	err := json.Unmarshal([]byte(message.Data), &blocks)
	if err != nil {
		fmt.Printf("Error parsing blocks: %v\n", err)
		return
	}
	
	// Create new blockchain
	newChain := &Blockchain{
		Blocks:     blocks,
		Difficulty: n.Blockchain.Difficulty,
	}
	
	// Replace chain if new one is valid and longer
	if n.Blockchain.ReplaceChain(newChain) {
		fmt.Printf("Node %s replaced blockchain with one from node %s\n", n.ID, message.NodeID)
	}
}

// handleResponseBlockMessage handles a response with a block
func (n *Node) handleResponseBlockMessage(message P2PMessage) {
	// Parse block from message
	var block Block
	err := json.Unmarshal([]byte(message.Data), &block)
	if err != nil {
		fmt.Printf("Error parsing block: %v\n", err)
		return
	}
	
	// Add block to blockchain
	if n.Blockchain.AddBlockToChain(&block) {
		fmt.Printf("Node %s added block with index %d and hash %s from node %s\n", 
			n.ID, block.Index, block.Hash, message.NodeID)
	}
}

// BroadcastBlock broadcasts a block to all neighbors
func (n *Node) BroadcastBlock(block *Block) {
	// Serialize block
	blockJSON, err := json.Marshal(block)
	if err != nil {
		fmt.Printf("Error serializing block: %v\n", err)
		return
	}
	
	// Create message
	message := P2PMessage{
		Type:   MessageTypeNewBlock,
		Data:   string(blockJSON),
		NodeID: n.ID,
	}
	
	// Broadcast to neighbors asynchronously
	go n.broadcast(message)
}

// BroadcastTransaction broadcasts a transaction to all neighbors
func (n *Node) BroadcastTransaction(tx *Transaction) {
	// Serialize transaction
	txJSON, err := json.Marshal(tx)
	if err != nil {
		fmt.Printf("Error serializing transaction: %v\n", err)
		return
	}
	
	// Create message
	message := P2PMessage{
		Type:   MessageTypeNewTx,
		Data:   string(txJSON),
		NodeID: n.ID,
	}
	
	// Broadcast to neighbors asynchronously 
	go n.broadcast(message)
}

// RequestChain requests the blockchain from a neighbor
func (n *Node) RequestChain(neighbor string) {
	// Create message
	message := P2PMessage{
		Type:   MessageTypeRequestChain,
		Data:   "",
		NodeID: n.ID,
	}
	
	// Send to neighbor
	n.sendToNeighbor(neighbor, message)
}

// RequestBlock requests a specific block from a neighbor
func (n *Node) RequestBlock(neighbor string, blockHash string) {
	// Create message
	message := P2PMessage{
		Type:   MessageTypeRequestBlock,
		Data:   blockHash,
		NodeID: n.ID,
	}
	
	// Send to neighbor
	n.sendToNeighbor(neighbor, message)
}

// AddPeer adds a new peer to the node's neighbors
func (n *Node) AddPeer(address string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	
	// Check if peer already exists
	for _, peer := range n.Neighbors {
		if peer == address {
			return
		}
	}
	
	// Add new peer
	n.Neighbors = append(n.Neighbors, address)
	fmt.Printf("Node %s added peer: %s\n", n.ID, address)
	
	// Try to connect to the peer
	go n.sendToNeighbor(address, P2PMessage{
		Type: MessageTypeRequestChain,
		Data: "",
		NodeID: n.ID,
	})
}

// broadcast sends a message to all neighbors
func (n *Node) broadcast(message P2PMessage) {
	for _, neighbor := range n.Neighbors {
		go n.sendToNeighbor(neighbor, message)
	}
}

// sendToNeighbor sends a message to a neighbor
func (n *Node) sendToNeighbor(neighbor string, message P2PMessage) {
	// Serialize message
	messageJSON, err := json.Marshal(message)
	if err != nil {
		fmt.Printf("Error serializing message: %v\n", err)
		return
	}
	
	// Connect to neighbor with timeout
	conn, err := net.DialTimeout("tcp", neighbor, 2*time.Second)
	if err != nil {
		// Not a critical error, neighbor might be offline
		fmt.Printf("Could not connect to %s: %v\n", neighbor, err)
		return
	}
	defer conn.Close()
	
	// Set write deadline
	conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	
	// Send message
	_, err = conn.Write(messageJSON)
	if err != nil {
		fmt.Printf("Error sending message to %s: %v\n", neighbor, err)
	} else {
		fmt.Printf("Message sent to %s\n", neighbor)
	}
}

// JoinNetwork connects to the network and synchronizes the blockchain
func (n *Node) JoinNetwork() {
	for _, neighbor := range n.Neighbors {
		n.RequestChain(neighbor)
	}
}

// Mine mines a new block with pending transactions
func (n *Node) Mine() *Block {
	// Mine new block
	fmt.Printf("Node %s is mining a new block...\n", n.ID)
	newBlock := n.Blockchain.MinePendingTransactions(n.Wallet.Address)
	fmt.Printf("Node %s mined block with hash %s\n", n.ID, newBlock.Hash)
	
	// Add to our known blocks to avoid re-processing when it comes back
	n.mu.Lock()
	n.knownBlocks[newBlock.Hash] = true
	n.mu.Unlock()
	
	// Broadcast immediately to all peers
	go func() {
		// Create message and broadcast
		blockJSON, err := json.Marshal(newBlock)
		if err != nil {
			fmt.Printf("Error serializing block: %v\n", err)
			return
		}
		
		message := P2PMessage{
			Type:   MessageTypeNewBlock,
			Data:   string(blockJSON),
			NodeID: n.ID,
		}
		
		// Broadcast to all neighbors
		for _, neighbor := range n.Neighbors {
			go n.sendToNeighbor(neighbor, message)
		}
	}()
	
	return newBlock
}

// CreateTransaction creates a transaction from this node's wallet
func (n *Node) CreateTransaction(recipient string, amount float64) (*Transaction, error) {
	// Create transaction
	tx, err := n.Wallet.CreateTransaction(recipient, amount)
	if err != nil {
		return nil, err
	}
	
	fmt.Printf("Node %s created transaction: %s -> %s: %.2f\n", n.ID, n.Wallet.Address, recipient, amount)
	
	// Add to blockchain's pending transactions pool (shared between nodes)
	success := n.Blockchain.AddTransaction(tx)
	if !success {
		return nil, fmt.Errorf("failed to add transaction to blockchain")
	}
	
	// Immediately broadcast to all peers in background
	go func() {
		// Mark as known to avoid processing our own transaction when it comes back
		n.mu.Lock()
		n.knownTxs[tx.ID] = true
		n.mu.Unlock()
		
		// Create message and broadcast
		txJSON, err := json.Marshal(tx)
		if err != nil {
			fmt.Printf("Error serializing transaction: %v\n", err)
			return
		}
		
		message := P2PMessage{
			Type:   MessageTypeNewTx,
			Data:   string(txJSON),
			NodeID: n.ID,
		}
		
		// Broadcast to all neighbors
		for _, neighbor := range n.Neighbors {
			go n.sendToNeighbor(neighbor, message)
		}
	}()
	
	return tx, nil
}

// Close closes the node server
func (n *Node) Close() {
	if n.server != nil {
		n.server.Close()
	}
}
