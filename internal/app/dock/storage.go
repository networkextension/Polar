package dock

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// AttachmentStorage is the interface for storing chat attachment files.
// Implementations: LocalAttachmentStorage (default) and R2AttachmentStorage (Cloudflare R2).
type AttachmentStorage interface {
	// Store saves a locally-staged file to the backing store and returns its public URL.
	// localPath is the path where the file has already been written to disk.
	// filename is the desired storage key / base filename.
	// mimeType is the content-type of the file.
	Store(ctx context.Context, localPath, filename, mimeType string) (publicURL string, err error)

	// IsRemote returns true when files are stored in remote object storage.
	// When true the caller is responsible for removing local staging files
	// after Store() succeeds.
	IsRemote() bool
}

// ─── Local storage ────────────────────────────────────────────────────────────

// LocalAttachmentStorage stores files on the local filesystem inside uploadDir.
// Files are already written to uploadDir by the handler before Store is called,
// so this implementation simply returns the public URL path.
type LocalAttachmentStorage struct {
	uploadDir string // absolute path, e.g. "data/uploads"
}

func NewLocalAttachmentStorage(uploadDir string) *LocalAttachmentStorage {
	return &LocalAttachmentStorage{uploadDir: uploadDir}
}

// Store returns the public URL for a file that is already in uploadDir.
func (s *LocalAttachmentStorage) Store(_ context.Context, _, filename, _ string) (string, error) {
	return "/uploads/" + filename, nil
}

func (s *LocalAttachmentStorage) IsRemote() bool { return false }

// ─── Cloudflare R2 storage ─────────────────────────────────────────────────

// R2AttachmentStorage uploads files to Cloudflare R2 (S3-compatible) and
// returns public URLs based on a configurable publicBase.
type R2AttachmentStorage struct {
	client     *s3.Client
	bucket     string
	publicBase string // e.g. "https://pub-xxxx.r2.dev" — no trailing slash
}

// NewR2AttachmentStorage creates an R2AttachmentStorage.
// accountID:   Cloudflare account ID
// accessKeyID: R2 access key ID
// secretKey:   R2 secret access key
// bucket:      R2 bucket name
// publicBase:  public URL base, e.g. "https://pub-xxxx.r2.dev" or a custom domain
func NewR2AttachmentStorage(accountID, accessKeyID, secretKey, bucket, publicBase string) (*R2AttachmentStorage, error) {
	if accountID == "" || accessKeyID == "" || secretKey == "" || bucket == "" || publicBase == "" {
		return nil, fmt.Errorf("storage: all Cloudflare R2 parameters are required")
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

// Store uploads the file at localPath to R2 and returns the public URL.
func (s *R2AttachmentStorage) Store(ctx context.Context, localPath, filename, mimeType string) (string, error) {
	f, err := os.Open(localPath)
	if err != nil {
		return "", fmt.Errorf("r2 open local file: %w", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("r2 stat local file: %w", err)
	}

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(filename),
		Body:          f,
		ContentLength: aws.Int64(stat.Size()),
		ContentType:   aws.String(mimeType),
	})
	if err != nil {
		return "", fmt.Errorf("r2 put object %q: %w", filename, err)
	}

	return s.publicBase + "/" + filename, nil
}

func (s *R2AttachmentStorage) IsRemote() bool { return true }

// ─── Constructor helper ────────────────────────────────────────────────────

// newAttachmentStorage returns the appropriate AttachmentStorage implementation.
// If all Cloudflare R2 parameters are provided it returns an R2AttachmentStorage,
// otherwise it falls back to LocalAttachmentStorage.
func newAttachmentStorage(uploadDir string, cfg Config) (AttachmentStorage, error) {
	r2Configured := cfg.CloudflareR2AccountID != "" &&
		cfg.CloudflareR2AccessKeyID != "" &&
		cfg.CloudflareR2SecretAccessKey != "" &&
		cfg.CloudflareR2Bucket != "" &&
		cfg.CloudflareR2PublicURL != ""

	if r2Configured {
		return NewR2AttachmentStorage(
			cfg.CloudflareR2AccountID,
			cfg.CloudflareR2AccessKeyID,
			cfg.CloudflareR2SecretAccessKey,
			cfg.CloudflareR2Bucket,
			cfg.CloudflareR2PublicURL,
		)
	}

	return NewLocalAttachmentStorage(uploadDir), nil
}

// removeLocalFile is a best-effort helper to delete a staging file.
func removeLocalFile(path string) {
	if path != "" {
		_ = os.Remove(path)
	}
}

// storeAttachmentFiles saves the main file and any generated variants (e.g.
// thumbnails) to the AttachmentStorage.  It returns the public URL for the main
// file and a map of localPath→publicURL for each extra path (variant).
//
// For remote storage the local staging files in extraLocalPaths are deleted
// after a successful upload.  The caller is responsible for removing the main
// localPath after this function returns when storage.IsRemote() is true.
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
		url, uploadErr := storage.Store(ctx, p, fn, "image/jpeg")
		if uploadErr == nil {
			extraURLs[p] = url
		}
		if storage.IsRemote() {
			removeLocalFile(p)
		}
	}

	return mainURL, extraURLs, nil
}
