package e2e

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	api "github.com/zgsm-ai/codebase-indexer/test/api_test"
	"testing"
	"time"
)

func TestLock(t *testing.T) {
	ctx := context.Background()
	svcCtx := api.InitSvcCtx(ctx, nil)
	lock := svcCtx.DistLock
	key := "test-" + uuid.New().String()
	mutex, err := lock.Lock(ctx, key, time.Minute)
	assert.NoError(t, err)
	err = lock.Unlock(ctx, mutex)
	assert.NoError(t, err)

}

func TestLockExpire(t *testing.T) {
	ctx := context.Background()
	svcCtx := api.InitSvcCtx(ctx, nil)
	lock := svcCtx.DistLock
	key := "test-" + uuid.New().String()
	mutex, err := lock.Lock(ctx, key, time.Second*10)
	assert.NoError(t, err)
	time.Sleep(time.Second * 15)
	err = lock.Unlock(ctx, mutex)
	assert.NoError(t, err)

}
