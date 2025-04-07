package wallethub

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Common error definitions
var (
	ErrWalletNotFound         = errors.New("wallet not found")
	ErrWalletInactive         = errors.New("wallet is not active")
	ErrWalletFrozen           = errors.New("wallet is frozen")
	ErrInsufficientBalance    = errors.New("insufficient balance")
	ErrTransactionNotFound    = errors.New("transaction not found")
	ErrInvalidAmount          = errors.New("amount must be positive")
	ErrPendingTransactionOnly = errors.New("only pending transactions can be modified")
)

// DefaultWalletManager implements the WalletManager interface
type DefaultWalletManager struct {
	store WalletStore
}

// Option defines a functional option pattern for configuring the wallet manager
type Option func(*DefaultWalletManager)

// WithStore sets the wallet store for the wallet manager
func WithStore(store WalletStore) Option {
	return func(m *DefaultWalletManager) {
		m.store = store
	}
}

// NewWalletManager creates a new instance of WalletManager with provided options
func NewWalletManager(options ...Option) *DefaultWalletManager {
	manager := &DefaultWalletManager{}

	for _, option := range options {
		option(manager)
	}

	return manager
}

// CreateWallet creates a new wallet for a user
func (m *DefaultWalletManager) CreateWallet(ctx context.Context, userID string, name string, description string, reference string) (*Wallet, error) {
	// Check if a wallet with the same reference already exists
	existingWallet, err := m.store.FindWalletByUserIDAndReference(ctx, userID, reference)
	if err != nil {
		return nil, err
	}
	if existingWallet != nil {
		return existingWallet, nil
	}

	// Start a transaction
	txn := m.store.Begin(ctx)
	defer txn.Rollback()

	// Check if this is the first wallet for the user (to set as primary)
	wallets, err := txn.FindWalletsByUserID(userID)
	if err != nil {
		return nil, err
	}

	isPrimary := len(wallets) == 0

	// Create the new wallet
	now := time.Now()
	wallet := &Wallet{
		ID:          GenerateID(), // Assuming a helper function exists
		UserID:      userID,
		Name:        name,
		Description: description,
		Reference:   reference,
		Balance:     0,
		Primary:     isPrimary,
		Active:      true,
		Frozen:      false,
		RiskFlagged: false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Save the wallet
	if err := txn.SaveWallet(wallet); err != nil {
		return nil, err
	}

	// Commit the transaction
	if err := txn.Commit(); err != nil {
		return nil, err
	}

	return wallet, nil
}

// GetWallet gets a wallet by ID
func (m *DefaultWalletManager) GetWallet(ctx context.Context, walletID string) (*Wallet, error) {
	return m.store.FindWallet(ctx, walletID)
}

// GetWalletsByUserID gets all wallets for a user
func (m *DefaultWalletManager) GetWalletsByUserID(ctx context.Context, userID string) ([]Wallet, error) {
	return m.store.FindWalletsByUserID(ctx, userID)
}

// GetWalletByUserIDAndReference gets a wallet by user ID and reference
func (m *DefaultWalletManager) GetWalletByUserIDAndReference(ctx context.Context, userID string, reference string) (*Wallet, error) {
	return m.store.FindWalletByUserIDAndReference(ctx, userID, reference)
}

// GetPrimaryWallet gets the primary wallet for a user
func (m *DefaultWalletManager) GetPrimaryWallet(ctx context.Context, userID string) (*Wallet, error) {
	return m.store.FindPrimaryWalletByUserID(ctx, userID)
}

// SetPrimaryWallet sets a wallet as the primary wallet for its user
func (m *DefaultWalletManager) SetPrimaryWallet(ctx context.Context, walletID string) error {
	// Start a transaction
	txn := m.store.Begin(ctx)
	defer txn.Rollback()

	// Get the wallet to be set as primary
	wallet, err := txn.FindWallet(walletID)
	if err != nil {
		return err
	}
	if wallet == nil {
		return ErrWalletNotFound
	}

	// Get the current primary wallet
	currentPrimary, err := txn.FindPrimaryWalletByUserID(wallet.UserID)
	if err != nil {
		return err
	}

	// Update the current primary wallet (if exists)
	if currentPrimary != nil && currentPrimary.ID != walletID {
		currentPrimary.Primary = false
		if err := txn.UpdateWallet(currentPrimary); err != nil {
			return err
		}
	}

	// Set the new wallet as primary
	wallet.Primary = true
	if err := txn.UpdateWallet(wallet); err != nil {
		return err
	}

	// Commit the transaction
	return txn.Commit()
}

// UpdateWalletActive updates the active status of a wallet
func (m *DefaultWalletManager) UpdateWalletActive(ctx context.Context, walletID string, active bool) error {
	// Get the wallet
	wallet, err := m.store.FindWallet(ctx, walletID)
	if err != nil {
		return err
	}
	if wallet == nil {
		return errors.New("wallet not found")
	}

	// Update the active status
	wallet.Active = active
	return m.store.UpdateWallet(ctx, wallet)
}

// UpdateWalletName updates the name of a wallet
func (m *DefaultWalletManager) UpdateWalletName(ctx context.Context, walletID string, name string) error {
	// Get the wallet
	wallet, err := m.store.FindWallet(ctx, walletID)
	if err != nil {
		return err
	}
	if wallet == nil {
		return errors.New("wallet not found")
	}

	// Update the name
	wallet.Name = name
	return m.store.UpdateWallet(ctx, wallet)
}

// UpdateWalletDescription updates the description of a wallet
func (m *DefaultWalletManager) UpdateWalletDescription(ctx context.Context, walletID string, description string) error {
	// Get the wallet
	wallet, err := m.store.FindWallet(ctx, walletID)
	if err != nil {
		return err
	}
	if wallet == nil {
		return errors.New("wallet not found")
	}

	// Update the description
	wallet.Description = description
	return m.store.UpdateWallet(ctx, wallet)
}

// UpdateWalletReference updates the reference of a wallet
func (m *DefaultWalletManager) UpdateWalletReference(ctx context.Context, walletID string, reference string) error {
	// Get the wallet
	wallet, err := m.store.FindWallet(ctx, walletID)
	if err != nil {
		return err
	}
	if wallet == nil {
		return errors.New("wallet not found")
	}

	// Update the reference
	wallet.Reference = reference
	return m.store.UpdateWallet(ctx, wallet)
}

// Credit adds points to a wallet
func (m *DefaultWalletManager) Credit(ctx context.Context, walletID string, amount int64, description string, note string, reference string, data map[string]interface{}) (*Transaction, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	// Start a transaction
	txn := m.store.Begin(ctx)
	defer txn.Rollback()

	// Get the wallet
	wallet, err := txn.FindWallet(walletID)
	if err != nil {
		return nil, err
	}
	if wallet == nil {
		return nil, ErrWalletNotFound
	}
	if !wallet.Active {
		return nil, ErrWalletInactive
	}
	if wallet.Frozen {
		return nil, ErrWalletFrozen
	}

	// Update wallet balance
	newBalance := wallet.Balance + amount
	wallet.Balance = newBalance
	if err := txn.UpdateWallet(wallet); err != nil {
		return nil, err
	}

	// Create the transaction
	now := time.Now()
	transaction := &Transaction{
		ID:          GenerateID(), // Assuming a helper function exists
		WalletID:    walletID,
		Type:        TransactionTypeCredit,
		Amount:      amount,
		Balance:     newBalance,
		Description: description,
		Note:        note,
		Reference:   reference,
		Status:      TransactionStatusCompleted,
		Data:        data,
		CreatedAt:   now,
		CompletedAt: now,
	}

	// Save the transaction
	if err := txn.SaveTransaction(transaction); err != nil {
		return nil, err
	}

	// Commit the transaction
	if err := txn.Commit(); err != nil {
		return nil, err
	}

	return transaction, nil
}

// Debit removes points from a wallet
func (m *DefaultWalletManager) Debit(ctx context.Context, walletID string, amount int64, description string, note string, reference string, data map[string]interface{}) (*Transaction, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	// Start a transaction
	txn := m.store.Begin(ctx)
	defer txn.Rollback()

	// Get the wallet
	wallet, err := txn.FindWallet(walletID)
	if err != nil {
		return nil, err
	}
	if wallet == nil {
		return nil, errors.New("wallet not found")
	}
	if !wallet.Active {
		return nil, errors.New("wallet is not active")
	}
	if wallet.Frozen {
		return nil, errors.New("wallet is frozen")
	}
	if wallet.Balance < amount {
		return nil, ErrInsufficientBalance
	}

	// Update wallet balance
	newBalance := wallet.Balance - amount
	wallet.Balance = newBalance
	if err := txn.UpdateWallet(wallet); err != nil {
		return nil, err
	}

	// Create the transaction
	now := time.Now()
	transaction := &Transaction{
		ID:          GenerateID(), // Assuming a helper function exists
		WalletID:    walletID,
		Type:        TransactionTypeDebit,
		Amount:      amount,
		Balance:     newBalance,
		Description: description,
		Note:        note,
		Reference:   reference,
		Status:      TransactionStatusCompleted,
		Data:        data,
		CreatedAt:   now,
		CompletedAt: now,
	}

	// Save the transaction
	if err := txn.SaveTransaction(transaction); err != nil {
		return nil, err
	}

	// Commit the transaction
	if err := txn.Commit(); err != nil {
		return nil, err
	}

	return transaction, nil
}

// GetTransaction gets a transaction by ID
func (m *DefaultWalletManager) GetTransaction(ctx context.Context, transactionID string) (*Transaction, error) {
	return m.store.FindTransaction(ctx, transactionID)
}

// ListTransactions lists transactions for a wallet with pagination
func (m *DefaultWalletManager) ListTransactions(ctx context.Context, walletID string, limit int, offset int) ([]Transaction, error) {
	return m.store.FindTransactionsByWalletID(ctx, walletID, limit, offset)
}

// ListUserTransactions lists transactions for a user with pagination
func (m *DefaultWalletManager) ListUserTransactions(ctx context.Context, userID string, limit int, offset int) ([]Transaction, error) {
	return m.store.FindTransactionsByUserID(ctx, userID, limit, offset)
}

// Transfer transfers points from one wallet to another
func (m *DefaultWalletManager) Transfer(ctx context.Context, fromWalletID string, toWalletID string, amount int64, description string, note string, data map[string]interface{}) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}

	// Start a transaction
	txn := m.store.Begin(ctx)
	defer txn.Rollback()

	// Get the source wallet
	fromWallet, err := txn.FindWallet(fromWalletID)
	if err != nil {
		return err
	}
	if fromWallet == nil {
		return errors.New("source wallet not found")
	}
	if !fromWallet.Active {
		return ErrWalletInactive
	}
	if fromWallet.Frozen {
		return ErrWalletFrozen
	}
	if fromWallet.Balance < amount {
		return ErrInsufficientBalance
	}

	// Get the destination wallet
	toWallet, err := txn.FindWallet(toWalletID)
	if err != nil {
		return err
	}
	if toWallet == nil {
		return errors.New("destination wallet not found")
	}
	if !toWallet.Active {
		return ErrWalletInactive
	}
	if toWallet.Frozen {
		return ErrWalletFrozen
	}

	// Update source wallet balance
	fromWallet.Balance -= amount
	if err := txn.UpdateWallet(fromWallet); err != nil {
		return err
	}

	// Update destination wallet balance
	toWallet.Balance += amount
	if err := txn.UpdateWallet(toWallet); err != nil {
		return err
	}

	// Create debit transaction for source wallet
	now := time.Now()
	debitTransaction := &Transaction{
		ID:          GenerateID(), // Assuming a helper function exists
		WalletID:    fromWalletID,
		Type:        TransactionTypeDebit,
		Amount:      amount,
		Balance:     fromWallet.Balance,
		Description: description + " (Transfer to " + toWalletID + ")",
		Note:        note,
		Reference:   GenerateID(), // Common reference for linked transactions
		Status:      TransactionStatusCompleted,
		Data:        data,
		CreatedAt:   now,
		CompletedAt: now,
	}

	if err := txn.SaveTransaction(debitTransaction); err != nil {
		return err
	}

	// Create credit transaction for destination wallet
	creditTransaction := &Transaction{
		ID:          GenerateID(), // Assuming a helper function exists
		WalletID:    toWalletID,
		Type:        TransactionTypeCredit,
		Amount:      amount,
		Balance:     toWallet.Balance,
		Description: description + " (Transfer from " + fromWalletID + ")",
		Note:        note,
		Reference:   debitTransaction.Reference, // Same reference for linked transactions
		Status:      TransactionStatusCompleted,
		Data:        data,
		CreatedAt:   now,
		CompletedAt: now,
	}

	if err := txn.SaveTransaction(creditTransaction); err != nil {
		return err
	}

	// Commit the transaction
	return txn.Commit()
}

