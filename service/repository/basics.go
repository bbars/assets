package repository

import (
	"errors"
	"github.com/bbars/assets/service/types"
)

const (
	InitialMigrationName = "0000-00-00-00-00-00-initial.sql"
	MigrationTableName   = "assets_migration"
)

var (
	ErrNotFound = errors.New("row not found")
)

type Repository interface {
	Migrate() (err error)
	GetByAssetKey(assetKey string) (asset *types.Asset, err error)
	GetByOriginalUrl(originalUrl string, allowError bool) (asset *types.Asset, err error)
	Insert(asset *types.Asset) (err error)
	Update(asset *types.Asset) (err error)
}
