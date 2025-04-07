package wallethub

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateWallet tests creating a new wallet
func TestCreateWallet(t *testing.T) {
	store := setupTestGormWalletStore(t)
	manager := NewWalletManager(WithStore(store))
	ctx := context.Background()

	// Test creating a wallet
	wallet, err := manager.CreateWallet(ctx, "test-user", "Test Wallet", "Test Description", "test-ref")
	assert.NoError(t, err)
	assert.NotNil(t, wallet)
	assert.Equal(t, "test-user", wallet.UserID)
	assert.Equal(t, "Test Wallet", wallet.Name)
	assert.Equal(t, "Test Description", wallet.Description)
	assert.Equal(t, "test-ref", wallet.Reference)
	assert.Equal(t, int64(0), wallet.Balance)
	assert.True(t, wallet.Primary)
	assert.True(t, wallet.Active)
	assert.False(t, wallet.Frozen)

	// Test idempotency - creating with same reference
	wallet2, err := manager.CreateWallet(ctx, "test-user", "Updated Name", "Updated Description", "test-ref")
	assert.NoError(t, err)
	assert.Equal(t, wallet.ID, wallet2.ID) // Should return the same wallet

	// Test creating a second wallet for same user (should not be primary)
	wallet3, err := manager.CreateWallet(ctx, "test-user", "Second Wallet", "Second Description", "test-ref-2")
	assert.NoError(t, err)
	assert.False(t, wallet3.Primary)
}

// TestGetWallet tests retrieving a wallet by ID
func TestGetWallet(t *testing.T) {
	store := setupTestGormWalletStore(t)
	manager := NewWalletManager(WithStore(store))
	ctx := context.Background()

	// Create a wallet first
	wallet, err := manager.CreateWallet(ctx, "test-user", "Test Wallet", "Test Description", "test-ref")
	require.NoError(t, err)

	// Test getting the wallet
	foundWallet, err := manager.GetWallet(ctx, wallet.ID)
	assert.NoError(t, err)
	assert.NotNil(t, foundWallet)
	assert.Equal(t, wallet.ID, foundWallet.ID)

	// Test getting non-existent wallet
	notFoundWallet, err := manager.GetWallet(ctx, "non-existent-id")
	assert.NoError(t, err)
	assert.Nil(t, notFoundWallet)
}

// TestGetWalletsByUserID tests retrieving all wallets for a user
func TestGetWalletsByUserID(t *testing.T) {
	store := setupTestGormWalletStore(t)
	manager := NewWalletManager(WithStore(store))
	ctx := context.Background()

	// Create multiple wallets for a user
	wallet1, err := manager.CreateWallet(ctx, "test-user", "Wallet 1", "Description 1", "ref-1")
	require.NoError(t, err)

	wallet2, err := manager.CreateWallet(ctx, "test-user", "Wallet 2", "Description 2", "ref-2")
	require.NoError(t, err)

	// Create wallet for different user
	_, err = manager.CreateWallet(ctx, "other-user", "Other Wallet", "Other Description", "other-ref")
	require.NoError(t, err)

	// Test getting wallets for test-user
	wallets, err := manager.GetWalletsByUserID(ctx, "test-user")
	assert.NoError(t, err)
	assert.Len(t, wallets, 2)

	// Verify the wallets returned match the ones we created
	walletIDs := []string{wallets[0].ID, wallets[1].ID}
	assert.Contains(t, walletIDs, wallet1.ID)
	assert.Contains(t, walletIDs, wallet2.ID)

	// Test getting wallets for user with no wallets
	noWallets, err := manager.GetWalletsByUserID(ctx, "non-existent-user")
	assert.NoError(t, err)
	assert.Empty(t, noWallets)
}

