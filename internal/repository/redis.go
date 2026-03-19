package repository

import (
	"context"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/danilovaalina/goshrink/internal/model"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	linkKeyPrefix = "link:"
	linkTTL       = 24 * time.Hour // Ссылки живут в кэше дольше уведомлений
)

// linkCache — структура для Redis Hash
type linkCache struct {
	ID        string `redis:"id"`
	OriginURL string `redis:"original_url"`
	ShortCode string `redis:"short_code"`
	Created   string `redis:"created"`
}

func convertToLinkCache(l model.Link) linkCache {
	return linkCache{
		ID:        l.ID.String(),
		OriginURL: l.OriginURL,
		ShortCode: l.ShortCode,
		Created:   l.Created.UTC().Format(time.RFC3339Nano),
	}
}

func convertToLink(lc linkCache) (model.Link, error) {
	id, err := uuid.Parse(lc.ID)
	if err != nil {
		return model.Link{}, err
	}
	created, err := time.Parse(time.RFC3339Nano, lc.Created)
	if err != nil {
		return model.Link{}, err
	}

	return model.Link{
		ID:        id,
		OriginURL: lc.OriginURL,
		ShortCode: lc.ShortCode,
		Created:   created,
	}, nil
}

func (r *Repository) storeLinkInRedis(ctx context.Context, l model.Link) error {
	lc := convertToLinkCache(l)
	key := linkKeyPrefix + l.ShortCode

	// Используем HSet для записи всей структуры
	err := r.redis.HSet(ctx, key, lc).Err()
	if err != nil {
		return errors.WithStack(err)
	}

	_ = r.redis.Expire(ctx, key, linkTTL)
	return nil
}

func (r *Repository) getLinkFromRedis(ctx context.Context, shortCode string) (model.Link, error) {
	key := linkKeyPrefix + shortCode

	// HGetAll считывает хэш в map
	var lc linkCache
	err := r.redis.HGetAll(ctx, key).Scan(&lc)
	if err != nil {
		return model.Link{}, errors.WithStack(err)
	}

	// Если в Redis ничего нет, вернется пустая структура (ID == "")
	if lc.ID == "" {
		return model.Link{}, redis.Nil
	}

	return convertToLink(lc)
}
