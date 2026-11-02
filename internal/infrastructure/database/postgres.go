package database

import (
	"context"
	"fmt"
	"time"

	"github.com/DextaAfrica/Backend/internal/config"
	"github.com/DextaAfrica/Backend/internal/domain/crm"
	"github.com/DextaAfrica/Backend/internal/domain/finance"
	"github.com/DextaAfrica/Backend/internal/domain/identity"
	"github.com/DextaAfrica/Backend/internal/domain/outbox"
	"github.com/DextaAfrica/Backend/internal/domain/property"
	"github.com/DextaAfrica/Backend/internal/infrastructure/logger"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type PostgresDB struct {
	DB *gorm.DB
}

// Connect initializes PostgreSQL with connection pooling and executes auto-migrations.
func Connect(cfg *config.DatabaseConfig) (*PostgresDB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode,
	)

	gormConfig := &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConnections)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConnections)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.MaxLifetimeMinutes) * time.Minute)

	// Ping database to verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	logger.Log.Info().
		Str("host", cfg.Host).
		Str("dbname", cfg.Name).
		Msg("PostgreSQL connected successfully with connection pool")

	return &PostgresDB{DB: db}, nil
}

// AutoMigrate runs schema migrations for all domain entities.
func (p *PostgresDB) AutoMigrate() error {
	logger.Log.Info().Msg("Running database schema auto-migrations...")

	err := p.DB.AutoMigrate(
		&identity.User{},
		&identity.RefreshToken{},
		&identity.AuditLog{},
		&property.Development{},
		&property.Unit{},
		&property.MediaAsset{},
		&property.Milestone{},
		&crm.Lead{},
		&crm.Inspection{},
		&finance.Transaction{},
		&finance.Invoice{},
		&outbox.OutboxEvent{},
	)
	if err != nil {
		return fmt.Errorf("auto-migration failed: %w", err)
	}

	logger.Log.Info().Msg("Database auto-migration completed successfully")
	return nil
}

// Ping checks whether database is responsive.
func (p *PostgresDB) Ping(ctx context.Context) error {
	sqlDB, err := p.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

