// core/bind/history.go
// BIND备份及恢复历史记录管理

package bind

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"SteadyDNS/core/common"
)

// 常量定义
const (
	// 备份文件路径
	BackupFilePath = "./backup/history.record"
	// 备份目录
	BackupDirPath = "./backup/"
	// 临时目录
	TempDirPath = "./backup/temp/"
	// 回退保护文件前缀
	RollbackProtectionPrefix = "history.record.rollback."
	// 30天过期时间（秒）
	ExpiryDuration = 30 * 24 * 60 * 60
	// 魔法数字
	MagicNumber = "STEADYDNS_BACKUP"
	// 当前版本
	CurrentVersion = 1
	// 默认回退保护最大记录数
	DefaultRollbackProtectionMaxRecords = 10
)

// 操作类型
const (
	OperationCreate uint8 = iota
	OperationUpdate
	OperationDelete
	OperationRollback
)

// FileHeader 文件头
type FileHeader struct {
	MagicNumber [16]byte // 文件标识
	Version     uint32   // 文件版本
	RecordCount uint64   // 记录数量
	IndexOffset uint64   // 索引区域偏移
	IndexSize   uint64   // 索引区域大小
	NextExpiry  int64    // 下一个过期时间戳
}

// IndexEntry 索引条目
type IndexEntry struct {
	RecordID   uint64 // 记录ID
	Offset     uint64 // 元数据偏移
	Size       uint64 // 元数据大小
	Operation  uint8  // 操作类型
	Domain     string // 操作域名
	Timestamp  int64  // 操作时间戳
	ExpiryTime int64  // 过期时间戳
}

// MetadataEntry 元数据条目
type MetadataEntry struct {
	RecordID      uint64     // 记录ID
	Operation     uint8      // 操作类型
	Domain        string     // 操作域名
	Timestamp     int64      // 操作时间戳
	ExpiryTime    int64      // 过期时间戳
	OperationData []byte     // 操作数据（JSON格式）
	Files         []FileInfo // 涉及的文件信息
}

// FileInfo 文件信息
type FileInfo struct {
	FileID       uint64 // 文件ID
	FileName     string // 文件名
	BeforeOffset uint64 // 修改前文件内容偏移
	BeforeSize   uint64 // 修改前文件内容大小
	AfterOffset  uint64 // 修改后文件内容偏移
	AfterSize    uint64 // 修改后文件内容大小
}

// HistoryManager 历史记录管理器
type HistoryManager struct {
	logger       *common.Logger
	mutex        sync.Mutex
	nextRecordID uint64
}

// NewHistoryManager 创建历史记录管理器实例
func NewHistoryManager() *HistoryManager {
	return &HistoryManager{
		logger:       common.NewLogger(),
		nextRecordID: 1,
	}
}

