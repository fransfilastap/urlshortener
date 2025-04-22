package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/fransfilastap/urlshortener/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements URLRepository using PostgreSQL
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL repository with connection retry
func NewPostgresRepository(connString string) (*PostgresRepository, error) {
	// Print connection string for debugging
	fmt.Printf("Using connection string: %s\n", connString)

	// Create the connection pool with a simple configuration
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Set a longer connection timeout
	config.ConnConfig.ConnectTimeout = 10 * time.Second

	// Retry parameters
	maxRetries := 5
	retryDelay := 3 * time.Second
	var pool *pgxpool.Pool

	// Retry loop for connection
	for i := 0; i < maxRetries; i++ {
		// Create the connection pool
		pool, err = pgxpool.NewWithConfig(context.Background(), config)
		if err == nil {
			// Test the connection
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err = pool.Ping(ctx)
			cancel()

			if err == nil {
				fmt.Printf("Successfully connected to database on attempt %d\n", i+1)
				return &PostgresRepository{pool: pool}, nil
			}
			pool.Close()
		}

		fmt.Printf("Failed to connect to database (attempt %d/%d): %v. Retrying in %v...\n",
			i+1, maxRetries, err, retryDelay)
		time.Sleep(retryDelay)
	}

	return nil, fmt.Errorf("failed to connect to database after %d attempts: %w", maxRetries, err)
}

