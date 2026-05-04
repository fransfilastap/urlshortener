package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/fransfilastap/urlshortener/internal/db/sqlc"
	"github.com/fransfilastap/urlshortener/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewPostgresRepository(connString string) (*PostgresRepository, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	config.ConnConfig.ConnectTimeout = 10 * time.Second

	maxRetries := 5
	retryDelay := 3 * time.Second
	var pool *pgxpool.Pool

	for i := 0; i < maxRetries; i++ {
		pool, err = pgxpool.NewWithConfig(context.Background(), config)
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err = pool.Ping(ctx)
			cancel()

			if err == nil {
				queries := sqlc.New(pool)
				return &PostgresRepository{pool: pool, queries: queries}, nil
			}
			pool.Close()
		}

		fmt.Printf("Failed to connect to database (attempt %d/%d): %v. Retrying in %v...\n",
			i+1, maxRetries, err, retryDelay)
		time.Sleep(retryDelay)
	}

	return nil, fmt.Errorf("failed to connect to database after %d attempts: %w", maxRetries, err)
}

func (r *PostgresRepository) Pool() *pgxpool.Pool {
	return r.pool
}

func pgtypeText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

func pgtypeTimestamp(t time.Time) pgtype.Timestamp {
	if t.IsZero() {
		return pgtype.Timestamp{Valid: false}
	}
	return pgtype.Timestamp{Time: t, Valid: true}
}

func pgtypeTimestampPtr(t *time.Time) pgtype.Timestamp {
	if t == nil {
		return pgtype.Timestamp{Valid: false}
	}
	return pgtype.Timestamp{Time: *t, Valid: true}
}

func sqlcURLToModel(u sqlc.Url) *models.URL {
	var expiresAt time.Time
	if u.ExpiresAt.Valid {
		expiresAt = u.ExpiresAt.Time
	}

	var deletedAt *time.Time
	if u.DeletedAt.Valid {
		deletedAt = &u.DeletedAt.Time
	}

	var title string
	if u.Title.Valid {
		title = u.Title.String
	}

	var creatorReference string
	if u.CreatorReference.Valid {
		creatorReference = u.CreatorReference.String
	}

	return &models.URL{
		ID:               int64(u.ID),
		Original:         u.Original,
		Short:            u.Short,
		Title:            title,
		CreatedAt:        u.CreatedAt.Time,
		ExpiresAt:        expiresAt,
		Clicks:           u.Clicks,
		CreatorReference: creatorReference,
		DeletedAt:        deletedAt,
	}
}

func sqlcClickToModel(c sqlc.Click) *models.Click {
	var location string
	if c.Location.Valid {
		location = c.Location.String
	}

	var browser string
	if c.Browser.Valid {
		browser = c.Browser.String
	}

	var device string
	if c.Device.Valid {
		device = c.Device.String
	}

	return &models.Click{
		ID:        int64(c.ID),
		URLID:     c.UrlID,
		URLShort:  c.UrlShort,
		IP:        c.Ip,
		Location:  location,
		Browser:   browser,
		Device:    device,
		Timestamp: c.Timestamp.Time,
	}
}

func (r *PostgresRepository) Create(ctx context.Context, url *models.URL) (*models.URL, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM urls WHERE short = $1 AND deleted_at IS NULL)", url.Short).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrURLExists
	}

	result, err := r.queries.CreateURL(ctx, sqlc.CreateURLParams{
		Original:         url.Original,
		Short:            url.Short,
		Title:            pgtypeText(url.Title),
		CreatedAt:        pgtypeTimestamp(url.CreatedAt),
		ExpiresAt:        pgtypeTimestamp(url.ExpiresAt),
		Clicks:            url.Clicks,
		CreatorReference: pgtypeText(url.CreatorReference),
		DeletedAt:        pgtypeTimestampPtr(url.DeletedAt),
	})
	if err != nil {
		return nil, err
	}

	return sqlcURLToModel(result), nil
}

func (r *PostgresRepository) GetByShort(ctx context.Context, short string) (*models.URL, error) {
	result, err := r.queries.GetURLByShort(ctx, short)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrURLNotFound
		}
		return nil, err
	}

	url := sqlcURLToModel(result)

	if !url.ExpiresAt.IsZero() && url.ExpiresAt.Before(time.Now()) {
		return nil, ErrURLNotFound
	}

	return url, nil
}

func (r *PostgresRepository) GetByOriginal(ctx context.Context, original string) (*models.URL, error) {
	result, err := r.queries.GetURLByOriginal(ctx, original)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrURLNotFound
		}
		return nil, err
	}

	url := sqlcURLToModel(result)

	if !url.ExpiresAt.IsZero() && url.ExpiresAt.Before(time.Now()) {
		return nil, ErrURLNotFound
	}

	return url, nil
}

func (r *PostgresRepository) GetByCreator(ctx context.Context, creatorReference string) ([]*models.URL, error) {
	results, err := r.queries.GetURLsByCreator(ctx, pgtypeText(creatorReference))
	if err != nil {
		return nil, err
	}

	var urls []*models.URL
	for _, u := range results {
		url := sqlcURLToModel(u)

		if !url.ExpiresAt.IsZero() && url.ExpiresAt.Before(time.Now()) {
			continue
		}

		urls = append(urls, url)
	}

	return urls, nil
}

