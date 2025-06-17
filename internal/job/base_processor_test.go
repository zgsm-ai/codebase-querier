package job

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/dao/query"
	"github.com/zgsm-ai/codebase-indexer/internal/errs"
	"github.com/zgsm-ai/codebase-indexer/internal/store/database/mocks"
	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

// testProcessor 测试用的处理器包装
type testProcessor struct {
	*baseProcessor
	mockDB *mocks.MockDB
	ctrl   *gomock.Controller
}

// setupTestProcessor 创建测试用的处理器
func setupTestProcessor(t *testing.T) *testProcessor {
	ctrl := gomock.NewController(t)
	mockDB, err := mocks.NewMockDB()
	assert.NoError(t, err)

	msg := &types.CodebaseSyncMessage{
		SyncID:       1,
		CodebaseID:   100,
		CodebasePath: "test/path",
	}

	svcCtx := &svc.ServiceContext{
		Querier: query.Use(mockDB.GormDB),
	}

	processor := &baseProcessor{
		svcCtx:          svcCtx,
		msg:             msg,
		syncFileModeMap: make(map[string]string),
	}

	return &testProcessor{
		baseProcessor: processor,
		mockDB:        mockDB,
		ctrl:          ctrl,
	}
}

// cleanup 清理测试资源
func (tp *testProcessor) cleanup() {
	tp.ctrl.Finish()
	tp.mockDB.Close()
}

// prepareFiles 准备测试文件
func prepareFiles(count int) map[string]string {
	files := make(map[string]string)
	for i := 0; i < count; i++ {
		files[fmt.Sprintf("file%d.txt", i)] = types.FileOpAdd
	}
	return files
}

// processFunc 创建处理函数
type processFunc struct {
	processTime  time.Duration // 处理时间
	errorRate    int           // 错误率 (0-100)
	deadlockRate int           // 死锁率 (0-100)
	stats        *processStats
	ctx          context.Context
}

// processStats 处理统计
type processStats struct {
	processed int32
	errors    int32
	deadlocks int32
}

// newProcessFunc 创建新的处理函数
func newProcessFunc(ctx context.Context, processTime time.Duration, errorRate, deadlockRate int) *processFunc {
	return &processFunc{
		processTime:  processTime,
		errorRate:    errorRate,
		deadlockRate: deadlockRate,
		stats:        &processStats{},
		ctx:          ctx,
	}
}

// handle 处理单个文件
func (pf *processFunc) handle(path string, op types.FileOp) error {
	atomic.AddInt32(&pf.stats.processed, 1)

	// 模拟死锁
	if pf.deadlockRate > 0 && rand.Intn(100) < pf.deadlockRate {
		atomic.AddInt32(&pf.stats.deadlocks, 1)
		select {
		case <-pf.ctx.Done():
			return pf.ctx.Err()
		case <-time.After(time.Hour):
			return nil
		}
	}

	// 模拟处理时间
	select {
	case <-pf.ctx.Done():
		return pf.ctx.Err()
	case <-time.After(pf.processTime):
	}

	// 模拟错误
	if pf.errorRate > 0 && rand.Intn(100) < pf.errorRate {
		atomic.AddInt32(&pf.stats.errors, 1)
		return fmt.Errorf("random error in %s", path)
	}

	return nil
}

// TestBaseProcessor_DatabaseOperations 测试数据库操作
func TestBaseProcessor_DatabaseOperations(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*testProcessor)
		run      func(tp *testProcessor) error
		validate func(*testing.T, *testProcessor, error)
	}{
		{
			name: "初始化任务历史",
			setup: func(tp *testProcessor) {
				tp.mockDB.Mock.ExpectBegin()
				tp.mockDB.Mock.ExpectQuery(`INSERT INTO "index_history"`).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
				tp.mockDB.Mock.ExpectCommit()
			},
			run: func(tp *testProcessor) error {
				return tp.initTaskHistory(context.Background(), "test_task")
			},
			validate: func(t *testing.T, tp *testProcessor, err error) {
				assert.NoError(t, err)
				assert.Equal(t, int32(1), tp.taskHistoryId)
				assert.NoError(t, tp.mockDB.Mock.ExpectationsWereMet())
			},
		},
		{
			name: "更新任务成功状态",
			setup: func(tp *testProcessor) {
				tp.taskHistoryId = 1
				tp.totalFileCnt = 100
				tp.successFileCnt = 95
				tp.failedFileCnt = 3
				tp.ignoreFileCnt = 2

				tp.mockDB.Mock.ExpectBegin()
				tp.mockDB.Mock.ExpectExec(`UPDATE "index_history"`).
					WillReturnResult(sqlmock.NewResult(0, 1))
				tp.mockDB.Mock.ExpectCommit()
			},
			run: func(tp *testProcessor) error {
				return tp.updateTaskSuccess(context.Background())
			},
			validate: func(t *testing.T, tp *testProcessor, err error) {
				tp.updateTaskSuccess(context.Background())
				assert.NoError(t, tp.mockDB.Mock.ExpectationsWereMet())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tp := setupTestProcessor(t)
			defer tp.cleanup()

			tt.setup(tp)
			err := tt.run(tp)
			tt.validate(t, tp, err)
		})
	}
}

