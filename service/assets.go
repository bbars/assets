package service

import (
	"context"
	"fmt"
	"github.com/bbars/assets/service/repository"
	"github.com/bbars/assets/service/storage"
	"github.com/bbars/assets/service/types"
	"github.com/bbars/assets/service/utils"
	"github.com/pkg/errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"time"
)

type Assets struct {
	Storage storage.Storage
	Repo    repository.Repository
	Config  AssetStorageConfig

	HttpClient                *http.Client
	contentDispositionMatcher *regexp.Regexp
}

//goland:noinspection GoUnusedParameter
func (a *Assets) DescribeByKey(ctx context.Context, assetKey string) (asset *types.Asset, err error) {
	defer RecoverService(&err)

	asset, err = a.Repo.GetByAssetKey(assetKey)
	if err != nil {
		err = errors.Wrapf(err, "query asset by asset_key=%+q", assetKey)
		return
	}
	return
}

//goland:noinspection GoUnusedParameter
func (a *Assets) GetByKey(ctx context.Context, assetKey string) (asset *types.Asset, rc io.ReadCloser, err error) {
	defer RecoverService(&err)

	asset, err = a.Repo.GetByAssetKey(assetKey)
	if err != nil {
		err = errors.Wrapf(err, "query asset by asset_key=%+q", assetKey)
		return
	}

	rc, err = a.readAsset(ctx, asset)
	return
}

func (a *Assets) GetByOriginalUrl(ctx context.Context, originalUrl string) (asset *types.Asset, rc io.ReadCloser, err error) {
	defer RecoverService(&err)

	asset, err = a.getByOriginalUrlOrNil(originalUrl)
	if err != nil {
		err = errors.Wrap(err, "find existing asset")
		return
	} else if asset != nil {
		rc, err = a.readAsset(ctx, asset)
		return
	}

	err = a.checkOriginalUrl(originalUrl)
	if err != nil {
		err = &SeeUrlError{
			Url: originalUrl,
		}
		err = errors.Wrap(err, "unable to store by original url")
		return
	}

	r, w := io.Pipe()
	rc = io.NopCloser(r)
	assetCh := make(chan *types.Asset)
	go func() {
		asset, err = a.storeByOriginalUrl(ctx, originalUrl, assetCh, w)
	}()
	asset = <-assetCh
	return
}

//goland:noinspection GoUnusedParameter
func (a *Assets) readAsset(ctx context.Context, asset *types.Asset) (rc io.ReadCloser, err error) {
	if asset.Status == types.AssetStatus_pending || asset.Status == types.AssetStatus_processing {
		if asset.OriginalUrl == "" {
			err = errors.Errorf("found asset is not done yet, status=%s", asset.Status)
		} else {
			err = &SeeUrlError{
				Url: asset.OriginalUrl,
			}
			err = errors.Wrapf(err, "found asset is not done yet, status=%s", asset.Status)
		}
		return
	}

	if asset.Error != "" {
		err = errors.Errorf("open asset content_hash=%+q: %s", asset.ContentHash, asset.Error)
		return
	}

	rc, err = a.Storage.OpenRead(asset.ContentHash)
	if err != nil {
		err = errors.Wrapf(err, "open asset content_hash=%+q", asset.ContentHash)
		return
	}
	return
}

//goland:noinspection GoUnusedParameter
func (a *Assets) Store(ctx context.Context, extra *types.Asset, data io.Reader) (asset *types.Asset, err error) {
	defer RecoverService(&err)

	_, contentHash, size, err := a.Storage.Write(data, a.Config.MaxSize)
	if err != nil {
		err = errors.Wrap(err, "write asset")
		return
	}

	asset = &types.Asset{
		AssetKey:     utils.GenerateQid(types.AssetKeyLen),
		Btime:        time.Now(),
		Mtime:        nil,
		Dtime:        nil,
		Size:         size,
		ContentHash:  contentHash,
		ContentType:  extra.ContentType,
		OriginalName: extra.OriginalName,
		UserId:       "",
		OriginalUrl:  extra.OriginalUrl,
		Deleted:      false,
		StorageName:  "",
		Status:       types.AssetStatus_done,
		Info:         extra.Info,
		Error:        "",
	}
	err = a.Repo.Insert(asset)
	if err != nil {
		err = errors.Wrap(err, "save done asset")
		return
	}
	return
}

