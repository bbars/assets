package service

import (
	"regexp"
)

type AssetStorageConfig struct {
	MaxRemoteSize      int64
	MaxRemoteWaitSize  int64
	MaxSize            int64
	OriginalUrlPattern *regexp.Regexp
	HttpUserAgent      string
}
