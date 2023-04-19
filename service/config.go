package service

import (
	"regexp"
)

type AssetStorageConfig struct {
	MaxRemoteSize      uint64
	MaxRemoteWaitSize  uint64
	MaxSize            uint64
	OriginalUrlPattern *regexp.Regexp
	HttpUserAgent      string
}
