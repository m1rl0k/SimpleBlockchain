# Advanced SimpleBlockchain

An enhanced blockchain implementation with advanced cryptographic features, consensus mechanisms, and P2P networking.

## Features

- **Transaction System**: Uses digital signatures to secure transactions between wallets
- **Merkle Trees**: Efficiently verifies transaction integrity within blocks
- **Dynamic Difficulty Adjustment**: Automatically adjusts mining difficulty to maintain consistent block times
- **Wallet Management**: Create and manage cryptocurrency wallets with public/private key pairs
- **P2P Network**: Fully functional peer-to-peer network for blockchain synchronization
- **Consensus Algorithm**: Ensures all nodes maintain the same blockchain state
- **Mining Rewards**: Miners receive rewards for successfully mining blocks
- **Balance Tracking**: Monitor wallet balances across the blockchain

## Architecture

The project is organized into several components:

- **Block**: Core data structure for storing transactions
- **Blockchain**: Chain of blocks with validation and consensus mechanisms
- **Transaction**: Represents transfers of value between wallets with cryptographic signatures
- **Wallet**: Manages cryptographic identities and signs transactions
- **Node**: Handles network communication and blockchain synchronization

## Getting Started

### Prerequisites

- Go 1.16 or higher

### Running the Demo

```bash
go run *.go
```

This will:
1. Create three nodes with their own wallets
2. Start a local P2P network
3. Mine blocks and process transactions
4. Display the final blockchain state

## API Overview

### Wallet Operations
- Create a new wallet
- Sign transactions
- Check balance

### Blockchain Operations
- Mine blocks
- Add transactions
- Validate blockchain integrity
- Get blockchain status

### Network Operations
- Join a network
- Broadcast transactions and blocks
- Synchronize with peers

## Technical Details

### Consensus

The network uses a Proof of Work (PoW) consensus algorithm with dynamic difficulty adjustment. The difficulty is adjusted every 10 blocks to maintain a target block time of 10 seconds.

### Transaction Verification

Transactions are verified using RSA digital signatures. Each wallet generates a public/private key pair, and transactions are signed with the sender's private key. The network verifies the signature using the sender's public key.

### Merkle Trees

Blocks use Merkle trees to efficiently verify transaction integrity. The Merkle root is included in the block header and is used for validation.

## Security Features

- Cryptographic signatures for all transactions
- Secure wallet management
- Blockchain validation at multiple levels
- Protection against common attack vectors

## Limitations and Future Improvements

- Add support for transaction fees
- Implement more sophisticated consensus algorithms (PoS, DPoS)
- Add smart contract functionality
- Improve network security and scalability
- Add a RESTful API for external integrations
- Create a web-based user interface

## License

This project is licensed under the MIT License - see the LICENSE file for details.