// TestGetPrimaryWallet tests retrieving the primary wallet for a user
func TestGetPrimaryWallet(t *testing.T) {
	store := setupTestGormWalletStore(t)
	manager := NewWalletManager(WithStore(store))
	ctx := context.Background()

	// Create wallets for a user (first one should be primary)
	wallet1, err := manager.CreateWallet(ctx, "test-user", "Primary Wallet", "Primary Description", "primary-ref")
	require.NoError(t, err)
	assert.True(t, wallet1.Primary)

	_, err = manager.CreateWallet(ctx, "test-user", "Secondary Wallet", "Secondary Description", "secondary-ref")
	require.NoError(t, err)

	// Test getting primary wallet
	primaryWallet, err := manager.GetPrimaryWallet(ctx, "test-user")
	assert.NoError(t, err)
	assert.NotNil(t, primaryWallet)
	assert.Equal(t, wallet1.ID, primaryWallet.ID)

	// Test getting primary wallet for user with no wallets
	noPrimaryWallet, err := manager.GetPrimaryWallet(ctx, "non-existent-user")
	assert.NoError(t, err)
	assert.Nil(t, noPrimaryWallet)
}

// TestSetPrimaryWallet tests setting a wallet as primary
func TestSetPrimaryWallet(t *testing.T) {
	store := setupTestGormWalletStore(t)
	manager := NewWalletManager(WithStore(store))
	ctx := context.Background()

	// Create multiple wallets for a user
	wallet1, err := manager.CreateWallet(ctx, "test-user", "Wallet 1", "Description 1", "ref-1")
	require.NoError(t, err)
	assert.True(t, wallet1.Primary)

	wallet2, err := manager.CreateWallet(ctx, "test-user", "Wallet 2", "Description 2", "ref-2")
	require.NoError(t, err)
	assert.False(t, wallet2.Primary)

	// Set wallet2 as primary
	err = manager.SetPrimaryWallet(ctx, wallet2.ID)
	assert.NoError(t, err)

	// Verify wallet2 is now primary
	updatedWallet2, err := manager.GetWallet(ctx, wallet2.ID)
	assert.NoError(t, err)
	assert.True(t, updatedWallet2.Primary)

	// Verify wallet1 is no longer primary
	updatedWallet1, err := manager.GetWallet(ctx, wallet1.ID)
	assert.NoError(t, err)
	assert.False(t, updatedWallet1.Primary)

	// Test setting non-existent wallet as primary
	err = manager.SetPrimaryWallet(ctx, "non-existent-id")
	assert.Error(t, err)
	assert.Equal(t, ErrWalletNotFound, err)
}

