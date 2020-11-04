package aslince

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/gographics/imagick.v2/imagick"
	tb "gopkg.in/tucnak/telebot.v2"
)

func (a *Aslince) paint(m *tb.Message) (*tb.Photo, error) {
	if m.Photo.FileID == "" {
		return nil, errors.New("empty file id")
	}
	imgURL, err := a.FileURLByID(m.Photo.FileID)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient().Get(imgURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, errors.New("status not ok")
	}

	rawBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	mw := imagick.NewMagickWand()
	defer mw.Destroy()

	err = mw.ReadImageBlob(rawBody)
	if err != nil {
		return nil, errors.Wrap(err, "error reading image")
	}

	err = mw.SetImageFormat("JPG")
	if err != nil {
		return nil, errors.Wrap(err, "error setting format")
	}

	mw.SetImageVirtualPixelMethod(imagick.VIRTUAL_PIXEL_DITHER)

	mw.DistortImage(imagick.DISTORTION_BARREL, []float64{0.3, 0.0, 0.0, 0.3}, true)

	err = mw.ModulateImage(100, 100, 166.3)
	if err != nil {
		return nil, errors.Wrap(err, "error modulating")
	}

	return &tb.Photo{File: tb.FromReader(bytes.NewBuffer(mw.GetImageBlob()))}, nil
}

func httpClient() *http.Client {
	client := http.Client{
		Timeout: 2 * time.Second,
	}
	return &client
}
