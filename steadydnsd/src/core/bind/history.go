/*
SteadyDNS - DNS服务器实现

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

// core/bind/history.go
// BIND备份及恢复历史记录管理 - 支持内容去重和多记录

package bind

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
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
	CurrentVersion = 3
	// 最小兼容版本
	MinCompatVersion = 1
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

// ContentBlock 内容块（用于去重）
type ContentBlock struct {
	Hash     [32]byte `json:"hash"`     // 内容哈希
	Size     uint64   `json:"size"`     // 原始大小
	Offset   uint64   `json:"offset"`   // 在文件中的偏移
	Length   uint64   `json:"length"`   // 压缩后长度
	RefCount uint64   `json:"refCount"` // 引用计数
}

// contentBlockMap 内容块映射（Hash -> ContentBlock）
type contentBlockMap map[[32]byte]*ContentBlock

// FileHeader 文件头
type FileHeader struct {
	MagicNumber        [16]byte // 文件标识
	Version            uint32   // 文件版本
	RecordCount        uint64   // 记录数量
	IndexOffset        uint64   // 索引区域偏移
	IndexSize          uint64   // 索引区域大小
	ContentBlockOffset uint64   // 内容块索引偏移
	ContentBlockSize   uint64   // 内容块索引大小
	DataOffset         uint64   // 数据区域偏移
	Checksum           [32]byte // 整体校验和
	Reserved           [64]byte // 保留字段
	TotalSize          uint64   // 文件总大小
}

// IndexEntry 索引条目
type IndexEntry struct {
	RecordID      uint64   // 记录ID
	Offset        uint64   // 元数据偏移
	Size          uint64   // 元数据大小
	Operation     uint8    // 操作类型
	Domain        string   // 操作域名
	Timestamp     int64    // 操作时间戳
	ExpiryTime    int64    // 过期时间戳
	ContentOffset uint64   // 内容偏移（旧格式兼容）
	ContentSize   uint64   // 内容大小（旧格式兼容）
	Checksum      [32]byte // 记录校验和
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
	FileID       uint64   // 文件ID
	FileName     string   // 文件名
	ContentHash  [32]byte // 内容哈希（新格式使用）
	LastModified int64    // 最后修改时间
}

// HistoryManager 历史记录管理器
type HistoryManager struct {
	logger            *common.Logger
	mutex             sync.Mutex
	nextRecordID      uint64
	bindManager       *BindManager
	pendingIndexEntry *IndexEntry
}

// NewHistoryManager 创建历史记录管理器实例
func NewHistoryManager() *HistoryManager {
	return &HistoryManager{
		logger:       common.NewLogger(),
		nextRecordID: 1,
	}
}

// SetBindManager 设置BindManager引用
func (hm *HistoryManager) SetBindManager(bm *BindManager) {
	hm.bindManager = bm
}

// CreateBackup 创建备份（支持内容去重和多记录）
func (hm *HistoryManager) CreateBackup(operation uint8, domain string, operationData []byte, files []string) (uint64, error) {
	// 执行自动清理
	if _, err := hm.CleanupExpiredRecords(); err != nil {
		hm.logger.Warn("自动清理过期记录失败: %v", err)
	}

	// 加锁保证并发安全
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	// 确保目录存在
	os.MkdirAll(BackupDirPath, 0755)
	os.MkdirAll(TempDirPath, 0755)

	// 2. 创建临时文件
	tempFile, err := os.CreateTemp(TempDirPath, "backup_*.tmp")
	if err != nil {
		hm.logger.Error("创建临时文件失败: %v", err)
		return 0, fmt.Errorf("创建临时文件失败: %v", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	// 3. 如果备份文件已存在，复制现有内容；否则写入占位文件头
	backupPath := filepath.Join(common.GetConfig("Server", "WORKING_DIR"), BackupFilePath)
	var existingRecords []IndexEntry
	var existingBlocks contentBlockMap
	if _, err := os.Stat(backupPath); err == nil {
		existingRecords, existingBlocks, err = hm.copyExistingRecordsWithBlocks(tempFile, backupPath)
		if err != nil {
			hm.logger.Error("复制现有记录失败: %v", err)
			return 0, fmt.Errorf("复制现有记录失败: %v", err)
		}
	} else {
		// 备份文件不存在，初始化空的内容块映射
		existingBlocks = make(contentBlockMap)
		// 写入占位文件头
		tempHeader := FileHeader{
			MagicNumber: [16]byte{},
			Version:     CurrentVersion,
		}
		copy(tempHeader.MagicNumber[:], MagicNumber)
		if err := binary.Write(tempFile, binary.BigEndian, tempHeader); err != nil {
			hm.logger.Error("写入占位文件头失败: %v", err)
			return 0, fmt.Errorf("写入占位文件头失败: %v", err)
		}
	}

	// 4. 追加新记录（传入existingBlocks用于去重）
	newRecordID := hm.getNextRecordID(existingRecords)
	if err := hm.appendNewRecord(tempFile, newRecordID, operation, domain, operationData, files, existingBlocks); err != nil {
		hm.logger.Error("追加新记录失败: %v", err)
		return 0, fmt.Errorf("追加新记录失败: %v", err)
	}

	// 5. 写入内容块索引
	contentBlockOffset, err := hm.writeContentBlockIndex(tempFile, existingBlocks)
	if err != nil {
		hm.logger.Error("写入内容块索引失败: %v", err)
		return 0, fmt.Errorf("写入内容块索引失败: %v", err)
	}

	// 6. 更新文件头和索引
	if err := hm.updateFileHeaderAndIndex(tempFile, existingRecords, contentBlockOffset, existingBlocks); err != nil {
		hm.logger.Error("更新文件头失败: %v", err)
		return 0, fmt.Errorf("更新文件头失败: %v", err)
	}

	// 7. 计算并写入校验和
	if err := hm.calculateAndWriteChecksum(tempFile); err != nil {
		hm.logger.Error("计算校验和失败: %v", err)
		return 0, fmt.Errorf("计算校验和失败: %v", err)
	}

	// 8. 同步到磁盘
	if err := tempFile.Sync(); err != nil {
		hm.logger.Error("同步文件失败: %v", err)
		return 0, fmt.Errorf("同步文件失败: %v", err)
	}
	tempFile.Close()

	// 9. 验证临时文件
	if err := hm.verifyBackupFile(tempPath); err != nil {
		hm.logger.Error("验证备份文件失败: %v", err)
		return 0, fmt.Errorf("验证备份文件失败: %v", err)
	}

	// 10. 原子性替换
	if err := os.Rename(tempPath, backupPath); err != nil {
		hm.logger.Error("替换备份文件失败: %v", err)
		return 0, fmt.Errorf("替换备份文件失败: %v", err)
	}

	hm.logger.Info("备份成功，记录ID: %d", newRecordID)
	return newRecordID, nil
}

// createRollbackBackup 创建回退操作备份（使用预读取的文件内容，而不是从磁盘读取）
// 这是 RestoreBackup 的辅助方法，用于在文件恢复后创建正确的回退记录
func (hm *HistoryManager) createRollbackBackup(operation uint8, domain string, operationData []byte, fileContents map[string][]byte) (uint64, error) {
	// 执行自动清理
	if _, err := hm.CleanupExpiredRecords(); err != nil {
		hm.logger.Warn("自动清理过期记录失败: %v", err)
	}

	// 加锁保证并发安全
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	// 确保目录存在
	os.MkdirAll(BackupDirPath, 0755)
	os.MkdirAll(TempDirPath, 0755)

	// 2. 创建临时文件
	tempFile, err := os.CreateTemp(TempDirPath, "backup_*.tmp")
	if err != nil {
		hm.logger.Error("创建临时文件失败: %v", err)
		return 0, fmt.Errorf("创建临时文件失败: %v", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	// 3. 如果备份文件已存在，复制现有内容；否则写入占位文件头
	backupPath := filepath.Join(common.GetConfig("Server", "WORKING_DIR"), BackupFilePath)
	var existingRecords []IndexEntry
	var existingBlocks contentBlockMap
	if _, err := os.Stat(backupPath); err == nil {
		existingRecords, existingBlocks, err = hm.copyExistingRecordsWithBlocks(tempFile, backupPath)
		if err != nil {
			hm.logger.Error("复制现有记录失败: %v", err)
			return 0, fmt.Errorf("复制现有记录失败: %v", err)
		}
	} else {
		// 备份文件不存在，初始化空的内容块映射
		existingBlocks = make(contentBlockMap)
		// 写入占位文件头
		tempHeader := FileHeader{
			MagicNumber: [16]byte{},
			Version:     CurrentVersion,
		}
		copy(tempHeader.MagicNumber[:], MagicNumber)
		if err := binary.Write(tempFile, binary.BigEndian, tempHeader); err != nil {
			hm.logger.Error("写入占位文件头失败: %v", err)
			return 0, fmt.Errorf("写入占位文件头失败: %v", err)
		}
	}

	// 4. 追加新记录（使用预读取的内容）
	newRecordID := hm.getNextRecordID(existingRecords)
	if err := hm.appendNewRecordWithContents(tempFile, newRecordID, operation, domain, operationData, fileContents, existingBlocks); err != nil {
		hm.logger.Error("追加新记录失败: %v", err)
		return 0, fmt.Errorf("追加新记录失败: %v", err)
	}

	// 5. 写入内容块索引
	contentBlockOffset, err := hm.writeContentBlockIndex(tempFile, existingBlocks)
	if err != nil {
		hm.logger.Error("写入内容块索引失败: %v", err)
		return 0, fmt.Errorf("写入内容块索引失败: %v", err)
	}

	// 6. 更新文件头和索引
	if err := hm.updateFileHeaderAndIndex(tempFile, existingRecords, contentBlockOffset, existingBlocks); err != nil {
		hm.logger.Error("更新文件头失败: %v", err)
		return 0, fmt.Errorf("更新文件头失败: %v", err)
	}

	// 7. 计算并写入校验和
	if err := hm.calculateAndWriteChecksum(tempFile); err != nil {
		hm.logger.Error("计算校验和失败: %v", err)
		return 0, fmt.Errorf("计算校验和失败: %v", err)
	}

	// 8. 同步到磁盘
	if err := tempFile.Sync(); err != nil {
		hm.logger.Error("同步文件失败: %v", err)
		return 0, fmt.Errorf("同步文件失败: %v", err)
	}
	tempFile.Close()

	// 9. 验证临时文件
	if err := hm.verifyBackupFile(tempPath); err != nil {
		hm.logger.Error("验证备份文件失败: %v", err)
		return 0, fmt.Errorf("验证备份文件失败: %v", err)
	}

	// 10. 原子性替换
	if err := os.Rename(tempPath, backupPath); err != nil {
		hm.logger.Error("替换备份文件失败: %v", err)
		return 0, fmt.Errorf("替换备份文件失败: %v", err)
	}

	hm.logger.Info("回退备份成功，记录ID: %d", newRecordID)
	return newRecordID, nil
}

// appendNewRecordWithContents 追加新记录（使用预读取的内容）
func (hm *HistoryManager) appendNewRecordWithContents(tempFile *os.File, recordID uint64, operation uint8, domain string, operationData []byte, fileContents map[string][]byte, existingBlocks contentBlockMap) error {
	hm.logger.Info("追加新记录 ID=%d, 操作=%d, 域名=%s, 文件数=%d", recordID, operation, domain, len(fileContents))

	// 准备文件信息列表（使用预读取的内容）
	var files []FileInfo
	for fileName, content := range fileContents {
		contentHash := sha256.Sum256(content)

		// 检查是否已存在相同内容
		if _, exists := existingBlocks[contentHash]; !exists {
			// 新内容，需要写入数据区域
			compressed := hm.compressContent(content)
			dataOffset, err := tempFile.Seek(0, io.SeekEnd)
			if err != nil {
				return fmt.Errorf("定位文件末尾失败: %v", err)
			}

			if _, err := tempFile.Write(compressed); err != nil {
				return fmt.Errorf("写入压缩内容失败: %v", err)
			}

			// 添加到内容块映射
			existingBlocks[contentHash] = &ContentBlock{
				Hash:     contentHash,
				Size:     uint64(len(content)),
				Offset:   uint64(dataOffset),
				Length:   uint64(len(compressed)),
				RefCount: 1,
			}
		} else {
			// 已存在的内容，增加引用计数
			existingBlocks[contentHash].RefCount++
		}

		files = append(files, FileInfo{
			FileID:       uint64(len(files) + 1),
			FileName:     fileName,
			ContentHash:  contentHash,
			LastModified: time.Now().Unix(),
		})
	}

	// 创建元数据
	metaEntry := MetadataEntry{
		RecordID:      recordID,
		Operation:     operation,
		Domain:        domain,
		Timestamp:     time.Now().Unix(),
		ExpiryTime:    time.Now().Unix() + ExpiryDuration,
		OperationData: operationData,
		Files:         files,
	}

	// 写入元数据
	metaData, err := json.Marshal(metaEntry)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %v", err)
	}

	metaOffset, err := tempFile.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("定位文件末尾失败: %v", err)
	}

	metaSize := uint64(len(metaData))
	if err := binary.Write(tempFile, binary.BigEndian, metaSize); err != nil {
		return fmt.Errorf("写入元数据大小失败: %v", err)
	}

	if _, err := tempFile.Write(metaData); err != nil {
		return fmt.Errorf("写入元数据失败: %v", err)
	}

	// 创建索引条目
	indexEntry := IndexEntry{
		RecordID:   recordID,
		Offset:     uint64(metaOffset),
		Size:       metaSize,
		Operation:  operation,
		Domain:     domain,
		Timestamp:  metaEntry.Timestamp,
		ExpiryTime: metaEntry.ExpiryTime,
	}

	// 计算记录校验和（只计算元数据，因为内容块有独立的引用）
	recordChecksum := sha256.Sum256(metaData)
	indexEntry.Checksum = recordChecksum

	// 保存索引条目供后续使用
	hm.pendingIndexEntry = &indexEntry

	return nil
}

// RestoreBackup 恢复指定事务ID的备份（支持多记录）
func (hm *HistoryManager) RestoreBackup(recordID uint64) error {
	// 加锁保证并发安全
	hm.mutex.Lock()

	hm.logger.Info("开始恢复记录ID: %d", recordID)

	// 在回退前备份当前history.record文件（使用无锁版本，避免死锁）
	if err := hm.backupHistoryRecordUnsafe(recordID); err != nil {
		hm.mutex.Unlock()
		hm.logger.Error("备份history.record文件失败: %v", err)
		return fmt.Errorf("备份history.record文件失败: %v", err)
	}

	// 读取备份文件
	backupPath := filepath.Join(common.GetConfig("Server", "WORKING_DIR"), BackupFilePath)
	file, err := os.Open(backupPath)
	if err != nil {
		hm.mutex.Unlock()
		hm.logger.Error("打开备份文件失败: %v", err)
		return fmt.Errorf("打开备份文件失败: %v", err)
	}

	// 读取文件头
	var header FileHeader
	if err := binary.Read(file, binary.BigEndian, &header); err != nil {
		file.Close()
		hm.mutex.Unlock()
		hm.logger.Error("读取文件头失败: %v", err)
		return fmt.Errorf("读取文件头失败: %v", err)
	}

	// 验证魔法数字
	if string(header.MagicNumber[:len(MagicNumber)]) != MagicNumber {
		file.Close()
		hm.mutex.Unlock()
		hm.logger.Error("无效的备份文件")
		return fmt.Errorf("无效的备份文件")
	}

	// 验证版本
	if header.Version < MinCompatVersion || header.Version > CurrentVersion {
		file.Close()
		hm.mutex.Unlock()
		return fmt.Errorf("不支持的文件版本: %d", header.Version)
	}

	// 加载内容块索引
	contentBlocks, err := hm.loadContentBlocks(file, header)
	if err != nil {
		file.Close()
		hm.mutex.Unlock()
		hm.logger.Error("加载内容块索引失败: %v", err)
		return fmt.Errorf("加载内容块索引失败: %v", err)
	}

	// 读取索引区域
	file.Seek(int64(header.IndexOffset), io.SeekStart)
	indexData := make([]byte, header.IndexSize)
	if _, err := file.Read(indexData); err != nil {
		file.Close()
		hm.mutex.Unlock()
		hm.logger.Error("读取索引区域失败: %v", err)
		return fmt.Errorf("读取索引区域失败: %v", err)
	}

	// 解析索引条目（支持多记录）
	var indexEntries []IndexEntry
	if err := json.Unmarshal(indexData, &indexEntries); err != nil {
		file.Close()
		hm.mutex.Unlock()
		hm.logger.Error("解析索引区域失败: %v", err)
		return fmt.Errorf("解析索引区域失败: %v", err)
	}

	// 查找指定记录
	var targetEntry *IndexEntry
	for i := range indexEntries {
		if indexEntries[i].RecordID == recordID {
			targetEntry = &indexEntries[i]
			break
		}
	}

	if targetEntry == nil {
		file.Close()
		hm.mutex.Unlock()
		hm.logger.Error("记录ID不存在: %d", recordID)
		return fmt.Errorf("记录ID不存在: %d", recordID)
	}

	// 验证记录校验和
	if err := hm.verifyRecord(file, *targetEntry, contentBlocks); err != nil {
		file.Close()
		hm.mutex.Unlock()
		hm.logger.Error("记录验证失败: %v", err)
		return fmt.Errorf("记录验证失败: %v", err)
	}

	// 读取元数据
	file.Seek(int64(targetEntry.Offset), io.SeekStart)
	var metaSize uint64
	if err := binary.Read(file, binary.BigEndian, &metaSize); err != nil {
		file.Close()
		hm.mutex.Unlock()
		hm.logger.Error("读取元数据大小失败: %v", err)
		return fmt.Errorf("读取元数据大小失败: %v", err)
	}

	metaData := make([]byte, metaSize)
	if _, err := file.Read(metaData); err != nil {
		file.Close()
		hm.mutex.Unlock()
		hm.logger.Error("读取元数据失败: %v", err)
		return fmt.Errorf("读取元数据失败: %v", err)
	}

	// 解析元数据
	var metaEntry MetadataEntry
	if err := json.Unmarshal(metaData, &metaEntry); err != nil {
		file.Close()
		hm.mutex.Unlock()
		hm.logger.Error("解析元数据失败: %v", err)
		return fmt.Errorf("解析元数据失败: %v", err)
	}

	// 检测是否是回退Rollback操作（即回退一个恢复操作）
	if metaEntry.Operation == OperationRollback {
		file.Close()
		hm.mutex.Unlock()
		return hm.restoreRollbackOperation(recordID, metaEntry)
	}

	// 获取当前所有zone文件（用于后续清理）
	currentZoneFiles := hm.getCurrentZoneFiles()

	// 预读取当前所有文件内容（用于创建回退操作记录，必须在恢复文件之前完成）
	currentFileContents := make(map[string][]byte)
	// 包括 named.conf
	namedConfPath := filepath.Join(common.GetConfig("BIND", "NAMED_CONF_PATH"), "named.conf")
	if content, err := os.ReadFile(namedConfPath); err == nil {
		currentFileContents[namedConfPath] = content
	}
	// 包括所有 zone 文件
	for _, zoneFile := range currentZoneFiles {
		if content, err := os.ReadFile(zoneFile); err == nil {
			currentFileContents[zoneFile] = content
		}
	}

	// 构建目标文件列表（文件名 -> ContentHash）- 包含所有文件（包括 named.conf）
	targetFiles := make(map[string][32]byte)
	for _, fileInfo := range metaEntry.Files {
		targetFiles[fileInfo.FileName] = fileInfo.ContentHash
	}

	// 恢复或更新目标文件
	for fileName, targetHash := range targetFiles {
		needRestore := false

		// 检查文件是否存在
		if _, err := os.Stat(fileName); os.IsNotExist(err) {
			needRestore = true
		} else {
			// 文件存在，读取并计算hash
			currentContent, err := os.ReadFile(fileName)
			if err != nil {
				needRestore = true
			} else {
				currentHash := sha256.Sum256(currentContent)
				if currentHash != targetHash {
					needRestore = true
				}
			}
		}

		// 需要恢复时从备份读取内容
		if needRestore {
			block, exists := contentBlocks[targetHash]
			if !exists {
				file.Close()
				hm.mutex.Unlock()
				return fmt.Errorf("内容块不存在: %x", targetHash)
			}

			// 读取并解压内容
			file.Seek(int64(block.Offset), io.SeekStart)
			compressedData := make([]byte, block.Length)
			if _, err := file.Read(compressedData); err != nil {
				file.Close()
				hm.mutex.Unlock()
				return fmt.Errorf("读取内容失败: %v", err)
			}

			content := hm.decompressContent(compressedData)

			// 写入文件
			if err := os.WriteFile(fileName, content, 0644); err != nil {
				file.Close()
				hm.mutex.Unlock()
				hm.logger.Error("写入文件失败: %v", err)
				return fmt.Errorf("写入文件失败: %v", err)
			}

			hm.logger.Info("恢复文件成功: %s", fileName)
		}
	}

	file.Close()

	// 删除多余的zone文件（当前有但目标记录没有的）
	for _, currentFile := range currentZoneFiles {
		if _, exists := targetFiles[currentFile]; !exists {
			if err := os.Remove(currentFile); err != nil {
				hm.logger.Warn("删除多余文件失败 %s: %v", currentFile, err)
			}
		}
	}

	// 准备回退操作数据
	rollbackData, _ := json.Marshal(map[string]interface{}{
		"rollback_record_id": recordID,
		"rollback_operation": metaEntry.Operation,
		"rollback_domain":    metaEntry.Domain,
	})

	// 释放锁，避免死锁（createRollbackBackup也会获取锁）
	hm.mutex.Unlock()

	// 1. 记录回退操作（使用预读取的文件内容，确保记录的是恢复前的状态）
	rollbackID, err := hm.createRollbackBackup(OperationRollback, metaEntry.Domain, rollbackData, currentFileContents)
	if err != nil {
		hm.logger.Warn("记录回退操作失败: %v", err)
	}

	// 2. 刷新BIND服务器使配置生效
	if hm.bindManager != nil {
		if err := hm.bindManager.ReloadBind(); err != nil {
			hm.logger.Error("刷新BIND服务器失败: %v", err)
		} else {
			hm.logger.Info("刷新BIND服务器成功")
		}
	}

	// 3. 恢复完成后，删除目标记录及之后的所有记录（包括目标记录本身），但保留回退记录
	if rollbackID > 0 {
		if err := hm.deleteRecordsAfterIncluding(recordID, rollbackID); err != nil {
			hm.logger.Warn("删除记录失败: %v", err)
		}
	} else {
		if err := hm.deleteRecordsAfter(recordID); err != nil {
			hm.logger.Warn("删除记录失败: %v", err)
		}
	}

	hm.logger.Info("恢复记录ID: %d 成功", recordID)
	return nil
}

// restoreRollbackOperation 回退Rollback操作（即回退一个恢复操作）
// 双路径恢复：1. 使用history.record.rollback文件 2. 使用嵌入的数据
func (hm *HistoryManager) restoreRollbackOperation(recordID uint64, metaEntry MetadataEntry) error {
	hm.logger.Info("回退Rollback操作，记录ID: %d", recordID)

	// 解析原始记录ID（被回退的原始操作）
	var rollbackInfo map[string]interface{}
	if err := json.Unmarshal(metaEntry.OperationData, &rollbackInfo); err != nil {
		return fmt.Errorf("解析回退信息失败: %v", err)
	}

	originalRecordIDFloat, ok := rollbackInfo["rollback_record_id"].(float64)
	if !ok {
		return fmt.Errorf("无法获取原始记录ID")
	}
	originalRecordID := uint64(originalRecordIDFloat)

	// 路径1：尝试使用history.record.rollback.{id}文件
	rollbackFilePath := filepath.Join(
		common.GetConfig("Server", "WORKING_DIR"),
		"./backup/",
		fmt.Sprintf("%s%d", RollbackProtectionPrefix, originalRecordID),
	)

	if _, err := os.Stat(rollbackFilePath); err == nil {
		hm.logger.Info("使用rollback文件回退: %s", rollbackFilePath)
		return hm.restoreFromRollbackFile(rollbackFilePath, recordID)
	}

	hm.logger.Warn("rollback文件不存在: %s，尝试使用嵌入数据", rollbackFilePath)

	// 路径2：从恢复操作记录中提取嵌入的文件数据
	return hm.restoreFromEmbeddedData(recordID, metaEntry, originalRecordID)
}

// restoreFromRollbackFile 从rollback文件恢复（不记录新操作）
func (hm *HistoryManager) restoreFromRollbackFile(rollbackFilePath string, currentRecordID uint64) error {
	hm.logger.Info("从rollback文件恢复: %s", rollbackFilePath)

	// 1. 从当前 history.record 读取回退记录并恢复文件
	// 回退记录中包含恢复前的完整状态（named.conf + zone文件）
	if err := hm.applyFilesFromCurrentRecord(currentRecordID); err != nil {
		return fmt.Errorf("从当前记录恢复文件失败: %v", err)
	}

	// 2. 用 rollback 文件恢复 history.record
	data, err := os.ReadFile(rollbackFilePath)
	if err != nil {
		return fmt.Errorf("读取rollback文件失败: %v", err)
	}

	backupPath := filepath.Join(common.GetConfig("Server", "WORKING_DIR"), BackupFilePath)
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("恢复history.record文件失败: %v", err)
	}

	hm.logger.Info("恢复history.record文件成功")

	// 3. 删除当前Rollback记录（因为是直接回退，不保留记录）
	if err := hm.deleteRecordsAfterIncluding(currentRecordID, 0); err != nil {
		hm.logger.Warn("删除Rollback记录失败: %v", err)
	}

	return nil
}

// restoreFromEmbeddedData 从嵌入数据恢复（记录新操作）
func (hm *HistoryManager) restoreFromEmbeddedData(recordID uint64, metaEntry MetadataEntry, originalRecordID uint64) error {
	hm.logger.Info("使用嵌入数据回退，原始记录ID: %d", originalRecordID)

	// 检查是否有嵌入的文件数据
	if len(metaEntry.Files) == 0 {
		return fmt.Errorf("备份已过期，无法回退此操作（没有嵌入的文件数据）")
	}

	// 加锁保证并发安全
	hm.mutex.Lock()

	// 在回退前备份当前history.record文件
	if err := hm.backupHistoryRecordUnsafe(recordID); err != nil {
		hm.mutex.Unlock()
		hm.logger.Error("备份history.record文件失败: %v", err)
		return fmt.Errorf("备份history.record文件失败: %v", err)
	}

	// 读取备份文件获取内容块
	backupPath := filepath.Join(common.GetConfig("Server", "WORKING_DIR"), BackupFilePath)
	file, err := os.Open(backupPath)
	if err != nil {
		hm.mutex.Unlock()
		return fmt.Errorf("打开备份文件失败: %v", err)
	}

	// 读取文件头
	var header FileHeader
	if err := binary.Read(file, binary.BigEndian, &header); err != nil {
		file.Close()
		hm.mutex.Unlock()
		return fmt.Errorf("读取文件头失败: %v", err)
	}

	// 加载内容块索引
	contentBlocks, err := hm.loadContentBlocks(file, header)
	if err != nil {
		file.Close()
		hm.mutex.Unlock()
		return fmt.Errorf("加载内容块索引失败: %v", err)
	}

	// 预读取当前所有文件内容（用于创建新的回退记录）
	currentFileContents := make(map[string][]byte)
	namedConfPath := filepath.Join(common.GetConfig("BIND", "NAMED_CONF_PATH"), "named.conf")
	if content, err := os.ReadFile(namedConfPath); err == nil {
		currentFileContents[namedConfPath] = content
	}
	currentZoneFiles := hm.getCurrentZoneFiles()
	for _, zoneFile := range currentZoneFiles {
		if content, err := os.ReadFile(zoneFile); err == nil {
			currentFileContents[zoneFile] = content
		}
	}

	// 从嵌入数据恢复文件
	for _, fileInfo := range metaEntry.Files {
		block, exists := contentBlocks[fileInfo.ContentHash]
		if !exists {
			file.Close()
			hm.mutex.Unlock()
			return fmt.Errorf("内容块不存在: %x", fileInfo.ContentHash)
		}

		// 读取并解压内容
		file.Seek(int64(block.Offset), io.SeekStart)
		compressedData := make([]byte, block.Length)
		if _, err := file.Read(compressedData); err != nil {
			file.Close()
			hm.mutex.Unlock()
			return fmt.Errorf("读取内容失败: %v", err)
		}

		content := hm.decompressContent(compressedData)

		// 写入文件
		if err := os.WriteFile(fileInfo.FileName, content, 0644); err != nil {
			file.Close()
			hm.mutex.Unlock()
			hm.logger.Error("写入文件失败: %v", err)
			return fmt.Errorf("写入文件失败: %v", err)
		}

		hm.logger.Info("恢复文件成功: %s", fileInfo.FileName)
	}

	file.Close()

	// 准备新的回退操作数据
	newRollbackData, _ := json.Marshal(map[string]interface{}{
		"rollback_record_id":    recordID,
		"rollback_operation":    metaEntry.Operation,
		"rollback_domain":       metaEntry.Domain,
		"restore_from_embedded": true,
	})

	// 释放锁
	hm.mutex.Unlock()

	// 记录新的回退操作（创建新的rollback文件）
	newRollbackID, err := hm.createRollbackBackup(OperationRollback, metaEntry.Domain, newRollbackData, currentFileContents)
	if err != nil {
		hm.logger.Warn("记录新的回退操作失败: %v", err)
	} else {
		hm.logger.Info("记录新的回退操作成功，记录ID: %d", newRollbackID)
	}

	// 刷新BIND服务器
	if hm.bindManager != nil {
		if err := hm.bindManager.ReloadBind(); err != nil {
			hm.logger.Error("刷新BIND服务器失败: %v", err)
		} else {
			hm.logger.Info("刷新BIND服务器成功")
		}
	}

	// 删除原Rollback记录及之后的记录，但保留新的回退记录
	if newRollbackID > 0 {
		if err := hm.deleteRecordsAfterIncluding(recordID, newRollbackID); err != nil {
			hm.logger.Warn("删除记录失败: %v", err)
		}
	}

	hm.logger.Info("使用嵌入数据回退成功")
	return nil
}

// applyFilesFromCurrentRecord 从当前history.record的指定记录恢复文件
func (hm *HistoryManager) applyFilesFromCurrentRecord(recordID uint64) error {
	hm.logger.Info("从当前history.record的记录ID %d 恢复文件", recordID)

	// 读取当前history.record
	backupPath := filepath.Join(common.GetConfig("Server", "WORKING_DIR"), BackupFilePath)
	file, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("打开history.record失败: %v", err)
	}
	defer file.Close()

	// 读取文件头
	var header FileHeader
	if err := binary.Read(file, binary.BigEndian, &header); err != nil {
		return fmt.Errorf("读取文件头失败: %v", err)
	}

	// 验证文件
	if string(header.MagicNumber[:len(MagicNumber)]) != MagicNumber {
		return fmt.Errorf("无效的备份文件")
	}

	// 加载内容块
	contentBlocks, err := hm.loadContentBlocks(file, header)
	if err != nil {
		return fmt.Errorf("加载内容块失败: %v", err)
	}

	// 读取索引
	file.Seek(int64(header.IndexOffset), io.SeekStart)
	indexData := make([]byte, header.IndexSize)
	if _, err := file.Read(indexData); err != nil {
		return fmt.Errorf("读取索引失败: %v", err)
	}

	var indexEntries []IndexEntry
	if err := json.Unmarshal(indexData, &indexEntries); err != nil {
		return fmt.Errorf("解析索引失败: %v", err)
	}

	// 查找指定记录
	var targetEntry *IndexEntry
	for i := range indexEntries {
		if indexEntries[i].RecordID == recordID {
			targetEntry = &indexEntries[i]
			break
		}
	}

	if targetEntry == nil {
		return fmt.Errorf("找不到记录ID: %d", recordID)
	}

	// 应用该记录的文件
	if err := hm.applyRecordFiles(file, *targetEntry, contentBlocks); err != nil {
		return fmt.Errorf("应用记录文件失败: %v", err)
	}

	hm.logger.Info("从记录ID %d 恢复文件成功", recordID)
	return nil
}

// applyRecordFiles 应用记录中的文件到系统
func (hm *HistoryManager) applyRecordFiles(file *os.File, entry IndexEntry, contentBlocks contentBlockMap) error {
	// 读取元数据
	file.Seek(int64(entry.Offset), io.SeekStart)
	var metaSize uint64
	if err := binary.Read(file, binary.BigEndian, &metaSize); err != nil {
		return err
	}

	metaData := make([]byte, metaSize)
	if _, err := file.Read(metaData); err != nil {
		return err
	}

	var metaEntry MetadataEntry
	if err := json.Unmarshal(metaData, &metaEntry); err != nil {
		return err
	}

	// 恢复文件
	for _, fileInfo := range metaEntry.Files {
		block, exists := contentBlocks[fileInfo.ContentHash]
		if !exists {
			return fmt.Errorf("内容块不存在: %x", fileInfo.ContentHash)
		}

		file.Seek(int64(block.Offset), io.SeekStart)
		compressedData := make([]byte, block.Length)
		if _, err := file.Read(compressedData); err != nil {
			return err
		}

		content := hm.decompressContent(compressedData)
		if err := os.WriteFile(fileInfo.FileName, content, 0644); err != nil {
			return err
		}

		hm.logger.Info("应用文件成功: %s", fileInfo.FileName)
	}

	return nil
}

// 辅助函数和工具方法

// getNextRecordID 获取下一个记录ID
func (hm *HistoryManager) getNextRecordID(existingRecords []IndexEntry) uint64 {
	maxID := uint64(0)
	for _, record := range existingRecords {
		if record.RecordID > maxID {
			maxID = record.RecordID
		}
	}
	return maxID + 1
}

// copyExistingRecordsWithBlocks 复制现有记录和内容块到临时文件（支持去重）
func (hm *HistoryManager) copyExistingRecordsWithBlocks(tempFile *os.File, backupPath string) ([]IndexEntry, contentBlockMap, error) {
	// 读取现有文件
	existingFile, err := os.Open(backupPath)
	if err != nil {
		return nil, nil, err
	}
	defer existingFile.Close()

	// 读取文件头
	var header FileHeader
	if err := binary.Read(existingFile, binary.BigEndian, &header); err != nil {
		return nil, nil, err
	}

	// 验证文件头
	if string(header.MagicNumber[:len(MagicNumber)]) != MagicNumber {
		return nil, nil, fmt.Errorf("无效的备份文件")
	}

	// 验证版本
	if header.Version < MinCompatVersion || header.Version > CurrentVersion {
		return nil, nil, fmt.Errorf("不支持的文件版本: %d", header.Version)
	}

	// 加载内容块索引
	existingBlocks, err := hm.loadContentBlocks(existingFile, header)
	if err != nil {
		return nil, nil, fmt.Errorf("加载内容块索引失败: %v", err)
	}

	// 读取记录索引
	existingFile.Seek(int64(header.IndexOffset), io.SeekStart)
	indexData := make([]byte, header.IndexSize)
	if _, err := existingFile.Read(indexData); err != nil {
		return nil, nil, err
	}

	var indexEntries []IndexEntry
	if err := json.Unmarshal(indexData, &indexEntries); err != nil {
		return nil, nil, err
	}

	// 复制数据区域（内容块数据）
	existingFile.Seek(int64(header.DataOffset), io.SeekStart)
	dataSize := header.TotalSize - header.DataOffset
	if header.ContentBlockOffset > 0 && header.ContentBlockSize > 0 {
		// 如果有内容块索引，只复制到内容块索引之前的数据
		dataSize = header.ContentBlockOffset - header.DataOffset
	}
	dataBuffer := make([]byte, dataSize)
	if _, err := existingFile.Read(dataBuffer); err != nil {
		return nil, nil, err
	}

	// 写入临时文件 - 先写入占位文件头
	tempHeader := FileHeader{
		MagicNumber: header.MagicNumber,
		Version:     CurrentVersion,
	}
	copy(tempHeader.MagicNumber[:], MagicNumber)

	if err := binary.Write(tempFile, binary.BigEndian, tempHeader); err != nil {
		return nil, nil, err
	}

	// 写入数据
	if _, err := tempFile.Write(dataBuffer); err != nil {
		return nil, nil, err
	}

	// 调整内容块偏移量，使其相对于新文件的DataOffset
	newDataOffset := uint64(binary.Size(FileHeader{}))
	oldDataOffset := header.DataOffset
	offsetDiff := int64(newDataOffset) - int64(oldDataOffset)

	for _, block := range existingBlocks {
		block.Offset = uint64(int64(block.Offset) + offsetDiff)
	}

	return indexEntries, existingBlocks, nil
}

// appendNewRecord 追加新记录到临时文件（支持内容去重）
func (hm *HistoryManager) appendNewRecord(tempFile *os.File, recordID uint64, operation uint8, domain string, operationData []byte, files []string, existingBlocks contentBlockMap) error {
	// 先收集所有文件信息和内容（但不写入）
	fileInfos := make([]FileInfo, 0, len(files))
	fileContents := make(map[string][]byte)
	fileID := uint64(1)

	for _, filePath := range files {
		// 读取文件内容
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("读取文件失败 %s: %v", filePath, err)
		}
		fileContents[filePath] = content

		// 写入内容（自动去重）
		block, err := hm.writeContentWithDeduplication(tempFile, content, existingBlocks)
		if err != nil {
			return err
		}

		// 获取文件信息
		fileInfo, _ := os.Stat(filePath)
		var modTime int64
		if fileInfo != nil {
			modTime = fileInfo.ModTime().Unix()
		}

		fileInfos = append(fileInfos, FileInfo{
			FileID:       fileID,
			FileName:     filePath,
			ContentHash:  block.Hash,
			LastModified: modTime,
		})
		fileID++
	}

	// 获取当前文件位置作为元数据偏移（在写入内容块之后）
	metaOffset, err := tempFile.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}

	// 创建元数据
	metaEntry := MetadataEntry{
		RecordID:      recordID,
		Operation:     operation,
		Domain:        domain,
		Timestamp:     time.Now().Unix(),
		ExpiryTime:    time.Now().Add(time.Duration(ExpiryDuration) * time.Second).Unix(),
		OperationData: operationData,
		Files:         fileInfos,
	}

	metaBytes, err := json.Marshal(metaEntry)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %v", err)
	}

	// 写入元数据大小和内容
	metaSize := uint64(len(metaBytes))
	if err := binary.Write(tempFile, binary.BigEndian, metaSize); err != nil {
		return err
	}
	if _, err := tempFile.Write(metaBytes); err != nil {
		return err
	}

	// 计算记录校验和（只计算元数据，因为内容块有独立的引用）
	recordChecksum := sha256.Sum256(metaBytes)

	// 创建索引条目
	indexEntry := IndexEntry{
		RecordID:      recordID,
		Offset:        uint64(metaOffset),
		Size:          metaSize,
		Operation:     operation,
		Domain:        domain,
		Timestamp:     metaEntry.Timestamp,
		ExpiryTime:    metaEntry.ExpiryTime,
		ContentOffset: 0, // 新格式不再使用
		ContentSize:   0, // 新格式不再使用
		Checksum:      recordChecksum,
	}

	// 存储索引条目供后续使用
	hm.pendingIndexEntry = &indexEntry

	return nil
}

// writeContentWithDeduplication 写入内容并去重
func (hm *HistoryManager) writeContentWithDeduplication(
	tempFile *os.File,
	content []byte,
	existingBlocks contentBlockMap,
) (*ContentBlock, error) {
	// 1. 计算内容哈希
	hash := sha256.Sum256(content)

	// 2. 检查是否已存在
	if block, exists := existingBlocks[hash]; exists {
		block.RefCount++
		return block, nil
	}

	// 3. 压缩内容
	compressed := hm.compressContent(content)

	// 4. 获取当前文件位置并写入
	offset, _ := tempFile.Seek(0, io.SeekCurrent)
	if _, err := tempFile.Write(compressed); err != nil {
		return nil, err
	}

	// 5. 创建新内容块
	block := &ContentBlock{
		Hash:     hash,
		Size:     uint64(len(content)),
		Offset:   uint64(offset),
		Length:   uint64(len(compressed)),
		RefCount: 1,
	}
	existingBlocks[hash] = block

	return block, nil
}

// writeContentBlockIndex 写入内容块索引
func (hm *HistoryManager) writeContentBlockIndex(tempFile *os.File, blocks contentBlockMap) (uint64, error) {
	// 转换为数组
	blockList := make([]ContentBlock, 0, len(blocks))
	for _, block := range blocks {
		blockList = append(blockList, *block)
	}

	// 序列化
	blockData, err := json.Marshal(blockList)
	if err != nil {
		return 0, err
	}

	// 写入
	offset, _ := tempFile.Seek(0, io.SeekCurrent)
	if _, err := tempFile.Write(blockData); err != nil {
		return 0, err
	}

	return uint64(offset), nil
}

// loadContentBlocks 从备份文件加载内容块索引
func (hm *HistoryManager) loadContentBlocks(file *os.File, header FileHeader) (contentBlockMap, error) {
	if header.ContentBlockSize == 0 {
		return make(contentBlockMap), nil
	}

	file.Seek(int64(header.ContentBlockOffset), io.SeekStart)
	blockData := make([]byte, header.ContentBlockSize)
	if _, err := file.Read(blockData); err != nil {
		return nil, err
	}

	var blocks []ContentBlock
	if err := json.Unmarshal(blockData, &blocks); err != nil {
		return nil, err
	}

	blockMap := make(contentBlockMap)
	for i := range blocks {
		// 创建副本，避免指向临时数组（防止垃圾回收后指针失效）
		block := &ContentBlock{
			Hash:     blocks[i].Hash,
			Size:     blocks[i].Size,
			Offset:   blocks[i].Offset,
			Length:   blocks[i].Length,
			RefCount: blocks[i].RefCount,
		}
		blockMap[block.Hash] = block
	}
	return blockMap, nil
}

// updateFileHeaderAndIndex 更新文件头和索引（支持内容块索引）
func (hm *HistoryManager) updateFileHeaderAndIndex(tempFile *os.File, existingRecords []IndexEntry, contentBlockOffset uint64, blocks contentBlockMap) error {
	// 获取当前文件位置（记录索引区域起始位置）
	indexOffset, _ := tempFile.Seek(0, io.SeekCurrent)

	// 构建索引列表：现有记录 + 新记录
	indexEntries := make([]IndexEntry, 0, len(existingRecords)+1)
	indexEntries = append(indexEntries, existingRecords...)

	// 添加新的索引条目
	if hm.pendingIndexEntry != nil {
		indexEntries = append(indexEntries, *hm.pendingIndexEntry)
	}

	// 序列化记录索引
	indexData, err := json.Marshal(indexEntries)
	if err != nil {
		return err
	}

	// 写入记录索引区域
	tempFile.Seek(int64(indexOffset), io.SeekStart)
	if _, err := tempFile.Write(indexData); err != nil {
		return err
	}

	// 计算内容块索引大小
	blockList := make([]ContentBlock, 0, len(blocks))
	for _, block := range blocks {
		blockList = append(blockList, *block)
	}
	blockData, _ := json.Marshal(blockList)

	// 获取文件总大小
	totalSize, _ := tempFile.Seek(0, io.SeekEnd)

	// 更新文件头
	tempFile.Seek(0, io.SeekStart)
	header := FileHeader{
		MagicNumber:        [16]byte{},
		Version:            CurrentVersion,
		RecordCount:        uint64(len(indexEntries)),
		IndexOffset:        uint64(indexOffset),
		IndexSize:          uint64(len(indexData)),
		ContentBlockOffset: contentBlockOffset,
		ContentBlockSize:   uint64(len(blockData)),
		DataOffset:         uint64(binary.Size(FileHeader{})),
		TotalSize:          uint64(totalSize),
	}
	copy(header.MagicNumber[:], MagicNumber)

	return binary.Write(tempFile, binary.BigEndian, header)
}

// calculateAndWriteChecksum 计算并写入整体校验和（流式计算，避免大内存占用）
func (hm *HistoryManager) calculateAndWriteChecksum(file *os.File) error {
	fileSize, _ := file.Seek(0, io.SeekEnd)
	checksumOffset := binary.Size(FileHeader{}) - 32 - 64
	checksumSize := 32

	// 提取存储的校验和
	file.Seek(int64(checksumOffset), io.SeekStart)
	storedChecksum := make([]byte, checksumSize)
	if _, err := io.ReadFull(file, storedChecksum); err != nil {
		return fmt.Errorf("读取校验和失败: %v", err)
	}

	// 使用SHA-256重新计算校验和
	hash := sha256.New()
	bufferSize := 64 * 1024 // 64KB缓冲区
	buffer := make([]byte, bufferSize)

	file.Seek(0, io.SeekStart)
	bytesRead := 0

	for bytesRead < int(fileSize) {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}

		// 检查是否包含校验和字段
		start := bytesRead
		end := bytesRead + n

		if start <= checksumOffset && checksumOffset < end {
			// 当前块包含校验和字段的开始
			beforeChecksumLen := checksumOffset - start
			if beforeChecksumLen > 0 {
				hash.Write(buffer[:beforeChecksumLen])
			}

			// 跳过校验和字段（视为0）
			zeroLen := checksumSize
			if end < checksumOffset+checksumSize {
				zeroLen = end - checksumOffset
			}
			zeroBytes := make([]byte, zeroLen)
			hash.Write(zeroBytes)

			afterChecksumStart := checksumOffset + checksumSize - start
			if afterChecksumStart < n {
				hash.Write(buffer[afterChecksumStart:])
			}
		} else if checksumOffset < start && start < checksumOffset+checksumSize {
			// 当前块在校验和字段内部
			remainingChecksum := checksumOffset + checksumSize - start
			zeroLen := remainingChecksum
			if n < zeroLen {
				zeroLen = n
			}
			zeroBytes := make([]byte, zeroLen)
			hash.Write(zeroBytes)

			if remainingChecksum < n {
				hash.Write(buffer[remainingChecksum:])
			}
		} else {
			// 当前块不包含校验和字段
			hash.Write(buffer[:n])
		}

		bytesRead += n
	}

	calculatedChecksum := hash.Sum(nil)

	// 写入计算后的校验和
	file.Seek(int64(checksumOffset), io.SeekStart)
	if _, err := file.Write(calculatedChecksum); err != nil {
		return fmt.Errorf("写入校验和失败: %v", err)
	}

	return nil
}

// verifyBackupFile 验证备份文件完整性
func (hm *HistoryManager) verifyBackupFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// 读取文件头
	var header FileHeader
	if err := binary.Read(file, binary.BigEndian, &header); err != nil {
		return fmt.Errorf("读取文件头失败: %v", err)
	}

	// 验证魔法数字
	if string(header.MagicNumber[:len(MagicNumber)]) != MagicNumber {
		return fmt.Errorf("无效的备份文件")
	}

	// 验证版本
	if header.Version < MinCompatVersion || header.Version > CurrentVersion {
		return fmt.Errorf("不支持的文件版本: %d", header.Version)
	}

	// 加载内容块索引
	contentBlocks, err := hm.loadContentBlocks(file, header)
	if err != nil {
		return fmt.Errorf("加载内容块索引失败: %v", err)
	}

	// 验证索引区域
	file.Seek(int64(header.IndexOffset), io.SeekStart)
	indexData := make([]byte, header.IndexSize)
	if _, err := file.Read(indexData); err != nil {
		return fmt.Errorf("读取索引区域失败: %v", err)
	}

	var indexEntries []IndexEntry
	if err := json.Unmarshal(indexData, &indexEntries); err != nil {
		return fmt.Errorf("解析索引区域失败: %v", err)
	}

	if uint64(len(indexEntries)) != header.RecordCount {
		return fmt.Errorf("记录数量不匹配")
	}

	// 验证每个记录
	for _, entry := range indexEntries {
		if err := hm.verifyRecord(file, entry, contentBlocks); err != nil {
			return fmt.Errorf("验证记录%d失败: %v", entry.RecordID, err)
		}
	}

	return nil
}

// verifyRecord 验证单个记录（新格式：通过ContentHash验证）
func (hm *HistoryManager) verifyRecord(file *os.File, entry IndexEntry, blocks contentBlockMap) error {
	// 读取元数据
	file.Seek(int64(entry.Offset), io.SeekStart)
	var metaSize uint64
	if err := binary.Read(file, binary.BigEndian, &metaSize); err != nil {
		return err
	}

	// 验证metaSize合理性（最大100MB）
	if metaSize == 0 || metaSize > 100*1024*1024 {
		return fmt.Errorf("无效的元数据大小: %d", metaSize)
	}

	metaData := make([]byte, metaSize)
	if _, err := file.Read(metaData); err != nil {
		return err
	}

	// 解析元数据获取文件列表
	var metaEntry MetadataEntry
	if err := json.Unmarshal(metaData, &metaEntry); err != nil {
		return fmt.Errorf("解析元数据失败: %v", err)
	}

	// 验证每个文件引用的内容块是否存在
	for _, fileInfo := range metaEntry.Files {
		if _, exists := blocks[fileInfo.ContentHash]; !exists {
			return fmt.Errorf("内容块不存在: %x", fileInfo.ContentHash)
		}
	}

	// 验证元数据校验和
	calculatedChecksum := sha256.Sum256(metaData)
	if calculatedChecksum != entry.Checksum {
		return fmt.Errorf("元数据校验和不匹配")
	}

	return nil
}

// getCurrentZoneFiles 获取当前所有zone文件路径
func (hm *HistoryManager) getCurrentZoneFiles() []string {
	// 使用配置的ZoneFilePath，而不是WORKING_DIR/zones
	zoneDir := common.GetConfig("BIND", "ZONE_FILE_PATH")
	if zoneDir == "" {
		// 如果配置为空，使用默认路径
		zoneDir = filepath.Join(common.GetConfig("Server", "WORKING_DIR"), "zones")
	}

	entries, err := os.ReadDir(zoneDir)
	if err != nil {
		return nil
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".zone") {
			files = append(files, filepath.Join(zoneDir, entry.Name()))
		}
	}
	return files
}

// compressContent 压缩内容
func (hm *HistoryManager) compressContent(content []byte) []byte {
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)

	if _, err := gzipWriter.Write(content); err != nil {
		return content // 压缩失败，返回原内容
	}

	gzipWriter.Close()
	return buffer.Bytes()
}

// decompressContent 解压内容
func (hm *HistoryManager) decompressContent(content []byte) []byte {
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

// DeleteBackupRecord 删除指定备份记录
func (hm *HistoryManager) DeleteBackupRecord(recordID uint64) error {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	backupPath := filepath.Join(common.GetConfig("Server", "WORKING_DIR"), BackupFilePath)

	// 检查文件是否存在
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return nil // 文件不存在，无需删除
	}

	// 打开备份文件
	file, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("打开备份文件失败: %v", err)
	}

	// 读取文件头
	var header FileHeader
	if err := binary.Read(file, binary.BigEndian, &header); err != nil {
		file.Close()
		return fmt.Errorf("读取文件头失败: %v", err)
	}

	// 加载内容块索引
	contentBlocks, err := hm.loadContentBlocks(file, header)
	if err != nil {
		file.Close()
		return fmt.Errorf("加载内容块索引失败: %v", err)
	}

	// 读取索引区域
	file.Seek(int64(header.IndexOffset), io.SeekStart)
	indexData := make([]byte, header.IndexSize)
	if _, err := file.Read(indexData); err != nil {
		file.Close()
		return fmt.Errorf("读取索引区域失败: %v", err)
	}
	file.Close()

	// 解析索引条目
	var indexEntries []IndexEntry
	if err := json.Unmarshal(indexData, &indexEntries); err != nil {
		return fmt.Errorf("解析索引区域失败: %v", err)
	}

	// 查找要删除的记录
	found := false
	newEntries := make([]IndexEntry, 0, len(indexEntries))
	for _, entry := range indexEntries {
		if entry.RecordID == recordID {
			found = true
			// 减少引用计数
			hm.decreaseRefCount(contentBlocks, entry)
		} else {
			newEntries = append(newEntries, entry)
		}
	}

	if !found {
		return fmt.Errorf("记录ID不存在: %d", recordID)
	}

	// 重新构建备份文件
	return hm.rebuildBackupFile(newEntries, contentBlocks)
}

// decreaseRefCount 减少内容块引用计数
func (hm *HistoryManager) decreaseRefCount(blocks contentBlockMap, entry IndexEntry) {
	// 读取元数据获取文件列表
	backupPath := filepath.Join(common.GetConfig("Server", "WORKING_DIR"), BackupFilePath)
	file, err := os.Open(backupPath)
	if err != nil {
		return
	}
	defer file.Close()

	file.Seek(int64(entry.Offset), io.SeekStart)
	var metaSize uint64
	if err := binary.Read(file, binary.BigEndian, &metaSize); err != nil {
		return
	}

	metaData := make([]byte, metaSize)
	if _, err := file.Read(metaData); err != nil {
		return
	}

	var metaEntry MetadataEntry
	if err := json.Unmarshal(metaData, &metaEntry); err != nil {
		return
	}

	// 减少每个文件引用计数
	for _, fileInfo := range metaEntry.Files {
		if block, exists := blocks[fileInfo.ContentHash]; exists {
			block.RefCount--
		}
	}
}

// rebuildBackupFile 重建备份文件（只复制需要保留的记录的数据）
func (hm *HistoryManager) rebuildBackupFile(entries []IndexEntry, blocks contentBlockMap) error {
	backupPath := filepath.Join(common.GetConfig("Server", "WORKING_DIR"), BackupFilePath)

	// 如果没有记录了，删除备份文件
	if len(entries) == 0 {
		return os.Remove(backupPath)
	}

	// 创建临时文件
	tempFile, err := os.CreateTemp(TempDirPath, "rebuild_*.tmp")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %v", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	// 写入占位文件头
	tempHeader := FileHeader{
		MagicNumber: [16]byte{},
		Version:     CurrentVersion,
	}
	copy(tempHeader.MagicNumber[:], MagicNumber)
	if err := binary.Write(tempFile, binary.BigEndian, tempHeader); err != nil {
		tempFile.Close()
		return fmt.Errorf("写入文件头失败: %v", err)
	}

	// 打开现有备份文件读取数据
	existingFile, err := os.Open(backupPath)
	if err != nil {
		tempFile.Close()
		return fmt.Errorf("打开备份文件失败: %v", err)
	}

	var header FileHeader
	if err := binary.Read(existingFile, binary.BigEndian, &header); err != nil {
		existingFile.Close()
		tempFile.Close()
		return fmt.Errorf("读取文件头失败: %v", err)
	}

	// 构建新的内容块映射（只包含需要保留的块）
	newBlocks := make(contentBlockMap)
	newEntries := make([]IndexEntry, 0, len(entries))

	// 收集所有需要保留的内容块哈希
	neededHashes := make(map[[32]byte]bool)
	for _, entry := range entries {
		// 读取元数据获取文件列表
		existingFile.Seek(int64(entry.Offset), io.SeekStart)
		var metaSize uint64
		if err := binary.Read(existingFile, binary.BigEndian, &metaSize); err != nil {
			continue
		}

		metaData := make([]byte, metaSize)
		if _, err := existingFile.Read(metaData); err != nil {
			continue
		}

		var metaEntry MetadataEntry
		if err := json.Unmarshal(metaData, &metaEntry); err != nil {
			continue
		}

		// 收集需要的内容块哈希
		for _, fileInfo := range metaEntry.Files {
			neededHashes[fileInfo.ContentHash] = true
		}
	}

	// 只复制需要的内容块
	dataStartOffset := uint64(binary.Size(FileHeader{}))

	for hash, block := range blocks {
		if !neededHashes[hash] || block.RefCount <= 0 {
			continue
		}

		// 读取内容块数据
		existingFile.Seek(int64(block.Offset), io.SeekStart)
		contentData := make([]byte, block.Length)
		if _, err := existingFile.Read(contentData); err != nil {
			continue
		}

		// 写入新位置
		newOffset, _ := tempFile.Seek(0, io.SeekCurrent)
		if _, err := tempFile.Write(contentData); err != nil {
			existingFile.Close()
			tempFile.Close()
			return fmt.Errorf("写入内容块失败: %v", err)
		}

		// 创建新的内容块（更新偏移量）
		newBlock := &ContentBlock{
			Hash:     block.Hash,
			Size:     block.Size,
			Offset:   uint64(newOffset),
			Length:   block.Length,
			RefCount: block.RefCount,
		}
		newBlocks[hash] = newBlock
	}

	// 复制元数据并更新偏移量
	for _, entry := range entries {
		// 读取元数据
		existingFile.Seek(int64(entry.Offset), io.SeekStart)
		var metaSize uint64
		if err := binary.Read(existingFile, binary.BigEndian, &metaSize); err != nil {
			continue
		}

		metaData := make([]byte, metaSize)
		if _, err := existingFile.Read(metaData); err != nil {
			continue
		}

		// 获取新的元数据偏移量
		newMetaOffset, _ := tempFile.Seek(0, io.SeekCurrent)

		// 写入元数据大小和内容
		if err := binary.Write(tempFile, binary.BigEndian, metaSize); err != nil {
			existingFile.Close()
			tempFile.Close()
			return fmt.Errorf("写入元数据大小失败: %v", err)
		}
		if _, err := tempFile.Write(metaData); err != nil {
			existingFile.Close()
			tempFile.Close()
			return fmt.Errorf("写入元数据失败: %v", err)
		}

		// 创建新的索引条目（更新偏移量）
		newEntry := IndexEntry{
			RecordID:      entry.RecordID,
			Offset:        uint64(newMetaOffset),
			Size:          entry.Size,
			Operation:     entry.Operation,
			Domain:        entry.Domain,
			Timestamp:     entry.Timestamp,
			ExpiryTime:    entry.ExpiryTime,
			ContentOffset: entry.ContentOffset,
			ContentSize:   entry.ContentSize,
			Checksum:      entry.Checksum,
		}
		newEntries = append(newEntries, newEntry)
	}

	existingFile.Close()

	// 写入内容块索引
	contentBlockOffset, err := hm.writeContentBlockIndex(tempFile, newBlocks)
	if err != nil {
		tempFile.Close()
		return fmt.Errorf("写入内容块索引失败: %v", err)
	}

	// 写入记录索引
	indexOffset, _ := tempFile.Seek(0, io.SeekCurrent)
	indexData, err := json.Marshal(newEntries)
	if err != nil {
		tempFile.Close()
		return fmt.Errorf("序列化索引失败: %v", err)
	}

	if _, err := tempFile.Write(indexData); err != nil {
		tempFile.Close()
		return fmt.Errorf("写入索引失败: %v", err)
	}

	// 更新文件头
	totalSize, _ := tempFile.Seek(0, io.SeekEnd)
	tempFile.Seek(0, io.SeekStart)
	newHeader := FileHeader{
		MagicNumber:        [16]byte{},
		Version:            CurrentVersion,
		RecordCount:        uint64(len(newEntries)),
		IndexOffset:        uint64(indexOffset),
		IndexSize:          uint64(len(indexData)),
		ContentBlockOffset: contentBlockOffset,
		ContentBlockSize:   0, // 将在writeContentBlockIndex中更新
		DataOffset:         dataStartOffset,
		TotalSize:          uint64(totalSize),
	}
	copy(newHeader.MagicNumber[:], MagicNumber)

	// 重新计算ContentBlockSize
	blockList := make([]ContentBlock, 0, len(newBlocks))
	for _, block := range newBlocks {
		blockList = append(blockList, *block)
	}
	blockData, _ := json.Marshal(blockList)
	newHeader.ContentBlockSize = uint64(len(blockData))

	if err := binary.Write(tempFile, binary.BigEndian, newHeader); err != nil {
		tempFile.Close()
		return fmt.Errorf("写入文件头失败: %v", err)
	}

	// 计算并写入校验和
	if err := hm.calculateAndWriteChecksum(tempFile); err != nil {
		tempFile.Close()
		return fmt.Errorf("计算校验和失败: %v", err)
	}

	tempFile.Close()

	// 验证临时文件
	if err := hm.verifyBackupFile(tempPath); err != nil {
		return fmt.Errorf("验证备份文件失败: %v", err)
	}

	// 原子性替换
	return os.Rename(tempPath, backupPath)
}

// CleanupExpiredRecords 清理过期记录
func (hm *HistoryManager) CleanupExpiredRecords() (int, error) {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	backupPath := filepath.Join(common.GetConfig("Server", "WORKING_DIR"), BackupFilePath)

	// 检查文件是否存在
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return 0, nil // 文件不存在，无需清理
	}

	// 打开备份文件
	file, err := os.Open(backupPath)
	if err != nil {
		return 0, fmt.Errorf("打开备份文件失败: %v", err)
	}

	// 读取文件头
	var header FileHeader
	if err := binary.Read(file, binary.BigEndian, &header); err != nil {
		file.Close()
		return 0, fmt.Errorf("读取文件头失败: %v", err)
	}

	// 加载内容块索引
	contentBlocks, err := hm.loadContentBlocks(file, header)
	if err != nil {
		file.Close()
		return 0, fmt.Errorf("加载内容块索引失败: %v", err)
	}

	// 读取索引区域
	file.Seek(int64(header.IndexOffset), io.SeekStart)
	indexData := make([]byte, header.IndexSize)
	if _, err := file.Read(indexData); err != nil {
		file.Close()
		return 0, fmt.Errorf("读取索引区域失败: %v", err)
	}
	file.Close()

	// 解析索引条目
	var indexEntries []IndexEntry
	if err := json.Unmarshal(indexData, &indexEntries); err != nil {
		return 0, fmt.Errorf("解析索引区域失败: %v", err)
	}

	// 筛选过期记录
	now := time.Now().Unix()
	newEntries := make([]IndexEntry, 0, len(indexEntries))
	deletedCount := 0

	for _, entry := range indexEntries {
		if entry.ExpiryTime < now {
			// 记录已过期，减少引用计数
			hm.decreaseRefCount(contentBlocks, entry)
			deletedCount++
		} else {
			newEntries = append(newEntries, entry)
		}
	}

	if deletedCount == 0 {
		return 0, nil // 没有过期记录
	}

	// 重建备份文件
	if err := hm.rebuildBackupFile(newEntries, contentBlocks); err != nil {
		return 0, err
	}

	return deletedCount, nil
}

// backupHistoryRecordUnsafe 无锁版本的备份函数
func (hm *HistoryManager) backupHistoryRecordUnsafe(transactionID uint64) error {
	srcPath := filepath.Join(common.GetConfig("Server", "WORKING_DIR"), BackupFilePath)

	// 检查源文件是否存在
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return nil // 文件不存在，跳过备份
	}

	// 确保备份目录存在
	backupDir := filepath.Join(common.GetConfig("Server", "WORKING_DIR"), "./backup/")
	os.MkdirAll(backupDir, 0755)

	// 目标文件路径
	dstPath := filepath.Join(backupDir, fmt.Sprintf("%s%d", RollbackProtectionPrefix, transactionID))

	// 读取源文件内容
	content, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("读取备份文件失败: %v", err)
	}

	// 写入备份文件
	if err := os.WriteFile(dstPath, content, 0644); err != nil {
		return fmt.Errorf("写入回退保护文件失败: %v", err)
	}

	return nil
}

// deleteRecordsAfter 删除指定记录之后的所有记录
func (hm *HistoryManager) deleteRecordsAfter(recordID uint64) error {
	backupPath := filepath.Join(common.GetConfig("Server", "WORKING_DIR"), BackupFilePath)

	// 打开备份文件
	file, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("打开备份文件失败: %v", err)
	}

	// 读取文件头
	var header FileHeader
	if err := binary.Read(file, binary.BigEndian, &header); err != nil {
		file.Close()
		return fmt.Errorf("读取文件头失败: %v", err)
	}

	// 加载内容块索引
	contentBlocks, err := hm.loadContentBlocks(file, header)
	if err != nil {
		file.Close()
		return fmt.Errorf("加载内容块索引失败: %v", err)
	}

	// 读取索引区域
	file.Seek(int64(header.IndexOffset), io.SeekStart)
	indexData := make([]byte, header.IndexSize)
	if _, err := file.Read(indexData); err != nil {
		file.Close()
		return fmt.Errorf("读取索引区域失败: %v", err)
	}
	file.Close()

	// 解析索引条目
	var indexEntries []IndexEntry
	if err := json.Unmarshal(indexData, &indexEntries); err != nil {
		return fmt.Errorf("解析索引区域失败: %v", err)
	}

	// 筛选记录（保留recordID及之前的记录）
	newEntries := make([]IndexEntry, 0, len(indexEntries))
	for _, entry := range indexEntries {
		if entry.RecordID <= recordID {
			newEntries = append(newEntries, entry)
		} else {
			// 减少引用计数
			hm.decreaseRefCount(contentBlocks, entry)
		}
	}

	// 重建备份文件
	return hm.rebuildBackupFile(newEntries, contentBlocks)
}

// deleteRecordsAfterIncluding 删除指定记录及之后的所有记录（但保留excludeRecordID指定的记录）
func (hm *HistoryManager) deleteRecordsAfterIncluding(recordID uint64, excludeRecordID uint64) error {
	backupPath := filepath.Join(common.GetConfig("Server", "WORKING_DIR"), BackupFilePath)

	// 打开备份文件
	file, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("打开备份文件失败: %v", err)
	}

	// 读取文件头
	var header FileHeader
	if err := binary.Read(file, binary.BigEndian, &header); err != nil {
		file.Close()
		return fmt.Errorf("读取文件头失败: %v", err)
	}

	// 加载内容块索引
	contentBlocks, err := hm.loadContentBlocks(file, header)
	if err != nil {
		file.Close()
		return fmt.Errorf("加载内容块索引失败: %v", err)
	}

	// 读取索引区域
	file.Seek(int64(header.IndexOffset), io.SeekStart)
	indexData := make([]byte, header.IndexSize)
	if _, err := file.Read(indexData); err != nil {
		file.Close()
		return fmt.Errorf("读取索引区域失败: %v", err)
	}
	file.Close()

	// 解析索引条目
	var indexEntries []IndexEntry
	if err := json.Unmarshal(indexData, &indexEntries); err != nil {
		return fmt.Errorf("解析索引区域失败: %v", err)
	}

	// 筛选记录（保留recordID之前的记录，以及excludeRecordID指定的记录）
	newEntries := make([]IndexEntry, 0, len(indexEntries))
	for _, entry := range indexEntries {
		if entry.RecordID < recordID || entry.RecordID == excludeRecordID {
			newEntries = append(newEntries, entry)
		} else {
			// 减少引用计数
			hm.decreaseRefCount(contentBlocks, entry)
		}
	}

	// 重建备份文件
	return hm.rebuildBackupFile(newEntries, contentBlocks)
}

// GetHistoryRecords 获取所有历史记录
func (hm *HistoryManager) GetHistoryRecords() ([]IndexEntry, error) {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	backupPath := filepath.Join(common.GetConfig("Server", "WORKING_DIR"), BackupFilePath)

	// 检查文件是否存在
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return []IndexEntry{}, nil
	}

	// 打开备份文件
	file, err := os.Open(backupPath)
	if err != nil {
		return nil, fmt.Errorf("打开备份文件失败: %v", err)
	}
	defer file.Close()

	// 读取文件头
	var header FileHeader
	if err := binary.Read(file, binary.BigEndian, &header); err != nil {
		return nil, fmt.Errorf("读取文件头失败: %v", err)
	}

	// 读取索引区域
	file.Seek(int64(header.IndexOffset), io.SeekStart)
	indexData := make([]byte, header.IndexSize)
	if _, err := file.Read(indexData); err != nil {
		return nil, fmt.Errorf("读取索引区域失败: %v", err)
	}

	// 解析索引条目
	var indexEntries []IndexEntry
	if err := json.Unmarshal(indexData, &indexEntries); err != nil {
		return nil, fmt.Errorf("解析索引区域失败: %v", err)
	}

	return indexEntries, nil
}

// GetHistoryRecord 获取单个历史记录详情
func (hm *HistoryManager) GetHistoryRecord(recordID uint64) (*MetadataEntry, error) {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	backupPath := filepath.Join(common.GetConfig("Server", "WORKING_DIR"), BackupFilePath)

	// 打开备份文件
	file, err := os.Open(backupPath)
	if err != nil {
		return nil, fmt.Errorf("打开备份文件失败: %v", err)
	}
	defer file.Close()

	// 读取文件头
	var header FileHeader
	if err := binary.Read(file, binary.BigEndian, &header); err != nil {
		return nil, fmt.Errorf("读取文件头失败: %v", err)
	}

	// 读取索引区域
	file.Seek(int64(header.IndexOffset), io.SeekStart)
	indexData := make([]byte, header.IndexSize)
	if _, err := file.Read(indexData); err != nil {
		return nil, fmt.Errorf("读取索引区域失败: %v", err)
	}

	// 解析索引条目
	var indexEntries []IndexEntry
	if err := json.Unmarshal(indexData, &indexEntries); err != nil {
		return nil, fmt.Errorf("解析索引区域失败: %v", err)
	}

	// 查找指定记录
	var targetEntry *IndexEntry
	for i := range indexEntries {
		if indexEntries[i].RecordID == recordID {
			targetEntry = &indexEntries[i]
			break
		}
	}

	if targetEntry == nil {
		return nil, fmt.Errorf("记录ID不存在: %d", recordID)
	}

	// 读取元数据
	file.Seek(int64(targetEntry.Offset), io.SeekStart)
	var metaSize uint64
	if err := binary.Read(file, binary.BigEndian, &metaSize); err != nil {
		return nil, fmt.Errorf("读取元数据大小失败: %v", err)
	}

	metaData := make([]byte, metaSize)
	if _, err := file.Read(metaData); err != nil {
		return nil, fmt.Errorf("读取元数据失败: %v", err)
	}

	// 解析元数据
	var metaEntry MetadataEntry
	if err := json.Unmarshal(metaData, &metaEntry); err != nil {
		return nil, fmt.Errorf("解析元数据失败: %v", err)
	}

	return &metaEntry, nil
}

// BatchRestore 批量回退到指定记录
func (hm *HistoryManager) BatchRestore(toRecordID uint64) error {
	return hm.RestoreBackup(toRecordID)
}

// BackupHistoryRecord 备份history.record文件
func (hm *HistoryManager) BackupHistoryRecord(transactionID uint64) error {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()
	return hm.backupHistoryRecordUnsafe(transactionID)
}

// RollbackRollback 回退回退操作
func (hm *HistoryManager) RollbackRollback(rollbackTransactionID uint64) error {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	backupDir := filepath.Join(common.GetConfig("Server", "WORKING_DIR"), "./backup/")
	backupPath := filepath.Join(backupDir, fmt.Sprintf("%s%d", RollbackProtectionPrefix, rollbackTransactionID))

	// 检查备份文件是否存在
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("回退保护备份文件不存在")
	}

	// 读取备份文件内容
	content, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("读取回退保护备份文件失败: %v", err)
	}

	// 写入目标文件
	dstPath := filepath.Join(common.GetConfig("Server", "WORKING_DIR"), BackupFilePath)
	if err := os.WriteFile(dstPath, content, 0644); err != nil {
		return fmt.Errorf("写入history.record文件失败: %v", err)
	}

	// 删除回退保护文件
	os.Remove(backupPath)

	return nil
}

// RestoreFromRollbackBackup 从回退保护备份恢复
func (hm *HistoryManager) RestoreFromRollbackBackup(transactionID uint64) error {
	return hm.RollbackRollback(transactionID)
}

// GetRollbackProtectionFiles 获取所有回退保护文件
func (hm *HistoryManager) GetRollbackProtectionFiles() ([]string, error) {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	backupDir := filepath.Join(common.GetConfig("Server", "WORKING_DIR"), "./backup/")

	// 读取备份目录
	files, err := os.ReadDir(backupDir)
	if err != nil {
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

// CleanupRollbackProtectionFiles 清理回退保护文件
func (hm *HistoryManager) CleanupRollbackProtectionFiles() error {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	backupDir := filepath.Join(common.GetConfig("Server", "WORKING_DIR"), "./backup/")

	// 获取配置的最大记录数
	maxRecordsStr := common.GetConfig("Backup", "ROLLBACK_PROTECTION_MAX_RECORDS")
	maxRecords := DefaultRollbackProtectionMaxRecords
	if maxRecordsStr != "" {
		if val, err := strconv.Atoi(maxRecordsStr); err == nil {
			maxRecords = val
		}
	}

	// 读取备份目录
	files, err := os.ReadDir(backupDir)
	if err != nil {
		return fmt.Errorf("读取备份目录失败: %v", err)
	}

	// 筛选回退保护文件
	rollbackFiles := make([]os.DirEntry, 0)
	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), RollbackProtectionPrefix) {
			rollbackFiles = append(rollbackFiles, file)
		}
	}

	// 按修改时间排序
	sort.Slice(rollbackFiles, func(i, j int) bool {
		infoI, _ := rollbackFiles[i].Info()
		infoJ, _ := rollbackFiles[j].Info()
		return infoI.ModTime().Before(infoJ.ModTime())
	})

	// 计算需要删除的文件数量
	deleteCount := len(rollbackFiles) - maxRecords
	if deleteCount <= 0 {
		return nil
	}

	// 删除最旧的文件
	for i := 0; i < deleteCount && i < len(rollbackFiles); i++ {
		filePath := filepath.Join(backupDir, rollbackFiles[i].Name())
		if err := os.Remove(filePath); err != nil {
			hm.logger.Warn("删除回退保护文件失败: %v", err)
		}
	}

	return nil
}

// GetHistoryRecordsForAPI 获取格式化的历史记录（用于API响应）
func (hm *HistoryManager) GetHistoryRecordsForAPI() ([]HistoryRecord, error) {
	entries, err := hm.GetHistoryRecords()
	if err != nil {
		return nil, err
	}

	records := make([]HistoryRecord, 0, len(entries))
	for _, entry := range entries {
		// 转换操作类型为字符串
		operationStr := "unknown"
		switch entry.Operation {
		case OperationCreate:
			operationStr = "create"
		case OperationUpdate:
			operationStr = "update"
		case OperationDelete:
			operationStr = "delete"
		case OperationRollback:
			operationStr = "rollback"
		}

		// 获取文件列表
		metaEntry, err := hm.GetHistoryRecord(entry.RecordID)
		files := []string{}
		if err == nil && metaEntry != nil {
			for _, fileInfo := range metaEntry.Files {
				files = append(files, fileInfo.FileName)
			}
		}

		record := HistoryRecord{
			ID:        int(entry.RecordID),
			Operation: operationStr,
			Domain:    entry.Domain,
			Timestamp: time.Unix(entry.Timestamp, 0).UTC(),
			Files:     files,
		}
		records = append(records, record)
	}

	return records, nil
}