// TestWalletUpdates tests updating wallet properties
func TestWalletUpdates(t *testing.T) {
	store := setupTestGormWalletStore(t)
	manager := NewWalletManager(WithStore(store))
	ctx := context.Background()

	// Create a wallet
	wallet, err := manager.CreateWallet(ctx, "test-user", "Test Wallet", "Test Description", "test-ref")
	require.NoError(t, err)

	// Test updating name
	err = manager.UpdateWalletName(ctx, wallet.ID, "Updated Name")
	assert.NoError(t, err)

	// Test updating description
	err = manager.UpdateWalletDescription(ctx, wallet.ID, "Updated Description")
	assert.NoError(t, err)

	// Test updating reference
	err = manager.UpdateWalletReference(ctx, wallet.ID, "updated-ref")
	assert.NoError(t, err)

	// Test updating active status
	err = manager.UpdateWalletActive(ctx, wallet.ID, false)
	assert.NoError(t, err)

	// Verify updates
	updatedWallet, err := manager.GetWallet(ctx, wallet.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Name", updatedWallet.Name)
	assert.Equal(t, "Updated Description", updatedWallet.Description)
	assert.Equal(t, "updated-ref", updatedWallet.Reference)
	assert.False(t, updatedWallet.Active)
}

// TestCreditDebit tests credit and debit operations
func TestCreditDebit(t *testing.T) {
	store := setupTestGormWalletStore(t)
	manager := NewWalletManager(WithStore(store))
	ctx := context.Background()

	// Create a wallet
	wallet, err := manager.CreateWallet(ctx, "test-user", "Test Wallet", "Test Description", "test-ref")
	require.NoError(t, err)

	// Test crediting wallet
	creditData := map[string]interface{}{"source": "test-credit"}
	creditTx, err := manager.Credit(ctx, wallet.ID, 1000, "Test Credit", "Credit Note", "credit-ref", creditData)
	assert.NoError(t, err)
	assert.NotNil(t, creditTx)
	assert.Equal(t, wallet.ID, creditTx.WalletID)
	assert.Equal(t, TransactionTypeCredit, creditTx.Type)
	assert.Equal(t, int64(1000), creditTx.Amount)
	assert.Equal(t, int64(1000), creditTx.Balance)
	assert.Equal(t, "Test Credit", creditTx.Description)
	assert.Equal(t, "credit-ref", creditTx.Reference)
	assert.Equal(t, TransactionStatusCompleted, creditTx.Status)
	assert.Equal(t, "test-credit", creditTx.Data["source"])

	// Verify wallet balance updated
	updatedWallet, err := manager.GetWallet(ctx, wallet.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(1000), updatedWallet.Balance)

	// Test debiting wallet
	debitData := map[string]interface{}{"purpose": "test-debit"}
	debitTx, err := manager.Debit(ctx, wallet.ID, 400, "Test Debit", "Debit Note", "debit-ref", debitData)
	assert.NoError(t, err)
	assert.NotNil(t, debitTx)
	assert.Equal(t, wallet.ID, debitTx.WalletID)
	assert.Equal(t, TransactionTypeDebit, debitTx.Type)
	assert.Equal(t, int64(400), debitTx.Amount)
	assert.Equal(t, int64(600), debitTx.Balance)
	assert.Equal(t, TransactionStatusCompleted, debitTx.Status)

	// Verify wallet balance updated
	updatedWallet, err = manager.GetWallet(ctx, wallet.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(600), updatedWallet.Balance)

	// Test insufficient balance
	_, err = manager.Debit(ctx, wallet.ID, 1000, "Invalid Debit", "Insufficient Funds", "invalid-ref", nil)
	assert.Error(t, err)
	assert.Equal(t, ErrInsufficientBalance, err)

	// Test invalid amount
	_, err = manager.Credit(ctx, wallet.ID, -500, "Invalid Credit", "Negative Amount", "invalid-ref", nil)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidAmount, err)

	_, err = manager.Debit(ctx, wallet.ID, 0, "Invalid Debit", "Zero Amount", "invalid-ref", nil)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidAmount, err)
}

// TestTransactionListing tests listing transactions
func TestTransactionListing(t *testing.T) {
	store := setupTestGormWalletStore(t)
	manager := NewWalletManager(WithStore(store))
	ctx := context.Background()

	// Create a wallet
	wallet, err := manager.CreateWallet(ctx, "test-user", "Test Wallet", "Test Description", "test-ref")
	require.NoError(t, err)

	// Create multiple transactions
	for i := 0; i < 5; i++ {
		_, err = manager.Credit(ctx, wallet.ID, 100, "Credit Transaction", "Note", "ref", nil)
		require.NoError(t, err)
	}

	// Test getting transaction by ID
	firstTx, err := manager.ListTransactions(ctx, wallet.ID, 1, 0)
	require.NoError(t, err)
	require.Len(t, firstTx, 1)

	tx, err := manager.GetTransaction(ctx, firstTx[0].ID)
	assert.NoError(t, err)
	assert.NotNil(t, tx)
	assert.Equal(t, firstTx[0].ID, tx.ID)

	// Test listing transactions with pagination
	txs, err := manager.ListTransactions(ctx, wallet.ID, 3, 0)
	assert.NoError(t, err)
	assert.Len(t, txs, 3)

	txs, err = manager.ListTransactions(ctx, wallet.ID, 3, 3)
	assert.NoError(t, err)
	assert.Len(t, txs, 2)

	// Test listing transactions for user
	userTxs, err := manager.ListUserTransactions(ctx, "test-user", 10, 0)
	assert.NoError(t, err)
	assert.Len(t, userTxs, 5)
}

