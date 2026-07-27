package retention

import (
	"context"
	"database/sql"
	"log/slog"
	"time"
)

const (
	purgeInterval  = time.Hour
	purgeBatchSize = 100
)

type ObjectDeleter interface {
	Delete(ctx context.Context, key string) error
}

type Worker struct {
	db   *sql.DB
	objs ObjectDeleter
	log  *slog.Logger
}

func New(db *sql.DB, objs ObjectDeleter, log *slog.Logger) *Worker {
	return &Worker{db: db, objs: objs, log: log}
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(purgeInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.purgeExpired(ctx); err != nil {
				w.log.Error("retention_purge_failed", "error", err)
			}
		}
	}
}

func (w *Worker) purgeExpired(ctx context.Context) error {
	rows, err := w.db.QueryContext(ctx, `
		SELECT id, storage_key, output_key
		FROM videos
		WHERE purged_at IS NULL
		  AND expires_at IS NOT NULL
		  AND expires_at < now()
		LIMIT $1`, purgeBatchSize)
	if err != nil {
		return err
	}
	defer rows.Close()

	type expiredVideo struct {
		id        string
		inputKey  string
		outputKey sql.NullString
	}
	var expired []expiredVideo
	for rows.Next() {
		var v expiredVideo
		if err := rows.Scan(&v.id, &v.inputKey, &v.outputKey); err != nil {
			return err
		}
		expired = append(expired, v)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, v := range expired {
		if err := w.objs.Delete(ctx, v.inputKey); err != nil {
			w.log.Error("retention_object_delete_failed", "key", v.inputKey, "error", err)
			continue
		}
		if v.outputKey.Valid {
			_ = w.objs.Delete(ctx, v.outputKey.String)
		}
		if _, err := w.db.ExecContext(ctx,
			`UPDATE videos SET purged_at = now() WHERE id = $1`, v.id); err != nil {
			w.log.Error("retention_mark_failed", "id", v.id, "error", err)
		}
	}

	if len(expired) > 0 {
		w.log.Info("retention_purged", "count", len(expired))
	}
	return nil
}

func (w *Worker) EraseUser(ctx context.Context, userID string) error {
	rows, err := w.db.QueryContext(ctx,
		`SELECT storage_key, output_key FROM videos WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var inputKey string
		var outputKey sql.NullString
		if err := rows.Scan(&inputKey, &outputKey); err != nil {
			return err
		}
		_ = w.objs.Delete(ctx, inputKey)
		if outputKey.Valid {
			_ = w.objs.Delete(ctx, outputKey.String)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = w.db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, userID)
	return err
}
