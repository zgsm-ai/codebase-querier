package codebase

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codebase/wrapper/mocks"

	"github.com/golang/mock/gomock"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

// mockReadCloser implements io.ReadCloser for testing
type mockReadCloser struct {
	io.Reader
}

func (m *mockReadCloser) Close() error {
	return nil
}

func newMockReadCloser() io.ReadCloser {
	return &mockReadCloser{
		Reader: strings.NewReader("test content"),
	}
}

func setupTestMinioCodebase(t *testing.T) (*mocks.MockMinioClient, *minioCodebase, string) {
	ctrl := gomock.NewController(t)
	mockClient := mocks.NewMockMinioClient(ctrl)

	cfg := config.CodeBaseStoreConf{
		Minio: config.MinioStoreConf{
			Endpoint:        "localhost:9000",
			AccessKeyID:     "minioadmin",
			SecretAccessKey: "minioadmin",
			UseSSL:          false,
			Bucket:          "test-bucket",
		},
	}

	codebase := &minioCodebase{
		cfg:    cfg,
		client: mockClient,
		logger: logx.WithContext(context.Background()),
	}

	// Set up default expectations for Init
	mockClient.EXPECT().
		PutObject(
			gomock.Any(),
			"test-bucket",
			"/test/test-client/test-path/",
			gomock.Any(),
			int64(0),
			gomock.Any(),
		).
		Return(minio.UploadInfo{}, nil).
		AnyTimes()

	ctx := context.Background()
	path, err := codebase.Init(ctx, "test-client", "test-path")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	return mockClient, codebase, path.FullPath
}

func TestMinioCodebase_Init(t *testing.T) {
	tests := []struct {
		name         string
		clientId     string
		codebasePath string
		setupMock    func(*mocks.MockMinioClient)
		want         string
		wantErr      bool
	}{
		{
			name:         "successful initialization",
			clientId:     "test-client",
			codebasePath: "test-path",
			setupMock: func(m *mocks.MockMinioClient) {
				uniquePath := generateUniquePath("test-client", "test-path")
				expectedPath := filepath.Join("test-bucket", uniquePath, "test-path", filepathSlash)
				m.EXPECT().PutObject(gomock.Any(), "test-bucket", expectedPath, gomock.Any(), int64(0), gomock.Any()).Return(minio.UploadInfo{}, nil)
			},
			want:    filepath.Join("test-bucket", generateUniquePath("test-client", "test-path"), "test-path", filepathSlash),
			wantErr: false,
		},
		{
			name:         "empty client id",
			clientId:     "",
			codebasePath: "test-path",
			setupMock:    func(m *mocks.MockMinioClient) {},
			want:         "",
			wantErr:      true,
		},
		{
			name:         "empty codebase path",
			clientId:     "test-client",
			codebasePath: "",
			setupMock:    func(m *mocks.MockMinioClient) {},
			want:         "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := mocks.NewMockMinioClient(ctrl)
			tt.setupMock(mockClient)

			codebase := &minioCodebase{
				client: mockClient,
				cfg: config.CodeBaseStoreConf{
					Minio: config.MinioStoreConf{
						Bucket: "test-bucket",
					},
				},
				logger: logx.WithContext(context.Background()),
			}

			got, err := codebase.Init(context.Background(), tt.clientId, tt.codebasePath)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got.FullPath)
			}
		})
	}
}