// CreateBackup 创建备份
// operation: 操作类型
// domain: 操作域名
// operationData: 操作数据（JSON格式）
// files: 涉及的文件列表
func (hm *HistoryManager) CreateBackup(operation uint8, domain string, operationData []byte, files []string) (uint64, error) {
	// 执行自动清理
	if _, err := hm.CleanupExpiredRecords(); err != nil {
		hm.logger.Warn("自动清理过期记录失败: %v", err)
		// 清理失败不影响备份操作，继续执行
	}

	// 加锁保证并发安全
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	// 确保备份目录存在
	os.MkdirAll(BackupDirPath, 0755)
	os.MkdirAll(TempDirPath, 0755)

	// 读取操作前的文件内容
	beforeFiles := make(map[string][]byte)

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			hm.logger.Error("读取文件失败: %v", err)
			return 0, fmt.Errorf("读取文件失败: %v", err)
		}
		beforeFiles[file] = content
	}

	// 这里会执行核心操作，由调用方负责
	// 调用方执行操作后，需要再次读取文件内容
	// 此处只是演示备份逻辑，实际操作由调用方执行

	// 创建临时文件用于存储压缩后的文件内容
	tempFile, err := os.CreateTemp(TempDirPath, "backup_")
	if err != nil {
		hm.logger.Error("创建临时文件失败: %v", err)
		return 0, fmt.Errorf("创建临时文件失败: %v", err)
	}
	defer tempFile.Close()
	defer os.Remove(tempFile.Name())

	// 写入文件头
	header := FileHeader{
		MagicNumber: [16]byte{},
		Version:     CurrentVersion,
		RecordCount: 0,
		IndexOffset: 0,
		IndexSize:   0,
		NextExpiry:  time.Now().Add(time.Duration(ExpiryDuration) * time.Second).Unix(),
	}
	copy(header.MagicNumber[:], MagicNumber)

	if err := binary.Write(tempFile, binary.BigEndian, header); err != nil {
		hm.logger.Error("写入文件头失败: %v", err)
		return 0, fmt.Errorf("写入文件头失败: %v", err)
	}

	// 写入索引区域（暂时为空）
	indexOffset64, _ := tempFile.Seek(0, io.SeekCurrent)
	indexOffset := uint64(indexOffset64)
	tempFile.Seek(int64(indexOffset+1024), io.SeekStart) // 预留索引空间

	// 写入元数据区域
	metaOffset64, _ := tempFile.Seek(0, io.SeekCurrent)
	metaOffset := uint64(metaOffset64)
	metaEntry := MetadataEntry{
		RecordID:      hm.nextRecordID,
		Operation:     operation,
		Domain:        domain,
		Timestamp:     time.Now().Unix(),
		ExpiryTime:    time.Now().Add(time.Duration(ExpiryDuration) * time.Second).Unix(),
		OperationData: operationData,
		Files:         make([]FileInfo, 0),
	}

	// 压缩文件内容并写入
	fileID := uint64(1)
	for fileName, content := range beforeFiles {
		// 压缩文件内容
		compressedContent := compressContent(content)

		// 写入压缩后的文件内容
		dataOffset64, _ := tempFile.Seek(0, io.SeekCurrent)
		dataOffset := uint64(dataOffset64)
		dataSize := uint64(len(compressedContent))

		if _, err := tempFile.Write(compressedContent); err != nil {
			hm.logger.Error("写入压缩文件内容失败: %v", err)
			return 0, fmt.Errorf("写入压缩文件内容失败: %v", err)
		}

		// 记录文件信息
		fileInfo := FileInfo{
			FileID:       fileID,
			FileName:     fileName,
			BeforeOffset: dataOffset,
			BeforeSize:   dataSize,
			AfterOffset:  dataOffset, // 临时值，实际操作后更新
			AfterSize:    dataSize,   // 临时值，实际操作后更新
		}
		metaEntry.Files = append(metaEntry.Files, fileInfo)
		fileID++
	}

	// 写入元数据
	metaBytes, err := json.Marshal(metaEntry)
	if err != nil {
		hm.logger.Error("序列化元数据失败: %v", err)
		return 0, fmt.Errorf("序列化元数据失败: %v", err)
	}

	metaSize := uint64(len(metaBytes))
	if _, err := tempFile.Write(metaBytes); err != nil {
		hm.logger.Error("写入元数据失败: %v", err)
		return 0, fmt.Errorf("写入元数据失败: %v", err)
	}

	// 写入索引条目
	indexEntry := IndexEntry{
		RecordID:   hm.nextRecordID,
		Offset:     metaOffset,
		Size:       metaSize,
		Operation:  operation,
		Domain:     domain,
		Timestamp:  time.Now().Unix(),
		ExpiryTime: metaEntry.ExpiryTime,
	}

	indexBytes, err := json.Marshal(indexEntry)
	if err != nil {
		hm.logger.Error("序列化索引条目失败: %v", err)
		return 0, fmt.Errorf("序列化索引条目失败: %v", err)
	}

	// 移动到索引区域起始位置
	tempFile.Seek(int64(indexOffset), io.SeekStart)
	if _, err := tempFile.Write(indexBytes); err != nil {
		hm.logger.Error("写入索引条目失败: %v", err)
		return 0, fmt.Errorf("写入索引条目失败: %v", err)
	}

	// 更新文件头
	tempFile.Seek(0, io.SeekStart)
	header.RecordCount = 1
	header.IndexSize = uint64(len(indexBytes))
	header.IndexOffset = indexOffset

	if err := binary.Write(tempFile, binary.BigEndian, header); err != nil {
		hm.logger.Error("更新文件头失败: %v", err)
		return 0, fmt.Errorf("更新文件头失败: %v", err)
	}

	// 复制临时文件到正式备份文件
	backupPath := filepath.Join(common.GetConfig("Server", "WORKING_DIR"), BackupFilePath)
	if err := os.Rename(tempFile.Name(), backupPath); err != nil {
		hm.logger.Error("保存备份文件失败: %v", err)
		return 0, fmt.Errorf("保存备份文件失败: %v", err)
	}

	recordID := hm.nextRecordID
	hm.nextRecordID++

	hm.logger.Info("备份成功，记录ID: %d", recordID)
	return recordID, nil
}

