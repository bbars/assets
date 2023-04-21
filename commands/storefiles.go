package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"github.com/bbars/assets/service"
	"github.com/bbars/assets/service/types"
	"github.com/urfave/cli/v2"
)

func NewStoreFilesCommand(initAssets InitAssets) *cli.Command {
	sf := storeFile{
		assets:  nil,
		jsonOut: json.NewEncoder(os.Stdout),
	}
	return &cli.Command{
		Name:   "storefiles",
		Usage:  "Store local files as assets",
		Action: sf.Action,
		Before: func(ctx *cli.Context) (err error) {
			sf.assets, err = initAssets(ctx)
			return
		},
		Flags: []cli.Flag{},
	}
}

type storeFile struct {
	assets  *service.Assets
	jsonOut *json.Encoder
}

func (sf storeFile) Action(ctx *cli.Context) (err error) {
	var scanner *bufio.Scanner

	args := ctx.Args()
	filePaths := make([]string, 0, args.Len())
	for i := 0; i < args.Len(); i++ {
		filePath := args.Get(i)
		if filePath == "-" {
			scanner = bufio.NewScanner(os.Stdin)
			continue
		}
		filePaths = append(filePaths, filePath)
	}

	for _, filePath := range filePaths {
		sf.processOne(ctx.Context, filePath)
	}

	if scanner != nil {
		for scanner.Scan() {
			sf.processOne(ctx.Context, scanner.Text())
		}
		err = scanner.Err()
		if err != nil {
			return
		}
	}

	return
}

func (sf storeFile) processOne(ctx context.Context, filePath string) {
	f, err := os.Open(filePath)
	if err != nil {
		log.Println("error", err)
		return
	}
	defer func() {
		err = f.Close()
		if err != nil {
			log.Println("error", err)
			return
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
	stat, err := f.Stat()
	if err != nil {
		log.Println("error", err)
		return
	}

	fileAssetInfo := make(map[string]map[string]any)
	fileAssetInfo["file"] = make(map[string]any)
	fileAssetInfo["file"]["absolutePath"], err = filepath.Abs(filePath)
	if err != nil {
		log.Println("error", err)
		return
	}
	fileAssetInfo["file"]["mtime"] = stat.ModTime()
	extraInfoJson, err := json.Marshal(fileAssetInfo)
	if err != nil {
		log.Println("error", err)
		return
	}
	extra.Info = string(extraInfoJson)

	extra.OriginalName = filepath.Base(filePath)
	extra.Size = stat.Size()
	asset, err := sf.assets.Store(
		ctx,
		extra,
		f,
	)
	if err != nil {
		log.Println("error", err)
	}
	if asset != nil {
		jsonErr := sf.jsonOut.Encode(asset)
		if jsonErr != nil {
			log.Println("error", "jsonErr", jsonErr)
		}
	}
}
