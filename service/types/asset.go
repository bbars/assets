package types

import (
	"github.com/bbars/assets/utils"
	"sync"
	"time"
)

type AssetStatus string

const (
	AssetStatus_pending    = AssetStatus("pending")
	AssetStatus_processing = AssetStatus("processing")
	AssetStatus_done       = AssetStatus("done")
)

const (
	AssetKeyLen = 32
)

type Asset struct {
	// AssetKey - secret unique identifier of the asset
	AssetKey string `json:"assetKey" db:"asset_key"`

	// Btime - birth time
	Btime time.Time `json:"btime" db:"btime"`

	// Mtime - modify time
	Mtime *time.Time `json:"mtime" db:"mtime"`

	// Dtime - delete time
	Dtime *time.Time `json:"dtime" db:"dtime"`

	// Size - asset size
	Size int64 `json:"size" db:"size"`

	// ContentHash - hash combination of the asset contents
	ContentHash string `json:"contentHash" db:"content_hash"`

	// ContentType - http-style content-type (mime + additional info)
	ContentType string `json:"contentType" db:"content_type"`

	// OriginalName - original asset name (e.g. file name)
	OriginalName string `json:"originalName" db:"original_name"`

	// UserId - owner user identifier
	UserId string `json:"userId" db:"user_id"`

	// OriginalUrl - original external url (for files uploaded via url)
	OriginalUrl string `json:"originalUrl" db:"original_url"`

	// Deleted - deletion flag
	Deleted bool `json:"deleted" db:"deleted"`

	// StorageName - current storage name containing the asset
	StorageName string `json:"storageName" db:"storage_name"`

	// Status - current processing status of the asset (used when downloading by external url)
	Status AssetStatus `json:"status" db:"status"`

	// Info - arbitrary short information: meta, description, etc
	Info string `json:"info" db:"info"`

	// Error - message describing an error occurred while processing the asset
	Error string `json:"error" db:"error"`
}

func (a *Asset) GenerateAssetKey() {
	a.AssetKey = utils.GenerateQid(AssetKeyLen)
}

//var _ sqlutil.Entity = &Asset{}

var assetPool = sync.Pool{
	New: func() any {
		return &Asset{}
	},
}

/*func (a *Asset) New() sqlutil.Entity {
	return assetPool.Get().(sqlutil.Entity)
}*/
func NewAsset() *Asset {
	return assetPool.Get().(*Asset)
}

func (a *Asset) TableName() string {
	return "asset"
}

func (a *Asset) Close() error {
	*a = Asset{}
	assetPool.Put(a)
	return nil
}