// RestoreBackup 恢复指定事务ID的备份
func (hm *HistoryManager) RestoreBackup(recordID uint64) error {
	// 加锁保证并发安全
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	hm.logger.Info("开始恢复记录ID: %d", recordID)

	// 在回退前备份当前history.record文件
	if err := hm.BackupHistoryRecord(recordID); err != nil {
		hm.logger.Error("备份history.record文件失败: %v", err)
		return fmt.Errorf("备份history.record文件失败: %v", err)
	}

	// 读取备份文件
	backupPath := filepath.Join(common.GetConfig("Server", "WORKING_DIR"), BackupFilePath)
	file, err := os.Open(backupPath)
	if err != nil {
		hm.logger.Error("打开备份文件失败: %v", err)
		return fmt.Errorf("打开备份文件失败: %v", err)
	}
	defer file.Close()

	// 读取文件头
	var header FileHeader
	if err := binary.Read(file, binary.BigEndian, &header); err != nil {
		hm.logger.Error("读取文件头失败: %v", err)
		return fmt.Errorf("读取文件头失败: %v", err)
	}

	// 验证魔法数字
	if string(header.MagicNumber[:len(MagicNumber)]) != MagicNumber {
		hm.logger.Error("无效的备份文件")
		return fmt.Errorf("无效的备份文件")
	}

	// 读取索引区域
	file.Seek(int64(header.IndexOffset), io.SeekStart)
	indexData := make([]byte, header.IndexSize)
	if _, err := file.Read(indexData); err != nil {
		hm.logger.Error("读取索引区域失败: %v", err)
		return fmt.Errorf("读取索引区域失败: %v", err)
	}

	// 解析索引条目
	var indexEntry IndexEntry
	if err := json.Unmarshal(indexData, &indexEntry); err != nil {
		hm.logger.Error("解析索引条目失败: %v", err)
		return fmt.Errorf("解析索引条目失败: %v", err)
	}

	// 检查记录ID是否匹配
	if indexEntry.RecordID != recordID {
		hm.logger.Error("记录ID不匹配")
		return fmt.Errorf("记录ID不匹配")
	}

	// 检查是否过期
	if indexEntry.ExpiryTime < time.Now().Unix() {
		hm.logger.Error("记录已过期")
		return fmt.Errorf("记录已过期")
	}

	// 读取元数据
	file.Seek(int64(indexEntry.Offset), io.SeekStart)
	metaData := make([]byte, indexEntry.Size)
	if _, err := file.Read(metaData); err != nil {
		hm.logger.Error("读取元数据失败: %v", err)
		return fmt.Errorf("读取元数据失败: %v", err)
	}

	// 解析元数据
	var metaEntry MetadataEntry
	if err := json.Unmarshal(metaData, &metaEntry); err != nil {
		hm.logger.Error("解析元数据失败: %v", err)
		return fmt.Errorf("解析元数据失败: %v", err)
	}

	// 恢复文件
	for _, fileInfo := range metaEntry.Files {
		// 读取修改前的文件内容
		file.Seek(int64(fileInfo.BeforeOffset), io.SeekStart)
		compressedContent := make([]byte, fileInfo.BeforeSize)
		if _, err := file.Read(compressedContent); err != nil {
			hm.logger.Error("读取压缩文件内容失败: %v", err)
			return fmt.Errorf("读取压缩文件内容失败: %v", err)
		}

		// 解压文件内容
		content := decompressContent(compressedContent)

		// 写入文件
		if err := os.WriteFile(fileInfo.FileName, content, 0644); err != nil {
			hm.logger.Error("写入文件失败: %v", err)
			return fmt.Errorf("写入文件失败: %v", err)
		}

		hm.logger.Info("恢复文件成功: %s", fileInfo.FileName)
	}

	// 记录回退操作
	rollbackData, _ := json.Marshal(map[string]interface{}{
		"rollback_record_id": recordID,
		"rollback_operation": metaEntry.Operation,
		"rollback_domain":    metaEntry.Domain,
	})

	files := make([]string, 0)
	for _, fileInfo := range metaEntry.Files {
		files = append(files, fileInfo.FileName)
	}

	// 回退操作也是事务性操作，会被记录
	if _, err := hm.CreateBackup(OperationRollback, metaEntry.Domain, rollbackData, files); err != nil {
		hm.logger.Warn("记录回退操作失败: %v", err)
		// 记录失败不影响回退操作，继续执行
	}

	hm.logger.Info("恢复记录ID: %d 成功", recordID)
	return nil
}

