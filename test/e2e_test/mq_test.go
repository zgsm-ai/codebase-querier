package e2e

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"sync"
	"testing"
	"time"

	"github.com/zgsm-ai/codebase-indexer/internal/types"
	api "github.com/zgsm-ai/codebase-indexer/test/api_test"
)

func TestAck(t *testing.T) {
	ctx := context.Background()
	svcCtx := api.InitSvcCtx(ctx, nil)
	mq := svcCtx.MessageQueue

	// 准备测试数据
	topic := "test-ack-topic"
	consumerGroup := svcCtx.Config.IndexTask.ConsumerGroup
	message := []byte("test message for ack")
	msgID := ""

	// 生产消息
	err := mq.Produce(ctx, topic, message, types.ProduceOptions{})
	if err != nil {
		t.Fatalf("生产消息失败: %v", err)
	}

	// 消费消息
	msg, err := mq.Consume(ctx, topic, consumerGroup, types.ConsumeOptions{})
	if err != nil {
		t.Fatalf("消费消息失败: %v", err)
	}
	msgID = msg.ID

	// 确认消息
	err = mq.Ack(ctx, topic, consumerGroup, msgID)
	if err != nil {
		t.Fatalf("确认消息失败: %v", err)
	}

	// 尝试再次消费，期望获取不到消息
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err = mq.Consume(ctx, topic, consumerGroup, types.ConsumeOptions{
		ReadTimeout: 5 * time.Second,
	})
	if err == nil {
		t.Fatalf("Ack后仍能消费到消息，测试失败")
	}
}

// 生成唯一消息内容（避免并发测试冲突）
func generateUniqueMessage() []byte {
	return []byte(uuid.New().String())
}

