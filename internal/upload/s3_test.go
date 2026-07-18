package upload

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/brandonkramer/netcup-cli/internal/scpclient"
)

func TestMultipartUpload(t *testing.T) {
	var parts []int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			w.Header().Set("ETag", fmt.Sprintf(`"etag-%d"`, len(parts)+1))
			parts = append(parts, int32(len(parts)+1))
			w.WriteHeader(200)
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "blob.bin")
	data := make([]byte, DefaultPartSize+100)
	for i := range data {
		data[i] = byte(i)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	uploadID := "u1"
	api := API{
		Prepare: func(ctx context.Context, key string, multipart bool) (*scpclient.S3Upload, error) {
			if !multipart {
				t.Fatal("expected multipart")
			}
			return &scpclient.S3Upload{UploadId: &uploadID}, nil
		},
		PartURL: func(ctx context.Context, key, id string, part int32) (string, error) {
			return srv.URL + fmt.Sprintf("/p/%d", part), nil
		},
		Complete: func(ctx context.Context, key, id string, completed []scpclient.S3CompletedPart) error {
			if len(completed) != 2 {
				t.Fatalf("parts: %d", len(completed))
			}
			return nil
		},
	}
	res, err := File(context.Background(), api, "k", path, DefaultPartSize, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Multipart || res.Parts != 2 || res.Bytes != int64(len(data)) {
		t.Fatalf("%+v", res)
	}
}
