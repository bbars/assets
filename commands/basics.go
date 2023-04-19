package commands

import (
	"github.com/bbars/assets/service"
	"github.com/bbars/assets/service/repository"
	"github.com/urfave/cli/v2"
)

type InitAssets func(ctx *cli.Context) (assets *service.Assets, err error)

type InitAssetRepo func(ctx *cli.Context) (assetRepo repository.Repository, err error)
