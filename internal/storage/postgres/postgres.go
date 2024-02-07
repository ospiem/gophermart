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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ospiem/gophermart/internal/models"
	"github.com/ospiem/gophermart/internal/models/status"
	"github.com/rs/zerolog"
)

const retryAttempts = 3
const connPGError = "cannot connect to postgres, will retry in"
const defaultSleepInterval = 500

type DB struct {
	pool *pgxpool.Pool
}

//go:embed migrations/*.sql
var migrationsDir embed.FS

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

	for attempt := 0; attempt < retryAttempts; attempt++ {
		tag, err := db.pool.Exec(ctx,
			`INSERT INTO orders (id, status, username) VALUES ($1, $2, $3)
				ON CONFLICT DO NOTHING`,
			order.ID, order.Status, order.Username,
		)
		if err != nil {
			if !isConnException(err) {
				return fmt.Errorf("cannot insert order: %w", err)
			}
			var sleepTime time.Duration
			sleepTime += defaultSleepInterval * time.Millisecond
			logger.Error().Err(err).Msgf("%s %v", connPGError, sleepTime)
			time.Sleep(sleepTime)
			continue
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

func (db *DB) SelectOrder(ctx context.Context, num string) (models.Order, error) {
	order := models.Order{}
	row := db.pool.QueryRow(ctx,
		`SELECT id, username, status, created_at, COALESCE(accrual, 0) AS accrual FROM orders WHERE id = $1`,
		num)
	if err := row.Scan(&order.ID, &order.Username, &order.Status, &order.CreatedAt, &order.Accrual); err != nil {
		return models.Order{}, fmt.Errorf("cannot select the order: %w", err)
	}
	return order, nil
}

func (db *DB) SelectOrders(ctx context.Context, login string) ([]models.Order, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, status, created_at, COALESCE(accrual, 0) AS accrual
			 FROM orders WHERE username = $1 ORDER BY created_at DESC`, login)
	if err != nil {
		return nil, fmt.Errorf("postgres failed to get orders: %w", err)
	}

	//TODO: paginate it.
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

func (db *DB) SelectUserBalance(ctx context.Context, login string) (models.UserBalance, error) {
	ub := models.UserBalance{}
	row := db.pool.QueryRow(ctx,
		`SELECT COALESCE(balance, 0) as balance, COALESCE(total_withdrawn, 0) as total_withdrawn
			from users where login = $1`, login)
	if err := row.Scan(&ub.Balance, &ub.Withdrawn); err != nil {
		return models.UserBalance{}, fmt.Errorf("cannot select user balance: %w", err)
	}
	return ub, nil
}

func (db *DB) SelectCreds(ctx context.Context, login string) (models.Credentials, error) {
	c := models.Credentials{}
	row := db.pool.QueryRow(ctx,
		`SELECT login, hash_password
			from users where login = $1`, login)
	if err := row.Scan(&c.Login, &c.Pass); err != nil {
		return models.Credentials{}, fmt.Errorf("cannot select user balance: %w", err)
	}
	return c, nil
}

func (db *DB) InsertUser(ctx context.Context, login string, hash string, l zerolog.Logger) error {
	logger := l.With().Str("func", "InsertUser").Logger()

	for attempt := 0; attempt < retryAttempts; attempt++ {
		tag, err := db.pool.Exec(ctx,
			`INSERT INTO users (login, hash_password) VALUES ($1, $2)
				ON CONFLICT DO NOTHING`,
			login, hash,
		)
		if err != nil {
			if !isConnException(err) {
				return fmt.Errorf("cannot insert order: %w", err)
			}
			var sleepTime time.Duration
			sleepTime += defaultSleepInterval * time.Millisecond
			logger.Error().Err(err).Msgf("%s %v", connPGError, sleepTime)
			time.Sleep(sleepTime)
			continue
		}
		rowsAffectedCount := tag.RowsAffected()
		if rowsAffectedCount != 1 {
			return fmt.Errorf("insertUser expected 1 row to be affected, actually affected %d", rowsAffectedCount)
		}
		break
	}
	return nil
}

func (db *DB) InsertWithdraw(ctx context.Context, w models.Withdraw, l zerolog.Logger) error {
	logger := l.With().Str("func", "InsertWithdraw").Logger()
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("cannot start a transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			l.Error().Err(err).Msg("cannot rollback tx")
		}
	}()

	var wID string
	row := tx.QueryRow(ctx,
		`INSERT INTO withdraws (username, withdrawn, order_number) VALUES ($1, $2, $3)
				RETURNING withdraws.id`,
		w.User, w.Sum, w.OrderNumber,
	)
	if err := row.Scan(&wID); err != nil {
		return fmt.Errorf("cannot insert withdraw: %w", err)
	}

	for attempt := 0; attempt < retryAttempts; attempt++ {
		tag, err := tx.Exec(ctx,
			`UPDATE users SET balance = users.balance - $1, 
                 total_withdrawn = COALESCE(users.total_withdrawn, 0) + $1 WHERE login = $2;`,
			w.Sum, w.User,
		)
		if err != nil {
			if !isConnException(err) {
				return fmt.Errorf("cannot update balance: %w", err)
			}
			var sleepTime time.Duration
			sleepTime += defaultSleepInterval * time.Millisecond
			logger.Error().Err(err).Msgf("%s %v", connPGError, sleepTime)
			time.Sleep(sleepTime)
			continue
		}
		rowsAffectedCount := tag.RowsAffected()
		if rowsAffectedCount != 1 {
			return fmt.Errorf("update balance expected 1 row to be affected, actually affected %d", rowsAffectedCount)
		}
		break
	}

	for attempt := 0; attempt < retryAttempts; attempt++ {
		tag, err := tx.Exec(ctx,
			`UPDATE orders SET  withdraw = $1 WHERE id = $2;`,
			wID, w.OrderNumber,
		)
		if err != nil {
			if !isConnException(err) {
				return fmt.Errorf("cannot insert withdraw: %w", err)
			}
			var sleepTime time.Duration
			sleepTime += defaultSleepInterval * time.Millisecond
			logger.Error().Err(err).Msgf("%s %v", connPGError, sleepTime)
			time.Sleep(sleepTime)
			continue
		}
		rowsAffectedCount := tag.RowsAffected()
		if rowsAffectedCount != 1 {
			return fmt.Errorf("insert withdraw expected 1 row to be affected, actually affected %d", rowsAffectedCount)
		}
		break
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("cannot commit transaction: %w", err)
	}
	return nil
}

func (db *DB) SelectWithdraws(ctx context.Context, login string) ([]models.WithdrawResponse, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT order_number, withdrawn,processed_at FROM withdraws where username = $1 
   		ORDER BY processed_at`, login)

	if err != nil {
		return nil, fmt.Errorf("postgres failed to get withdrawls: %w", err)
	}

	withdrawls := make([]models.WithdrawResponse, 0, rows.CommandTag().RowsAffected())
	for rows.Next() {
		wr := models.WithdrawResponse{}
		if err := rows.Scan(&wr.Order, &wr.Sum, &wr.ProcessedAt); err != nil {
			return nil, fmt.Errorf("cannot select the withdrawl: %w", err)
		}
		withdrawls = append(withdrawls, wr)
	}
	return withdrawls, nil
}

func (db *DB) SelectOrdersToProceed(ctx context.Context, pagination int, offset *int) ([]models.Order, error) {
	var totalRows int
	err := db.pool.QueryRow(ctx, `SELECT count(*) FROM orders WHERE status NOT IN ($1, $2)`,
		status.PROCESSED, status.INVALID).Scan(&totalRows)
	if err != nil {
		return nil, fmt.Errorf("failed to get total amount of rows: %w", err)
	}
	if *offset >= totalRows {
		*offset = 0
	}

	rows, err := db.pool.Query(ctx,
		`SELECT id, status, created_at, COALESCE(accrual, 0) as accrual, username FROM orders 
            WHERE status NOT IN ($1, $2) ORDER BY created_at LIMIT $3 OFFSET $4`,
		status.PROCESSED, status.INVALID, pagination, *offset,
	)
	if err != nil {
		return nil, fmt.Errorf("cannot get orders from db: %w", err)
	}

	orders := make([]models.Order, 0, rows.CommandTag().RowsAffected())
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(&o.ID, &o.Status, &o.CreatedAt, &o.Accrual, &o.Username); err != nil {
			return nil, fmt.Errorf("cannot scan order: %w", err)
		}
		orders = append(orders, o)
	}

	return orders, nil
}