// BatchRestore 批量回退到指定事务ID
func (hm *HistoryManager) BatchRestore(toRecordID uint64) error {
	// 加锁保证并发安全
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	hm.logger.Info("开始批量回退到记录ID: %d", toRecordID)

	// 读取备份文件
	backupPath := filepath.Join(common.GetConfig("Server", "WORKING_DIR"), BackupFilePath)
	file, err := os.Open(backupPath)
	if err != nil {
		hm.logger.Error("打开备份文件失败: %v", err)
		return fmt.Errorf("打开备份文件失败: %v", err)
	}
	defer file.Close()

	// 读取文件头
	var header FileHeader
	if err := binary.Read(file, binary.BigEndian, &header); err != nil {
		hm.logger.Error("读取文件头失败: %v", err)
		return fmt.Errorf("读取文件头失败: %v", err)
	}

	// 验证魔法数字
	if string(header.MagicNumber[:len(MagicNumber)]) != MagicNumber {
		hm.logger.Error("无效的备份文件")
		return fmt.Errorf("无效的备份文件")
	}

	// 读取索引区域
	file.Seek(int64(header.IndexOffset), io.SeekStart)
	indexData := make([]byte, header.IndexSize)
	if _, err := file.Read(indexData); err != nil {
		hm.logger.Error("读取索引区域失败: %v", err)
		return fmt.Errorf("读取索引区域失败: %v", err)
	}

	// 解析索引条目
	var indexEntry IndexEntry
	if err := json.Unmarshal(indexData, &indexEntry); err != nil {
		hm.logger.Error("解析索引条目失败: %v", err)
		return fmt.Errorf("解析索引条目失败: %v", err)
	}

	// 检查是否需要回退
	if indexEntry.RecordID <= toRecordID {
		hm.logger.Info("当前记录ID: %d 已小于等于目标记录ID: %d，无需回退", indexEntry.RecordID, toRecordID)
		return nil
	}

	// 执行回退
	return hm.RestoreBackup(toRecordID)
}

// GetHistoryRecords 获取历史记录
func (hm *HistoryManager) GetHistoryRecords() ([]IndexEntry, error) {
	// 加锁保证并发安全
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	records := make([]IndexEntry, 0)

	// 读取备份文件
	backupPath := filepath.Join(common.GetConfig("Server", "WORKING_DIR"), BackupFilePath)
	file, err := os.Open(backupPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 备份文件不存在，返回空列表
			return records, nil
		}
		hm.logger.Error("打开备份文件失败: %v", err)
		return nil, fmt.Errorf("打开备份文件失败: %v", err)
	}
	defer file.Close()

	// 读取文件头
	var header FileHeader
	if err := binary.Read(file, binary.BigEndian, &header); err != nil {
		hm.logger.Error("读取文件头失败: %v", err)
		return nil, fmt.Errorf("读取文件头失败: %v", err)
	}

	// 验证魔法数字
	if string(header.MagicNumber[:len(MagicNumber)]) != MagicNumber {
		hm.logger.Error("无效的备份文件")
		return nil, fmt.Errorf("无效的备份文件")
	}

	// 读取索引区域
	file.Seek(int64(header.IndexOffset), io.SeekStart)
	indexData := make([]byte, header.IndexSize)
	if _, err := file.Read(indexData); err != nil {
		hm.logger.Error("读取索引区域失败: %v", err)
		return nil, fmt.Errorf("读取索引区域失败: %v", err)
	}

	// 解析索引条目
	var indexEntry IndexEntry
	if err := json.Unmarshal(indexData, &indexEntry); err != nil {
		hm.logger.Error("解析索引条目失败: %v", err)
		return nil, fmt.Errorf("解析索引条目失败: %v", err)
	}

	// 检查是否过期
	if indexEntry.ExpiryTime >= time.Now().Unix() {
		records = append(records, indexEntry)
	}

	return records, nil
}

