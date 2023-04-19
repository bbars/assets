package repository

import (
	"database/sql"
	"fmt"
	"github.com/bbars/assets/service/types"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"io"
	"io/fs"
	"path/filepath"
	"time"
)

const (
	SqliteMigrationsDir = "migrations/sqlite3"
)

type sqlite struct {
	Db         *sqlx.DB
	migrations fs.ReadDirFS
}

func NewSqlite(db *sql.DB, migrations fs.ReadDirFS) *sqlite {
	return &sqlite{
		Db:         sqlx.NewDb(db, "sqlite3"),
		migrations: migrations,
	}
}

var _ Repository = &sqlite{}

func (sq *sqlite) Migrate() (err error) {
	dirEntries, err := sq.migrations.ReadDir(SqliteMigrationsDir)
	if err != nil {
		err = errors.Wrap(err, "list migrations")
		return
	}

	// apply initial migration
	_ = sq.applyMigration(SqliteMigrationsDir+"/"+InitialMigrationName, true)

	// apply pending migrations
	for _, dirEntry := range dirEntries {
		migrationName := dirEntry.Name()
		if migrationName == InitialMigrationName {
			continue
		}

		err = sq.applyMigration(SqliteMigrationsDir+"/"+migrationName, false)
		if err != nil {
			err = errors.Wrapf(err, "apply migration %+q", migrationName)
			break
		}
	}

	return
}

func (sq *sqlite) applyMigration(filePath string, skipPreCheck bool) (err error) {
	migrationName := filepath.Base(filePath)

	if !skipPreCheck {
		var applied bool
		err = sq.Db.Get(
			&applied,
			fmt.Sprintf(
				`
				SELECT`+` 1 FROM %s
				WHERE name = $1
				AND error = ''
				`,
				MigrationTableName,
			),
			migrationName,
		)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				err = nil
			} else {
				err = errors.Wrapf(err, "pre-check migration %+q", filePath)
				return
			}
		}
		if applied {
			return
		}
	}

	tx, err := sq.Db.Begin()
	if err != nil {
		err = errors.Wrapf(err, "begin transaction for migration %+q", filePath)
		return
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
		}

		errorMessage := ""
		if err != nil {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				err = errors.Wrapf(err, "rollback migration %+q: %s", filePath, rollbackErr.Error())
			}
			errorMessage = err.Error()
		}

		_, errFix := sq.Db.Exec(
			fmt.Sprintf(
				`
				INSERT`+` INTO %s
				(name, btime, error)
				VALUES
				($1, $2, $3)
				`,
				MigrationTableName,
			),
			migrationName,
			time.Now().UTC(),
			errorMessage,
		)
		if errFix != nil && err == nil {
			err = errors.Wrapf(errFix, "fix migration %+q", filePath)
			return
		}
	}()

	f, err := sq.migrations.Open(filePath)
	if err != nil {
		err = errors.Wrapf(err, "open migration %+q", filePath)
		return
	}

	query, err := io.ReadAll(f)
	_ = f.Close()
	if err != nil {
		err = errors.Wrapf(err, "read migration %+q", filePath)
		return
	}

	_, err = tx.Exec(string(query))
	if err != nil {
		err = errors.Wrapf(err, "execute migration %+q", filePath)
		return
	}
	return
}

func (sq *sqlite) GetByAssetKey(assetKey string) (asset *types.Asset, err error) {
	asset = types.NewAsset()
	err = sq.Db.Get(
		asset,
		fmt.Sprintf(
			`
			SELECT`+` * FROM %s
			WHERE asset_key = $1
			`,
			asset.TableName(),
		),
		assetKey,
	)
	if errors.Is(err, sql.ErrNoRows) {
		asset = nil
		err = errors.Wrap(ErrNotFound, "select asset by asset key")
	}
	//asset, err = sqlutil.QueryOne[*types.Asset](sq.Db, "SELECT * FROM ?? WHERE asset_key = $1", assetKey)
	return
}

func (sq *sqlite) GetByOriginalUrl(originalUrl string, allowError bool) (asset *types.Asset, err error) {
	asset = types.NewAsset()
	err = sq.Db.Get(
		asset,
		fmt.Sprintf(
			`
			SELECT`+` * FROM %s
			WHERE original_url = $1
			AND ($2 OR error = "")
			`,
			asset.TableName(),
		),
		originalUrl,
		allowError,
	)
	if errors.Is(err, sql.ErrNoRows) {
		asset = nil
		err = errors.Wrap(ErrNotFound, "select asset by original url")
	}
	//asset, err = sqlutil.QueryOne[*types.Asset](sq.Db, "SELECT * FROM ?? WHERE original_url = $1 and error = ''", originalUrl)
	return
}

func (sq *sqlite) Insert(asset *types.Asset) (err error) {
	_, err = sq.Db.NamedExec(
		fmt.Sprintf(
			`
			INSERT`+` INTO %s
			(asset_key, btime, size, content_hash, content_type, original_name, user_id, original_url, deleted, storage_name, status, info, error)
			VALUES
			(:asset_key, :btime, :size, :content_hash, :content_type, :original_name, :user_id, :original_url, :deleted, :storage_name, :status, :info, :error)
			`,
			asset.TableName(),
		),
		asset,
	)
	//_, err = sqlite.Insert(sq.Db, asset)
	return
}

func (sq *sqlite) Update(asset *types.Asset) (err error) {
	now := time.Now()
	asset.Mtime = &now

	_, err = sq.Db.NamedExec(
		fmt.Sprintf(
			`
			UPDATE`+` %s
			SET
			  mtime = :mtime
			, dtime = :dtime
			, size = :size
			, content_hash = :content_hash
			, content_type = :content_type
			, original_name = :original_name
			, user_id = :user_id
			, original_url = :original_url
			, deleted = :deleted
			, storage_name = :storage_name
			, status = :status
			, info = :info
			, error = :error
			WHERE asset_key = :asset_key
			`,
			asset.TableName(),
		),
		asset,
	)
	/*
		_, err = sqlite.UpdateOne(sq.Db,
			[]string{
				"mtime",
				"dtime",
				"size",
				"content_hash",
				"content_type",
				"original_name",
				"user_id",
				"original_url",
				"deleted",
				"storage_name",
				"status",
				"info",
				"error",
			},
			asset,
		)
		return
	*/
	return
}
