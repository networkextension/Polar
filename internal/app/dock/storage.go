package dock

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// AttachmentStorage is the interface for storing and retrieving chat attachment files.
// Two implementations are provided:
//   - LocalAttachmentStorage  – files stay on the local filesystem (default)
//   - R2AttachmentStorage     – files are uploaded to Cloudflare R2
type AttachmentStorage interface {
	// Store saves a locally-staged file to the backing store and returns the
	// URL that clients should use to download the file.
	// localPath  – path where the file has already been written to disk
	// filename   – desired storage key / base filename
	// mimeType   – content-type of the file
	Store(ctx context.Context, localPath, filename, mimeType string) (downloadURL string, err error)

	// GetObject fetches the raw file bytes from the store.
	// Returns (body, contentLength, contentType, error).
	// Used by the backend proxy endpoint for remote stores; local stores
	// are served directly by the static-file handler and never call this.
	GetObject(ctx context.Context, filename string) (body io.ReadCloser, size int64, contentType string, err error)

	// IsRemote returns true when files are stored in remote object storage.
	// When true the caller is responsible for removing local staging files
	// after Store() succeeds.
	IsRemote() bool
}

// ─── Local storage ─────────────────────────────────────────────────────────

// LocalAttachmentStorage stores files on the local filesystem inside uploadDir.
// The handler already writes the file to uploadDir before calling Store, so
// Store just returns the static /uploads/ URL.
type LocalAttachmentStorage struct {
	uploadDir string
}

func NewLocalAttachmentStorage(uploadDir string) *LocalAttachmentStorage {
	return &LocalAttachmentStorage{uploadDir: uploadDir}
}

func (s *LocalAttachmentStorage) Store(_ context.Context, _, filename, _ string) (string, error) {
	return "/uploads/" + filename, nil
}

func (s *LocalAttachmentStorage) GetObject(_ context.Context, filename string) (io.ReadCloser, int64, string, error) {
	path := filepath.Join(s.uploadDir, filename)
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, "", err
	}
	stat, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, 0, "", err
	}
	return f, stat.Size(), "", nil // caller detects content-type
}

func (s *LocalAttachmentStorage) IsRemote() bool { return false }

// ─── Cloudflare R2 storage ─────────────────────────────────────────────────

// R2AttachmentStorage uploads files to Cloudflare R2 (S3-compatible).
//
// Download URL strategy (decided at construction time):
//   - If publicBase is set (CF_R2_PUBLIC_URL) → use "<publicBase>/<filename>"
//     (the R2 bucket must have public-access enabled, or a CDN in front).
//   - If publicBase is empty → use "/chat-files/<filename>"
//     (the backend proxy endpoint fetches from R2 with credentials and streams
//     the response; the R2 bucket does NOT need to be public).
type R2AttachmentStorage struct {
	client     *s3.Client
	bucket     string
	publicBase string // optional, e.g. "https://pub-xxxx.r2.dev" — no trailing slash
}

// NewR2AttachmentStorage creates an R2AttachmentStorage.
//   - accountID   : Cloudflare account ID
//   - accessKeyID : R2 access key ID
//   - secretKey   : R2 secret access key
//   - bucket      : R2 bucket name
//   - publicBase  : optional public URL base; empty = proxy mode
func NewR2AttachmentStorage(accountID, accessKeyID, secretKey, bucket, publicBase string) (*R2AttachmentStorage, error) {
	if accountID == "" || accessKeyID == "" || secretKey == "" || bucket == "" {
		return nil, fmt.Errorf("storage: accountID, accessKeyID, secretKey and bucket are required for R2")
	}

	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)
	creds := credentials.NewStaticCredentialsProvider(accessKeyID, secretKey, "")
	cfg := aws.Config{
		Region:      "auto",
		Credentials: creds,
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})

	return &R2AttachmentStorage{
		client:     client,
		bucket:     bucket,
		publicBase: strings.TrimRight(publicBase, "/"),
	}, nil
}

