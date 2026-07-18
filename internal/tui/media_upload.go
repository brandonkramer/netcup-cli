package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/brandonkramer/netcup-cli/internal/scpclient"
	"github.com/brandonkramer/netcup-cli/internal/upload"
	tea "github.com/charmbracelet/bubbletea"
)

func uploadMediaCmd(ctx context.Context, c *scpclient.ClientWithResponses, uid int32, kind, path string) tea.Cmd {
	return func() tea.Msg {
		key := filepath.Base(path)
		api := upload.API{
			Prepare: func(ctx context.Context, key string, multipart bool) (*scpclient.S3Upload, error) {
				if kind == "iso" {
					r, e := c.PostApiV1UsersUserIdIsosKeyWithResponse(ctx, uid, key, &scpclient.PostApiV1UsersUserIdIsosKeyParams{Multipart: &multipart})
					if e != nil {
						return nil, e
					}
					if r == nil {
						return nil, tuiError("empty response")
					}
					if r.StatusCode() != 201 {
						return nil, apiStatusErr("prepare ISO upload", r.StatusCode(), r.Body)
					}
					return decodeUpload(r.JSON201, r.HALJSON201, r.Body)
				}
				r, e := c.PostApiV1UsersUserIdImagesKeyWithResponse(ctx, uid, key, &scpclient.PostApiV1UsersUserIdImagesKeyParams{Multipart: &multipart})
				if e != nil {
					return nil, e
				}
				if r == nil {
					return nil, tuiError("empty response")
				}
				if r.StatusCode() != 201 {
					return nil, apiStatusErr("prepare image upload", r.StatusCode(), r.Body)
				}
				return decodeUpload(r.JSON201, r.HALJSON201, r.Body)
			},
			PartURL: func(ctx context.Context, key, id string, part int32) (string, error) {
				if kind == "iso" {
					r, e := c.GetApiV1UsersUserIdIsosKeyUploadIdPartsPartNumberWithResponse(ctx, uid, key, id, part)
					if e != nil {
						return "", e
					}
					if r == nil {
						return "", tuiError("empty response")
					}
					return decodePartURL(r.JSON200, r.HALJSON200, r.Body)
				}
				r, e := c.GetApiV1UsersUserIdImagesKeyUploadIdPartsPartNumberWithResponse(ctx, uid, key, id, part)
				if e != nil {
					return "", e
				}
				if r == nil {
					return "", tuiError("empty response")
				}
				return decodePartURL(r.JSON200, r.HALJSON200, r.Body)
			},
			Complete: func(ctx context.Context, key, id string, parts []scpclient.S3CompletedPart) error {
				if kind == "iso" {
					r, e := c.PutApiV1UsersUserIdIsosKeyUploadIdWithResponse(ctx, uid, key, id, parts)
					if e != nil {
						return e
					}
					if r == nil {
						return tuiError("empty response")
					}
					if r.StatusCode() != 200 && r.StatusCode() != 204 {
						return apiStatusErr("complete ISO upload", r.StatusCode(), r.Body)
					}
					return nil
				}
				r, e := c.PutApiV1UsersUserIdImagesKeyUploadIdWithResponse(ctx, uid, key, id, parts)
				if e != nil {
					return e
				}
				if r == nil {
					return tuiError("empty response")
				}
				if r.StatusCode() != 200 && r.StatusCode() != 204 {
					return apiStatusErr("complete image upload", r.StatusCode(), r.Body)
				}
				return nil
			},
		}
		_, err := upload.File(ctx, api, key, path, upload.DefaultPartSize, func(part, total int, uploaded, size int64) {})
		return resourceActionMsg{action: kind + "-upload", err: err}
	}
}
func decodeUpload(a, b *scpclient.S3Upload, body []byte) (*scpclient.S3Upload, error) {
	if a != nil {
		return a, nil
	}
	if b != nil {
		return b, nil
	}
	var out scpclient.S3Upload
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
func decodePartURL(a, b *scpclient.S3SignPartURL, body []byte) (string, error) {
	v := a
	if v == nil {
		v = b
	}
	if v == nil {
		v = &scpclient.S3SignPartURL{}
		if err := json.Unmarshal(body, v); err != nil {
			return "", err
		}
	}
	if v.Url == nil {
		return "", fmt.Errorf("empty part URL")
	}
	return *v.Url, nil
}
func deleteMediaCmd(ctx context.Context, c *scpclient.ClientWithResponses, uid int32, kind, key string) tea.Cmd {
	return func() tea.Msg {
		if kind == "iso" {
			r, e := c.DeleteApiV1UsersUserIdIsosKeyWithResponse(ctx, uid, key)
			if e != nil {
				return resourceActionMsg{action: "iso-delete", err: e}
			}
			if r == nil {
				return resourceActionMsg{action: "iso-delete", err: tuiError("empty response")}
			}
			status := r.StatusCode()
			if status != 200 && status != 204 {
				return resourceActionMsg{action: "iso-delete", status: status, err: apiStatusErr("delete ISO", status, r.Body)}
			}
			return resourceActionMsg{action: "iso-delete", status: status}
		}
		r, e := c.DeleteApiV1UsersUserIdImagesKeyWithResponse(ctx, uid, key)
		if e != nil {
			return resourceActionMsg{action: "image-delete", err: e}
		}
		if r == nil {
			return resourceActionMsg{action: "image-delete", err: tuiError("empty response")}
		}
		status := r.StatusCode()
		if status != 200 && status != 204 {
			return resourceActionMsg{action: "image-delete", status: status, err: apiStatusErr("delete image", status, r.Body)}
		}
		return resourceActionMsg{action: "image-delete", status: status}
	}
}
