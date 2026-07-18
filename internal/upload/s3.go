package upload

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/brandonkramer/netcup-cli/internal/scpclient"
)

const DefaultPartSize = 8 << 20 // 8 MiB (S3 min 5 MiB except last)

type ProgressFunc func(part, totalParts int, uploaded, total int64)

type API struct {
	Prepare  func(ctx context.Context, key string, multipart bool) (*scpclient.S3Upload, error)
	PartURL  func(ctx context.Context, key, uploadID string, part int32) (string, error)
	Complete func(ctx context.Context, key, uploadID string, parts []scpclient.S3CompletedPart) error
}

type Result struct {
	Key       string                      `json:"key"`
	UploadID  string                      `json:"upload_id,omitempty"`
	Multipart bool                        `json:"multipart"`
	Bytes     int64                       `json:"bytes"`
	Parts     int                         `json:"parts,omitempty"`
	Completed []scpclient.S3CompletedPart `json:"completed_parts,omitempty"`
	PartSize  int64                       `json:"part_size,omitempty"`
}

func File(ctx context.Context, api API, key, path string, partSize int64, onProgress ProgressFunc) (*Result, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	size := fi.Size()
	if size == 0 {
		return nil, fmt.Errorf("empty file")
	}
	if partSize <= 0 {
		partSize = DefaultPartSize
	}

	multipart := size > partSize
	up, err := api.Prepare(ctx, key, multipart)
	if err != nil {
		return nil, err
	}

	if !multipart {
		if up.PresignedUrl != nil && *up.PresignedUrl != "" {
			if err := putPresigned(ctx, *up.PresignedUrl, path, size, onProgress); err != nil {
				return nil, err
			}
			return &Result{Key: key, Multipart: false, Bytes: size}, nil
		}
		// Fall back to multipart if server only returned uploadId.
		if up.UploadId == nil || *up.UploadId == "" {
			return nil, fmt.Errorf("prepare upload: missing presignedUrl and uploadId")
		}
		multipart = true
	}

	if up.UploadId == nil || *up.UploadId == "" {
		return nil, fmt.Errorf("prepare multipart: missing uploadId")
	}
	uploadID := *up.UploadId

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	totalParts := int((size + partSize - 1) / partSize)
	parts := make([]scpclient.S3CompletedPart, 0, totalParts)
	var uploaded int64
	buf := make([]byte, partSize)

	for n := int32(1); ; n++ {
		nr, readErr := io.ReadFull(f, buf)
		if nr == 0 && readErr == io.EOF {
			break
		}
		if readErr != nil && readErr != io.ErrUnexpectedEOF && readErr != io.EOF {
			return nil, readErr
		}
		url, err := api.PartURL(ctx, key, uploadID, n)
		if err != nil {
			return nil, fmt.Errorf("part %d url: %w", n, err)
		}
		etag, err := putBytes(ctx, url, buf[:nr])
		if err != nil {
			return nil, fmt.Errorf("part %d upload: %w", n, err)
		}
		pn, et := n, etag
		parts = append(parts, scpclient.S3CompletedPart{PartNumber: &pn, ETag: &et})
		uploaded += int64(nr)
		if onProgress != nil {
			onProgress(int(n), totalParts, uploaded, size)
		}
		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			break
		}
	}

	if err := api.Complete(ctx, key, uploadID, parts); err != nil {
		return nil, err
	}
	return &Result{
		Key:       key,
		UploadID:  uploadID,
		Multipart: true,
		Bytes:     size,
		Parts:     len(parts),
		Completed: parts,
		PartSize:  partSize,
	}, nil
}

func putPresigned(ctx context.Context, url, path string, size int64, onProgress ProgressFunc) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, f)
	if err != nil {
		return err
	}
	req.ContentLength = size
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("presigned PUT HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if onProgress != nil {
		onProgress(1, 1, size, size)
	}
	return nil
}

func putBytes(ctx context.Context, url string, data []byte) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.ContentLength = int64(len(data))
	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	etag := resp.Header.Get("ETag")
	if etag == "" {
		etag = resp.Header.Get("Etag")
	}
	if etag == "" {
		return "", fmt.Errorf("missing ETag in part upload response")
	}
	return etag, nil
}