func (db *DB) ProcessOrderWithBonuses(ctx context.Context, order models.Order, l *zerolog.Logger) error {
	logger := l.With().Str("func", "UpdateOrders").Logger()
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("cannot start a transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			logger.Error().Err(err).Msg("cannot rollback tx")
		}
	}()
	if order.Status != status.PROCESSED {
		err := updateWithRetry(ctx, tx, `UPDATE orders set status = $1 where id = $2`,
			order.Status, order.ID)
		if err != nil {
			return fmt.Errorf("cannot update status: %w", err)
		}

		return nil
	}

	err = updateWithRetry(ctx, tx, `UPDATE orders SET status = $1, accrual = $2 where id = $3`,
		order.Status, order.Accrual, order.ID)
	if err != nil {
		return fmt.Errorf("cannot update status and accrual: %w", err)
	}

	err = updateWithRetry(ctx, tx, `UPDATE users SET balance = users.balance + $1`, order.Accrual)
	if err != nil {
		return fmt.Errorf("cannot update user's balance: %w", err)
	}

	return nil
}

func updateWithRetry(ctx context.Context, tx pgx.Tx, query string, args ...interface{}) error {
	for attempt := 0; attempt < retryAttempts; attempt++ {
		tag, err := tx.Exec(ctx, query, args...)
		if err != nil {
			if !isConnException(err) {
				return fmt.Errorf("failed to execute query: %w", err)
			}
			var sleepTime time.Duration
			sleepTime += defaultSleepInterval * time.Millisecond
			time.Sleep(sleepTime)
			continue
		}

		rowsAffectedCount := tag.RowsAffected()
		if rowsAffectedCount != 1 {
			return fmt.Errorf("expected 1 row affected, got %d", rowsAffectedCount)
		}
		return nil
	}

	return fmt.Errorf("reached maximum retry attempts")
}