// GetHistoryRecord 获取单个历史记录详情
func (hm *HistoryManager) GetHistoryRecord(recordID uint64) (*MetadataEntry, error) {
	// 加锁保证并发安全
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	// 读取备份文件
	backupPath := filepath.Join(common.GetConfig("Server", "WORKING_DIR"), BackupFilePath)
	file, err := os.Open(backupPath)
	if err != nil {
		hm.logger.Error("打开备份文件失败: %v", err)
		return nil, fmt.Errorf("打开备份文件失败: %v", err)
	}
	defer file.Close()

	// 读取文件头
	var header FileHeader
	if err := binary.Read(file, binary.BigEndian, &header); err != nil {
		hm.logger.Error("读取文件头失败: %v", err)
		return nil, fmt.Errorf("读取文件头失败: %v", err)
	}

	// 验证魔法数字
	if string(header.MagicNumber[:len(MagicNumber)]) != MagicNumber {
		hm.logger.Error("无效的备份文件")
		return nil, fmt.Errorf("无效的备份文件")
	}

	// 读取索引区域
	file.Seek(int64(header.IndexOffset), io.SeekStart)
	indexData := make([]byte, header.IndexSize)
	if _, err := file.Read(indexData); err != nil {
		hm.logger.Error("读取索引区域失败: %v", err)
		return nil, fmt.Errorf("读取索引区域失败: %v", err)
	}

	// 解析索引条目
	var indexEntry IndexEntry
	if err := json.Unmarshal(indexData, &indexEntry); err != nil {
		hm.logger.Error("解析索引条目失败: %v", err)
		return nil, fmt.Errorf("解析索引条目失败: %v", err)
	}

	// 检查记录ID是否匹配
	if indexEntry.RecordID != recordID {
		hm.logger.Error("记录ID不匹配")
		return nil, fmt.Errorf("记录ID不匹配")
	}

	// 检查是否过期
	if indexEntry.ExpiryTime < time.Now().Unix() {
		hm.logger.Error("记录已过期")
		return nil, fmt.Errorf("记录已过期")
	}

	// 读取元数据
	file.Seek(int64(indexEntry.Offset), io.SeekStart)
	metaData := make([]byte, indexEntry.Size)
	if _, err := file.Read(metaData); err != nil {
		hm.logger.Error("读取元数据失败: %v", err)
		return nil, fmt.Errorf("读取元数据失败: %v", err)
	}

	// 解析元数据
	var metaEntry MetadataEntry
	if err := json.Unmarshal(metaData, &metaEntry); err != nil {
		hm.logger.Error("解析元数据失败: %v", err)
		return nil, fmt.Errorf("解析元数据失败: %v", err)
	}

	return &metaEntry, nil
}

// BackupHistoryRecord 备份history.record文件
func (hm *HistoryManager) BackupHistoryRecord(transactionID uint64) error {
	// 加锁保证并发安全
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	hm.logger.Info("开始备份history.record文件，事务ID: %d", transactionID)

	// 确保备份目录存在
	os.MkdirAll(BackupDirPath, 0755)

	// 源文件路径
	srcPath := filepath.Join(common.GetConfig("Server", "WORKING_DIR"), BackupFilePath)

	// 检查源文件是否存在
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		hm.logger.Warn("history.record文件不存在，跳过备份")
		return nil
	}

	// 目标文件路径：history.record.rollback.{transaction_id}
	dstPath := filepath.Join(common.GetConfig("Server", "WORKING_DIR"), "./backup/", fmt.Sprintf("%s%d", RollbackProtectionPrefix, transactionID))

	// 读取源文件内容
	content, err := os.ReadFile(srcPath)
	if err != nil {
		hm.logger.Error("读取history.record文件失败: %v", err)
		return fmt.Errorf("读取history.record文件失败: %v", err)
	}

	// 写入备份文件
	if err := os.WriteFile(dstPath, content, 0644); err != nil {
		hm.logger.Error("写入回退保护文件失败: %v", err)
		return fmt.Errorf("写入回退保护文件失败: %v", err)
	}

	// 清理过期的回退保护文件
	if err := hm.CleanupRollbackProtectionFiles(); err != nil {
		hm.logger.Warn("清理回退保护文件失败: %v", err)
		// 清理失败不影响备份操作，继续执行
	}

	hm.logger.Info("备份history.record文件成功，备份文件: %s", dstPath)
	return nil
}