// FreezeWallet freezes a wallet
func (m *DefaultWalletManager) FreezeWallet(ctx context.Context, walletID string, reason string) error {
	// Get the wallet
	wallet, err := m.store.FindWallet(ctx, walletID)
	if err != nil {
		return err
	}
	if wallet == nil {
		return errors.New("wallet not found")
	}

	// Update the frozen status
	wallet.Frozen = true
	return m.store.UpdateWallet(ctx, wallet)
}

// UnfreezeWallet unfreezes a wallet
func (m *DefaultWalletManager) UnfreezeWallet(ctx context.Context, walletID string) error {
	// Get the wallet
	wallet, err := m.store.FindWallet(ctx, walletID)
	if err != nil {
		return err
	}
	if wallet == nil {
		return errors.New("wallet not found")
	}

	// Update the frozen status
	wallet.Frozen = false
	return m.store.UpdateWallet(ctx, wallet)
}

// CancelTransaction cancels a pending transaction
func (m *DefaultWalletManager) CancelTransaction(ctx context.Context, transactionID string, reason string) error {
	// Start a transaction
	txn := m.store.Begin(ctx)
	defer txn.Rollback()

	// Get the transaction
	transaction, err := txn.FindTransaction(transactionID)
	if err != nil {
		return err
	}
	if transaction == nil {
		return ErrTransactionNotFound
	}
	if transaction.Status != TransactionStatusPending {
		return ErrPendingTransactionOnly
	}

	// Update the transaction status
	transaction.Status = TransactionStatusCancelled
	transaction.FailedReason = reason
	if err := txn.UpdateTransaction(transaction); err != nil {
		return err
	}

	// Commit the transaction
	return txn.Commit()
}

