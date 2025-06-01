package codebase

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codebase/mocks"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

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
				m.EXPECT().
					PutObject(
						gomock.Any(),
						"test-bucket",
						"/test/test-client/test-path/",
						gomock.Any(),
						int64(0),
						gomock.Any(),
					).
					Return(minio.UploadInfo{}, nil)
			},
			want:    "/test/test-client/test-path/",
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
			mockClient := mocks.NewMockMinioClient(ctrl)
			tt.setupMock(mockClient)

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

			got, err := codebase.Init(context.Background(), tt.clientId, tt.codebasePath)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestMinioCodebase_Add(t *testing.T) {
	tests := []struct {
		name         string
		codebasePath string
		target       string
		content      string
		setupMock    func(*mocks.MockMinioClient)
		wantErr      bool
	}{
		{
			name:         "successful add",
			codebasePath: "/test/test-client/test-path",
			target:       "test.txt",
			content:      "test content",
			setupMock: func(m *mocks.MockMinioClient) {
				m.EXPECT().
					PutObject(
						gomock.Any(),
						"test-bucket",
						"/test/test-client/test-path/test.txt",
						gomock.Any(),
						int64(-1),
						gomock.Any(),
					).
					Return(minio.UploadInfo{}, nil)
			},
			wantErr: false,
		},
		{
			name:         "empty codebase path",
			codebasePath: "",
			target:       "test.txt",
			content:      "test content",
			setupMock:    func(m *mocks.MockMinioClient) {},
			wantErr:      true,
		},
		{
			name:         "empty target",
			codebasePath: "/test/test-client/test-path",
			target:       "",
			content:      "test content",
			setupMock:    func(m *mocks.MockMinioClient) {},
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient, codebase, _ := setupTestMinioCodebase(t)
			tt.setupMock(mockClient)

			err := codebase.Add(context.Background(), tt.codebasePath, strings.NewReader(tt.content), tt.target)
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
			codebasePath: "/test/test-client/test-path",
			path:         "test.txt",
			setupMock: func(m *mocks.MockMinioClient) {
				m.EXPECT().
					RemoveObject(
						gomock.Any(),
						"test-bucket",
						"/test/test-client/test-path/test.txt",
						gomock.Any(),
					).
					Return(nil)
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
			codebasePath: "/test/test-client/test-path",
			path:         "",
			setupMock:    func(m *mocks.MockMinioClient) {},
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient, codebase, _ := setupTestMinioCodebase(t)
			tt.setupMock(mockClient)

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
			codebasePath: "/test/test-client/test-path",
			dir:          "",
			option:       types.ListOptions{},
			setupMock: func(m *mocks.MockMinioClient) {
				ch := make(chan minio.ObjectInfo, 2)
				ch <- minio.ObjectInfo{
					Key:          "/test/test-client/test-path/file1.txt",
					Size:         100,
					LastModified: time.Time{},
				}
				ch <- minio.ObjectInfo{
					Key:          "/test/test-client/test-path/file2.txt",
					Size:         200,
					LastModified: time.Time{},
				}
				close(ch)

				m.EXPECT().
					ListObjects(
						gomock.Any(),
						"test-bucket",
						gomock.Any(),
					).
					Return(ch)
			},
			want: []*types.FileInfo{
				{
					Name:    "file1.txt",
					Size:    100,
					IsDir:   false,
					ModTime: time.Time{},
					Mode:    defaultFileMode,
				},
				{
					Name:    "file2.txt",
					Size:    200,
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
			mockClient, codebase, _ := setupTestMinioCodebase(t)
			tt.setupMock(mockClient)

			got, err := codebase.List(context.Background(), tt.codebasePath, tt.dir, tt.option)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
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
			codebasePath: "/test/test-client/test-path",
			dir:          "",
			option:       types.TreeOptions{},
			setupMock: func(m *mocks.MockMinioClient) {
				ch := make(chan minio.ObjectInfo, 3)
				ch <- minio.ObjectInfo{
					Key:          "/test/test-client/test-path/file1.txt",
					Size:         100,
					LastModified: time.Time{},
				}
				ch <- minio.ObjectInfo{
					Key:          "/test/test-client/test-path/dir1/file2.txt",
					Size:         200,
					LastModified: time.Time{},
				}
				ch <- minio.ObjectInfo{
					Key:          "/test/test-client/test-path/dir2/file3.txt",
					Size:         300,
					LastModified: time.Time{},
				}
				close(ch)

				m.EXPECT().
					ListObjects(
						gomock.Any(),
						"test-bucket",
						gomock.Any(),
					).
					Return(ch)
			},
			want: []*types.TreeNode{
				{
					FileInfo: types.FileInfo{
						Name:    "file1.txt",
						Path:    "file1.txt",
						Size:    100,
						IsDir:   false,
						ModTime: time.Time{},
						Mode:    defaultFileMode,
					},
					Children: []types.TreeNode{},
				},
				{
					FileInfo: types.FileInfo{
						Name:    "dir1",
						Path:    "dir1",
						Size:    0,
						IsDir:   true,
						ModTime: time.Time{},
						Mode:    defaultFileMode,
					},
					Children: []types.TreeNode{
						{
							FileInfo: types.FileInfo{
								Name:    "file2.txt",
								Path:    "dir1/file2.txt",
								Size:    200,
								IsDir:   false,
								ModTime: time.Time{},
								Mode:    defaultFileMode,
							},
							Children: []types.TreeNode{},
						},
					},
				},
				{
					FileInfo: types.FileInfo{
						Name:    "dir2",
						Path:    "dir2",
						Size:    0,
						IsDir:   true,
						ModTime: time.Time{},
						Mode:    defaultFileMode,
					},
					Children: []types.TreeNode{
						{
							FileInfo: types.FileInfo{
								Name:    "file3.txt",
								Path:    "dir2/file3.txt",
								Size:    300,
								IsDir:   false,
								ModTime: time.Time{},
								Mode:    defaultFileMode,
							},
							Children: []types.TreeNode{},
						},
					},
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
			mockClient, codebase, _ := setupTestMinioCodebase(t)
			tt.setupMock(mockClient)

			got, err := codebase.Tree(context.Background(), tt.codebasePath, tt.dir, tt.option)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestMinioCodebase_Walk(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	tests := []struct {
		name         string
		codebasePath string
		dir          string
		setupMock    func(*mocks.MockMinioClient)
		wantPaths    []string
		wantErr      bool
	}{
		{
			name:         "successful walk",
			codebasePath: "/test/test-client/test-path",
			dir:          "",
			setupMock: func(m *mocks.MockMinioClient) {
				// Create a channel for object listing
				ch := make(chan minio.ObjectInfo, 4)
				ch <- minio.ObjectInfo{
					Key:          "/test/test-client/test-path/file1.txt",
					Size:         100,
					LastModified: time.Time{},
				}
				ch <- minio.ObjectInfo{
					Key:          "/test/test-client/test-path/dir1/file2.txt",
					Size:         200,
					LastModified: time.Time{},
				}
				ch <- minio.ObjectInfo{
					Key:          "/test/test-client/test-path/dir1/dir2/file3.txt",
					Size:         300,
					LastModified: time.Time{},
				}
				ch <- minio.ObjectInfo{
					Key:          "/test/test-client/test-path/.hidden/file4.txt",
					Size:         400,
					LastModified: time.Time{},
				}
				close(ch)

				// Set up ListObjects expectation
				m.EXPECT().
					ListObjects(
						gomock.Any(),
						"test-bucket",
						gomock.Any(),
					).
					Return(ch)

				// Set up GetObject expectations for each file
				for _, key := range []string{
					"/test/test-client/test-path/file1.txt",
					"/test/test-client/test-path/dir1/file2.txt",
					"/test/test-client/test-path/dir1/dir2/file3.txt",
				} {
					mockObj := mocks.NewMockStore(ctrl)
					mockObj.EXPECT().Walk(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
					m.EXPECT().
						GetObject(
							gomock.Any(),
							"test-bucket",
							key,
							gomock.Any(),
						).
						Return(mockObj, nil)
				}
			},
			wantPaths: []string{
				"file1.txt",
				"dir1/file2.txt",
				"dir1/dir2/file3.txt",
			},
			wantErr: false,
		},
		{
			name:         "empty codebase path",
			codebasePath: "",
			dir:          "",
			setupMock:    func(m *mocks.MockMinioClient) {},
			wantPaths:    nil,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient, codebase, _ := setupTestMinioCodebase(t)
			tt.setupMock(mockClient)

			var visitedPaths []string
			err := codebase.Walk(context.Background(), tt.codebasePath, tt.dir, func(walkCtx *WalkContext, reader io.ReadCloser) error {
				// Skip hidden files and directories
				if strings.HasPrefix(walkCtx.Info.Name, ".") {
					if walkCtx.Info.IsDir {
						return SkipDir
					}
					return nil
				}

				// Only record file paths
				if !walkCtx.Info.IsDir {
					visitedPaths = append(visitedPaths, walkCtx.RelativePath)
				}
				return nil
			})

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.ElementsMatch(t, tt.wantPaths, visitedPaths)
			}
		})
	}
}
