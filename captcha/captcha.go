package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang/freetype/truetype"
	"github.com/wenlng/go-captcha-assets/bindata/chars"
	"github.com/wenlng/go-captcha-assets/helper"
	"github.com/wenlng/go-captcha-assets/resources/fonts/fzshengsksjw"
	"github.com/wenlng/go-captcha-assets/resources/images"
	"github.com/wenlng/go-captcha-assets/resources/tiles"
	"github.com/wenlng/go-captcha/v2/base/option"
	"github.com/wenlng/go-captcha/v2/click"
	"github.com/wenlng/go-captcha/v2/rotate"
	"github.com/wenlng/go-captcha/v2/slide"
)

// Client 负责生成和校验验证码，缓存细节由 Store 实现提供。
type Client struct {
	ctx   context.Context
	store Store
	ttl   time.Duration
}

// NewCaptcha 创建验证码客户端。opts 可用于覆盖默认缓存有效期等配置。
func NewCaptcha(ctx context.Context, store Store, opts ...Option) *Client {
	client := &Client{
		ctx:   ctx,
		store: store,
		ttl:   DefaultTTL,
	}
	for _, opt := range opts {
		opt(client)
	}
	return client
}

func (l *Client) ClickCapt() (err error, resp map[string]interface{}) {
	builder := click.NewBuilder()

	// fonts
	fonts, err := fzshengsksjw.GetFont()
	if err != nil {
		return err, nil
	}

	// background images
	imgs, err := images.GetImages()
	if err != nil {
		return err, nil
	}

	builder.SetResources(
		click.WithChars(chars.GetChineseChars()),
		click.WithFonts([]*truetype.Font{fonts}),
		click.WithBackgrounds(imgs),
	)

	textCapt := builder.Make()

	captData, err := textCapt.Generate()
	if err != nil {
		return err, nil
	}

	dotData := captData.GetData()
	if dotData == nil {
		return err, nil
	}

	dotsByte, _ := json.Marshal(dotData)
	key := helper.StringToMD5(string(dotsByte))
	if err := l.saveData(key, string(dotsByte)); err != nil {
		return err, nil
	}

	masterImageBase64, _ := captData.GetMasterImage().ToBase64()
	tileImageBase64, _ := captData.GetThumbImage().ToBase64()

	return nil, map[string]interface{}{
		"captcha_key":  key,
		"image_base64": masterImageBase64,
		"thumb_base64": tileImageBase64,
	}
}

func (l *Client) SlideCapt() (err error, resp map[string]interface{}) {
	builder := slide.NewBuilder(
		//slide.WithGenGraphNumber(2),
		slide.WithEnableGraphVerticalRandom(true),
	)

	// background images
	imgs, err := images.GetImages()
	if err != nil {
		return err, nil
	}

	graphs, err := tiles.GetTiles()
	if err != nil {
		return err, nil
	}

	var newGraphs = make([]*slide.GraphImage, 0, len(graphs))
	for i := 0; i < len(graphs); i++ {
		graph := graphs[i]
		newGraphs = append(newGraphs, &slide.GraphImage{
			OverlayImage: graph.OverlayImage,
			MaskImage:    graph.MaskImage,
			ShadowImage:  graph.ShadowImage,
		})
	}

	// set resources
	builder.SetResources(
		slide.WithGraphImages(newGraphs),
		slide.WithBackgrounds(imgs),
	)

	slideCapt := builder.Make()

	captData, err := slideCapt.Generate()
	if err != nil {
		return err, nil
	}

	dotData := captData.GetData()
	if dotData == nil {
		return err, nil
	}

	dotsByte, _ := json.Marshal(dotData)
	key := helper.StringToMD5(string(dotsByte))
	if err := l.saveData(key, string(dotsByte)); err != nil {
		return err, nil
	}

	masterImageBase64, _ := captData.GetMasterImage().ToBase64()
	tileImageBase64, _ := captData.GetTileImage().ToBase64()

	return nil, map[string]interface{}{
		"captcha_key":  key,
		"image_base64": masterImageBase64,
		"tile_base64":  tileImageBase64,
		"tile_width":   dotData.Width,
		"tile_height":  dotData.Height,
		"tile_x":       dotData.TileX,
		"tile_y":       dotData.TileY,
	}
}