func (a *Assets) StoreByOriginalUrl(ctx context.Context, originalUrl string, wait bool) (asset *types.Asset, err error) {
	defer RecoverService(&err)

	asset, err = a.getByOriginalUrlOrNil(originalUrl)
	if err != nil {
		err = errors.Wrap(err, "find existing asset")
		return
	}
	if asset != nil {
		if asset.Error == "" {
			return
		} else {
			// another try to fetch
			asset = nil
		}
	}

	err = a.checkOriginalUrl(originalUrl)
	if err != nil {
		err = errors.Wrap(err, "unable to store by original url")
		return
	}

	//ctxStore, cancel := context.WithCancel(ctx)
	//defer cancel()
	prepAssetCh := make(chan *types.Asset)
	done := make(chan struct{})
	go func() {
		asset, err = a.storeByOriginalUrl(ctx, originalUrl, prepAssetCh, nil)
		close(done)
		log.Printf("background asset done with asset_key=%+q", asset.AssetKey)
		if err != nil {
			log.Printf("background error: %s\n", err.Error())
		}
	}()

	select {
	case <-ctx.Done():
		err = errors.New("cancelled")
	case asset = <-prepAssetCh:
	}
	if wait {
		<-done
	}
	return
}

func (a *Assets) checkOriginalUrl(originalUrl string) (err error) {
	if originalUrl == "" {
		err = errors.New("value of originalUrl can't be empty")
		return
	}
	if a.Config.OriginalUrlPattern == nil {
		err = errors.New("not allowed, because OriginalUrlPattern is nil")
		return
	}
	if !a.Config.OriginalUrlPattern.Match([]byte(originalUrl)) {
		err = errors.New("not allowed, because originalUrl doesn't match OriginalUrlPattern")
		return
	}
	return
}

func (a *Assets) getByOriginalUrlOrNil(originalUrl string) (asset *types.Asset, err error) {
	// find an asset without error first
	asset, err = a.Repo.GetByOriginalUrl(originalUrl, false)
	if err != nil && errors.Is(err, repository.ErrNotFound) {
		// if nothing found, then allow error and try again
		asset, err = a.Repo.GetByOriginalUrl(originalUrl, true)
		if err != nil && errors.Is(err, repository.ErrNotFound) {
			err = nil
		}
	}
	if err != nil {
		err = errors.Wrapf(err, "query asset by original_url=%+q", originalUrl)
		return
	}
	return
}