func TestMinioCodebase_Add(t *testing.T) {
	tests := []struct {
		name         string
		codebasePath string
		target       string
		setupMock    func(*mocks.MockMinioClient)
		wantErr      bool
	}{
		{
			name:         "successful add",
			codebasePath: filepath.Join("test-bucket", generateUniquePath("test-client", "test-path"), "test-path", filepathSlash),
			target:       "test.txt",
			setupMock: func(m *mocks.MockMinioClient) {
				uniquePath := generateUniquePath("test-client", "test-path")
				expectedPath := filepath.Join("test-bucket", uniquePath, "test-path", "test.txt")
				m.EXPECT().PutObject(gomock.Any(), "test-bucket", expectedPath, gomock.Any(), int64(-1), gomock.Any()).Return(minio.UploadInfo{}, nil)
			},
			wantErr: false,
		},
		{
			name:         "empty codebase path",
			codebasePath: "",
			target:       "test.txt",
			setupMock:    func(m *mocks.MockMinioClient) {},
			wantErr:      true,
		},
		{
			name:         "empty target",
			codebasePath: filepath.Join("test-bucket", generateUniquePath("test-client", "test-path"), "test-path", filepathSlash),
			target:       "",
			setupMock:    func(m *mocks.MockMinioClient) {},
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := mocks.NewMockMinioClient(ctrl)
			tt.setupMock(mockClient)

			codebase := &minioCodebase{
				client: mockClient,
				cfg: config.CodeBaseStoreConf{
					Minio: config.MinioStoreConf{
						Bucket: "test-bucket",
					},
				},
				logger: logx.WithContext(context.Background()),
			}

			err := codebase.Add(context.Background(), tt.codebasePath, strings.NewReader("test content"), tt.target)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMinioCodebase_Delete(t *testing.T) {
	tests := []struct {
		name         string
		codebasePath string
		path         string
		setupMock    func(*mocks.MockMinioClient)
		wantErr      bool
	}{
		{
			name:         "successful delete",
			codebasePath: filepath.Join("test-bucket", generateUniquePath("test-client", "test-path"), "test-path", filepathSlash),
			path:         "test.txt",
			setupMock: func(m *mocks.MockMinioClient) {
				uniquePath := generateUniquePath("test-client", "test-path")
				expectedPath := filepath.Join("test-bucket", uniquePath, "test-path", "test.txt")
				m.EXPECT().RemoveObject(gomock.Any(), "test-bucket", expectedPath, gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name:         "empty codebase path",
			codebasePath: "",
			path:         "test.txt",
			setupMock:    func(m *mocks.MockMinioClient) {},
			wantErr:      true,
		},
		{
			name:         "empty path",
			codebasePath: filepath.Join("test-bucket", generateUniquePath("test-client", "test-path"), "test-path", filepathSlash),
			path:         "",
			setupMock:    func(m *mocks.MockMinioClient) {},
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := mocks.NewMockMinioClient(ctrl)
			tt.setupMock(mockClient)

			codebase := &minioCodebase{
				client: mockClient,
				cfg: config.CodeBaseStoreConf{
					Minio: config.MinioStoreConf{
						Bucket: "test-bucket",
					},
				},
				logger: logx.WithContext(context.Background()),
			}

			err := codebase.Delete(context.Background(), tt.codebasePath, tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMinioCodebase_List(t *testing.T) {
	tests := []struct {
		name         string
		codebasePath string
		dir          string
		option       types.ListOptions
		setupMock    func(*mocks.MockMinioClient)
		want         []*types.FileInfo
		wantErr      bool
	}{
		{
			name:         "successful list",
			codebasePath: filepath.Join("test-bucket", generateUniquePath("test-client", "test-path"), "test-path", filepathSlash),
			dir:          "",
			option:       types.ListOptions{},
			setupMock: func(m *mocks.MockMinioClient) {
				uniquePath := generateUniquePath("test-client", "test-path")
				expectedPath := filepath.Join("test-bucket", uniquePath, "test-path")
				ch := make(chan minio.ObjectInfo, 1)
				ch <- minio.ObjectInfo{
					Key:          filepath.Join(expectedPath, "test.txt"),
					Size:         12,
					LastModified: time.Time{},
				}
				close(ch)
				m.EXPECT().ListObjects(gomock.Any(), "test-bucket", gomock.Any()).Return(ch)
			},
			want: []*types.FileInfo{
				{
					Name:    "test.txt",
					Path:    "test.txt",
					Size:    12,
					IsDir:   false,
					ModTime: time.Time{},
					Mode:    defaultFileMode,
				},
			},
			wantErr: false,
		},
		{
			name:         "empty codebase path",
			codebasePath: "",
			dir:          "",
			option:       types.ListOptions{},
			setupMock:    func(m *mocks.MockMinioClient) {},
			want:         nil,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := mocks.NewMockMinioClient(ctrl)
			tt.setupMock(mockClient)

			codebase := &minioCodebase{
				client: mockClient,
				cfg: config.CodeBaseStoreConf{
					Minio: config.MinioStoreConf{
						Bucket: "test-bucket",
					},
				},
				logger: logx.WithContext(context.Background()),
			}

			got, err := codebase.List(context.Background(), tt.codebasePath, tt.dir, tt.option)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.want), len(got))
				if len(got) > 0 {
					assert.Equal(t, tt.want[0].Name, got[0].Name)
					assert.Equal(t, tt.want[0].Size, got[0].Size)
					assert.Equal(t, tt.want[0].IsDir, got[0].IsDir)
				}
			}
		})
	}
}

func TestMinioCodebase_Tree(t *testing.T) {
	tests := []struct {
		name         string
		codebasePath string
		dir          string
		option       types.TreeOptions
		setupMock    func(*mocks.MockMinioClient)
		want         []*types.TreeNode
		wantErr      bool
	}{
		{
			name:         "successful tree",
			codebasePath: filepath.Join("test-bucket", generateUniquePath("test-client", "test-path"), "test-path", filepathSlash),
			dir:          "",
			option:       types.TreeOptions{},
			setupMock: func(m *mocks.MockMinioClient) {
				ch := make(chan minio.ObjectInfo, 1)
				ch <- minio.ObjectInfo{
					Key:          "test.txt",
					Size:         12,
					LastModified: time.Time{},
				}
				close(ch)
				m.EXPECT().ListObjects(gomock.Any(), "test-bucket", gomock.Any()).Return(ch)
			},
			want: []*types.TreeNode{
				{
					FileInfo: types.FileInfo{
						Name:    "test.txt",
						Path:    "test.txt",
						Size:    12,
						IsDir:   false,
						ModTime: time.Time{},
						Mode:    defaultFileMode,
					},
					Children: []types.TreeNode{},
				},
			},
			wantErr: false,
		},
		{
			name:         "empty codebase path",
			codebasePath: "",
			dir:          "",
			option:       types.TreeOptions{},
			setupMock:    func(m *mocks.MockMinioClient) {},
			want:         nil,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := mocks.NewMockMinioClient(ctrl)
			tt.setupMock(mockClient)

			codebase := &minioCodebase{
				client: mockClient,
				cfg: config.CodeBaseStoreConf{
					Minio: config.MinioStoreConf{
						Bucket: "test-bucket",
					},
				},
				logger: logx.WithContext(context.Background()),
			}

			got, err := codebase.Tree(context.Background(), tt.codebasePath, tt.dir, tt.option)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.want), len(got))
				if len(got) > 0 {
					assert.Equal(t, tt.want[0].Name, got[0].Name)
					assert.Equal(t, tt.want[0].Path, got[0].Path)
					assert.Equal(t, tt.want[0].Size, got[0].Size)
					assert.Equal(t, tt.want[0].IsDir, got[0].IsDir)
				}
			}
		})
	}
}

func TestMinioCodebase_Walk(t *testing.T) {
	tests := []struct {
		name         string
		codebasePath string
		dir          string
		setupMock    func(*mocks.MockMinioClient)
		wantPaths    []string
		wantErr      bool
	}{
		{
			name:         "empty directory",
			codebasePath: filepath.Join("test-bucket", "test-path"),
			dir:          "",
			setupMock: func(m *mocks.MockMinioClient) {
				m.EXPECT().ListObjects(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, bucket string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
						ch := make(chan minio.ObjectInfo)
						close(ch)
						return ch
					})
			},
			wantPaths: make([]string, 0),
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := mocks.NewMockMinioClient(ctrl)
			tt.setupMock(mockClient)

			c := &minioCodebase{
				client: mockClient,
				cfg: config.CodeBaseStoreConf{
					Minio: config.MinioStoreConf{
						Bucket: "test-bucket",
					},
				},
				logger: logx.WithContext(context.Background()),
			}

			var paths []string
			err := c.Walk(context.Background(), tt.codebasePath, tt.dir, func(walkCtx *WalkContext, reader io.ReadCloser) error {
				if !walkCtx.Info.IsDir {
					paths = append(paths, walkCtx.RelativePath)
				}
				return nil
			})

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			if paths == nil {
				paths = make([]string, 0)
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantPaths, paths)
		})
	}
}
