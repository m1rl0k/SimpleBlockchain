package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

func (bc *Blockchain) IsValid() bool {
    for i := 1; i < len(bc.Blocks); i++ {
        currentBlock := bc.Blocks[i]
        prevBlock := bc.Blocks[i-1]

        if currentBlock.Hash != currentBlock.CalculateHash() {
            return false
        }

        if currentBlock.PrevHash != prevBlock.Hash {
            return false
        }
    }
    return true
}

func (bc *Blockchain) AddBlockToChain(newBlock *Block) bool {
    bc.mu.Lock()
    defer bc.mu.Unlock()

    if newBlock.PrevHash != bc.LatestBlock().Hash {
        return false
    }

    if !strings.HasPrefix(newBlock.Hash, strings.Repeat("0", Difficulty)) {
        return false
    }

    bc.Blocks = append(bc.Blocks, newBlock)
    return true
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

func (bc *Blockchain) Print() {
	for _, block := range bc.Blocks {
		fmt.Printf("Index: %d, Timestamp: %d, Data: %s, Hash: %s, PrevHash: %s, Nonce: %d\n", block.Index, block.Timestamp, block.Data, block.Hash, block.PrevHash, block.Nonce)
	}
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
                numBytes, err := conn.Read(buf)
                if err != nil {
                    return
                }

                msg := string(buf[:numBytes])
                if strings.HasPrefix(msg, "BLOCK") {
                    blockStr := strings.Split(msg, " ")[1]
                    var block Block
                    if err := json.Unmarshal([]byte(blockStr), &block); err != nil {
                        fmt.Println(err)
                        return
                    }
                    if !n.Blockchain.AddBlockToChain(&block) {
                        fmt.Println("Failed to add block to chain!")
                        return
                    }
                    fmt.Printf("Node %s added block with index %d and hash %s\n", n.ID, block.Index, block.Hash)
                    n.Broadcast(msg)
                } else if strings.HasPrefix(msg, "REQUEST") {
                    n.sendChainToNeighbor(conn)
                }
            }(conn)
        }
    }()

    wg.Wait()
}


func (n *Node) sendChainToNeighbor(conn net.Conn) {
	n.Blockchain.mu.Lock()
	defer n.Blockchain.mu.Unlock()

	for _, block := range n.Blockchain.Blocks {
		blockBytes, err := json.Marshal(block)
		if err != nil {
			return
		}
		conn.Write(blockBytes)
	}
}

func (n *Node) StartServer() error {
	l, err := net.Listen("tcp", ":"+n.ID)
	if err != nil {
		return err
	}
	n.server = l
	return nil
}

func (n *Node) JoinNetwork() {
	for _, neighbor := range n.Neighbors {
		conn, err := net.Dial("tcp", neighbor)
		if err != nil {
			continue
		}

		// Request blockchain from neighbor
		conn.Write([]byte("REQUEST"))

		// Receive blockchain from neighbor
		buf := make([]byte, 4096)
		var blocks []*Block
		for {
			n, err := conn.Read(buf)
			if err != nil {
				break
			}
			var block Block
			if err := json.Unmarshal(buf[:n], &block); err != nil {
				fmt.Println(err)
				break
			}
			blocks = append(blocks, &block)
		}
		conn.Close()

		// Rebuild blockchain
		newChain := &Blockchain{Blocks: blocks}
		if len(newChain.Blocks) > len(n.Blockchain.Blocks) && newChain.IsValid() {
			n.Blockchain = newChain
		}
	}
}

func main() {
	// create nodes
	node1 := NewNode("3000", []string{"localhost:3001", "localhost:3002"})
	node2 := NewNode("3001", []string{"localhost:3000", "localhost:3002"})
	node3 := NewNode("3002", []string{"localhost:3000", "localhost:3001"})

	// start servers and join network
	node1.StartServer()
	node2.StartServer()
	node3.StartServer()

	node1.JoinNetwork()
	node2.JoinNetwork()
	node3.JoinNetwork()

	// add some blocks
	node1.Blockchain.AddBlock("Block from node1")
		node2.Blockchain.AddBlock("Block from node2")
	node3.Blockchain.AddBlock("Block from node3")

	time.Sleep(5 * time.Second)

	// print blockchain on all nodes
	fmt.Println("Blockchain on node 1:")
	node1.Blockchain.Print()
	fmt.Println("Blockchain on node 2:")
	node2.Blockchain.Print()
	fmt.Println("Blockchain on node 3:")
	node3.Blockchain.Print()
}