// Store uploads the file at localPath to R2 and returns the appropriate download URL.
func (s *R2AttachmentStorage) Store(ctx context.Context, localPath, filename, mimeType string) (string, error) {
	f, err := os.Open(localPath)
	if err != nil {
		return "", fmt.Errorf("r2: open local file: %w", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("r2: stat local file: %w", err)
	}

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(filename),
		Body:          f,
		ContentLength: aws.Int64(stat.Size()),
		ContentType:   aws.String(mimeType),
	})
	if err != nil {
		return "", fmt.Errorf("r2: put object %q: %w", filename, err)
	}

	if s.publicBase != "" {
		// Direct CDN / public-bucket access.
		return s.publicBase + "/" + filename, nil
	}
	// Proxy mode: backend fetches from R2 on behalf of the client.
	return "/chat-files/" + filename, nil
}

// GetObject fetches the object from R2 so the backend proxy can stream it.
func (s *R2AttachmentStorage) GetObject(ctx context.Context, filename string) (io.ReadCloser, int64, string, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(filename),
	})
	if err != nil {
		return nil, 0, "", fmt.Errorf("r2: get object %q: %w", filename, err)
	}
	var size int64
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	ct := ""
	if out.ContentType != nil {
		ct = *out.ContentType
	}
	return out.Body, size, ct, nil
}

func (s *R2AttachmentStorage) IsRemote() bool { return true }

// ─── Constructor helper ────────────────────────────────────────────────────

// newAttachmentStorage returns the appropriate AttachmentStorage backend.
//
// R2 is selected when CF_R2_ACCOUNT_ID, CF_R2_ACCESS_KEY_ID,
// CF_R2_SECRET_ACCESS_KEY and CF_R2_BUCKET are all set.
// CF_R2_PUBLIC_URL is optional:
//   - set  → files are served directly from the CDN/public bucket URL
//   - unset → files are proxied through the /chat-files/:filename endpoint
//
// If any of the four required R2 fields is missing the server falls back to
// local filesystem storage.
func newAttachmentStorage(uploadDir string, cfg Config) (AttachmentStorage, error) {
	r2Configured := cfg.CloudflareR2AccountID != "" &&
		cfg.CloudflareR2AccessKeyID != "" &&
		cfg.CloudflareR2SecretAccessKey != "" &&
		cfg.CloudflareR2Bucket != ""

	if r2Configured {
		return NewR2AttachmentStorage(
			cfg.CloudflareR2AccountID,
			cfg.CloudflareR2AccessKeyID,
			cfg.CloudflareR2SecretAccessKey,
			cfg.CloudflareR2Bucket,
			cfg.CloudflareR2PublicURL, // empty = proxy mode
		)
	}

	return NewLocalAttachmentStorage(uploadDir), nil
}

// ─── Helpers ───────────────────────────────────────────────────────────────

// removeLocalFile is a best-effort helper to delete a local staging file.
func removeLocalFile(path string) {
	if path != "" {
		_ = os.Remove(path)
	}
}

// storeAttachmentFiles saves the main file and any generated variant files
// (e.g. image thumbnails) to AttachmentStorage.
//
// Returns the public URL for the main file and a map of localPath→URL for
// each extra variant.  For remote storage, local variant files are deleted
// after successful upload; the caller is responsible for removing the main
// localPath when storage.IsRemote() is true.
func storeAttachmentFiles(
	ctx context.Context,
	storage AttachmentStorage,
	mainLocalPath, mainFilename, mainMIME string,
	extraLocalPaths []string,
) (mainURL string, extraURLs map[string]string, err error) {
	mainURL, err = storage.Store(ctx, mainLocalPath, mainFilename, mainMIME)
	if err != nil {
		return "", nil, err
	}

	extraURLs = make(map[string]string, len(extraLocalPaths))
	for _, p := range extraLocalPaths {
		fn := filepath.Base(p)
		u, uploadErr := storage.Store(ctx, p, fn, "image/jpeg")
		if uploadErr == nil {
			extraURLs[p] = u
		}
		if storage.IsRemote() {
			removeLocalFile(p)
		}
	}

	return mainURL, extraURLs, nil
}