func TestNack(t *testing.T) {
	ctx := context.Background()
	svcCtx := api.InitSvcCtx(ctx, nil)
	mq := svcCtx.MessageQueue
	consumerGroup := svcCtx.Config.IndexTask.ConsumerGroup
	topic := "test-nack-topic"

	// 生成唯一测试消息
	message := generateUniqueMessage()

	// 生产消息
	err := mq.Produce(ctx, topic, message, types.ProduceOptions{})
	if err != nil {
		t.Fatalf("生产消息失败: %v", err)
	}

	// 消费消息
	msg, err := mq.Consume(ctx, topic, consumerGroup, types.ConsumeOptions{
		ReadTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("消费消息失败: %v", err)
	}

	// 执行Nack（Redis实现：先Ack再重新Produce）
	err = mq.Nack(ctx, topic, consumerGroup, msg.ID, message, types.NackOptions{})
	if err != nil {
		t.Fatalf("拒绝确认消息失败: %v", err)
	}

	// 等待消息重新入队（Redis需要时间处理重新Produce）
	time.Sleep(100 * time.Millisecond)

	// 再次消费，验证消息内容一致性
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	msg2, err := mq.Consume(ctx, topic, consumerGroup, types.ConsumeOptions{
		ReadTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("Nack后无法再次消费消息: %v", err)
	}

	//  核心断言：比对消息内容而非ID
	if string(msg2.Body) != string(message) {
		t.Fatalf("消息内容不一致，期望: %s，实际: %s", message, msg2.Body)
	}
}

func TestNackConcurrentConsumers(t *testing.T) {
	ctx := context.Background()
	svcCtx := api.InitSvcCtx(ctx, nil)
	mq := svcCtx.MessageQueue
	consumerGroup := svcCtx.Config.IndexTask.ConsumerGroup
	topic := "test-nack-concurrent-topic"

	// 生成唯一测试消息
	message := generateUniqueMessage()

	// 生产消息
	err := mq.Produce(ctx, topic, message, types.ProduceOptions{})
	if err != nil {
		t.Fatalf("生产消息失败: %v", err)
	}

	var wg sync.WaitGroup
	var firstMsgContent, secondMsgContent string
	var firstErr, secondErr error

	// 第一个消费者：消费并Nack
	wg.Add(1)
	go func() {
		defer wg.Done()
		msg, err := mq.Consume(ctx, topic, consumerGroup, types.ConsumeOptions{
			ReadTimeout: 2 * time.Second,
		})
		if err != nil {
			firstErr = fmt.Errorf("第一个消费者消费失败: %v", err)
			return
		}
		firstMsgContent = string(msg.Body)

		// 执行Nack
		err = mq.Nack(ctx, topic, consumerGroup, msg.ID, message, types.NackOptions{})
		if err != nil {
			firstErr = fmt.Errorf("第一个消费者Nack失败: %v", err)
			return
		}
	}()

	wg.Wait()
	if firstErr != nil {
		t.Fatal(firstErr)
	}

	// 第二个消费者：消费Nack后的消息
	wg.Add(1)
	go func() {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		// 等待消息重新入队
		time.Sleep(100 * time.Millisecond)

		msg, err := mq.Consume(ctx, topic, consumerGroup, types.ConsumeOptions{
			ReadTimeout: 2 * time.Second,
		})
		if err != nil {
			secondErr = fmt.Errorf("第二个消费者消费失败: %v", err)
			return
		}
		secondMsgContent = string(msg.Body)
	}()

	wg.Wait()
	if secondErr != nil {
		t.Fatal(secondErr)
	}

	//  核心断言：两个消费者收到的消息内容一致
	if firstMsgContent != secondMsgContent {
		t.Fatalf("并发消费消息内容不一致，消费者1: %s，消费者2: %s",
			firstMsgContent, secondMsgContent)
	}

	// 清理：确认最后一条消息，避免影响其他测试
	// 注意：此处无法通过ID关联，需重新消费最新消息后Ack
	cleanupCtx, cleanupCancel := context.WithTimeout(ctx, 2*time.Second)
	defer cleanupCancel()
	cleanupMsg, err := mq.Consume(cleanupCtx, topic, consumerGroup, types.ConsumeOptions{ReadTimeout: time.Second * 5})
	if err == nil {
		_ = mq.Ack(cleanupCtx, topic, consumerGroup, cleanupMsg.ID)
	}
}

// TestConsumerGroupCompetition 验证同一消费者组内多个消费者的竞争消费特性
func TestConsumerGroupCompetition(t *testing.T) {
	ctx := context.Background()
	svcCtx := api.InitSvcCtx(ctx, nil)
	mq := svcCtx.MessageQueue

	// 测试配置
	topic := "test-consumer-group-competition"
	consumerGroup := "test-competition-group"
	message := generateUniqueMessage()

	// 生产消息
	err := mq.Produce(ctx, topic, message, types.ProduceOptions{})
	if err != nil {
		t.Fatalf("生产消息失败: %v", err)
	}

	var wg sync.WaitGroup
	var consumer1MsgContent string
	var consumer2MsgContent string
	var consumer1Err, consumer2Err error

	// 消费者1：属于同一消费者组
	wg.Add(1)
	go func() {
		defer wg.Done()
		consumeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		msg, err := mq.Consume(consumeCtx, topic, consumerGroup, types.ConsumeOptions{
			ReadTimeout: 2 * time.Second,
		})
		if err != nil {
			consumer1Err = fmt.Errorf("消费者1消费失败: %v", err)
			return
		}
		consumer1MsgContent = string(msg.Body)

		// 确认消息，避免影响后续验证
		err = mq.Ack(consumeCtx, topic, consumerGroup, msg.ID)
		if err != nil {
			consumer1Err = fmt.Errorf("消费者1确认消息失败: %v", err)
			return
		}
	}()

	// 消费者2：属于同一消费者组（竞争关系）
	wg.Add(1)
	go func() {
		defer wg.Done()
		// 等待消费者1先尝试消费
		time.Sleep(200 * time.Millisecond)

		consumeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()

		// 尝试消费同一主题和消费者组的消息
		_, err := mq.Consume(consumeCtx, topic, consumerGroup, types.ConsumeOptions{
			ReadTimeout: 5 * time.Second,
		})

		// 预期消费者2消费失败（消息已被消费者1获取）
		if err == nil {
			consumer2Err = fmt.Errorf("消费者2意外获取到消息，违反竞争消费原则")
			return
		}
		consumer2MsgContent = "消费失败（预期结果）"
	}()

	wg.Wait()

	// 验证结果
	if consumer1Err != nil {
		t.Fatal(consumer1Err)
	}
	if consumer2Err != nil {
		t.Fatal(consumer2Err)
	}

	// 核心断言：消费者1获取到消息，消费者2未获取到
	if consumer1MsgContent == "" {
		t.Fatalf("消费者1未获取到消息")
	}
	if consumer2MsgContent != "消费失败（预期结果）" {
		t.Fatalf("消费者2意外获取到消息，内容为: %s", consumer2MsgContent)
	}

	// 验证消息内容一致性
	if string(message) != consumer1MsgContent {
		t.Fatalf("消息内容不一致，生产内容: %s，消费内容: %s", message, consumer1MsgContent)
	}
}
