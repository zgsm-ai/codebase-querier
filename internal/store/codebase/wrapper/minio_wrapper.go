package wrapper

import (
	"context"
	"io"

	"github.com/minio/minio-go/v7"
)

// MinioClient defines the interface for MinIO operations
type MinioClient interface {
	PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, size int64, opts minio.PutObjectOptions) (info minio.UploadInfo, err error)
	GetObject(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (*minio.Object, error)
	RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error
	StatObject(ctx context.Context, bucketName, objectName string, opts minio.StatObjectOptions) (minio.ObjectInfo, error)
	ListObjects(ctx context.Context, bucketName string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo
	BucketExists(ctx context.Context, bucketName string) (bool, error)
	MakeBucket(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) error
	RemoveObjects(ctx context.Context, bucket string, ch <-chan minio.ObjectInfo, options minio.RemoveObjectsOptions) <-chan minio.RemoveObjectError
}

// minioClientWrapper wraps the minio.Client to implement MinioClient interface
type minioClientWrapper struct {
	client *minio.Client
}

// NewMinioClientWrapper creates a new wrapper for minio.Client
func NewMinioClientWrapper(client *minio.Client) MinioClient {
	return &minioClientWrapper{client: client}
}

func (w *minioClientWrapper) PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, size int64, opts minio.PutObjectOptions) (info minio.UploadInfo, err error) {
	return w.client.PutObject(ctx, bucketName, objectName, reader, size, opts)
}

func (w *minioClientWrapper) GetObject(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (*minio.Object, error) {
	return w.client.GetObject(ctx, bucketName, objectName, opts)
}

func (w *minioClientWrapper) RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error {
	return w.client.RemoveObject(ctx, bucketName, objectName, opts)
}

func (w *minioClientWrapper) StatObject(ctx context.Context, bucketName, objectName string, opts minio.StatObjectOptions) (minio.ObjectInfo, error) {
	return w.client.StatObject(ctx, bucketName, objectName, opts)
}

func (w *minioClientWrapper) ListObjects(ctx context.Context, bucketName string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
	return w.client.ListObjects(ctx, bucketName, opts)
}

func (w *minioClientWrapper) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	return w.client.BucketExists(ctx, bucketName)
}

func (w *minioClientWrapper) MakeBucket(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) error {
	return w.client.MakeBucket(ctx, bucketName, opts)
}

func (w *minioClientWrapper) RemoveObjects(ctx context.Context, bucket string, ch <-chan minio.ObjectInfo, options minio.RemoveObjectsOptions) <-chan minio.RemoveObjectError {
	return w.client.RemoveObjects(ctx, bucket, ch, options)
}