// GetRollbackProtectionFiles 获取所有回退保护文件
func (hm *HistoryManager) GetRollbackProtectionFiles() ([]string, error) {
	// 加锁保证并发安全
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	// 备份目录路径
	backupDir := filepath.Join(common.GetConfig("Server", "WORKING_DIR"), "./backup/")

	// 读取备份目录
	files, err := os.ReadDir(backupDir)
	if err != nil {
		hm.logger.Error("读取备份目录失败: %v", err)
		return nil, fmt.Errorf("读取备份目录失败: %v", err)
	}

	// 筛选回退保护文件
	rollbackFiles := make([]string, 0)
	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), RollbackProtectionPrefix) {
			rollbackFiles = append(rollbackFiles, file.Name())
		}
	}

	return rollbackFiles, nil
}

// CleanupRollbackProtectionFiles 清理过期的回退保护文件
func (hm *HistoryManager) CleanupRollbackProtectionFiles() error {
	// 加锁保证并发安全
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	hm.logger.Info("开始清理回退保护文件")

	// 获取配置的最大记录数
	maxRecordsStr := common.GetConfig("Backup", "ROLLBACK_PROTECTION_MAX_RECORDS")
	maxRecords := DefaultRollbackProtectionMaxRecords
	if maxRecordsStr != "" {
		if val, err := strconv.Atoi(maxRecordsStr); err == nil {
			maxRecords = val
		}
	}

	// 备份目录路径
	backupDir := filepath.Join(common.GetConfig("Server", "WORKING_DIR"), "./backup/")

	// 读取备份目录
	files, err := os.ReadDir(backupDir)
	if err != nil {
		hm.logger.Error("读取备份目录失败: %v", err)
		return fmt.Errorf("读取备份目录失败: %v", err)
	}

	// 筛选回退保护文件并排序
	rollbackFiles := make([]os.DirEntry, 0)
	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), RollbackProtectionPrefix) {
			rollbackFiles = append(rollbackFiles, file)
		}
	}

	// 按修改时间排序， oldest first
	sort.Slice(rollbackFiles, func(i, j int) bool {
		infoI, _ := rollbackFiles[i].Info()
		infoJ, _ := rollbackFiles[j].Info()
		return infoI.ModTime().Before(infoJ.ModTime())
	})

	// 计算需要删除的文件数量
	deleteCount := len(rollbackFiles) - maxRecords
	if deleteCount <= 0 {
		hm.logger.Info("回退保护文件数量未超过限制，无需清理")
		return nil
	}

	// 删除最旧的文件
	deletedCount := 0
	for i := 0; i < deleteCount && i < len(rollbackFiles); i++ {
		filePath := filepath.Join(backupDir, rollbackFiles[i].Name())
		if err := os.Remove(filePath); err != nil {
			hm.logger.Error("删除回退保护文件失败: %v", err)
			continue
		}
		deletedCount++
		hm.logger.Info("删除回退保护文件成功: %s", rollbackFiles[i].Name())
	}

	hm.logger.Info("清理回退保护文件完成，删除了 %d 个文件", deletedCount)
	return nil
}

// RestoreFromRollbackBackup 从回退保护备份恢复
func (hm *HistoryManager) RestoreFromRollbackBackup(transactionID uint64) error {
	// 加锁保证并发安全
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	hm.logger.Info("开始从回退保护备份恢复，事务ID: %d", transactionID)

	// 备份文件路径
	backupPath := filepath.Join(common.GetConfig("Server", "WORKING_DIR"), "./backup/", fmt.Sprintf("%s%d", RollbackProtectionPrefix, transactionID))

	// 检查备份文件是否存在
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		hm.logger.Error("回退保护备份文件不存在: %s", backupPath)
		return fmt.Errorf("回退保护备份文件不存在: %s", backupPath)
	}

	// 目标文件路径
	dstPath := filepath.Join(common.GetConfig("Server", "WORKING_DIR"), BackupFilePath)

	// 读取备份文件内容
	content, err := os.ReadFile(backupPath)
	if err != nil {
		hm.logger.Error("读取回退保护备份文件失败: %v", err)
		return fmt.Errorf("读取回退保护备份文件失败: %v", err)
	}

	// 写入目标文件
	if err := os.WriteFile(dstPath, content, 0644); err != nil {
		hm.logger.Error("写入history.record文件失败: %v", err)
		return fmt.Errorf("写入history.record文件失败: %v", err)
	}

	hm.logger.Info("从回退保护备份恢复成功，恢复文件: %s", backupPath)
	return nil
}