func (r *PostgresRepository) IncrementClicks(ctx context.Context, short string) error {
	_, err := r.queries.IncrementClicks(ctx, short)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrURLNotFound
		}
		return err
	}
	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, short string) error {
	rowsAffected, err := r.queries.SoftDeleteURL(ctx, short)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrURLNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteWithCreator(ctx context.Context, short string, creatorReference string) error {
	existingURL, err := r.GetByShort(ctx, short)
	if err != nil {
		return err
	}

	if existingURL.CreatorReference != creatorReference {
		return errors.New("unauthorized: creator reference does not match")
	}

	rowsAffected, err := r.queries.SoftDeleteURLWithCreator(ctx, sqlc.SoftDeleteURLWithCreatorParams{
		Short:            short,
		CreatorReference: pgtypeText(creatorReference),
	})
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrURLNotFound
	}
	return nil
}

func (r *PostgresRepository) HardDelete(ctx context.Context, short string) error {
	_, err := r.queries.HardDeleteURL(ctx, short)
	return err
}

func (r *PostgresRepository) StoreClick(ctx context.Context, click *models.Click) error {
	_, err := r.queries.StoreClick(ctx, sqlc.StoreClickParams{
		UrlID:     click.URLID,
		UrlShort:  click.URLShort,
		Ip:        click.IP,
		Location:  pgtypeText(click.Location),
		Browser:   pgtypeText(click.Browser),
		Device:    pgtypeText(click.Device),
		Timestamp: pgtypeTimestamp(click.Timestamp),
	})
	return err
}

func (r *PostgresRepository) GetClicksByShort(ctx context.Context, short string) ([]*models.Click, error) {
	results, err := r.queries.GetClicksByShort(ctx, short)
	if err != nil {
		return nil, err
	}

	var clicks []*models.Click
	for _, c := range results {
		clicks = append(clicks, sqlcClickToModel(c))
	}

	return clicks, nil
}

func (r *PostgresRepository) HasRecentClick(ctx context.Context, short string, ip string, browser string, device string) (bool, error) {
	return r.queries.HasRecentClick(ctx, sqlc.HasRecentClickParams{
		UrlShort: short,
		Ip:       ip,
		Browser:  pgtypeText(browser),
		Device:   pgtypeText(device),
	})
}

func (r *PostgresRepository) UpdateURL(ctx context.Context, short string, url *models.URL) error {
	result, err := r.queries.UpdateURL(ctx, sqlc.UpdateURLParams{
		Original:  url.Original,
		Title:     pgtypeText(url.Title),
		ExpiresAt: pgtypeTimestamp(url.ExpiresAt),
		Short:     short,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrURLNotFound
		}
		return err
	}

	*url = *sqlcURLToModel(result)
	return nil
}

func (r *PostgresRepository) UpdateURLWithCreator(ctx context.Context, short string, url *models.URL, creatorReference string) error {
	existingURL, err := r.GetByShort(ctx, short)
	if err != nil {
		return err
	}

	if existingURL.CreatorReference != creatorReference {
		return errors.New("unauthorized: creator reference does not match")
	}

	result, err := r.queries.UpdateURLWithCreator(ctx, sqlc.UpdateURLWithCreatorParams{
		Original:         url.Original,
		Title:            pgtypeText(url.Title),
		ExpiresAt:        pgtypeTimestamp(url.ExpiresAt),
		Short:            short,
		CreatorReference: pgtypeText(creatorReference),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrURLNotFound
		}
		return err
	}

	*url = *sqlcURLToModel(result)
	return nil
}

func (r *PostgresRepository) LogURLHistory(ctx context.Context, urlID int64, short string, action string, oldValue, newValue interface{}, modifiedBy string) error {
	oldValueJSON, err := json.Marshal(oldValue)
	if err != nil {
		return err
	}

	newValueJSON, err := json.Marshal(newValue)
	if err != nil {
		return err
	}

	_, err = r.queries.LogURLHistory(ctx, sqlc.LogURLHistoryParams{
		UrlID:      urlID,
		UrlShort:   short,
		Action:     action,
		OldValue:   oldValueJSON,
		NewValue:   newValueJSON,
		ModifiedBy: pgtypeText(modifiedBy),
	})
	return err
}

func (r *PostgresRepository) GetClickAnalytics(ctx context.Context, short string) (map[string]interface{}, error) {
	totalClicks, err := r.queries.GetTotalClicks(ctx, short)
	if err != nil {
		return nil, err
	}

	browserRows, err := r.queries.GetClicksByBrowser(ctx, short)
	if err != nil {
		return nil, err
	}

	browserStats := make(map[string]int64)
	for _, row := range browserRows {
		var browser string
		if row.Browser.Valid {
			browser = row.Browser.String
		}
		browserStats[browser] = row.Count
	}

	deviceRows, err := r.queries.GetClicksByDevice(ctx, short)
	if err != nil {
		return nil, err
	}

	deviceStats := make(map[string]int64)
	for _, row := range deviceRows {
		var device string
		if row.Device.Valid {
			device = row.Device.String
		}
		deviceStats[device] = row.Count
	}

	locationRows, err := r.queries.GetClicksByLocation(ctx, short)
	if err != nil {
		return nil, err
	}

	locationStats := make(map[string]int64)
	for _, row := range locationRows {
		var location string
		if row.Location.Valid {
			location = row.Location.String
		}
		locationStats[location] = row.Count
	}

	return map[string]interface{}{
		"total_clicks": totalClicks,
		"browsers":     browserStats,
		"devices":      deviceStats,
		"locations":    locationStats,
	}, nil
}

func (r *PostgresRepository) Close() {
	r.pool.Close()
}