func (a *Assets) storeByOriginalUrl(ctx context.Context, originalUrl string, prepAssetCh chan<- *types.Asset, wc io.WriteCloser) (asset *types.Asset, err error) {
	defer RecoverService(&err)

	defer func() {
		close(prepAssetCh)
	}()

	asset = &types.Asset{
		AssetKey:    "",
		Btime:       time.Now(),
		OriginalUrl: originalUrl,
		Status:      types.AssetStatus_processing,
	}
	asset.GenerateAssetKey()

	err = a.Repo.Insert(asset)
	if err != nil {
		err = errors.Wrap(err, "save processing asset")
		return
	}
	defer func() {
		asset.Status = types.AssetStatus_done
		if err != nil {
			asset.Error = fmt.Sprintf("%s", err.Error())
		}
		updErr := a.Repo.Update(asset)
		if updErr != nil && err == nil {
			err = errors.Wrap(updErr, "save done asset")
		}
	}()

	httpCtx, cancel := context.WithCancel(ctx)
	defer func() {
		cancel()
	}()
	request, err := http.NewRequestWithContext(httpCtx, http.MethodGet, originalUrl, nil)
	if err != nil {
		err = errors.Wrapf(err, "prepare request to pull remote object %+q", originalUrl)
		return
	}
	request.Header.Set("user-agent", a.Config.HttpUserAgent)
	response, err := a.getHttpClient().Do(request)
	if err != nil {
		err = errors.Wrapf(err, "fetch remote object %+q", originalUrl)
		return
	}

	if 200 > response.StatusCode || response.StatusCode >= 300 {
		err = errors.Errorf("http status %s", response.Status)
		return
	}

	asset.ContentType = response.Header.Get("content-type")

	asset.OriginalName = a.extractOriginalName(response.Header.Get("content-disposition"), originalUrl)

	contentLength := response.ContentLength
	if a.Config.MaxRemoteSize > 0 {
		if contentLength < 0 {
			err = errors.Errorf("remote size is unknown while the limit is enabled (max-remote-size=%d)", a.Config.MaxRemoteSize)
			return
		}
		if contentLength > a.Config.MaxRemoteSize {
			err = errors.Errorf("remote size %d exceeds limit max-remote-size=%d", contentLength, a.Config.MaxRemoteSize)
			return
		}
	}
	asset.Size = contentLength

	if a.Config.MaxSize > 0 {
		if contentLength < 0 {
			err = errors.Errorf("remote size is unknown while the limit is enabled (max-size=%d)", a.Config.MaxSize)
			return
		}
		if contentLength > a.Config.MaxSize {
			err = errors.Errorf("remote size %d exceeds limit max-size=%d", contentLength, a.Config.MaxSize)
			return
		}
	}

	if prepAssetCh != nil {
		assetCopy := &types.Asset{}
		*assetCopy = *asset
		prepAssetCh <- assetCopy
	}

	var contentHash string
	var size int64
	if wc == nil {
		_, contentHash, size, err = a.Storage.Write(response.Body, a.Config.MaxSize)
	} else {
		defer func() {
			closeErr := wc.Close()
			if closeErr != nil && err == nil {
				err = errors.Wrap(closeErr, "close writer")
				return
			}
		}()
		tee := io.TeeReader(response.Body, wc)

		_, contentHash, size, err = a.Storage.Write(tee, a.Config.MaxSize)
	}

	asset.ContentHash = contentHash
	asset.Size = size
	if err != nil {
		err = errors.Wrapf(err, "storage write for asset asset_key=%+q", asset.AssetKey)
		return
	}

	return
}

func (a *Assets) getHttpClient() *http.Client {
	if a.HttpClient == nil {
		return http.DefaultClient
	}
	return a.HttpClient
}

func (a *Assets) extractOriginalName(contentDisposition string, originalUrl string) string {
	// TODO optimize
	if a.contentDispositionMatcher == nil {
		a.contentDispositionMatcher = regexp.MustCompile(
			`\*?filename="([^"]+)|\*?filename='([^']+)|\*?filename=([^;]+)`,
		)
	}
	m := a.contentDispositionMatcher.FindStringSubmatch(contentDisposition)
	for _, s := range m {
		if s != "" {
			return s
		}
	}
	if originalUrl != "" {
		if u, err := url.Parse(originalUrl); err == nil {
			return filepath.Base(u.Path)
		}
	}
	return ""
}

func RecoverService(err *error) {
	if r := recover(); r != nil {
		log.Printf(
			"RECOVERED PANIC: %v\n%s\n",
			r,
			string(debug.Stack()),
		)
		*err = fmt.Errorf("internal error: %+v", r)
	}
}

type SeeUrlError struct {
	Url string
}

var _ error = &SeeUrlError{}

func (err *SeeUrlError) Error() string {
	return "see " + err.Url
}
