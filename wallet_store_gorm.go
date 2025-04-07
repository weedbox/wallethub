package wallethub

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// WalletModel is the GORM model for Wallet entity
type WalletModel struct {
	ID          string    `gorm:"primaryKey;type:varchar(36)"`
	UserID      string    `gorm:"index;type:varchar(36)"`
	Name        string    `gorm:"type:varchar(100)"`
	Description string    `gorm:"type:text"`
	Reference   string    `gorm:"index;type:varchar(100)"`
	Balance     int64     `gorm:"type:bigint"`
	IsPrimary   bool      `gorm:"default:false"`
	Active      bool      `gorm:"default:true"`
	Frozen      bool      `gorm:"default:false"`
	RiskFlagged bool      `gorm:"default:false"`
	ClosedAt    time.Time `gorm:"type:timestamp"`
	CreatedAt   time.Time `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt   time.Time `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
}

// TransactionModel is the GORM model for Transaction entity
type TransactionModel struct {
	ID           string            `gorm:"primaryKey;type:varchar(36)"`
	WalletID     string            `gorm:"index;type:varchar(36)"`
	Type         TransactionType   `gorm:"type:varchar(10);not null"`
	Amount       int64             `gorm:"type:bigint;not null"`
	Balance      int64             `gorm:"type:bigint;not null"`
	Description  string            `gorm:"type:varchar(255)"`
	Note         string            `gorm:"type:text"`
	Reference    string            `gorm:"index;type:varchar(100)"`
	Status       TransactionStatus `gorm:"type:varchar(20);not null"`
	Data         datatypes.JSON    `gorm:"type:json"`
	CreatedAt    time.Time         `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
	CompletedAt  time.Time         `gorm:"type:timestamp"`
	FailedReason string            `gorm:"type:text"`
}

// ToWallet converts a WalletModel to a Wallet entity
func (m *WalletModel) ToWallet() *Wallet {
	return &Wallet{
		ID:          m.ID,
		UserID:      m.UserID,
		Name:        m.Name,
		Description: m.Description,
		Reference:   m.Reference,
		Balance:     m.Balance,
		Primary:     m.IsPrimary,
		Active:      m.Active,
		Frozen:      m.Frozen,
		RiskFlagged: m.RiskFlagged,
		ClosedAt:    m.ClosedAt,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// FromWallet initializes a WalletModel from a Wallet entity
func (m *WalletModel) FromWallet(wallet *Wallet) {
	m.ID = wallet.ID
	m.UserID = wallet.UserID
	m.Name = wallet.Name
	m.Description = wallet.Description
	m.Reference = wallet.Reference
	m.Balance = wallet.Balance
	m.IsPrimary = wallet.Primary
	m.Active = wallet.Active
	m.Frozen = wallet.Frozen
	m.RiskFlagged = wallet.RiskFlagged
	m.ClosedAt = wallet.ClosedAt
	m.CreatedAt = wallet.CreatedAt
	m.UpdatedAt = wallet.UpdatedAt
}

// ToTransaction converts a TransactionModel to a Transaction entity
func (m *TransactionModel) ToTransaction() *Transaction {
	data := make(map[string]interface{})
	if len(m.Data) > 0 {
		// Unmarshal the JSON data into the map
		if err := json.Unmarshal(m.Data, &data); err != nil {
			// If there's an error, just return an empty map
			data = make(map[string]interface{})
		}
	}

	return &Transaction{
		ID:           m.ID,
		WalletID:     m.WalletID,
		Type:         m.Type,
		Amount:       m.Amount,
		Balance:      m.Balance,
		Description:  m.Description,
		Note:         m.Note,
		Reference:    m.Reference,
		Status:       m.Status,
		Data:         data,
		CreatedAt:    m.CreatedAt,
		CompletedAt:  m.CompletedAt,
		FailedReason: m.FailedReason,
	}
}

// FromTransaction initializes a TransactionModel from a Transaction entity
func (m *TransactionModel) FromTransaction(transaction *Transaction) error {
	if transaction.Data != nil {
		// Convert the map to JSON bytes
		jsonBytes, err := json.Marshal(transaction.Data)
		if err != nil {
			return err
		}
		// Set the JSON data
		err = m.Data.UnmarshalJSON(jsonBytes)
		if err != nil {
			return err
		}
	}

	m.ID = transaction.ID
	m.WalletID = transaction.WalletID
	m.Type = transaction.Type
	m.Amount = transaction.Amount
	m.Balance = transaction.Balance
	m.Description = transaction.Description
	m.Note = transaction.Note
	m.Reference = transaction.Reference
	m.Status = transaction.Status
	m.CreatedAt = transaction.CreatedAt
	m.CompletedAt = transaction.CompletedAt
	m.FailedReason = transaction.FailedReason

	return nil
}

// GormWalletStore implements WalletStore interface using GORM
type GormWalletStore struct {
	db               *gorm.DB
	walletTable      string
	transactionTable string
}

// NewGormWalletStore creates a new instance of GormWalletStore with custom table names
func NewGormWalletStore(db *gorm.DB, walletTable, transactionTable string) *GormWalletStore {
	if walletTable == "" {
		walletTable = "wallets"
	}
	if transactionTable == "" {
		transactionTable = "transactions"
	}

	return &GormWalletStore{
		db:               db,
		walletTable:      walletTable,
		transactionTable: transactionTable,
	}
}

// AutoMigrate creates or updates the necessary database tables
func (s *GormWalletStore) AutoMigrate(ctx context.Context) error {
	// Use context with DB
	db := s.db.WithContext(ctx)

	// Create or update the wallet table
	if err := db.Table(s.walletTable).AutoMigrate(&WalletModel{}); err != nil {
		return err
	}

	// Create or update the transaction table
	if err := db.Table(s.transactionTable).AutoMigrate(&TransactionModel{}); err != nil {
		return err
	}

	return nil
}

// GormTxn implements Txn interface using GORM
type GormTxn struct {
	tx               *gorm.DB
	walletTable      string
	transactionTable string
}

// Begin starts a new database transaction
func (s *GormWalletStore) Begin(ctx context.Context) Txn {
	return &GormTxn{
		tx:               s.db.WithContext(ctx).Begin(),
		walletTable:      s.walletTable,
		transactionTable: s.transactionTable,
	}
}

// Commit commits the transaction
func (t *GormTxn) Commit() error {
	return t.tx.Commit().Error
}

// Rollback aborts the transaction
func (t *GormTxn) Rollback() error {
	return t.tx.Rollback().Error
}

// SaveWallet saves a wallet to the database (transactional)
func (t *GormTxn) SaveWallet(wallet *Wallet) error {
	if wallet.CreatedAt.IsZero() {
		wallet.CreatedAt = time.Now()
	}
	wallet.UpdatedAt = time.Now()

	model := &WalletModel{}
	model.FromWallet(wallet)

	err := t.tx.Table(t.walletTable).Create(model).Error
	return err
}

// FindWallet finds a wallet by ID (transactional)
func (t *GormTxn) FindWallet(walletID string) (*Wallet, error) {
	var model WalletModel
	result := t.tx.Table(t.walletTable).Where("id = ?", walletID).First(&model)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return model.ToWallet(), nil
}

// FindWalletsByUserID finds all wallets for a user (transactional)
func (t *GormTxn) FindWalletsByUserID(userID string) ([]Wallet, error) {
	var models []WalletModel
	result := t.tx.Table(t.walletTable).Where("user_id = ?", userID).Find(&models)
	if result.Error != nil {
		return nil, result.Error
	}

	wallets := make([]Wallet, len(models))
	for i, model := range models {
		wallet := model.ToWallet()
		wallets[i] = *wallet
	}
	return wallets, nil
}

// FindWalletByUserIDAndReference finds a wallet by user ID and reference (transactional)
func (t *GormTxn) FindWalletByUserIDAndReference(userID string, reference string) (*Wallet, error) {
	var model WalletModel
	result := t.tx.Table(t.walletTable).Where("user_id = ? AND reference = ?", userID, reference).First(&model)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return model.ToWallet(), nil
}

// FindPrimaryWalletByUserID finds the primary wallet for a user (transactional)
func (t *GormTxn) FindPrimaryWalletByUserID(userID string) (*Wallet, error) {
	var model WalletModel
	result := t.tx.Table(t.walletTable).Where("user_id = ? AND is_primary = ? AND active = ?", userID, true, true).First(&model)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return model.ToWallet(), nil
}

// UpdateWallet updates an existing wallet (transactional)
func (t *GormTxn) UpdateWallet(wallet *Wallet) error {
	wallet.UpdatedAt = time.Now()

	model := &WalletModel{}
	model.FromWallet(wallet)

	return t.tx.Table(t.walletTable).Save(model).Error
}

// SaveTransaction saves a transaction to the database (transactional)
func (t *GormTxn) SaveTransaction(transaction *Transaction) error {
	if transaction.CreatedAt.IsZero() {
		transaction.CreatedAt = time.Now()
	}

	model := &TransactionModel{}
	if err := model.FromTransaction(transaction); err != nil {
		return err
	}

	return t.tx.Table(t.transactionTable).Create(model).Error
}

// FindTransaction finds a transaction by ID (transactional)
func (t *GormTxn) FindTransaction(transactionID string) (*Transaction, error) {
	var model TransactionModel
	result := t.tx.Table(t.transactionTable).Where("id = ?", transactionID).First(&model)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return model.ToTransaction(), nil
}

// FindTransactionsByWalletID finds transactions for a wallet with pagination (transactional)
func (t *GormTxn) FindTransactionsByWalletID(walletID string, limit int, offset int) ([]Transaction, error) {
	var models []TransactionModel
	result := t.tx.Table(t.transactionTable).Where("wallet_id = ?", walletID).Order("created_at DESC").Limit(limit).Offset(offset).Find(&models)
	if result.Error != nil {
		return nil, result.Error
	}

	transactions := make([]Transaction, len(models))
	for i, model := range models {
		transaction := model.ToTransaction()
		transactions[i] = *transaction
	}
	return transactions, nil
}

// FindTransactionsByUserID finds transactions for a user with pagination (transactional)
func (t *GormTxn) FindTransactionsByUserID(userID string, limit int, offset int) ([]Transaction, error) {
	var models []TransactionModel
	result := t.tx.Table(t.transactionTable).
		Joins("JOIN "+t.walletTable+" ON "+t.transactionTable+".wallet_id = "+t.walletTable+".id").
		Where(t.walletTable+".user_id = ?", userID).
		Order(t.transactionTable + ".created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&models)
	if result.Error != nil {
		return nil, result.Error
	}

	transactions := make([]Transaction, len(models))
	for i, model := range models {
		transaction := model.ToTransaction()
		transactions[i] = *transaction
	}
	return transactions, nil
}

// UpdateTransaction updates an existing transaction (transactional)
func (t *GormTxn) UpdateTransaction(transaction *Transaction) error {
	model := &TransactionModel{}
	if err := model.FromTransaction(transaction); err != nil {
		return err
	}

	return t.tx.Table(t.transactionTable).Save(model).Error
}

// SaveWallet saves a wallet to the database (non-transactional)
func (s *GormWalletStore) SaveWallet(ctx context.Context, wallet *Wallet) error {
	if wallet.CreatedAt.IsZero() {
		wallet.CreatedAt = time.Now()
	}
	wallet.UpdatedAt = time.Now()

	model := &WalletModel{}
	model.FromWallet(wallet)

	return s.db.WithContext(ctx).Table(s.walletTable).Create(model).Error
}

// FindWallet finds a wallet by ID (non-transactional)
func (s *GormWalletStore) FindWallet(ctx context.Context, walletID string) (*Wallet, error) {
	var model WalletModel
	result := s.db.WithContext(ctx).Table(s.walletTable).Where("id = ?", walletID).First(&model)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return model.ToWallet(), nil
}

// FindWalletsByUserID finds all wallets for a user (non-transactional)
func (s *GormWalletStore) FindWalletsByUserID(ctx context.Context, userID string) ([]Wallet, error) {
	var models []WalletModel
	result := s.db.WithContext(ctx).Table(s.walletTable).Where("user_id = ?", userID).Find(&models)
	if result.Error != nil {
		return nil, result.Error
	}

	wallets := make([]Wallet, len(models))
	for i, model := range models {
		wallet := model.ToWallet()
		wallets[i] = *wallet
	}
	return wallets, nil
}

// FindWalletByUserIDAndReference finds a wallet by user ID and reference (non-transactional)
func (s *GormWalletStore) FindWalletByUserIDAndReference(ctx context.Context, userID string, reference string) (*Wallet, error) {
	var model WalletModel
	result := s.db.WithContext(ctx).Table(s.walletTable).Where("user_id = ? AND reference = ?", userID, reference).First(&model)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return model.ToWallet(), nil
}

// FindPrimaryWalletByUserID finds the primary wallet for a user (non-transactional)
func (s *GormWalletStore) FindPrimaryWalletByUserID(ctx context.Context, userID string) (*Wallet, error) {
	var model WalletModel
	result := s.db.WithContext(ctx).Table(s.walletTable).Where("user_id = ? AND is_primary = ? AND active = ?", userID, true, true).First(&model)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return model.ToWallet(), nil
}

// UpdateWallet updates an existing wallet (non-transactional)
func (s *GormWalletStore) UpdateWallet(ctx context.Context, wallet *Wallet) error {
	wallet.UpdatedAt = time.Now()

	model := &WalletModel{}
	model.FromWallet(wallet)

	return s.db.WithContext(ctx).Table(s.walletTable).Save(model).Error
}

// SaveTransaction saves a transaction to the database (non-transactional)
func (s *GormWalletStore) SaveTransaction(ctx context.Context, transaction *Transaction) error {
	if transaction.CreatedAt.IsZero() {
		transaction.CreatedAt = time.Now()
	}

	model := &TransactionModel{}
	if err := model.FromTransaction(transaction); err != nil {
		return err
	}

	return s.db.WithContext(ctx).Table(s.transactionTable).Create(model).Error
}

// FindTransaction finds a transaction by ID (non-transactional)
func (s *GormWalletStore) FindTransaction(ctx context.Context, transactionID string) (*Transaction, error) {
	var model TransactionModel
	result := s.db.WithContext(ctx).Table(s.transactionTable).Where("id = ?", transactionID).First(&model)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return model.ToTransaction(), nil
}

// FindTransactionsByWalletID finds transactions for a wallet with pagination (non-transactional)
func (s *GormWalletStore) FindTransactionsByWalletID(ctx context.Context, walletID string, limit int, offset int) ([]Transaction, error) {
	var models []TransactionModel
	result := s.db.WithContext(ctx).Table(s.transactionTable).Where("wallet_id = ?", walletID).Order("created_at DESC").Limit(limit).Offset(offset).Find(&models)
	if result.Error != nil {
		return nil, result.Error
	}

	transactions := make([]Transaction, len(models))
	for i, model := range models {
		transaction := model.ToTransaction()
		transactions[i] = *transaction
	}
	return transactions, nil
}

// FindTransactionsByUserID finds transactions for a user with pagination (non-transactional)
func (s *GormWalletStore) FindTransactionsByUserID(ctx context.Context, userID string, limit int, offset int) ([]Transaction, error) {
	var models []TransactionModel
	result := s.db.WithContext(ctx).Table(s.transactionTable).
		Joins("JOIN "+s.walletTable+" ON "+s.transactionTable+".wallet_id = "+s.walletTable+".id").
		Where(s.walletTable+".user_id = ?", userID).
		Order(s.transactionTable + ".created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&models)
	if result.Error != nil {
		return nil, result.Error
	}

	transactions := make([]Transaction, len(models))
	for i, model := range models {
		transaction := model.ToTransaction()
		transactions[i] = *transaction
	}
	return transactions, nil
}

// UpdateTransaction updates an existing transaction (non-transactional)
func (s *GormWalletStore) UpdateTransaction(ctx context.Context, transaction *Transaction) error {
	model := &TransactionModel{}
	if err := model.FromTransaction(transaction); err != nil {
		return err
	}

	return s.db.WithContext(ctx).Table(s.transactionTable).Save(model).Error
}
