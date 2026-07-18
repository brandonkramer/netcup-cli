package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brandonkramer/netcup-cli/internal/output"
	"github.com/brandonkramer/netcup-cli/internal/scpclient"
	"github.com/brandonkramer/netcup-cli/internal/upload"
)

type s3Kind string

const (
	s3ISO   s3Kind = "iso"
	s3Image s3Kind = "image"
)

func s3API(kind s3Kind, userID int32) upload.API {
	return upload.API{
		Prepare: func(ctx context.Context, key string, multipart bool) (*scpclient.S3Upload, error) {
			mp := multipart
			switch kind {
			case s3ISO:
				resp, err := app.Client.PostApiV1UsersUserIdIsosKeyWithResponse(ctx, userID, key, &scpclient.PostApiV1UsersUserIdIsosKeyParams{Multipart: &mp})
				if err != nil {
					return nil, err
				}
				if resp.StatusCode() != 201 {
					return nil, fmt.Errorf("prepare ISO upload: HTTP %d: %s", resp.StatusCode(), truncate(string(resp.Body), 300))
				}
				return decodeS3Upload(resp.JSON201, resp.HALJSON201, resp.Body)
			default:
				resp, err := app.Client.PostApiV1UsersUserIdImagesKeyWithResponse(ctx, userID, key, &scpclient.PostApiV1UsersUserIdImagesKeyParams{Multipart: &mp})
				if err != nil {
					return nil, err
				}
				if resp.StatusCode() != 201 {
					return nil, fmt.Errorf("prepare image upload: HTTP %d: %s", resp.StatusCode(), truncate(string(resp.Body), 300))
				}
				return decodeS3Upload(resp.JSON201, resp.HALJSON201, resp.Body)
			}
		},
		PartURL: func(ctx context.Context, key, uploadID string, part int32) (string, error) {
			switch kind {
			case s3ISO:
				resp, err := app.Client.GetApiV1UsersUserIdIsosKeyUploadIdPartsPartNumberWithResponse(ctx, userID, key, uploadID, part)
				if err != nil {
					return "", err
				}
				if resp.StatusCode() != 200 {
					return "", fmt.Errorf("part URL: HTTP %d: %s", resp.StatusCode(), truncate(string(resp.Body), 300))
				}
				return decodePartURL(resp.JSON200, resp.HALJSON200, resp.Body)
			default:
				resp, err := app.Client.GetApiV1UsersUserIdImagesKeyUploadIdPartsPartNumberWithResponse(ctx, userID, key, uploadID, part)
				if err != nil {
					return "", err
				}
				if resp.StatusCode() != 200 {
					return "", fmt.Errorf("part URL: HTTP %d: %s", resp.StatusCode(), truncate(string(resp.Body), 300))
				}
				return decodePartURL(resp.JSON200, resp.HALJSON200, resp.Body)
			}
		},
		Complete: func(ctx context.Context, key, uploadID string, parts []scpclient.S3CompletedPart) error {
			switch kind {
			case s3ISO:
				resp, err := app.Client.PutApiV1UsersUserIdIsosKeyUploadIdWithResponse(ctx, userID, key, uploadID, parts)
				if err != nil {
					return err
				}
				if resp.StatusCode() != 200 && resp.StatusCode() != 204 {
					return fmt.Errorf("complete ISO upload: HTTP %d: %s", resp.StatusCode(), truncate(string(resp.Body), 300))
				}
				return nil
			default:
				resp, err := app.Client.PutApiV1UsersUserIdImagesKeyUploadIdWithResponse(ctx, userID, key, uploadID, parts)
				if err != nil {
					return err
				}
				if resp.StatusCode() != 200 && resp.StatusCode() != 204 {
					return fmt.Errorf("complete image upload: HTTP %d: %s", resp.StatusCode(), truncate(string(resp.Body), 300))
				}
				return nil
			}
		},
	}
}

func decodeS3Upload(a, b *scpclient.S3Upload, body []byte) (*scpclient.S3Upload, error) {
	if a != nil {
		return a, nil
	}
	if b != nil {
		return b, nil
	}
	var u scpclient.S3Upload
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

func decodePartURL(a, b *scpclient.S3SignPartURL, body []byte) (string, error) {
	var u *scpclient.S3SignPartURL
	switch {
	case a != nil:
		u = a
	case b != nil:
		u = b
	default:
		var tmp scpclient.S3SignPartURL
		if err := json.Unmarshal(body, &tmp); err != nil {
			return "", err
		}
		u = &tmp
	}
	if u.Url == nil || *u.Url == "" {
		return "", fmt.Errorf("empty part URL")
	}
	return *u.Url, nil
}

func uploadProgress() upload.ProgressFunc {
	return func(part, totalParts int, uploaded, total int64) {
		if app.Out.Format == output.FormatJSON || app.Out.Format == output.FormatJSONL || app.Out.Quiet {
			return
		}
		pct := float64(0)
		if total > 0 {
			pct = 100 * float64(uploaded) / float64(total)
		}
		app.Out.Info(fmt.Sprintf("upload part %d/%d (%.1f%%)", part, totalParts, pct))
	}
}