// CompleteTransaction completes a pending transaction
func (m *DefaultWalletManager) CompleteTransaction(ctx context.Context, transactionID string) error {
	// Start a transaction
	txn := m.store.Begin(ctx)
	defer txn.Rollback()

	// Get the transaction
	transaction, err := txn.FindTransaction(transactionID)
	if err != nil {
		return err
	}
	if transaction == nil {
		return ErrTransactionNotFound
	}
	if transaction.Status != TransactionStatusPending {
		return ErrPendingTransactionOnly
	}

	// Get the wallet
	wallet, err := txn.FindWallet(transaction.WalletID)
	if err != nil {
		return err
	}
	if wallet == nil {
		return errors.New("wallet not found")
	}

	// Update the wallet balance based on transaction type
	if transaction.Type == TransactionTypeCredit {
		wallet.Balance += transaction.Amount
	} else if transaction.Type == TransactionTypeDebit {
		if wallet.Balance < transaction.Amount {
			return ErrInsufficientBalance
		}
		wallet.Balance -= transaction.Amount
	}

	// Update the wallet
	if err := txn.UpdateWallet(wallet); err != nil {
		return err
	}

	// Update the transaction
	transaction.Status = TransactionStatusCompleted
	transaction.CompletedAt = time.Now()
	transaction.Balance = wallet.Balance
	if err := txn.UpdateTransaction(transaction); err != nil {
		return err
	}

	// Commit the transaction
	return txn.Commit()
}

