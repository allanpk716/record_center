package device

import (
	"fmt"
	"strings"

	"github.com/allanpk716/record_center/internal/logger"
)

// CommandExecutor 命令执行器接口
type CommandExecutor interface {
	// ExecuteCommand 执行PowerShell命令
	ExecuteCommand(command string, args ...string) (*ExecutionResult, error)

	// ExecuteScript 执行PowerShell脚本
	ExecuteScript(script string) (*ExecutionResult, error)

	// TestExecution 测试执行器是否可用
	TestExecution() error

	// GetVersion 获取当前使用的PowerShell版本信息
	GetVersion() (*PowerShellVersion, error)

	// Close 关闭执行器，释放资源
	Close() error
}

// PowerShellExecutor PowerShell执行器实现
type PowerShellExecutor struct {
	manager *PowerShellManager
	log     *logger.Logger
	closed  bool
}

// NewPowerShellExecutor 创建PowerShell执行器
func NewPowerShellExecutor(log *logger.Logger, config *PowerShellConfig) *PowerShellExecutor {
	return &PowerShellExecutor{
		manager: NewPowerShellManager(log, config),
		log:     log,
		closed:  false,
	}
}

// ExecuteCommand 执行PowerShell命令
func (pe *PowerShellExecutor) ExecuteCommand(command string, args ...string) (*ExecutionResult, error) {
	if pe.closed {
		return nil, fmt.Errorf("执行器已关闭")
	}

	pe.log.Debug("PowerShellExecutor: 执行命令 %s %v", command, args)

	result, err := pe.manager.ExecuteCommand(command, args...)
	if err != nil {
		pe.log.Error("PowerShellExecutor: 命令执行失败: %v", err)
		return nil, err
	}

	pe.log.Debug("PowerShellExecutor: 命令执行成功，输出长度: %d", len(result.Output))
	return result, nil
}

// ExecuteScript 执行PowerShell脚本
func (pe *PowerShellExecutor) ExecuteScript(script string) (*ExecutionResult, error) {
	if pe.closed {
		return nil, fmt.Errorf("执行器已关闭")
	}

	pe.log.Debug("PowerShellExecutor: 执行脚本，长度: %d", len(script))

	result, err := pe.manager.ExecuteScript(script)
	if err != nil {
		pe.log.Error("PowerShellExecutor: 脚本执行失败: %v", err)
		return nil, err
	}

	pe.log.Debug("PowerShellExecutor: 脚本执行成功，输出长度: %d", len(result.Output))
	return result, nil
}

// TestExecution 测试执行器是否可用
func (pe *PowerShellExecutor) TestExecution() error {
	if pe.closed {
		return fmt.Errorf("执行器已关闭")
	}

	pe.log.Debug("PowerShellExecutor: 开始执行测试")

	err := pe.manager.TestExecution()
	if err != nil {
		pe.log.Error("PowerShellExecutor: 执行测试失败: %v", err)
		return fmt.Errorf("PowerShell执行器测试失败: %w", err)
	}

	pe.log.Info("PowerShellExecutor: 执行测试成功")
	return nil
}

// GetVersion 获取当前使用的PowerShell版本信息
func (pe *PowerShellExecutor) GetVersion() (*PowerShellVersion, error) {
	if pe.closed {
		return nil, fmt.Errorf("执行器已关闭")
	}

	version, err := pe.manager.GetPreferredVersion()
	if err != nil {
		return nil, fmt.Errorf("获取PowerShell版本失败: %w", err)
	}

	return version, nil
}

// Close 关闭执行器，释放资源
func (pe *PowerShellExecutor) Close() error {
	if pe.closed {
		return nil
	}

	pe.log.Debug("PowerShellExecutor: 关闭执行器")

	// 清理资源
	pe.manager.ClearCache()
	pe.closed = true

	return nil
}

// LegacyExecutorWrapper 传统执行器包装器，用于向后兼容
type LegacyExecutorWrapper struct {
	executor CommandExecutor
	log      *logger.Logger
}

// NewLegacyExecutorWrapper 创建传统执行器包装器
func NewLegacyExecutorWrapper(executor CommandExecutor, log *logger.Logger) *LegacyExecutorWrapper {
	return &LegacyExecutorWrapper{
		executor: executor,
		log:      log,
	}
}

// ExecuteCommandString 执行命令并返回字符串输出（兼容旧接口）
func (lew *LegacyExecutorWrapper) ExecuteCommandString(command string) (string, error) {
	// 解析命令字符串为命令和参数
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "", fmt.Errorf("空命令")
	}

	cmd := parts[0]
	args := parts[1:]

	result, err := lew.executor.ExecuteCommand(cmd, args...)
	if err != nil {
		return "", err
	}

	return result.Output, nil
}

// ExecuteCommandStringWithPolicy 使用执行策略执行命令
func (lew *LegacyExecutorWrapper) ExecuteCommandStringWithPolicy(command string, policy string) (string, error) {
	// 添加执行策略参数
	args := []string{"-ExecutionPolicy", policy, "-Command", command}

	result, err := lew.executor.ExecuteCommand("powershell", args...)
	if err != nil {
		return "", err
	}

	return result.Output, nil
}