// InitSchema initializes the database schema
func (r *PostgresRepository) InitSchema(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS urls (
			id SERIAL PRIMARY KEY,
			original TEXT NOT NULL,
			short TEXT NOT NULL UNIQUE,
			title TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			expires_at TIMESTAMP,
			clicks BIGINT NOT NULL DEFAULT 0,
			creator_reference TEXT,
			deleted_at TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_urls_short ON urls(short);
		CREATE INDEX IF NOT EXISTS idx_urls_original ON urls(original);

		CREATE TABLE IF NOT EXISTS clicks (
			id SERIAL PRIMARY KEY,
			url_id BIGINT NOT NULL REFERENCES urls(id) ON DELETE CASCADE,
			url_short TEXT NOT NULL,
			ip TEXT NOT NULL,
			location TEXT,
			browser TEXT,
			device TEXT,
			timestamp TIMESTAMP NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_clicks_url_id ON clicks(url_id);
		CREATE INDEX IF NOT EXISTS idx_clicks_url_short ON clicks(url_short);

		CREATE TABLE IF NOT EXISTS url_history (
			id SERIAL PRIMARY KEY,
			url_id BIGINT NOT NULL REFERENCES urls(id) ON DELETE CASCADE,
			url_short TEXT NOT NULL,
			action TEXT NOT NULL,
			old_value JSONB,
			new_value JSONB,
			modified_at TIMESTAMP NOT NULL DEFAULT NOW(),
			modified_by TEXT
		);
		CREATE INDEX IF NOT EXISTS idx_url_history_url_id ON url_history(url_id);
		CREATE INDEX IF NOT EXISTS idx_url_history_url_short ON url_history(url_short);
	`)
	return err
}

// Create stores a new URL
func (r *PostgresRepository) Create(ctx context.Context, url *models.URL) error {
	// Check if short URL already exists
	var exists bool
	err := r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM urls WHERE short = $1 AND deleted_at IS NULL)", url.Short).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return ErrURLExists
	}

	// Insert new URL
	_, err = r.pool.Exec(ctx,
		"INSERT INTO urls (original, short, title, created_at, expires_at, clicks, creator_reference, deleted_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
		url.Original, url.Short, url.Title, url.CreatedAt, url.ExpiresAt, url.Clicks, url.CreatorReference, url.DeletedAt)
	return err
}

// GetByShort retrieves a URL by its short code
func (r *PostgresRepository) GetByShort(ctx context.Context, short string) (*models.URL, error) {
	url := &models.URL{}
	err := r.pool.QueryRow(ctx,
		"SELECT id, original, short, title, created_at, expires_at, clicks, creator_reference, deleted_at FROM urls WHERE short = $1 AND deleted_at IS NULL",
		short).Scan(&url.ID, &url.Original, &url.Short, &url.Title, &url.CreatedAt, &url.ExpiresAt, &url.Clicks, &url.CreatorReference, &url.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrURLNotFound
		}
		return nil, err
	}

	// Check if URL has expired
	if !url.ExpiresAt.IsZero() && url.ExpiresAt.Before(time.Now()) {
		return nil, ErrURLNotFound
	}

	return url, nil
}

// GetByOriginal retrieves a URL by its original URL
func (r *PostgresRepository) GetByOriginal(ctx context.Context, original string) (*models.URL, error) {
	url := &models.URL{}
	err := r.pool.QueryRow(ctx,
		"SELECT id, original, short, title, created_at, expires_at, clicks, creator_reference, deleted_at FROM urls WHERE original = $1 AND deleted_at IS NULL",
		original).Scan(&url.ID, &url.Original, &url.Short, &url.Title, &url.CreatedAt, &url.ExpiresAt, &url.Clicks, &url.CreatorReference, &url.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrURLNotFound
		}
		return nil, err
	}

	// Check if URL has expired
	if !url.ExpiresAt.IsZero() && url.ExpiresAt.Before(time.Now()) {
		return nil, ErrURLNotFound
	}

	return url, nil
}

// GetByCreator retrieves URLs by their creator reference
func (r *PostgresRepository) GetByCreator(ctx context.Context, creatorReference string) ([]*models.URL, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT id, original, short, title, created_at, expires_at, clicks, creator_reference, deleted_at FROM urls WHERE creator_reference = $1 AND deleted_at IS NULL",
		creatorReference)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var urls []*models.URL
	for rows.Next() {
		url := &models.URL{}
		err := rows.Scan(&url.ID, &url.Original, &url.Short, &url.Title, &url.CreatedAt, &url.ExpiresAt, &url.Clicks, &url.CreatorReference, &url.DeletedAt)
		if err != nil {
			return nil, err
		}

		// Skip expired URLs
		if !url.ExpiresAt.IsZero() && url.ExpiresAt.Before(time.Now()) {
			continue
		}

		urls = append(urls, url)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return urls, nil
}

// IncrementClicks increments the click count for a URL
func (r *PostgresRepository) IncrementClicks(ctx context.Context, short string) error {
	_, err := r.pool.Exec(ctx, "UPDATE urls SET clicks = clicks + 1 WHERE short = $1 AND deleted_at IS NULL", short)
	return err
}

// Delete soft deletes a URL by setting its DeletedAt field
func (r *PostgresRepository) Delete(ctx context.Context, short string) error {
	_, err := r.pool.Exec(ctx, "UPDATE urls SET deleted_at = NOW() WHERE short = $1 AND deleted_at IS NULL", short)
	return err
}

// DeleteWithCreator soft deletes a URL if the creator_reference matches
func (r *PostgresRepository) DeleteWithCreator(ctx context.Context, short string, creatorReference string) error {
	// Check if URL exists and belongs to the creator
	existingURL, err := r.GetByShort(ctx, short)
	if err != nil {
		return err
	}

	// Check if the creator_reference matches
	if existingURL.CreatorReference != creatorReference {
		return errors.New("unauthorized: creator reference does not match")
	}

	// Soft delete URL
	_, err = r.pool.Exec(ctx, "UPDATE urls SET deleted_at = NOW() WHERE short = $1 AND creator_reference = $2 AND deleted_at IS NULL", short, creatorReference)
	return err
}

// HardDelete permanently removes a URL from the database
func (r *PostgresRepository) HardDelete(ctx context.Context, short string) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM urls WHERE short = $1", short)
	return err
}

// StoreClick stores click analytics data
func (r *PostgresRepository) StoreClick(ctx context.Context, click *models.Click) error {
	_, err := r.pool.Exec(ctx,
		"INSERT INTO clicks (url_id, url_short, ip, location, browser, device, timestamp) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		click.URLID, click.URLShort, click.IP, click.Location, click.Browser, click.Device, click.Timestamp)
	return err
}

// GetClicksByShort retrieves click analytics data for a URL
func (r *PostgresRepository) GetClicksByShort(ctx context.Context, short string) ([]*models.Click, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT id, url_id, url_short, ip, location, browser, device, timestamp FROM clicks WHERE url_short = $1 ORDER BY timestamp DESC",
		short)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clicks []*models.Click
	for rows.Next() {
		click := &models.Click{}
		err := rows.Scan(&click.ID, &click.URLID, &click.URLShort, &click.IP, &click.Location, &click.Browser, &click.Device, &click.Timestamp)
		if err != nil {
			return nil, err
		}
		clicks = append(clicks, click)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return clicks, nil
}

// HasRecentClick checks if there's a recent click from the same visitor
func (r *PostgresRepository) HasRecentClick(ctx context.Context, short string, ip string, browser string, device string) (bool, error) {
	// Check if there's a click from the same visitor (IP + browser + device) within the last hour
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM clicks 
			WHERE url_short = $1 
			AND ip = $2 
			AND browser = $3 
			AND device = $4 
			AND timestamp > NOW() - INTERVAL '1 hour'
		)
	`, short, ip, browser, device).Scan(&exists)

	if err != nil {
		return false, err
	}

	return exists, nil
}

// UpdateURL updates an existing URL
func (r *PostgresRepository) UpdateURL(ctx context.Context, short string, url *models.URL) error {
	// Check if URL exists
	_, err := r.GetByShort(ctx, short)
	if err != nil {
		return err
	}

	// Update URL
	_, err = r.pool.Exec(ctx,
		"UPDATE urls SET original = $1, title = $2, expires_at = $3 WHERE short = $4 AND deleted_at IS NULL",
		url.Original, url.Title, url.ExpiresAt, short)
	return err
}

// UpdateURLWithCreator updates an existing URL if the creator_reference matches
func (r *PostgresRepository) UpdateURLWithCreator(ctx context.Context, short string, url *models.URL, creatorReference string) error {
	// Check if URL exists and belongs to the creator
	existingURL, err := r.GetByShort(ctx, short)
	if err != nil {
		return err
	}

	// Check if the creator_reference matches
	if existingURL.CreatorReference != creatorReference {
		return errors.New("unauthorized: creator reference does not match")
	}

	// Update URL
	_, err = r.pool.Exec(ctx,
		"UPDATE urls SET original = $1, title = $2, expires_at = $3 WHERE short = $4 AND creator_reference = $5 AND deleted_at IS NULL",
		url.Original, url.Title, url.ExpiresAt, short, creatorReference)
	return err
}

// LogURLHistory logs a URL modification
func (r *PostgresRepository) LogURLHistory(ctx context.Context, urlID int64, short string, action string, oldValue, newValue interface{}, modifiedBy string) error {
	// Convert values to JSON
	oldValueJSON, err := json.Marshal(oldValue)
	if err != nil {
		return err
	}

	newValueJSON, err := json.Marshal(newValue)
	if err != nil {
		return err
	}

	// Insert history record
	_, err = r.pool.Exec(ctx,
		"INSERT INTO url_history (url_id, url_short, action, old_value, new_value, modified_at, modified_by) VALUES ($1, $2, $3, $4, $5, NOW(), $6)",
		urlID, short, action, oldValueJSON, newValueJSON, modifiedBy)
	return err
}

// GetClickAnalytics retrieves aggregated click analytics data for a URL
func (r *PostgresRepository) GetClickAnalytics(ctx context.Context, short string) (map[string]interface{}, error) {
	// Get total clicks
	var totalClicks int64
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM clicks WHERE url_short = $1", short).Scan(&totalClicks)
	if err != nil {
		return nil, err
	}

	// Get clicks by browser
	rows, err := r.pool.Query(ctx, "SELECT browser, COUNT(*) FROM clicks WHERE url_short = $1 GROUP BY browser", short)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	browserStats := make(map[string]int64)
	for rows.Next() {
		var browser string
		var count int64
		if err := rows.Scan(&browser, &count); err != nil {
			return nil, err
		}
		browserStats[browser] = count
	}

	// Get clicks by device
	rows, err = r.pool.Query(ctx, "SELECT device, COUNT(*) FROM clicks WHERE url_short = $1 GROUP BY device", short)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	deviceStats := make(map[string]int64)
	for rows.Next() {
		var device string
		var count int64
		if err := rows.Scan(&device, &count); err != nil {
			return nil, err
		}
		deviceStats[device] = count
	}

	// Get clicks by location
	rows, err = r.pool.Query(ctx, "SELECT location, COUNT(*) FROM clicks WHERE url_short = $1 GROUP BY location", short)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	locationStats := make(map[string]int64)
	for rows.Next() {
		var location string
		var count int64
		if err := rows.Scan(&location, &count); err != nil {
			return nil, err
		}
		locationStats[location] = count
	}

	// Return aggregated data
	return map[string]interface{}{
		"total_clicks": totalClicks,
		"browsers":     browserStats,
		"devices":      deviceStats,
		"locations":    locationStats,
	}, nil
}

// Close closes the database connection
func (r *PostgresRepository) Close() {
	r.pool.Close()
}
