package infrastructure

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/alakkaya/openscout/internal/domain"
)

func NewDatabase(cfg DatabaseConfig) (*gorm.DB, error) {
    var db *gorm.DB
    var err error

    if strings.TrimSpace(cfg.DSN) != "" {
        db, err = gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{
            Logger: logger.Default.LogMode(logger.Info),
        })
        if err != nil {
            return nil, fmt.Errorf("open postgres: %w", err)
        }
        sqlDB, _ := db.DB()
        sqlDB.SetMaxOpenConns(25)
        sqlDB.SetMaxIdleConns(5)
        sqlDB.SetConnMaxLifetime(5 * time.Minute)
    } else {
        db, err = gorm.Open(sqlite.Open(cfg.Path), &gorm.Config{})
        if err != nil {
            return nil, fmt.Errorf("open sqlite: %w", err)
        }
    }

    if err := db.AutoMigrate(
        &domain.User{},
        &domain.UserPreference{},
        &domain.UserIssueFeedback{},
        &domain.SentNotification{},
        &domain.IssueAnalysis{},
    ); err != nil {
        return nil, fmt.Errorf("automigrate: %w", err)
    }
    return db, nil
}