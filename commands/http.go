package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/bbars/assets/ctxutil"
	"github.com/bbars/assets/service"
	"github.com/bbars/assets/service/types"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

func NewHttpCommand(initAssets InitAssets) *cli.Command {
	sh := serveHttp{
		assets: nil,
	}
	return &cli.Command{
		Name:   "http",
		Usage:  "Start pure HTTP server",
		Action: sh.Action,
		Before: func(ctx *cli.Context) (err error) {
			sh.assets, err = initAssets(ctx)
			return
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "bind",
				Usage:   "Address to bind HTTP server.",
				Value:   ":8080",
				EnvVars: []string{"ASSETS_HTTP_BIND"},
			},
			&cli.StringFlag{
				Name:    "fallback-mimetype",
				Usage:   "Fallback value for response Content-Type header.",
				Value:   "application/octet-stream",
				EnvVars: []string{"ASSETS_HTTP_FALLBACK_MIMETYPE"},
			},
		},
	}
}

type serveHttp struct {
	assets *service.Assets
	cliCtx *cli.Context
}

func (sh *serveHttp) Action(ctx *cli.Context) error {
	var err error
	bind := ctx.String("bind")
	sh.cliCtx = ctx

	hm := http.NewServeMux()

	hm.HandleFunc("/describeByKey", sh.describeByKey)
	hm.HandleFunc("/getByKey", sh.getByKey)
	hm.HandleFunc("/getByOriginalUrl", sh.getByOriginalUrl)
	hm.HandleFunc("/storeByOriginalUrl", sh.storeByOriginalUrl)
	hm.HandleFunc("/store", sh.store)

	lis, err := net.Listen("tcp", bind)
	if err != nil {
		return err
	}

	fmt.Println(lis.Addr().String())

	httpServer := &http.Server{
		Handler: hm,
		ConnContext: func(httpCtx context.Context, c net.Conn) context.Context {
			// Save *current context* and feed it to Conn.
			// Conn will wrap it with cancel that will fire when client disconnects.
			// We are about to pop *current context* in some situations
			// to bypass http request context, when we want to ignore client disconnects.
			return ctxutil.Push(ctx.Context)
		},
	}
	closed := make(chan struct{})
	go func() {
		httpServerErr := httpServer.Serve(lis)
		if httpServerErr != nil {
			log.Println("error", "httpServerErr", httpServerErr)
		}
		close(closed)
	}()

	select {
	case <-ctx.Context.Done():
		err = httpServer.Close()
	case <-closed:
	}
	return err
}

func (sh *serveHttp) describeByKey(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	ctx := r.Context()
	asset, err := sh.assets.DescribeByKey(
		ctx,
		q.Get("assetKey"),
	)
	sh.respondJson(w, asset, err)
}

func (sh *serveHttp) getByKey(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	ctx := r.Context()
	asset, rc, err := sh.assets.GetByKey(
		ctx,
		q.Get("assetKey"),
	)
	sh.respondAsset(w, r, asset, rc, err)
}

func (sh *serveHttp) getByOriginalUrl(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	ctx := r.Context()
	asset, rc, err := sh.assets.GetByOriginalUrl(
		ctx,
		q.Get("originalUrl"),
	)
	sh.respondAsset(w, r, asset, rc, err)
}

func (sh *serveHttp) storeByOriginalUrl(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	wait := q.Get("wait") != ""
	ctx := r.Context()
	if !wait {
		// Bypass http request context to ignore client disconnects
		ctx = ctxutil.Pop(ctx)
	}
	prepAsset, err := sh.assets.StoreByOriginalUrl(
		ctx,
		q.Get("originalUrl"),
		wait,
	)
	sh.respondJson(w, prepAsset, err)
}

func (sh *serveHttp) store(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	ctx := r.Context()
	var data io.Reader
	switch {
	case r.Method == http.MethodPost || r.Method == http.MethodPut:
		data = r.Body
	case ctxutil.IsDebug(ctx):
		data = strings.NewReader(q.Get("data"))
	default:
		sh.respondJson(w, nil, errors.New("invalid method"))
		return
	}
	extra := &types.Asset{
		Size:         r.ContentLength,
		ContentType:  q.Get("contentType"),
		OriginalName: q.Get("originalName"),
		UserId:       "", // TODO
		OriginalUrl:  q.Get("originalUrl"),
		StorageName:  "", // TODO
		Info:         q.Get("info"),
	}
	asset, err := sh.assets.Store(
		ctx,
		extra,
		data,
	)
	sh.respondJson(w, asset, err)
}

func (sh *serveHttp) respondJson(w http.ResponseWriter, res any, err error) {
	w.Header().Set("content-type", "application/json")
	errStr := ""
	if err != nil {
		errStr = err.Error()
		w.WriteHeader(http.StatusBadRequest)
	}
	data := struct {
		Res any    `json:"res"`
		Err string `json:"err,omitempty"`
	}{
		Res: res,
		Err: errStr,
	}
	respondErr := json.NewEncoder(w).Encode(data)
	if respondErr != nil {
		log.Println("error", "respondErr", respondErr)
	}
}

func (sh *serveHttp) respondAsset(w http.ResponseWriter, r *http.Request, asset *types.Asset, rc io.Reader, err error) {
	if closer, ok := rc.(io.Closer); ok {
		defer func() {
			closeErr := closer.Close()
			if closeErr != nil {
				log.Println("error", "closeErr", closeErr)
			}
		}()
	}
	if asset != nil {
		w.Header().Set("x-asset-btime", asset.Btime.Format(time.RFC3339Nano))
		if asset.Mtime != nil {
			w.Header().Set("x-asset-mtime", asset.Mtime.Format(time.RFC3339Nano))
		}
		if asset.OriginalUrl != "" {
			w.Header().Set("x-asset-original-url", asset.OriginalUrl)
		}
		if asset.OriginalName != "" {
			w.Header().Set("x-asset-original-name", asset.OriginalName)
		}
	}
	if err != nil {
		seeUrlErr := &service.SeeUrlError{}
		if errors.As(err, &seeUrlErr) {
			http.Redirect(w, r, seeUrlErr.Url, http.StatusTemporaryRedirect)
			return
		}

		sh.respondJson(w, nil, err)
		return
	}
	if asset.ContentType != "" {
		w.Header().Set("content-type", asset.ContentType)
	} else {
		fallbackMimetype := sh.cliCtx.String("fallback-mimetype")
		w.Header().Set("content-type", fallbackMimetype)
	}
	if asset.OriginalName != "" {
		w.Header().Set("content-disposition", fmt.Sprintf("inline; *filename='%s'", asset.OriginalName))
	}
	if asset.Size > 0 {
		w.Header().Set("content-length", fmt.Sprintf("%d", asset.Size))
	}

	_, writeErr := io.Copy(w, rc)
	if writeErr != nil {
		log.Println("error", "writeErr", writeErr)
	}
}