// TestTransfer tests transferring between wallets
func TestTransfer(t *testing.T) {
	store := setupTestGormWalletStore(t)
	manager := NewWalletManager(WithStore(store))
	ctx := context.Background()

	// Create two wallets
	wallet1, err := manager.CreateWallet(ctx, "test-user", "Wallet 1", "Description 1", "ref-1")
	require.NoError(t, err)

	wallet2, err := manager.CreateWallet(ctx, "test-user", "Wallet 2", "Description 2", "ref-2")
	require.NoError(t, err)

	// Add credit to first wallet
	_, err = manager.Credit(ctx, wallet1.ID, 1000, "Initial Credit", "Note", "credit-ref", nil)
	require.NoError(t, err)

	// Test transfer
	err = manager.Transfer(ctx, wallet1.ID, wallet2.ID, 600, "Test Transfer", "Transfer Note", nil)
	assert.NoError(t, err)

	// Verify balances
	updatedWallet1, err := manager.GetWallet(ctx, wallet1.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(400), updatedWallet1.Balance)

	updatedWallet2, err := manager.GetWallet(ctx, wallet2.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(600), updatedWallet2.Balance)

	// Verify transactions were created
	txs1, err := manager.ListTransactions(ctx, wallet1.ID, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, txs1, 2) // Initial credit + transfer debit

	txs2, err := manager.ListTransactions(ctx, wallet2.ID, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, txs2, 1) // Transfer credit

	// Test transfer with insufficient balance
	err = manager.Transfer(ctx, wallet1.ID, wallet2.ID, 500, "Invalid Transfer", "Insufficient Funds", nil)
	assert.Error(t, err)
	assert.Equal(t, ErrInsufficientBalance, err)
}

// TestWalletFreeze tests freezing and unfreezing a wallet
func TestWalletFreeze(t *testing.T) {
	store := setupTestGormWalletStore(t)
	manager := NewWalletManager(WithStore(store))
	ctx := context.Background()

	// Create a wallet and add funds
	wallet, err := manager.CreateWallet(ctx, "test-user", "Test Wallet", "Test Description", "test-ref")
	require.NoError(t, err)

	_, err = manager.Credit(ctx, wallet.ID, 1000, "Initial Credit", "Note", "credit-ref", nil)
	require.NoError(t, err)

	// Freeze the wallet
	err = manager.FreezeWallet(ctx, wallet.ID, "Test freeze")
	assert.NoError(t, err)

	// Verify the wallet is frozen
	frozenWallet, err := manager.GetWallet(ctx, wallet.ID)
	assert.NoError(t, err)
	assert.True(t, frozenWallet.Frozen)

	// Test that operations on frozen wallet fail
	_, err = manager.Credit(ctx, wallet.ID, 500, "Should Fail", "Frozen Wallet", "fail-ref", nil)
	assert.Error(t, err)
	assert.Equal(t, ErrWalletFrozen, err)

	// Unfreeze the wallet
	err = manager.UnfreezeWallet(ctx, wallet.ID)
	assert.NoError(t, err)

	// Verify the wallet is unfrozen
	unfrozenWallet, err := manager.GetWallet(ctx, wallet.ID)
	assert.NoError(t, err)
	assert.False(t, unfrozenWallet.Frozen)

	// Test that operations work again
	_, err = manager.Credit(ctx, wallet.ID, 500, "Should Work", "Unfrozen Wallet", "success-ref", nil)
	assert.NoError(t, err)
}

// TestRiskFlagging tests flagging and clearing risk on a wallet
func TestRiskFlagging(t *testing.T) {
	store := setupTestGormWalletStore(t)
	manager := NewWalletManager(WithStore(store))
	ctx := context.Background()

	// Create a wallet
	wallet, err := manager.CreateWallet(ctx, "test-user", "Test Wallet", "Test Description", "test-ref")
	require.NoError(t, err)
	assert.False(t, wallet.RiskFlagged)

	// Flag wallet for risk
	err = manager.FlagWalletRisk(ctx, wallet.ID, "Suspicious activity")
	assert.NoError(t, err)

	// Verify wallet is flagged
	flaggedWallet, err := manager.GetWallet(ctx, wallet.ID)
	assert.NoError(t, err)
	assert.True(t, flaggedWallet.RiskFlagged)

	// Clear risk flag
	err = manager.ClearWalletRiskFlag(ctx, wallet.ID)
	assert.NoError(t, err)

	// Verify risk flag is cleared
	clearedWallet, err := manager.GetWallet(ctx, wallet.ID)
	assert.NoError(t, err)
	assert.False(t, clearedWallet.RiskFlagged)
}