// GetUserWalletSummary gets the total balance across all wallets for a user
func (m *DefaultWalletManager) GetUserWalletSummary(ctx context.Context, userID string) (int64, error) {
	// Get all wallets for the user
	wallets, err := m.store.FindWalletsByUserID(ctx, userID)
	if err != nil {
		return 0, err
	}

	// Calculate the total balance
	var totalBalance int64 = 0
	for _, wallet := range wallets {
		if wallet.Active && !wallet.Frozen {
			totalBalance += wallet.Balance
		}
	}

	return totalBalance, nil
}

// FlagWalletRisk flags a wallet for risk
func (m *DefaultWalletManager) FlagWalletRisk(ctx context.Context, walletID string, reason string) error {
	// Get the wallet
	wallet, err := m.store.FindWallet(ctx, walletID)
	if err != nil {
		return err
	}
	if wallet == nil {
		return errors.New("wallet not found")
	}

	// Update the risk flag
	wallet.RiskFlagged = true
	return m.store.UpdateWallet(ctx, wallet)
}

// ClearWalletRiskFlag clears the risk flag from a wallet
func (m *DefaultWalletManager) ClearWalletRiskFlag(ctx context.Context, walletID string) error {
	// Get the wallet
	wallet, err := m.store.FindWallet(ctx, walletID)
	if err != nil {
		return err
	}
	if wallet == nil {
		return errors.New("wallet not found")
	}

	// Clear the risk flag
	wallet.RiskFlagged = false
	return m.store.UpdateWallet(ctx, wallet)
}

// GenerateID generates a unique ID for wallets and transactions using UUID v4
func GenerateID() string {
	return uuid.New().String()
}
