package service

import (
	"regexp"
)

type AssetsConfig struct {
	MaxRemoteSize      int64
	MaxRemoteWaitSize  int64
	MaxSize            int64
	OriginalUrlPattern *regexp.Regexp
	HttpUserAgent      string
}
