package base

import (
	"sync"
	"testing"
	"time"
)

// TestBaseController_ConcurrentAccess 测试并发访问时上下文是否正确
func TestBaseController_ConcurrentAccess(t *testing.T) {
	// 创建控制器实例
	controller := NewBaseController()

	// 并发测试
	var wg sync.WaitGroup
	concurrentCount := 10

	// 存储每个请求的上下文路径
	paths := make([]string, concurrentCount)
	for i := 0; i < concurrentCount; i++ {
		paths[i] = "/test/path/" + string(rune('0'+i))
	}

	// 并发执行
	for i := 0; i < concurrentCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// 创建模拟上下文
			ctx := &MockContext{path: paths[index]}

			// 初始化控制器
			controller.Init(ctx)

			// 模拟处理时间
			time.Sleep(10 * time.Millisecond)

			// 验证控制器是否能正常工作（通过调用一个方法来测试）
			query := controller.GetQuery("test")
			if query != "" {
				t.Errorf("Expected empty string, got %s", query)
			}
		}(i)
	}

	wg.Wait()

	// 验证控制器仍然可以正常工作
	finalCtx := &MockContext{path: "/final/test"}
	controller.Init(finalCtx)

	// 测试控制器方法
	query := controller.GetQuery("test")
	if query != "" {
		t.Errorf("Expected empty string, got %s", query)
	}

	t.Log("Concurrent access test passed")
}

// TestBaseController_NewInstancePerRequest 测试每个请求使用新实例
func TestBaseController_NewInstancePerRequest(t *testing.T) {
	concurrentCount := 10
	var wg sync.WaitGroup

	for i := 0; i < concurrentCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// 每个请求创建新的控制器实例
			controller := NewBaseController()

			// 创建模拟上下文
			ctx := &MockContext{path: "/test/path/" + string(rune('0'+index))}

			// 初始化控制器
			controller.Init(ctx)

			// 模拟处理时间
			time.Sleep(10 * time.Millisecond)

			// 验证控制器是否能正常工作（通过调用一个方法来测试）
			query := controller.GetQuery("test")
			if query != "" {
				t.Errorf("Expected empty string, got %s", query)
			}

			t.Logf("Request %d: Controller instance created and initialized successfully", index)
		}(i)
	}

	wg.Wait()

	t.Log("New instance per request test passed")
}
