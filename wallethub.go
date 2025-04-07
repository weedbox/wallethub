package wallethub

import (
	"context"
	"time"
)

// TransactionType defines the types of transactions
type TransactionType string

const (
	TransactionTypeCredit TransactionType = "credit"
	TransactionTypeDebit  TransactionType = "debit"
)

// TransactionStatus defines the possible statuses of a transaction
type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusCompleted TransactionStatus = "completed"
	TransactionStatusFailed    TransactionStatus = "failed"
	TransactionStatusCancelled TransactionStatus = "cancelled"
)

// Transaction represents a wallet transaction
type Transaction struct {
	ID           string                 `json:"id"`
	WalletID     string                 `json:"wallet_id"`
	Type         TransactionType        `json:"type"`
	Amount       int64                  `json:"amount"`      // Points amount (positive number)
	Balance      int64                  `json:"balance"`     // Balance after transaction
	Description  string                 `json:"description"` // Brief description of the transaction
	Note         string                 `json:"note"`        // Additional notes or remarks
	Reference    string                 `json:"reference"`   // External reference (order ID, etc.)
	Status       TransactionStatus      `json:"status"`
	Data         map[string]interface{} `json:"data"` // Flexible field for additional data
	CreatedAt    time.Time              `json:"created_at"`
	CompletedAt  time.Time              `json:"completed_at,omitempty"`
	FailedReason string                 `json:"failed_reason,omitempty"`
}

// Wallet represents a point wallet
type Wallet struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Name        string    `json:"name"`                // Custom name for the wallet
	Description string    `json:"description"`         // Detailed description of the wallet
	Reference   string    `json:"reference"`           // External reference for associating with external systems
	Balance     int64     `json:"balance"`             // Current balance
	Primary     bool      `json:"primary"`             // Whether this is the primary/default wallet for the user
	Active      bool      `json:"active"`              // Whether the wallet is active
	Frozen      bool      `json:"frozen"`              // Whether the wallet is frozen
	RiskFlagged bool      `json:"risk_flagged"`        // Whether the wallet is flagged for risk control
	ClosedAt    time.Time `json:"closed_at,omitempty"` // When the wallet was closed, if applicable
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// WalletManager defines the interface for wallet operations
type WalletManager interface {
	// Wallet management
	CreateWallet(ctx context.Context, userID string, name string, description string, reference string) (*Wallet, error)
	GetWallet(ctx context.Context, walletID string) (*Wallet, error)
	GetWalletsByUserID(ctx context.Context, userID string) ([]Wallet, error)
	GetWalletByUserIDAndReference(ctx context.Context, userID string, reference string) (*Wallet, error)
	GetPrimaryWallet(ctx context.Context, userID string) (*Wallet, error)
	SetPrimaryWallet(ctx context.Context, walletID string) error
	UpdateWalletActive(ctx context.Context, walletID string, active bool) error
	UpdateWalletName(ctx context.Context, walletID string, name string) error
	UpdateWalletDescription(ctx context.Context, walletID string, description string) error
	UpdateWalletReference(ctx context.Context, walletID string, reference string) error

	// Transaction operations
	Credit(ctx context.Context, walletID string, amount int64, description string, note string, reference string, data map[string]interface{}) (*Transaction, error)
	Debit(ctx context.Context, walletID string, amount int64, description string, note string, reference string, data map[string]interface{}) (*Transaction, error)
	GetTransaction(ctx context.Context, transactionID string) (*Transaction, error)
	ListTransactions(ctx context.Context, walletID string, limit int, offset int) ([]Transaction, error)
	ListUserTransactions(ctx context.Context, userID string, limit int, offset int) ([]Transaction, error)

	// Advanced operations
	Transfer(ctx context.Context, fromWalletID string, toWalletID string, amount int64, description string, note string, data map[string]interface{}) error
	FreezeWallet(ctx context.Context, walletID string, reason string) error
	UnfreezeWallet(ctx context.Context, walletID string) error

	// Transaction lifecycle
	CancelTransaction(ctx context.Context, transactionID string, reason string) error
	CompleteTransaction(ctx context.Context, transactionID string) error

	// User wallet summary
	GetUserWalletSummary(ctx context.Context, userID string) (int64, error) // Returns total balance for all user wallets

	// Risk management
	FlagWalletRisk(ctx context.Context, walletID string, reason string) error
	ClearWalletRiskFlag(ctx context.Context, walletID string) error
}

// Txn defines transaction operations for wallet data
type Txn interface {
	// Wallet operations
	SaveWallet(wallet *Wallet) error
	FindWallet(walletID string) (*Wallet, error)
	FindWalletsByUserID(userID string) ([]Wallet, error)
	FindWalletByUserIDAndReference(userID string, reference string) (*Wallet, error)
	FindPrimaryWalletByUserID(userID string) (*Wallet, error)
	UpdateWallet(wallet *Wallet) error

	// Transaction operations
	SaveTransaction(transaction *Transaction) error
	FindTransaction(transactionID string) (*Transaction, error)
	FindTransactionsByWalletID(walletID string, limit int, offset int) ([]Transaction, error)
	FindTransactionsByUserID(userID string, limit int, offset int) ([]Transaction, error)
	UpdateTransaction(transaction *Transaction) error

	// Transaction control
	Commit() error
	Rollback() error
}

// WalletStore defines the data access layer interface
type WalletStore interface {
	// Begin a new transaction
	Begin(ctx context.Context) Txn

	// Non-transactional wallet operations
	SaveWallet(ctx context.Context, wallet *Wallet) error
	FindWallet(ctx context.Context, walletID string) (*Wallet, error)
	FindWalletsByUserID(ctx context.Context, userID string) ([]Wallet, error)
	FindWalletByUserIDAndReference(ctx context.Context, userID string, reference string) (*Wallet, error)
	FindPrimaryWalletByUserID(ctx context.Context, userID string) (*Wallet, error)
	UpdateWallet(ctx context.Context, wallet *Wallet) error

	// Non-transactional transaction operations
	SaveTransaction(ctx context.Context, transaction *Transaction) error
	FindTransaction(ctx context.Context, transactionID string) (*Transaction, error)
	FindTransactionsByWalletID(ctx context.Context, walletID string, limit int, offset int) ([]Transaction, error)
	FindTransactionsByUserID(ctx context.Context, userID string, limit int, offset int) ([]Transaction, error)
	UpdateTransaction(ctx context.Context, transaction *Transaction) error
}