func (l *Client) RotateCapt() (err error, resp map[string]interface{}) {
	builder := rotate.NewBuilder(rotate.WithRangeAnglePos([]option.RangeVal{
		{Min: 20, Max: 330},
	}))

	// background images
	imgs, err := images.GetImages()
	if err != nil {
		return err, nil
	}

	// set resources
	builder.SetResources(
		rotate.WithImages(imgs),
	)

	rotateCapt := builder.Make()

	captData, err := rotateCapt.Generate()
	if err != nil {
		return err, nil
	}

	dotData := captData.GetData()
	if dotData == nil {
		return err, nil
	}

	dotsByte, _ := json.Marshal(dotData)
	key := helper.StringToMD5(string(dotsByte))
	if err := l.saveData(key, string(dotsByte)); err != nil {
		return err, nil
	}

	masterImageBase64, _ := captData.GetMasterImage().ToBase64()
	thumbImageBase64, _ := captData.GetThumbImage().ToBase64()

	return nil, map[string]interface{}{
		"captcha_key":  key,
		"image_base64": masterImageBase64,
		"thumb_base64": thumbImageBase64,
	}
}

func (l *Client) CheckData(points string) error {
	pd := strings.Split(points, "|")
	if len(pd) != 3 {
		return fmt.Errorf("参数有误")
	}
	if pd[0] == "click" {
		return l.CheckClickData(pd[1], pd[2])
	} else if pd[0] == "slide" {
		return l.CheckSlideData(pd[1], pd[2])
	} else if pd[0] == "rotate" {
		return l.CheckRotateData(pd[1], pd[2])
	}
	return fmt.Errorf("类型有误")
}

func (l *Client) CheckClickData(key, points string) error {
	var dct map[int]*click.Dot
	cacheDataByte, err := l.loadData(key)
	if err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(cacheDataByte), &dct); err != nil {
		return err
	}

	src := strings.Split(points, ",")

	chkRet := false
	if (len(dct) * 2) == len(src) {
		for i := 0; i < len(dct); i++ {
			dot := dct[i]
			j := i * 2
			k := i*2 + 1
			sx, _ := strconv.ParseFloat(fmt.Sprintf("%v", src[j]), 64)
			sy, _ := strconv.ParseFloat(fmt.Sprintf("%v", src[k]), 64)

			chkRet = click.CheckPoint(int64(sx), int64(sy), int64(dot.X), int64(dot.Y), int64(dot.Width), int64(dot.Height), 0)
			if !chkRet {
				break
			}
		}
	}

	if chkRet {
		return nil
	}
	return fmt.Errorf("验证码验证失败")
}

func (l *Client) CheckSlideData(key, points string) error {
	var dct *slide.Block
	cacheDataByte, err := l.loadData(key)
	if err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(cacheDataByte), &dct); err != nil {
		return err
	}

	src := strings.Split(points, ",")

	chkRet := false
	if 2 == len(src) {
		sx, _ := strconv.ParseFloat(fmt.Sprintf("%v", src[0]), 64)
		sy, _ := strconv.ParseFloat(fmt.Sprintf("%v", src[1]), 64)
		chkRet = slide.CheckPoint(int64(sx), int64(sy), int64(dct.X), int64(dct.Y), 4)
	}

	if chkRet {
		return nil
	}
	return fmt.Errorf("验证码验证失败")
}

func (l *Client) CheckRotateData(key, points string) error {
	var dct *rotate.Block
	cacheDataByte, err := l.loadData(key)
	if err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(cacheDataByte), &dct); err != nil {
		return err
	}

	sAngle, _ := strconv.ParseFloat(fmt.Sprintf("%v", points), 64)
	chkRet := rotate.CheckAngle(int64(sAngle), int64(dct.Angle), 2)

	if chkRet {
		return nil
	}
	return fmt.Errorf("验证码验证失败")
}

func (l *Client) saveData(key, value string) error {
	if l.store == nil {
		return ErrMissingStore
	}
	return l.store.SetNX(l.ctx, key, value, l.ttl)
}

func (l *Client) loadData(key string) (string, error) {
	if l.store == nil {
		return "", ErrMissingStore
	}
	return l.store.Get(l.ctx, key)
}
