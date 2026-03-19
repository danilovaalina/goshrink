package repository

import (
	"context"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/danilovaalina/goshrink/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type Repository struct {
	pool  *pgxpool.Pool
	redis *redis.Client
}

func New(pool *pgxpool.Pool, redis *redis.Client) *Repository {
	return &Repository{
		pool:  pool,
		redis: redis,
	}
}

func (r *Repository) CreateLink(ctx context.Context, link model.Link) (model.Link, error) {
	query := `
		insert into links (id, original_url, short_code)
		values ($1, $2, $3)
		returning id, original_url, short_code, created
	`

	rows, err := r.pool.Query(ctx, query, link.ID, link.OriginURL, link.ShortCode)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return model.Link{}, model.ErrAlreadyExists
		}
		return model.Link{}, errors.WithStack(err)
	}
	defer rows.Close()

	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByNameLax[linkRow])
	if err != nil {
		return model.Link{}, errors.WithStack(err)
	}

	return r.linkModel(row), nil
}

func (r *Repository) Link(ctx context.Context, shortCode string) (model.Link, error) {
	l, err := r.getLinkFromRedis(ctx, shortCode)
	if err == nil {
		return l, nil
	} else if !errors.Is(err, redis.Nil) {
		// Если ошибка Redis логируем, но продолжаем идти в бд
		log.Warn().Err(err).Send()
	}

	query := `
		select id, original_url, short_code, created
		from links
		where short_code = $1
	`

	rows, err := r.pool.Query(ctx, query, shortCode)
	if err != nil {
		return model.Link{}, errors.WithStack(err)
	}
	defer rows.Close()

	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByNameLax[linkRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Link{}, model.ErrLinkNotFound
		}
		return model.Link{}, errors.WithStack(err)
	}

	l = r.linkModel(row)

	if err = r.storeLinkInRedis(ctx, l); err != nil {
		log.Warn().Err(err).Str("short_code", shortCode).Send()
	}

	return l, nil
}

func (r *Repository) SaveClick(ctx context.Context, click model.Click) error {
	query := `
		insert into clicks (id, link_id, user_agent, ip_address)
		values ($1, $2, $3, $4)
	`
	_, err := r.pool.Exec(ctx, query, click.ID, click.LinkID, click.UserAgent, click.IPAddress)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (r *Repository) linkModel(row linkRow) model.Link {
	return model.Link{
		ID:        row.ID,
		OriginURL: row.OriginURL,
		ShortCode: row.ShortCode,
		Created:   row.Created,
	}
}

func (r *Repository) Analytics(ctx context.Context, shortCode string) (model.Analytics, error) {
	var stats model.Analytics
	stats.ShortCode = shortCode

	// Получаем общее кол-во кликов
	totalQuery := `
		select count(c.id) 
		from links l 
		left join clicks c on l.id = c.link_id 
		where l.short_code = $1 
		group by l.id`

	err := r.pool.QueryRow(ctx, totalQuery, shortCode).Scan(&stats.TotalClicks)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Analytics{}, model.ErrLinkNotFound
		}
		return model.Analytics{}, errors.WithStack(err)
	}

	// Получаем статистику по дням
	dayQuery := `
		select to_char(clicked, 'yyyy-mm-dd') as day, count(*) as count
		from clicks c
		join links l on l.id = c.link_id
		where l.short_code = $1
		group by day order by day desC`

	dayRows, err := r.pool.Query(ctx, dayQuery, shortCode)
	if err != nil {
		return model.Analytics{}, errors.WithStack(err)
	}
	dRows, err := pgx.CollectRows(dayRows, pgx.RowToStructByNameLax[dailyStatRow])
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return model.Analytics{}, errors.WithStack(err)
	}

	stats.ByDay = make([]model.DailyStat, 0, len(dRows))
	for _, dr := range dRows {
		stats.ByDay = append(stats.ByDay, model.DailyStat{Date: dr.Date, Count: dr.Count})
	}

	// Получаем статистику по User-Agent
	uaQuery := `
		select user_agent, count(*) as count
		from clicks c
		join links l on l.id = c.link_id
		where l.short_code = $1
		group by user_agent order by count desc`

	uaRows, err := r.pool.Query(ctx, uaQuery, shortCode)
	if err != nil {
		return model.Analytics{}, errors.WithStack(err)
	}
	uRows, err := pgx.CollectRows(uaRows, pgx.RowToStructByNameLax[uaStatRow])
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return model.Analytics{}, errors.WithStack(err)
	}

	stats.ByUserAgent = make([]model.UserAgentStat, 0, len(uRows))
	for _, ur := range uRows {
		stats.ByUserAgent = append(stats.ByUserAgent, model.UserAgentStat{UserAgent: ur.UserAgent, Count: ur.Count})
	}

	return stats, nil
}

type dailyStatRow struct {
	Date  string `db:"day"`
	Count int    `db:"count"`
}

type uaStatRow struct {
	UserAgent string `db:"user_agent"`
	Count     int    `db:"count"`
}

type linkRow struct {
	ID        uuid.UUID `db:"id"`
	OriginURL string    `db:"original_url"`
	ShortCode string    `db:"short_code"`
	Created   time.Time `db:"created"`
}