// TestUserWalletSummary tests getting the total balance for a user
func TestUserWalletSummary(t *testing.T) {
	store := setupTestGormWalletStore(t)
	manager := NewWalletManager(WithStore(store))
	ctx := context.Background()

	// Create multiple wallets for a user with different balances
	wallet1, err := manager.CreateWallet(ctx, "test-user", "Wallet 1", "Description 1", "ref-1")
	require.NoError(t, err)

	wallet2, err := manager.CreateWallet(ctx, "test-user", "Wallet 2", "Description 2", "ref-2")
	require.NoError(t, err)

	// Add funds to wallets
	_, err = manager.Credit(ctx, wallet1.ID, 1000, "Credit 1", "Note", "ref1", nil)
	require.NoError(t, err)

	_, err = manager.Credit(ctx, wallet2.ID, 500, "Credit 2", "Note", "ref2", nil)
	require.NoError(t, err)

	// Test getting total balance
	totalBalance, err := manager.GetUserWalletSummary(ctx, "test-user")
	assert.NoError(t, err)
	assert.Equal(t, int64(1500), totalBalance)

	// Deactivate one wallet and test again
	err = manager.UpdateWalletActive(ctx, wallet1.ID, false)
	require.NoError(t, err)

	totalBalance, err = manager.GetUserWalletSummary(ctx, "test-user")
	assert.NoError(t, err)
	assert.Equal(t, int64(500), totalBalance) // Only the active wallet balance
}

// TestPendingTransactions tests handling of pending transactions
func TestPendingTransactions(t *testing.T) {
	store := setupTestGormWalletStore(t)
	manager := NewWalletManager(WithStore(store))
	ctx := context.Background()

	// Create a wallet
	wallet, err := manager.CreateWallet(ctx, "test-user", "Test Wallet", "Test Description", "test-ref")
	require.NoError(t, err)

	// Create a pending transaction manually
	pendingTx := &Transaction{
		ID:          GenerateID(),
		WalletID:    wallet.ID,
		Type:        TransactionTypeCredit,
		Amount:      1000,
		Balance:     0, // Will be set when completed
		Description: "Pending Credit",
		Reference:   "pending-ref",
		Status:      TransactionStatusPending,
		CreatedAt:   time.Now(),
	}

	err = store.SaveTransaction(ctx, pendingTx)
	require.NoError(t, err)

	// Test completing transaction
	err = manager.CompleteTransaction(ctx, pendingTx.ID)
	assert.NoError(t, err)

	// Verify transaction is completed
	completedTx, err := manager.GetTransaction(ctx, pendingTx.ID)
	assert.NoError(t, err)
	assert.Equal(t, TransactionStatusCompleted, completedTx.Status)
	assert.Equal(t, int64(1000), completedTx.Balance)
	assert.False(t, completedTx.CompletedAt.IsZero())

	// Verify wallet balance updated
	updatedWallet, err := manager.GetWallet(ctx, wallet.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(1000), updatedWallet.Balance)

	// Create another pending transaction
	pendingTx2 := &Transaction{
		ID:          GenerateID(),
		WalletID:    wallet.ID,
		Type:        TransactionTypeCredit,
		Amount:      500,
		Balance:     0,
		Description: "Pending Credit 2",
		Reference:   "pending-ref-2",
		Status:      TransactionStatusPending,
		CreatedAt:   time.Now(),
	}

	err = store.SaveTransaction(ctx, pendingTx2)
	require.NoError(t, err)

	// Test cancelling transaction
	err = manager.CancelTransaction(ctx, pendingTx2.ID, "Test cancellation")
	assert.NoError(t, err)

	// Verify transaction is cancelled
	cancelledTx, err := manager.GetTransaction(ctx, pendingTx2.ID)
	assert.NoError(t, err)
	assert.Equal(t, TransactionStatusCancelled, cancelledTx.Status)
	assert.Equal(t, "Test cancellation", cancelledTx.FailedReason)

	// Verify wallet balance unchanged
	updatedWallet, err = manager.GetWallet(ctx, wallet.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(1000), updatedWallet.Balance) // Still 1000 from first transaction
}
