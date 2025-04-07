package wallethub

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestGormWalletStore creates a new GormWalletStore with an in-memory SQLite database for testing
func setupTestGormWalletStore(t *testing.T) *GormWalletStore {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	store := NewGormWalletStore(db, "", "")

	// Migrate tables using store's method
	ctx := context.Background()
	err = store.AutoMigrate(ctx)
	require.NoError(t, err)

	return store
}

// createTestWallet creates a test wallet for use in tests
func createTestWallet() *Wallet {
	return &Wallet{
		ID:          "test-wallet-id",
		UserID:      "test-user-id",
		Name:        "Test Wallet",
		Description: "Test wallet for unit tests",
		Reference:   "test-reference",
		Balance:     1000,
		Primary:     true,
		Active:      true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// createTestTransaction creates a test transaction for use in tests
func createTestTransaction(walletID string) *Transaction {
	return &Transaction{
		ID:          "test-transaction-id",
		WalletID:    walletID,
		Type:        TransactionTypeCredit,
		Amount:      500,
		Balance:     1500,
		Description: "Test transaction",
		Reference:   "test-tx-reference",
		Status:      TransactionStatusCompleted,
		Data: map[string]interface{}{
			"test_key": "test_value",
		},
		CreatedAt:   time.Now(),
		CompletedAt: time.Now(),
	}
}

// TestGormWalletStore_Begin tests the Begin method of GormWalletStore
func TestGormWalletStore_Begin(t *testing.T) {
	store := setupTestGormWalletStore(t)

	ctx := context.Background()
	txn := store.Begin(ctx)

	assert.NotNil(t, txn)
	require.IsType(t, &GormTxn{}, txn)

	gormTxn := txn.(*GormTxn)
	assert.Equal(t, "wallets", gormTxn.walletTable)
	assert.Equal(t, "transactions", gormTxn.transactionTable)

	// Test rollback works
	err := txn.Rollback()
	assert.NoError(t, err)
}

// TestGormTxn_CommitRollback tests the Commit and Rollback methods of GormTxn
func TestGormTxn_CommitRollback(t *testing.T) {
	store := setupTestGormWalletStore(t)

	// Test commit
	ctx := context.Background()
	txn := store.Begin(ctx)

	wallet := createTestWallet()
	err := txn.SaveWallet(wallet)
	require.NoError(t, err)

	err = txn.Commit()
	assert.NoError(t, err)

	// Verify commit worked by querying outside transaction
	foundWallet, err := store.FindWallet(ctx, wallet.ID)
	assert.NoError(t, err)
	assert.NotNil(t, foundWallet)
	assert.Equal(t, wallet.ID, foundWallet.ID)

	// Test rollback
	txn = store.Begin(ctx)

	wallet2 := createTestWallet()
	wallet2.ID = "test-wallet-id-2"
	err = txn.SaveWallet(wallet2)
	require.NoError(t, err)

	err = txn.Rollback()
	assert.NoError(t, err)

	// Verify rollback worked by querying outside transaction
	foundWallet2, err := store.FindWallet(ctx, wallet2.ID)
	assert.NoError(t, err)
	assert.Nil(t, foundWallet2) // Should not be found after rollback
}

// TestGormTxn_SaveWallet tests the SaveWallet method of GormTxn
func TestGormTxn_SaveWallet(t *testing.T) {
	store := setupTestGormWalletStore(t)

	ctx := context.Background()
	txn := store.Begin(ctx)

	wallet := createTestWallet()

	// Test saving wallet
	err := txn.SaveWallet(wallet)
	assert.NoError(t, err)

	// Verify wallet was saved within transaction
	foundWallet, err := txn.FindWallet(wallet.ID)
	assert.NoError(t, err)
	assert.NotNil(t, foundWallet)
	assert.Equal(t, wallet.ID, foundWallet.ID)
	assert.Equal(t, wallet.Balance, foundWallet.Balance)

	err = txn.Commit()
	assert.NoError(t, err)
}

// TestGormTxn_FindWallet tests the FindWallet method of GormTxn
func TestGormTxn_FindWallet(t *testing.T) {
	store := setupTestGormWalletStore(t)

	ctx := context.Background()
	txn := store.Begin(ctx)

	wallet := createTestWallet()
	err := txn.SaveWallet(wallet)
	require.NoError(t, err)

	// Test finding existing wallet
	foundWallet, err := txn.FindWallet(wallet.ID)
	assert.NoError(t, err)
	assert.NotNil(t, foundWallet)
	assert.Equal(t, wallet.ID, foundWallet.ID)
	assert.Equal(t, wallet.UserID, foundWallet.UserID)
	assert.Equal(t, wallet.Name, foundWallet.Name)

	// Test finding non-existent wallet
	notFoundWallet, err := txn.FindWallet("non-existent-id")
	assert.NoError(t, err)
	assert.Nil(t, notFoundWallet)

	err = txn.Commit()
	assert.NoError(t, err)
}

// TestGormTxn_FindWalletsByUserID tests the FindWalletsByUserID method of GormTxn
func TestGormTxn_FindWalletsByUserID(t *testing.T) {
	store := setupTestGormWalletStore(t)

	ctx := context.Background()
	txn := store.Begin(ctx)

	// Create multiple wallets for the same user
	wallet1 := createTestWallet()
	wallet1.ID = "test-wallet-id-1"
	err := txn.SaveWallet(wallet1)
	require.NoError(t, err)

	wallet2 := createTestWallet()
	wallet2.ID = "test-wallet-id-2"
	wallet2.Primary = false
	err = txn.SaveWallet(wallet2)
	require.NoError(t, err)

	// Create wallet for different user
	wallet3 := createTestWallet()
	wallet3.ID = "test-wallet-id-3"
	wallet3.UserID = "different-user-id"
	err = txn.SaveWallet(wallet3)
	require.NoError(t, err)

	// Test finding wallets by user ID
	wallets, err := txn.FindWalletsByUserID(wallet1.UserID)
	assert.NoError(t, err)
	assert.Len(t, wallets, 2)

	// Test finding wallets for user with no wallets
	noWallets, err := txn.FindWalletsByUserID("non-existent-user-id")
	assert.NoError(t, err)
	assert.Len(t, noWallets, 0)

	err = txn.Commit()
	assert.NoError(t, err)
}

// TestGormTxn_FindWalletByUserIDAndReference tests the FindWalletByUserIDAndReference method of GormTxn
func TestGormTxn_FindWalletByUserIDAndReference(t *testing.T) {
	store := setupTestGormWalletStore(t)

	ctx := context.Background()
	txn := store.Begin(ctx)

	wallet := createTestWallet()
	err := txn.SaveWallet(wallet)
	require.NoError(t, err)

	// Test finding wallet by user ID and reference
	foundWallet, err := txn.FindWalletByUserIDAndReference(wallet.UserID, wallet.Reference)
	assert.NoError(t, err)
	assert.NotNil(t, foundWallet)
	assert.Equal(t, wallet.ID, foundWallet.ID)

	// Test with correct user ID but wrong reference
	notFoundWallet, err := txn.FindWalletByUserIDAndReference(wallet.UserID, "wrong-reference")
	assert.NoError(t, err)
	assert.Nil(t, notFoundWallet)

	// Test with correct reference but wrong user ID
	notFoundWallet, err = txn.FindWalletByUserIDAndReference("wrong-user-id", wallet.Reference)
	assert.NoError(t, err)
	assert.Nil(t, notFoundWallet)

	err = txn.Commit()
	assert.NoError(t, err)
}

// TestGormTxn_FindPrimaryWalletByUserID tests the FindPrimaryWalletByUserID method of GormTxn
func TestGormTxn_FindPrimaryWalletByUserID(t *testing.T) {
	store := setupTestGormWalletStore(t)

	ctx := context.Background()
	txn := store.Begin(ctx)

	// Create primary wallet
	primaryWallet := createTestWallet()
	primaryWallet.Primary = true
	err := txn.SaveWallet(primaryWallet)
	require.NoError(t, err)

	// Create non-primary wallet for same user
	nonPrimaryWallet := createTestWallet()
	nonPrimaryWallet.ID = "non-primary-wallet-id"
	nonPrimaryWallet.Primary = false
	err = txn.SaveWallet(nonPrimaryWallet)
	require.NoError(t, err)

	// Test finding primary wallet
	foundWallet, err := txn.FindPrimaryWalletByUserID(primaryWallet.UserID)
	assert.NoError(t, err)
	assert.NotNil(t, foundWallet)
	assert.Equal(t, primaryWallet.ID, foundWallet.ID)
	assert.True(t, foundWallet.Primary)

	// Test finding primary wallet for user without one
	noWallet, err := txn.FindPrimaryWalletByUserID("user-without-primary-wallet")
	assert.NoError(t, err)
	assert.Nil(t, noWallet)

	// Test with inactive primary wallet
	primaryWallet.Active = false
	err = txn.UpdateWallet(primaryWallet)
	require.NoError(t, err)

	inactiveWallet, err := txn.FindPrimaryWalletByUserID(primaryWallet.UserID)
	assert.NoError(t, err)
	assert.Nil(t, inactiveWallet)

	err = txn.Commit()
	assert.NoError(t, err)
}

// TestGormTxn_UpdateWallet tests the UpdateWallet method of GormTxn
func TestGormTxn_UpdateWallet(t *testing.T) {
	store := setupTestGormWalletStore(t)

	ctx := context.Background()
	txn := store.Begin(ctx)

	wallet := createTestWallet()
	err := txn.SaveWallet(wallet)
	require.NoError(t, err)

	// Update wallet properties
	wallet.Name = "Updated Name"
	wallet.Balance = 2000
	wallet.Active = false

	// Test updating wallet
	err = txn.UpdateWallet(wallet)
	assert.NoError(t, err)

	// Verify updates were applied
	updatedWallet, err := txn.FindWallet(wallet.ID)
	assert.NoError(t, err)
	assert.NotNil(t, updatedWallet)
	assert.Equal(t, "Updated Name", updatedWallet.Name)
	assert.Equal(t, int64(2000), updatedWallet.Balance)
	assert.False(t, updatedWallet.Active)

	err = txn.Commit()
	assert.NoError(t, err)
}

// TestGormTxn_SaveTransaction tests the SaveTransaction method of GormTxn
func TestGormTxn_SaveTransaction(t *testing.T) {
	store := setupTestGormWalletStore(t)

	ctx := context.Background()
	txn := store.Begin(ctx)

	wallet := createTestWallet()
	err := txn.SaveWallet(wallet)
	require.NoError(t, err)

	transaction := createTestTransaction(wallet.ID)

	// Test saving transaction
	err = txn.SaveTransaction(transaction)
	assert.NoError(t, err)

	// Verify transaction was saved
	foundTransaction, err := txn.FindTransaction(transaction.ID)
	assert.NoError(t, err)
	assert.NotNil(t, foundTransaction)
	assert.Equal(t, transaction.ID, foundTransaction.ID)
	assert.Equal(t, transaction.WalletID, foundTransaction.WalletID)
	assert.Equal(t, transaction.Amount, foundTransaction.Amount)
	assert.Equal(t, transaction.Type, foundTransaction.Type)

	err = txn.Commit()
	assert.NoError(t, err)
}

// TestGormTxn_FindTransaction tests the FindTransaction method of GormTxn
func TestGormTxn_FindTransaction(t *testing.T) {
	store := setupTestGormWalletStore(t)

	ctx := context.Background()
	txn := store.Begin(ctx)

	wallet := createTestWallet()
	err := txn.SaveWallet(wallet)
	require.NoError(t, err)

	transaction := createTestTransaction(wallet.ID)
	err = txn.SaveTransaction(transaction)
	require.NoError(t, err)

	// Test finding existing transaction
	foundTransaction, err := txn.FindTransaction(transaction.ID)
	assert.NoError(t, err)
	assert.NotNil(t, foundTransaction)
	assert.Equal(t, transaction.ID, foundTransaction.ID)

	// Test finding non-existent transaction
	notFoundTransaction, err := txn.FindTransaction("non-existent-id")
	assert.NoError(t, err)
	assert.Nil(t, notFoundTransaction)

	err = txn.Commit()
	assert.NoError(t, err)
}

// TestGormTxn_FindTransactionsByWalletID tests the FindTransactionsByWalletID method of GormTxn
func TestGormTxn_FindTransactionsByWalletID(t *testing.T) {
	store := setupTestGormWalletStore(t)

	ctx := context.Background()
	txn := store.Begin(ctx)

	wallet := createTestWallet()
	err := txn.SaveWallet(wallet)
	require.NoError(t, err)

	// Create multiple transactions for the wallet
	for i := 0; i < 5; i++ {
		transaction := createTestTransaction(wallet.ID)
		transaction.ID = "tx-id-" + string(rune('1'+i))
		err = txn.SaveTransaction(transaction)
		require.NoError(t, err)
	}

	// Test finding transactions with pagination
	transactions, err := txn.FindTransactionsByWalletID(wallet.ID, 3, 0)
	assert.NoError(t, err)
	assert.Len(t, transactions, 3) // Limit 3, offset 0

	transactions, err = txn.FindTransactionsByWalletID(wallet.ID, 3, 3)
	assert.NoError(t, err)
	assert.Len(t, transactions, 2) // Limit 3, offset 3, only 2 remaining

	// Test with wallet that has no transactions
	emptyWallet := createTestWallet()
	emptyWallet.ID = "empty-wallet-id"
	err = txn.SaveWallet(emptyWallet)
	require.NoError(t, err)

	noTransactions, err := txn.FindTransactionsByWalletID(emptyWallet.ID, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, noTransactions, 0)

	err = txn.Commit()
	assert.NoError(t, err)
}

// TestGormTxn_FindTransactionsByUserID tests the FindTransactionsByUserID method of GormTxn
func TestGormTxn_FindTransactionsByUserID(t *testing.T) {
	store := setupTestGormWalletStore(t)

	ctx := context.Background()
	txn := store.Begin(ctx)

	// Create wallets for two different users
	wallet1 := createTestWallet()
	wallet1.UserID = "user-id-1"
	err := txn.SaveWallet(wallet1)
	require.NoError(t, err)

	wallet2 := createTestWallet()
	wallet2.ID = "wallet-id-2"
	wallet2.UserID = "user-id-2"
	err = txn.SaveWallet(wallet2)
	require.NoError(t, err)

	// Create transactions for first user
	for i := 0; i < 3; i++ {
		transaction := createTestTransaction(wallet1.ID)
		transaction.ID = "tx-user1-" + string(rune('1'+i))
		err = txn.SaveTransaction(transaction)
		require.NoError(t, err)
	}

	// Create transactions for second user
	for i := 0; i < 2; i++ {
		transaction := createTestTransaction(wallet2.ID)
		transaction.ID = "tx-user2-" + string(rune('1'+i))
		err = txn.SaveTransaction(transaction)
		require.NoError(t, err)
	}

	// Test finding transactions for first user
	transactions1, err := txn.FindTransactionsByUserID("user-id-1", 10, 0)
	assert.NoError(t, err)
	assert.Len(t, transactions1, 3)

	// Test finding transactions for second user
	transactions2, err := txn.FindTransactionsByUserID("user-id-2", 10, 0)
	assert.NoError(t, err)
	assert.Len(t, transactions2, 2)

	// Test with pagination
	paginatedTx, err := txn.FindTransactionsByUserID("user-id-1", 2, 0)
	assert.NoError(t, err)
	assert.Len(t, paginatedTx, 2)

	paginatedTx, err = txn.FindTransactionsByUserID("user-id-1", 2, 2)
	assert.NoError(t, err)
	assert.Len(t, paginatedTx, 1)

	err = txn.Commit()
	assert.NoError(t, err)
}

// TestGormTxn_UpdateTransaction tests the UpdateTransaction method of GormTxn
func TestGormTxn_UpdateTransaction(t *testing.T) {
	store := setupTestGormWalletStore(t)

	ctx := context.Background()
	txn := store.Begin(ctx)

	wallet := createTestWallet()
	err := txn.SaveWallet(wallet)
	require.NoError(t, err)

	transaction := createTestTransaction(wallet.ID)
	err = txn.SaveTransaction(transaction)
	require.NoError(t, err)

	// Update transaction properties
	transaction.Status = TransactionStatusFailed
	transaction.FailedReason = "Test failure reason"
	transaction.Description = "Updated description"

	// Test updating transaction
	err = txn.UpdateTransaction(transaction)
	assert.NoError(t, err)

	// Verify updates were applied
	updatedTransaction, err := txn.FindTransaction(transaction.ID)
	assert.NoError(t, err)
	assert.NotNil(t, updatedTransaction)
	assert.Equal(t, TransactionStatusFailed, updatedTransaction.Status)
	assert.Equal(t, "Test failure reason", updatedTransaction.FailedReason)
	assert.Equal(t, "Updated description", updatedTransaction.Description)

	err = txn.Commit()
	assert.NoError(t, err)
}

// TestGormWalletStore_SaveWallet tests the non-transactional SaveWallet method
func TestGormWalletStore_SaveWallet(t *testing.T) {
	store := setupTestGormWalletStore(t)

	ctx := context.Background()
	wallet := createTestWallet()

	// Test saving wallet
	err := store.SaveWallet(ctx, wallet)
	assert.NoError(t, err)

	// Verify wallet was saved
	foundWallet, err := store.FindWallet(ctx, wallet.ID)
	assert.NoError(t, err)
	assert.NotNil(t, foundWallet)
	assert.Equal(t, wallet.ID, foundWallet.ID)
}

// TestGormWalletStore_FindWallet tests the non-transactional FindWallet method
func TestGormWalletStore_FindWallet(t *testing.T) {
	store := setupTestGormWalletStore(t)

	ctx := context.Background()
	wallet := createTestWallet()

	err := store.SaveWallet(ctx, wallet)
	require.NoError(t, err)

	// Test finding existing wallet
	foundWallet, err := store.FindWallet(ctx, wallet.ID)
	assert.NoError(t, err)
	assert.NotNil(t, foundWallet)
	assert.Equal(t, wallet.ID, foundWallet.ID)

	// Test finding non-existent wallet
	notFoundWallet, err := store.FindWallet(ctx, "non-existent-id")
	assert.NoError(t, err)
	assert.Nil(t, notFoundWallet)
}

// TestGormWalletStore_FindWalletsByUserID tests the non-transactional FindWalletsByUserID method
func TestGormWalletStore_FindWalletsByUserID(t *testing.T) {
	store := setupTestGormWalletStore(t)

	ctx := context.Background()

	// Create multiple wallets for the same user
	wallet1 := createTestWallet()
	wallet1.ID = "test-wallet-id-1"
	err := store.SaveWallet(ctx, wallet1)
	require.NoError(t, err)

	wallet2 := createTestWallet()
	wallet2.ID = "test-wallet-id-2"
	wallet2.Primary = false
	err = store.SaveWallet(ctx, wallet2)
	require.NoError(t, err)

	// Create wallet for different user
	wallet3 := createTestWallet()
	wallet3.ID = "test-wallet-id-3"
	wallet3.UserID = "different-user-id"
	err = store.SaveWallet(ctx, wallet3)
	require.NoError(t, err)

	// Test finding wallets by user ID
	wallets, err := store.FindWalletsByUserID(ctx, wallet1.UserID)
	assert.NoError(t, err)
	assert.Len(t, wallets, 2)

	// Test finding wallets for user with no wallets
	noWallets, err := store.FindWalletsByUserID(ctx, "non-existent-user-id")
	assert.NoError(t, err)
	assert.Len(t, noWallets, 0)
}

// TestGormWalletStore_FindWalletByUserIDAndReference tests the non-transactional FindWalletByUserIDAndReference method
func TestGormWalletStore_FindWalletByUserIDAndReference(t *testing.T) {
	store := setupTestGormWalletStore(t)

	ctx := context.Background()
	wallet := createTestWallet()

	err := store.SaveWallet(ctx, wallet)
	require.NoError(t, err)

	// Test finding wallet by user ID and reference
	foundWallet, err := store.FindWalletByUserIDAndReference(ctx, wallet.UserID, wallet.Reference)
	assert.NoError(t, err)
	assert.NotNil(t, foundWallet)
	assert.Equal(t, wallet.ID, foundWallet.ID)

	// Test with correct user ID but wrong reference
	notFoundWallet, err := store.FindWalletByUserIDAndReference(ctx, wallet.UserID, "wrong-reference")
	assert.NoError(t, err)
	assert.Nil(t, notFoundWallet)
}

// TestGormWalletStore_FindPrimaryWalletByUserID tests the non-transactional FindPrimaryWalletByUserID method
func TestGormWalletStore_FindPrimaryWalletByUserID(t *testing.T) {
	store := setupTestGormWalletStore(t)

	ctx := context.Background()

	// Create primary wallet
	primaryWallet := createTestWallet()
	err := store.SaveWallet(ctx, primaryWallet)
	require.NoError(t, err)

	// Create non-primary wallet for same user
	nonPrimaryWallet := createTestWallet()
	nonPrimaryWallet.ID = "non-primary-wallet-id"
	nonPrimaryWallet.Primary = false
	err = store.SaveWallet(ctx, nonPrimaryWallet)
	require.NoError(t, err)

	// Test finding primary wallet
	foundWallet, err := store.FindPrimaryWalletByUserID(ctx, primaryWallet.UserID)
	assert.NoError(t, err)
	assert.NotNil(t, foundWallet)
	assert.Equal(t, primaryWallet.ID, foundWallet.ID)
	assert.True(t, foundWallet.Primary)
}

// TestGormWalletStore_UpdateWallet tests the non-transactional UpdateWallet method
func TestGormWalletStore_UpdateWallet(t *testing.T) {
	store := setupTestGormWalletStore(t)

	ctx := context.Background()
	wallet := createTestWallet()

	err := store.SaveWallet(ctx, wallet)
	require.NoError(t, err)

	// Update wallet properties
	wallet.Name = "Updated Name"
	wallet.Balance = 2000
	wallet.Active = false

	// Test updating wallet
	err = store.UpdateWallet(ctx, wallet)
	assert.NoError(t, err)

	// Verify updates were applied
	updatedWallet, err := store.FindWallet(ctx, wallet.ID)
	assert.NoError(t, err)
	assert.NotNil(t, updatedWallet)
	assert.Equal(t, "Updated Name", updatedWallet.Name)
	assert.Equal(t, int64(2000), updatedWallet.Balance)
	assert.False(t, updatedWallet.Active)
}

// TestGormWalletStore_SaveTransaction tests the non-transactional SaveTransaction method
func TestGormWalletStore_SaveTransaction(t *testing.T) {
	store := setupTestGormWalletStore(t)

	ctx := context.Background()
	wallet := createTestWallet()

	err := store.SaveWallet(ctx, wallet)
	require.NoError(t, err)

	transaction := createTestTransaction(wallet.ID)

	// Test saving transaction
	err = store.SaveTransaction(ctx, transaction)
	assert.NoError(t, err)

	// Verify transaction was saved
	foundTransaction, err := store.FindTransaction(ctx, transaction.ID)
	assert.NoError(t, err)
	assert.NotNil(t, foundTransaction)
	assert.Equal(t, transaction.ID, foundTransaction.ID)
}

// TestGormWalletStore_FindTransaction tests the non-transactional FindTransaction method
func TestGormWalletStore_FindTransaction(t *testing.T) {
	store := setupTestGormWalletStore(t)

	ctx := context.Background()
	wallet := createTestWallet()

	err := store.SaveWallet(ctx, wallet)
	require.NoError(t, err)

	transaction := createTestTransaction(wallet.ID)
	err = store.SaveTransaction(ctx, transaction)
	require.NoError(t, err)

	// Test finding existing transaction
	foundTransaction, err := store.FindTransaction(ctx, transaction.ID)
	assert.NoError(t, err)
	assert.NotNil(t, foundTransaction)
	assert.Equal(t, transaction.ID, foundTransaction.ID)

	// Test finding non-existent transaction
	notFoundTransaction, err := store.FindTransaction(ctx, "non-existent-id")
	assert.NoError(t, err)
	assert.Nil(t, notFoundTransaction)
}

// TestGormWalletStore_FindTransactionsByWalletID tests the non-transactional FindTransactionsByWalletID method
func TestGormWalletStore_FindTransactionsByWalletID(t *testing.T) {
	store := setupTestGormWalletStore(t)

	ctx := context.Background()
	wallet := createTestWallet()

	err := store.SaveWallet(ctx, wallet)
	require.NoError(t, err)

	// Create multiple transactions for the wallet
	for i := 0; i < 5; i++ {
		transaction := createTestTransaction(wallet.ID)
		transaction.ID = "tx-id-" + string(rune('1'+i))
		err = store.SaveTransaction(ctx, transaction)
		require.NoError(t, err)
	}

	// Test finding transactions with pagination
	transactions, err := store.FindTransactionsByWalletID(ctx, wallet.ID, 3, 0)
	assert.NoError(t, err)
	assert.Len(t, transactions, 3) // Limit 3, offset 0

	transactions, err = store.FindTransactionsByWalletID(ctx, wallet.ID, 3, 3)
	assert.NoError(t, err)
	assert.Len(t, transactions, 2) // Limit 3, offset 3, only 2 remaining

	// Test with wallet that has no transactions
	emptyWallet := createTestWallet()
	emptyWallet.ID = "empty-wallet-id"
	err = store.SaveWallet(ctx, emptyWallet)
	require.NoError(t, err)

	noTransactions, err := store.FindTransactionsByWalletID(ctx, emptyWallet.ID, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, noTransactions, 0)
}

// TestGormWalletStore_FindTransactionsByUserID tests the non-transactional FindTransactionsByUserID method
func TestGormWalletStore_FindTransactionsByUserID(t *testing.T) {
	store := setupTestGormWalletStore(t)

	ctx := context.Background()

	// Create wallets for two different users
	wallet1 := createTestWallet()
	wallet1.UserID = "user-id-1"
	err := store.SaveWallet(ctx, wallet1)
	require.NoError(t, err)

	wallet2 := createTestWallet()
	wallet2.ID = "wallet-id-2"
	wallet2.UserID = "user-id-2"
	err = store.SaveWallet(ctx, wallet2)
	require.NoError(t, err)

	// Create transactions for first user
	for i := 0; i < 3; i++ {
		transaction := createTestTransaction(wallet1.ID)
		transaction.ID = "tx-user1-" + string(rune('1'+i))
		err = store.SaveTransaction(ctx, transaction)
		require.NoError(t, err)
	}

	// Create transactions for second user
	for i := 0; i < 2; i++ {
		transaction := createTestTransaction(wallet2.ID)
		transaction.ID = "tx-user2-" + string(rune('1'+i))
		err = store.SaveTransaction(ctx, transaction)
		require.NoError(t, err)
	}

	// Test finding transactions for first user
	transactions1, err := store.FindTransactionsByUserID(ctx, "user-id-1", 10, 0)
	assert.NoError(t, err)
	assert.Len(t, transactions1, 3)

	// Test finding transactions for second user
	transactions2, err := store.FindTransactionsByUserID(ctx, "user-id-2", 10, 0)
	assert.NoError(t, err)
	assert.Len(t, transactions2, 2)

	// Test with pagination
	paginatedTx, err := store.FindTransactionsByUserID(ctx, "user-id-1", 2, 0)
	assert.NoError(t, err)
	assert.Len(t, paginatedTx, 2)

	paginatedTx, err = store.FindTransactionsByUserID(ctx, "user-id-1", 2, 2)
	assert.NoError(t, err)
	assert.Len(t, paginatedTx, 1)
}

// TestGormWalletStore_UpdateTransaction tests the non-transactional UpdateTransaction method
func TestGormWalletStore_UpdateTransaction(t *testing.T) {
	store := setupTestGormWalletStore(t)

	ctx := context.Background()
	wallet := createTestWallet()

	err := store.SaveWallet(ctx, wallet)
	require.NoError(t, err)

	transaction := createTestTransaction(wallet.ID)
	err = store.SaveTransaction(ctx, transaction)
	require.NoError(t, err)

	// Update transaction properties
	transaction.Status = TransactionStatusFailed
	transaction.FailedReason = "Test failure reason"
	transaction.Description = "Updated description"

	// Test updating transaction
	err = store.UpdateTransaction(ctx, transaction)
	assert.NoError(t, err)

	// Verify updates were applied
	updatedTransaction, err := store.FindTransaction(ctx, transaction.ID)
	assert.NoError(t, err)
	assert.NotNil(t, updatedTransaction)
	assert.Equal(t, TransactionStatusFailed, updatedTransaction.Status)
	assert.Equal(t, "Test failure reason", updatedTransaction.FailedReason)
	assert.Equal(t, "Updated description", updatedTransaction.Description)
}
