package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"github.com/bbars/assets/service"
	"github.com/urfave/cli/v2"
	"log"
	"os"
)

func NewStoreUrlsCommand(initAssets InitAssets) *cli.Command {
	su := storeUrl{
		assets:  nil,
		jsonOut: json.NewEncoder(os.Stdout),
	}
	return &cli.Command{
		Name:   "storeurls",
		Usage:  "store assets by original urls",
		Action: su.Action,
		Before: func(ctx *cli.Context) (err error) {
			su.assets, err = initAssets(ctx)
			return
		},
		Flags: []cli.Flag{},
	}
}

type storeUrl struct {
	assets  *service.Assets
	jsonOut *json.Encoder
}

func (su storeUrl) Action(ctx *cli.Context) (err error) {
	var scanner *bufio.Scanner

	args := ctx.Args()
	originalUrls := make([]string, 0, args.Len())
	for i := 0; i < args.Len(); i++ {
		url := args.Get(i)
		if url == "-" {
			scanner = bufio.NewScanner(os.Stdin)
			continue
		}
		originalUrls = append(originalUrls, url)
	}

	for _, originalUrl := range originalUrls {
		su.processOne(ctx.Context, originalUrl)
	}

	if scanner != nil {
		for scanner.Scan() {
			su.processOne(ctx.Context, scanner.Text())
		}
		err = scanner.Err()
		if err != nil {
			return
		}
	}

	return
}

func (su storeUrl) processOne(ctx context.Context, originalUrl string) {
	asset, err := su.assets.StoreByOriginalUrl(
		ctx, // TODO wrap? handle Done?
		originalUrl,
		true,
	)
	if err != nil {
		log.Println("error", err)
	}
	if asset != nil {
		jsonErr := su.jsonOut.Encode(asset)
		if jsonErr != nil {
			log.Println("error", "jsonErr", jsonErr)
		}
	}
}
