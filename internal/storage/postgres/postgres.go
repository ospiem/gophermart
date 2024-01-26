package postgres

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ospiem/gophermart/internal/models"
	"github.com/rs/zerolog"
)

const retryAttempts = 3
const connPGError = "cannot connect to postgres, will retry in"

type DB struct {
	pool *pgxpool.Pool
}

func NewDB(ctx context.Context, dsn string) (*DB, error) {
	if err := runMigrations(dsn); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	pool, err := initPool(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to inizialize pool: %w", err)
	}

	return &DB{
		pool: pool,
	}, nil
}

func initPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pgConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, pgConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping DB: %w", err)
	}

	return pool, nil
}

//go:embed migrations/*.sql
var migrationsDir embed.FS

func runMigrations(dsn string) error {
	d, err := iofs.New(migrationsDir, "migrations")
	if err != nil {
		return fmt.Errorf("failed to return an iofs driver: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, dsn)
	if err != nil {
		return fmt.Errorf("failed to get new migrate instance: %w", err)
	}

	if err := m.Up(); err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("failed to apply migrations to the DB: %w", err)
		}
	}

	return nil
}

func (db *DB) InsertOrder(ctx context.Context, order models.Order, l zerolog.Logger) error {
	logger := l.With().Str("DB method", "InsertOrder").Logger()
	attempt := 0

	for {
		tag, err := db.pool.Exec(ctx,
			`INSERT INTO orders (id, status, username) VALUES ($1, $2, $3)
				ON CONFLICT DO NOTHING`,
			order.ID, order.Status, order.Username,
		)
		if err != nil {
			if !isConnExp(err) {
				return fmt.Errorf("cannot insert order: %w", err)
			}
			var sleepTime time.Duration
			if attempt < retryAttempts {
				sleepTime += 500 * time.Millisecond
				logger.Error().Err(err).Msgf("%s %v", connPGError, sleepTime)
				attempt++
				time.Sleep(sleepTime)
			}
		}
		rowsAffectedCount := tag.RowsAffected()
		if rowsAffectedCount != 1 {
			return fmt.Errorf("insertOrder expected 1 row to be affected, actually affected %d", rowsAffectedCount)
		}
		break
	}

	return nil
}

func (db *DB) Close() {
	db.pool.Close()
}

func (db *DB) SelectOrder(ctx context.Context, num uint64) (models.Order, error) {
	order := models.Order{}
	row := db.pool.QueryRow(ctx,
		`SELECT id, username, status, created_at, COALESCE(accrual, 0) AS accrual FROM orders WHERE id = $1`,
		num)
	if err := row.Scan(&order.ID, &order.Username, &order.Status, &order.CreatedAt, &order.Accrual); err != nil {
		return models.Order{}, fmt.Errorf("cannot select the order: %w", err)
	}
	return order, nil
}

func (db *DB) SelectOrders(ctx context.Context, user string) ([]models.Order, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, status, created_at, COALESCE(accrual, 0) AS accrual
			 FROM orders WHERE username = $1 ORDER BY created_at DESC`, user)
	if err != nil {
		return nil, fmt.Errorf("postgres failed to get orders: %w", err)
	}

	orders := make([]models.Order, 0, rows.CommandTag().RowsAffected())
	for rows.Next() {
		order := models.Order{}
		if err := rows.Scan(&order.ID, &order.Status, &order.CreatedAt, &order.Accrual); err != nil {
			return nil, fmt.Errorf("cannot select the order: %w", err)
		}
		orders = append(orders, order)
	}
	return orders, nil
}

func (db *DB) SelectUser(ctx context.Context, login string) (models.User, error) {
	user := models.User{}
	row := db.pool.QueryRow(ctx,
		`SELECT login, hash_password, COALESCE(balance, 0) as balance,
			COALESCE(withdrawn, 0) as withdrawn from users where login = $1`, login)
	if err := row.Scan(&user.Login, &user.Pass, &user.Balance, &user.Withdrawn); err != nil {
		return models.User{}, fmt.Errorf("cannot select user: %w", err)
	}
	return user, nil
}

func (db *DB) InsertUser(ctx context.Context, login string, hash string, l zerolog.Logger) error {
	logger := l.With().Str("func", "InsertUser").Logger()
	attempt := 0

	for {
		tag, err := db.pool.Exec(ctx,
			`INSERT INTO users (login, hash_password) VALUES ($1, $2)
				ON CONFLICT DO NOTHING`,
			login, hash,
		)
		if err != nil {
			if !isConnExp(err) {
				return fmt.Errorf("cannot insert order: %w", err)
			}
			var sleepTime time.Duration
			if attempt < retryAttempts {
				sleepTime += 500 * time.Millisecond
				logger.Error().Err(err).Msgf("%s %v", connPGError, sleepTime)
				attempt++
				time.Sleep(sleepTime)
			}
		}
		rowsAffectedCount := tag.RowsAffected()
		if rowsAffectedCount != 1 {
			return fmt.Errorf("insertUser expected 1 row to be affected, actually affected %d", rowsAffectedCount)
		}
		break
	}

	return nil
}