// ExecuteScriptString 执行脚本字符串
func (lew *LegacyExecutorWrapper) ExecuteScriptString(script string) (string, error) {
	result, err := lew.executor.ExecuteScript(script)
	if err != nil {
		return "", err
	}

	return result.Output, nil
}

// TestLegacyCompatibility 测试传统兼容性
func (lew *LegacyExecutorWrapper) TestLegacyCompatibility() error {
	// 测试基本的执行策略检查
	output, err := lew.ExecuteCommandString("Get-ExecutionPolicy")
	if err != nil {
		return fmt.Errorf("传统兼容性测试失败 - 执行策略检查: %w", err)
	}

	lew.log.Debug("传统兼容性测试 - 执行策略: %s", strings.TrimSpace(output))

	// 测试COM对象访问
	script := `
$shell = New-Object -ComObject Shell.Application
if ($shell) {
    "COM_OK"
} else {
    "COM_FAILED"
}
`
	output, err = lew.ExecuteScriptString(script)
	if err != nil {
		return fmt.Errorf("传统兼容性测试失败 - COM对象访问: %w", err)
	}

	if !strings.Contains(output, "COM_OK") {
		return fmt.Errorf("传统兼容性测试失败 - COM对象访问不可用")
	}

	lew.log.Debug("传统兼容性测试通过")
	return nil
}

// ExecutorPool 执行器池，用于管理多个执行器实例
type ExecutorPool struct {
	executors map[string]CommandExecutor
	log       *logger.Logger
}

// NewExecutorPool 创建执行器池
func NewExecutorPool(log *logger.Logger) *ExecutorPool {
	return &ExecutorPool{
		executors: make(map[string]CommandExecutor),
		log:       log,
	}
}

// GetExecutor 获取指定名称的执行器
func (ep *ExecutorPool) GetExecutor(name string) (CommandExecutor, error) {
	executor, exists := ep.executors[name]
	if !exists {
		return nil, fmt.Errorf("执行器 '%s' 不存在", name)
	}
	return executor, nil
}

// AddExecutor 添加执行器到池中
func (ep *ExecutorPool) AddExecutor(name string, executor CommandExecutor) error {
	if _, exists := ep.executors[name]; exists {
		return fmt.Errorf("执行器 '%s' 已存在", name)
	}

	ep.executors[name] = executor
	ep.log.Debug("执行器 '%s' 已添加到池中", name)
	return nil
}

// RemoveExecutor 从池中移除执行器
func (ep *ExecutorPool) RemoveExecutor(name string) error {
	executor, exists := ep.executors[name]
	if !exists {
		return fmt.Errorf("执行器 '%s' 不存在", name)
	}

	// 关闭执行器
	if err := executor.Close(); err != nil {
		ep.log.Warn("关闭执行器 '%s' 时出错: %v", name, err)
	}

	delete(ep.executors, name)
	ep.log.Debug("执行器 '%s' 已从池中移除", name)
	return nil
}

// TestAllExecutors 测试池中所有执行器
func (ep *ExecutorPool) TestAllExecutors() map[string]error {
	results := make(map[string]error)

	for name, executor := range ep.executors {
		if err := executor.TestExecution(); err != nil {
			results[name] = err
			ep.log.Error("执行器 '%s' 测试失败: %v", name, err)
		} else {
			results[name] = nil
			ep.log.Debug("执行器 '%s' 测试成功", name)
		}
	}

	return results
}

// CloseAll 关闭所有执行器
func (ep *ExecutorPool) CloseAll() {
	for name, executor := range ep.executors {
		if err := executor.Close(); err != nil {
			ep.log.Warn("关闭执行器 '%s' 时出错: %v", name, err)
		}
	}

	ep.executors = make(map[string]CommandExecutor)
	ep.log.Debug("所有执行器已关闭")
}

// GetExecutorStats 获取执行器统计信息
func (ep *ExecutorPool) GetExecutorStats() map[string]interface{} {
	stats := make(map[string]interface{})

	executorCount := len(ep.executors)
	stats["total_executors"] = executorCount
	stats["executor_names"] = make([]string, 0, executorCount)

	for name := range ep.executors {
		stats["executor_names"] = append(stats["executor_names"].([]string), name)
	}

	return stats
}

// GlobalExecutorPool 全局执行器池实例
var GlobalExecutorPool *ExecutorPool

// InitGlobalExecutorPool 初始化全局执行器池
func InitGlobalExecutorPool(log *logger.Logger) {
	GlobalExecutorPool = NewExecutorPool(log)
	log.Debug("全局执行器池已初始化")
}

// GetGlobalExecutorPool 获取全局执行器池
func GetGlobalExecutorPool() *ExecutorPool {
	if GlobalExecutorPool == nil {
		// 如果未初始化，使用默认日志创建
		defaultLog := logger.NewLogger(false)
		GlobalExecutorPool = NewExecutorPool(defaultLog)
	}
	return GlobalExecutorPool
}