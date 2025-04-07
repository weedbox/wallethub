# WalletHub

WalletHub is a comprehensive wallet management system for handling digital currency or point-based wallets. It provides a robust foundation for creating and managing user wallets, performing various transaction operations, and maintaining transaction history.

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

## Features

- **Wallet Management**: Create, retrieve, update, and manage user wallets
- **Transaction Operations**: Credit, debit, and transfer operations with comprehensive transaction tracking
- **Risk Management**: Wallet freezing, risk flagging, and other security features
- **Transaction Lifecycle**: Support for pending, completed, failed, and cancelled transactions
- **Database Flexibility**: GORM-based implementation with support for various database backends
- **Transactional Integrity**: Full transactional support to ensure data consistency

## Installation

```bash
go get github.com/weedbox/wallethub
```

## Usage

### Basic Wallet Operations

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/weedbox/wallethub"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// Setup database
	db, err := gorm.Open(sqlite.Open("wallet.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Migrate database schema
	err = db.AutoMigrate(&wallethub.WalletModel{}, &wallethub.TransactionModel{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Create wallet store
	store := wallethub.NewGormWalletStore(db, "", "")

	// Create wallet manager
	manager := wallethub.NewWalletManager(wallethub.WithStore(store))

	ctx := context.Background()

	// Create a new wallet
	wallet, err := manager.CreateWallet(ctx, "user123", "Main Wallet", "User's primary wallet", "main-wallet")
	if err != nil {
		log.Fatalf("Failed to create wallet: %v", err)
	}
	fmt.Printf("Created wallet with ID: %s\n", wallet.ID)

	// Add funds to the wallet
	creditTx, err := manager.Credit(ctx, wallet.ID, 1000, "Initial deposit", "Welcome bonus", "deposit-001", nil)
	if err != nil {
		log.Fatalf("Failed to credit wallet: %v", err)
	}
	fmt.Printf("Credited %d points, new balance: %d\n", creditTx.Amount, creditTx.Balance)

	// Debit funds from the wallet
	debitTx, err := manager.Debit(ctx, wallet.ID, 300, "Purchase", "Product XYZ", "order-001", nil)
	if err != nil {
		log.Fatalf("Failed to debit wallet: %v", err)
	}
	fmt.Printf("Debited %d points, new balance: %d\n", debitTx.Amount, debitTx.Balance)

	// Get wallet balance
	updatedWallet, err := manager.GetWallet(ctx, wallet.ID)
	if err != nil {
		log.Fatalf("Failed to get wallet: %v", err)
	}
	fmt.Printf("Current wallet balance: %d\n", updatedWallet.Balance)

	// List recent transactions
	transactions, err := manager.ListTransactions(ctx, wallet.ID, 10, 0)
	if err != nil {
		log.Fatalf("Failed to list transactions: %v", err)
	}
	fmt.Printf("Recent transactions: %d\n", len(transactions))
	for i, tx := range transactions {
		fmt.Printf("  %d. %s: %d points (%s)\n", i+1, tx.Type, tx.Amount, tx.Description)
	}
}
```

### Creating Multiple Wallets and Transferring Between Them

```go
// Create a second wallet for the same user
secondWallet, err := manager.CreateWallet(ctx, "user123", "Savings", "Long-term savings wallet", "savings-wallet")
if err != nil {
    log.Fatalf("Failed to create second wallet: %v", err)
}

// Add some funds to the second wallet
_, err = manager.Credit(ctx, secondWallet.ID, 500, "Initial funding", "Setup", "initial-funding", nil)
if err != nil {
    log.Fatalf("Failed to credit second wallet: %v", err)
}

// Transfer funds between wallets
err = manager.Transfer(ctx, wallet.ID, secondWallet.ID, 200, "Savings transfer", "Monthly savings", nil)
if err != nil {
    log.Fatalf("Failed to transfer funds: %v", err)
}

// Check balances after transfer
mainWallet, _ := manager.GetWallet(ctx, wallet.ID)
savingsWallet, _ := manager.GetWallet(ctx, secondWallet.ID)
fmt.Printf("Main wallet balance: %d\n", mainWallet.Balance)
fmt.Printf("Savings wallet balance: %d\n", savingsWallet.Balance)
```

### Advanced Operations

```go
// Freeze a wallet (e.g., for security concerns)
err = manager.FreezeWallet(ctx, wallet.ID, "Suspicious activity detected")
if err != nil {
    log.Fatalf("Failed to freeze wallet: %v", err)
}

// Get user's total balance across all active, unfrozen wallets
totalBalance, err := manager.GetUserWalletSummary(ctx, "user123")
if err != nil {
    log.Fatalf("Failed to get user wallet summary: %v", err)
}
fmt.Printf("Total user balance: %d\n", totalBalance)

// Unfreeze wallet
err = manager.UnfreezeWallet(ctx, wallet.ID)
if err != nil {
    log.Fatalf("Failed to unfreeze wallet: %v", err)
}
```

## Architecture

WalletHub follows a clean architecture approach with the following key components:

- **Models**: `Wallet` and `Transaction` entities
- **Store Interface**: Data access layer abstraction
- **GORM Implementation**: Concrete implementation of the store interface using GORM
- **Manager**: Business logic for wallet operations
- **Transaction Support**: Database transaction management for atomic operations

## Database Support

WalletHub uses GORM as its ORM layer, which supports multiple database backends:

- SQLite (as shown in examples)
- PostgreSQL
- MySQL
- SQL Server

## Configuration

The library provides several configuration options:

```go
// Custom table names
store := wallethub.NewGormWalletStore(db, "custom_wallets_table", "custom_transactions_table")

// Create wallet manager with custom store
manager := wallethub.NewWalletManager(wallethub.WithStore(store))
```

## License

This project is licensed under the Apache License 2.0 - see the LICENSE file for details.