// TestBaseProcessor_ProcessFilesConcurrently_Scenarios 测试各种场景
func TestBaseProcessor_ProcessFilesConcurrently_Scenarios(t *testing.T) {
	// 设置随机种子以保证测试可重复
	rand.Seed(time.Now().UnixNano())

	tests := []struct {
		name           string
		fileCount      int           // 文件数量
		maxConcurrency int           // 最大并发数
		timeout        time.Duration // 上下文超时时间
		processTime    time.Duration // 每个任务处理时间
		errorRate      int           // 错误率(0-100)
		deadlockRate   int           // 死锁率(0-100)
		expectTimeout  bool          // 是否期望超时
		expectErrors   bool          // 是否期望错误
	}{
		{
			name:           "正常场景_小规模",
			fileCount:      10,
			maxConcurrency: 5,
			timeout:        5 * time.Second,
			processTime:    100 * time.Millisecond,
			expectTimeout:  false,
			expectErrors:   false,
		},
		{
			name:           "超时场景_任务处理过慢",
			fileCount:      10,
			maxConcurrency: 2,
			timeout:        1 * time.Second,
			processTime:    500 * time.Millisecond,
			expectTimeout:  true,
			expectErrors:   false,
		},
		{
			name:           "错误场景_高错误率",
			fileCount:      100,
			maxConcurrency: 10,
			timeout:        5 * time.Second,
			processTime:    10 * time.Millisecond,
			errorRate:      20,
			expectTimeout:  false,
			expectErrors:   true,
		},
		{
			name:           "死锁场景_部分任务永久阻塞",
			fileCount:      20,
			maxConcurrency: 5,
			timeout:        2 * time.Second,
			processTime:    100 * time.Millisecond,
			deadlockRate:   10,
			expectTimeout:  true,
			expectErrors:   false,
		},
		{
			name:           "大规模场景_正常处理",
			fileCount:      1000,
			maxConcurrency: 100,
			timeout:        10 * time.Second,
			processTime:    time.Millisecond,
			expectTimeout:  false,
			expectErrors:   false,
		},
		{
			name:           "混合场景_错误和超时",
			fileCount:      100,
			maxConcurrency: 10,
			timeout:        2 * time.Second,
			processTime:    50 * time.Millisecond,
			errorRate:      10,
			deadlockRate:   5,
			expectTimeout:  true,
			expectErrors:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tp := setupTestProcessor(t)
			defer tp.cleanup()

			// 设置上下文和测试数据
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()
			tp.syncFileModeMap = prepareFiles(tt.fileCount)

			// 创建处理函数
			pf := newProcessFunc(ctx, tt.processTime, tt.errorRate, tt.deadlockRate)

			// 执行测试
			start := time.Now()
			err := tp.processFilesConcurrently(ctx, pf.handle, tt.maxConcurrency)
			duration := time.Since(start)

			// 输出统计信息
			t.Logf("Test completed in %v", duration)
			t.Logf("Stats: processed=%d, errors=%d, deadlocks=%d",
				atomic.LoadInt32(&pf.stats.processed),
				atomic.LoadInt32(&pf.stats.errors),
				atomic.LoadInt32(&pf.stats.deadlocks))

			// 验证结果
			if tt.expectTimeout {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, context.DeadlineExceeded) || errors.Is(err, errs.RunTimeout))
			}

			if tt.expectErrors {
				assert.Error(t, err)
				assert.True(t, atomic.LoadInt32(&pf.stats.errors) > 0)
			}

			if !tt.expectTimeout && !tt.expectErrors {
				assert.NoError(t, err)
				assert.Equal(t, int32(tt.fileCount), atomic.LoadInt32(&pf.stats.processed))
			}
		})
	}
}
