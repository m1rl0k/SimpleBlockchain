package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	Difficulty       = 4
	BlockchainLength = 10
)

type Block struct {
	Index     int
	Timestamp int64
	Data      string
	Hash      string
	PrevHash  string
	Nonce     int
}

func NewBlock(index int, data string, prevHash string) *Block {
	block := &Block{
		Index:     index,
		Timestamp: time.Now().UnixNano(),
		Data:      data,
		PrevHash:  prevHash,
	}

	block.Hash = block.CalculateHash()
	return block
}

func (b *Block) CalculateHash() string {
	record := strconv.Itoa(b.Index) + strconv.FormatInt(b.Timestamp, 10) + b.Data + b.PrevHash + strconv.Itoa(b.Nonce)
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

func (b *Block) MineBlock() {
	target := strings.Repeat("0", Difficulty)
	for {
		if strings.HasPrefix(b.Hash, target) {
			return
		}
		b.Nonce++
		b.Hash = b.CalculateHash()
	}
}

type Blockchain struct {
	Blocks []*Block
	mu     sync.Mutex
}

func NewBlockchain() *Blockchain {
	blockchain := &Blockchain{
		Blocks: []*Block{NewBlock(0, "Genesis Block", "")},
	}
	for i := 1; i < BlockchainLength; i++ {
		blockchain.AddBlock("Block " + strconv.Itoa(i))
	}
	return blockchain
}

func (bc *Blockchain) AddBlock(data string) {
	prevBlock := bc.Blocks[len(bc.Blocks)-1]
	newBlock := NewBlock(prevBlock.Index+1, data, prevBlock.Hash)
	newBlock.MineBlock()
	bc.Blocks = append(bc.Blocks, newBlock)
}

func (bc *Blockchain) LatestBlock() *Block {
	return bc.Blocks[len(bc.Blocks)-1]
}

type Node struct {
	ID          string
	Blockchain  *Blockchain
	Neighbors   []string
	server      net.Listener
}

func NewNode(id string, neighbors []string) *Node {
	node := &Node{
		ID:         id,
		Blockchain: NewBlockchain(),
		Neighbors:  neighbors,
	}

	go node.Listen()

	return node
}

func (n *Node) Broadcast(msg string) {
	for _, neighbor := range n.Neighbors {
		go n.SendToNeighbor(neighbor, msg)
	}
}

func (n *Node) SendToNeighbor(neighbor string, msg string) {
	conn, err := net.Dial("tcp", neighbor)
	if err != nil {
		return
	}
	defer conn.Close()

	conn.Write([]byte(msg))
}

func (n *Node) Listen() {
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		for {
			conn, err := n.server.Accept()
			if err != nil {
				return
			}

			go func(conn net.Conn) {
				defer conn.Close()

				buf := make([]byte, 4096)
				n, err := conn.Read(buf)
				if err != nil {
					return
				}

				msg := string(buf[:n])
				if strings.HasPrefix(msg,
