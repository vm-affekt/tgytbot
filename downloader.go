package main

import (
	"context"
	"fmt"
	"github.com/kkdai/youtube/v2"
	"io"
	"log"
	"net/http"
	"strconv"
)

// ErrUnexpectedStatusCode is returned on unexpected HTTP status codes
type ErrUnexpectedStatusCode int

func (err ErrUnexpectedStatusCode) Error() string {
	return fmt.Sprintf("unexpected status code: %d", err)
}

// GetStreamContext returns the stream and the total size for a specific format with a context.
func GetStreamContext(ctx context.Context, client *youtube.Client, video *youtube.Video, format *youtube.Format) (io.ReadCloser, int64, error) {
	url, err := client.GetStreamURL(video, format)
	if err != nil {
		return nil, 0, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36")

	r, w := io.Pipe()
	contentLength := format.ContentLength

	//contentLength = downloadOnce(client, req, w, format)

	if contentLength == 0 {
		// some videos don't have length information
		contentLength = downloadOnce(client, req, w, format)
	} else {
		// we have length information, let's download by chunks!
		go downloadChunked(client, req, w, format)
	}

	return r, contentLength, nil
}

func downloadOnce(c *youtube.Client, req *http.Request, w *io.PipeWriter, format *youtube.Format) int64 {
	resp, err := httpDo(c, req)
	if err != nil {
		//nolint:errcheck
		w.CloseWithError(err)
		return 0
	}

	go func() {
		defer resp.Body.Close()
		_, err := io.Copy(w, resp.Body)
		if err == nil {
			w.Close()
		} else {
			//nolint:errcheck
			w.CloseWithError(err)
		}
	}()

	contentLength := resp.Header.Get("Content-Length")
	len, _ := strconv.ParseInt(contentLength, 10, 64)

	return len
}

func downloadChunked(c *youtube.Client, req *http.Request, w *io.PipeWriter, format *youtube.Format) {
	const chunkSize int64 = 10_000_000
	// Loads a chunk a returns the written bytes.
	// Downloading in multiple chunks is much faster:
	// https://github.com/kkdai/youtube/pull/190
	loadChunk := func(pos int64) (int64, error) {
		req.Header.Set("Range", fmt.Sprintf("bytes=%v-%v", pos, pos+chunkSize-1))

		resp, err := httpDo(c, req)
		if err != nil {
			return 0, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusPartialContent {
			return 0, ErrUnexpectedStatusCode(resp.StatusCode)
		}

		return io.Copy(w, resp.Body)
	}

	defer w.Close()

	//nolint:revive,errcheck
	// load all the chunks
	for pos := int64(0); pos < format.ContentLength; {
		written, err := loadChunk(pos)
		if err != nil {
			w.CloseWithError(err)
			return
		}

		pos += written
	}
}

// httpDo sends an HTTP request and returns an HTTP response.
func httpDo(c *youtube.Client, req *http.Request) (*http.Response, error) {
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	if c.Debug {
		log.Println(req.Method, req.URL)
	}

	res, err := client.Do(req)

	if c.Debug && res != nil {
		log.Println(res.Status)
	}

	return res, err
}
