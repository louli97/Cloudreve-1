package aria2

import (
	"sync"

	model "github.com/cloudreve/Cloudreve/v3/models"
	"github.com/cloudreve/Cloudreve/v3/pkg/aria2/rpc"
	"github.com/cloudreve/Cloudreve/v3/pkg/balancer"
	"github.com/cloudreve/Cloudreve/v3/pkg/serializer"
)

// Instance 默认使用的Aria2处理实例
var Instance Aria2 = &DummyAria2{}

// LB 获取 Aria2 节点的负载均衡器
var LB balancer.Balancer

// Lock Instance的读写锁
var Lock sync.RWMutex

// EventNotifier 任务状态更新通知处理器
var EventNotifier = &Notifier{}

// Aria2 离线下载处理接口
type Aria2 interface {
	// Init 初始化客户端连接
	Init() error
	// CreateTask 创建新的任务
	CreateTask(task *model.Download, options map[string]interface{}) (string, error)
	// 返回状态信息
	Status(task *model.Download) (rpc.StatusInfo, error)
	// 取消任务
	Cancel(task *model.Download) error
	// 选择要下载的文件
	Select(task *model.Download, files []int) error
}

const (
	// URLTask 从URL添加的任务
	URLTask = iota
	// TorrentTask 种子任务
	TorrentTask
)

const (
	// Ready 准备就绪
	Ready = iota
	// Downloading 下载中
	Downloading
	// Paused 暂停中
	Paused
	// Error 出错
	Error
	// Complete 完成
	Complete
	// Canceled 取消/停止
	Canceled
	// Unknown 未知状态
	Unknown
)

var (
	// ErrNotEnabled 功能未开启错误
	ErrNotEnabled = serializer.NewError(serializer.CodeNoPermissionErr, "离线下载功能未开启", nil)
	// ErrUserNotFound 未找到下载任务创建者
	ErrUserNotFound = serializer.NewError(serializer.CodeNotFound, "无法找到任务创建者", nil)
)

// DummyAria2 未开启Aria2功能时使用的默认处理器
type DummyAria2 struct {
}

func (instance *DummyAria2) Init() error {
	return nil
}

// CreateTask 创建新任务，此处直接返回未开启错误
func (instance *DummyAria2) CreateTask(model *model.Download, options map[string]interface{}) (string, error) {
	return "", ErrNotEnabled
}

// Status 返回未开启错误
func (instance *DummyAria2) Status(task *model.Download) (rpc.StatusInfo, error) {
	return rpc.StatusInfo{}, ErrNotEnabled
}

// Cancel 返回未开启错误
func (instance *DummyAria2) Cancel(task *model.Download) error {
	return ErrNotEnabled
}

// Select 返回未开启错误
func (instance *DummyAria2) Select(task *model.Download, files []int) error {
	return ErrNotEnabled
}

// Init 初始化
func Init(isReload bool) {
	Lock.Lock()
	LB = balancer.NewBalancer("RoundRobin")
	Lock.Unlock()

	if !isReload {
		// 从数据库中读取未完成任务，创建监控
		unfinished := model.GetDownloadsByStatus(Ready, Paused, Downloading)

		for i := 0; i < len(unfinished); i++ {
			// 创建任务监控
			NewMonitor(&unfinished[i])
		}
	}
}

// getStatus 将给定的状态字符串转换为状态标识数字
func getStatus(status string) int {
	switch status {
	case "complete":
		return Complete
	case "active":
		return Downloading
	case "waiting":
		return Ready
	case "paused":
		return Paused
	case "error":
		return Error
	case "removed":
		return Canceled
	default:
		return Unknown
	}
}
