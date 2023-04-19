package commands

import (
	"encoding/json"
	"github.com/bbars/assets/service"
	"github.com/bbars/assets/service/types"
	"github.com/urfave/cli/v2"
	"log"
	"os"
)

func NewStorePipeCommand(initAssets InitAssets) *cli.Command {
	sp := storePipe{
		assets:  nil,
		jsonOut: json.NewEncoder(os.Stdout),
	}
	return &cli.Command{
		Name:   "storepipe",
		Usage:  "store local file as an asset",
		Action: sp.Action,
		Before: func(ctx *cli.Context) (err error) {
			sp.assets, err = initAssets(ctx)
			return
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "content-type",
				Aliases: []string{"type", "mime"},
				Usage:   "value for asset's content_type field",
			},
			&cli.StringFlag{
				Name:    "original-name",
				Aliases: []string{"name"},
				Usage:   "value for asset's original_name field",
			},
			&cli.StringFlag{
				Name:    "original-url",
				Aliases: []string{"url"},
				Usage:   "value for asset's original_url field",
			},
			&cli.StringFlag{
				Name:  "info",
				Usage: "value for asset's info field",
			},
		},
	}
}

type storePipe struct {
	assets  *service.Assets
	jsonOut *json.Encoder
}

func (sp storePipe) Action(ctx *cli.Context) (err error) {
	defer func() {
		closeErr := os.Stdin.Close()
		if closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	extra := types.NewAsset()
	defer func() {
		err = extra.Close()
		if err != nil {
			log.Println("error", err)
			return
		}
	}()

	extra.Info = ctx.String("info")
	extra.OriginalName = ctx.String("original-name")
	extra.OriginalUrl = ctx.String("original-url")
	extra.ContentType = ctx.String("content-type")
	asset, err := sp.assets.Store(
		ctx.Context,
		extra,
		os.Stdin,
	)
	if err != nil {
		log.Println("error", err)
	}
	if asset != nil {
		jsonErr := sp.jsonOut.Encode(asset)
		if jsonErr != nil {
			log.Println("error", "jsonErr", jsonErr)
		}
	}
	return
}