// RollbackRollback 回退回退操作
func (hm *HistoryManager) RollbackRollback(rollbackTransactionID uint64) error {
	// 加锁保证并发安全
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	hm.logger.Info("开始回退回退操作，回退事务ID: %d", rollbackTransactionID)

	// 从回退保护备份恢复
	if err := hm.RestoreFromRollbackBackup(rollbackTransactionID); err != nil {
		hm.logger.Error("从回退保护备份恢复失败: %v", err)
		return fmt.Errorf("从回退保护备份恢复失败: %v", err)
	}

	// 删除对应的回退保护文件
	backupPath := filepath.Join(common.GetConfig("Server", "WORKING_DIR"), "./backup/", fmt.Sprintf("%s%d", RollbackProtectionPrefix, rollbackTransactionID))
	if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
		hm.logger.Warn("删除回退保护文件失败: %v", err)
		// 删除失败不影响回退操作，继续执行
	} else {
		hm.logger.Info("删除回退保护文件成功: %s", backupPath)
	}

	hm.logger.Info("回退回退操作成功，回退事务ID: %d", rollbackTransactionID)
	return nil
}

// CleanupExpiredRecords 清理过期记录
func (hm *HistoryManager) CleanupExpiredRecords() (int, error) {
	// 加锁保证并发安全
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	hm.logger.Info("开始清理过期记录")

	// 检查备份文件是否存在
	backupPath := filepath.Join(common.GetConfig("Server", "WORKING_DIR"), BackupFilePath)
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		hm.logger.Info("备份文件不存在，无需清理")
		return 0, nil
	}

	// 这里简化实现，实际需要扫描所有记录并清理过期记录
	// 由于当前只支持单条记录，所以只需要检查当前记录是否过期

	// 读取备份文件
	file, err := os.Open(backupPath)
	if err != nil {
		hm.logger.Error("打开备份文件失败: %v", err)
		return 0, fmt.Errorf("打开备份文件失败: %v", err)
	}
	defer file.Close()

	// 读取文件头
	var header FileHeader
	if err := binary.Read(file, binary.BigEndian, &header); err != nil {
		hm.logger.Error("读取文件头失败: %v", err)
		return 0, fmt.Errorf("读取文件头失败: %v", err)
	}

	// 验证魔法数字
	if string(header.MagicNumber[:len(MagicNumber)]) != MagicNumber {
		hm.logger.Error("无效的备份文件")
		return 0, fmt.Errorf("无效的备份文件")
	}

	// 读取索引区域
	file.Seek(int64(header.IndexOffset), io.SeekStart)
	indexData := make([]byte, header.IndexSize)
	if _, err := file.Read(indexData); err != nil {
		hm.logger.Error("读取索引区域失败: %v", err)
		return 0, fmt.Errorf("读取索引区域失败: %v", err)
	}

	// 解析索引条目
	var indexEntry IndexEntry
	if err := json.Unmarshal(indexData, &indexEntry); err != nil {
		hm.logger.Error("解析索引条目失败: %v", err)
		return 0, fmt.Errorf("解析索引条目失败: %v", err)
	}

	// 检查是否过期
	if indexEntry.ExpiryTime < time.Now().Unix() {
		// 过期记录，删除备份文件
		if err := os.Remove(backupPath); err != nil {
			hm.logger.Error("删除过期备份文件失败: %v", err)
			return 0, fmt.Errorf("删除过期备份文件失败: %v", err)
		}
		hm.logger.Info("清理过期记录ID: %d 成功", indexEntry.RecordID)
		return 1, nil
	}

	hm.logger.Info("没有过期记录需要清理")
	return 0, nil
}

// compressContent 压缩内容
func compressContent(content []byte) []byte {
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)

	if _, err := gzipWriter.Write(content); err != nil {
		return content // 压缩失败，返回原内容
	}

	gzipWriter.Close()
	return buffer.Bytes()
}

// decompressContent 解压内容
func decompressContent(content []byte) []byte {
	reader, err := gzip.NewReader(bytes.NewReader(content))
	if err != nil {
		return content // 解压失败，返回原内容
	}

	decompressed, err := io.ReadAll(reader)
	reader.Close()

	if err != nil {
		return content // 解压失败，返回原内容
	}

	return decompressed
}
