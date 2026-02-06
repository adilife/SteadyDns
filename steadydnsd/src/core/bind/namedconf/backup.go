// core/bind/namedconf/backup.go
// named.conf 文件备份模块

package namedconf

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BackupManager 备份管理器
type BackupManager struct {
	backupDir string
	maxBackups int
}

// BackupInfo 备份信息
type BackupInfo struct {
	FilePath string    `json:"filePath"`
	Timestamp time.Time `json:"timestamp"`
	Size     int64    `json:"size"`
}

// NewBackupManager 创建新的备份管理器实例
func NewBackupManager(backupDir string, maxBackups int) *BackupManager {
	if backupDir == "" {
		backupDir = "./backup"
	}

	if maxBackups <= 0 {
		maxBackups = 10
	}

	// 确保备份目录存在
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		fmt.Printf("创建备份目录失败: %v\n", err)
	}

	return &BackupManager{
		backupDir: backupDir,
		maxBackups: maxBackups,
	}
}

// BackupFile 备份文件
func (bm *BackupManager) BackupFile(filePath string) (*BackupInfo, error) {
	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("文件不存在: %s", filePath)
	}

	// 获取文件信息
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %v", err)
	}

	// 生成备份文件名
	timestamp := time.Now()
	fileName := filepath.Base(filePath)
	ext := filepath.Ext(fileName)
	nameWithoutExt := strings.TrimSuffix(fileName, ext)

	backupFileName := fmt.Sprintf("%s%s.%s.bak", 
		nameWithoutExt, 
		ext, 
		timestamp.Format("20060102150405"))

	backupPath := filepath.Join(bm.backupDir, backupFileName)

	// 复制文件
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %v", err)
	}

	if err := os.WriteFile(backupPath, content, fileInfo.Mode()); err != nil {
		return nil, fmt.Errorf("写入备份文件失败: %v", err)
	}

	// 清理旧备份
	if err := bm.cleanupOldBackups(fileName); err != nil {
		fmt.Printf("清理旧备份失败: %v\n", err)
	}

	return &BackupInfo{
		FilePath: backupPath,
		Timestamp: timestamp,
		Size:     fileInfo.Size(),
	}, nil
}

// ListBackups 列出指定文件的所有备份
func (bm *BackupManager) ListBackups(originalFilePath string) ([]BackupInfo, error) {
	fileName := filepath.Base(originalFilePath)
	ext := filepath.Ext(fileName)
	nameWithoutExt := strings.TrimSuffix(fileName, ext)

	// 匹配模式
	pattern := fmt.Sprintf("%s%s.*.bak", nameWithoutExt, ext)

	// 读取备份目录
	files, err := os.ReadDir(bm.backupDir)
	if err != nil {
		return nil, fmt.Errorf("读取备份目录失败: %v", err)
	}

	// 过滤备份文件
	var backups []BackupInfo
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if matched, _ := filepath.Match(pattern, file.Name()); matched {
			filePath := filepath.Join(bm.backupDir, file.Name())
			fileInfo, err := file.Info()
			if err != nil {
				continue
			}

			// 解析时间戳
			timestampStr := strings.TrimSuffix(strings.TrimPrefix(file.Name(), nameWithoutExt+ext+"."), ".bak")
			timestamp, err := time.Parse("20060102150405", timestampStr)
			if err != nil {
				continue
			}

			backups = append(backups, BackupInfo{
				FilePath: filePath,
				Timestamp: timestamp,
				Size:     fileInfo.Size(),
			})
		}
	}

	// 按时间戳排序（最新的在前）
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	return backups, nil
}

// RestoreBackup 从备份恢复文件
func (bm *BackupManager) RestoreBackup(backupPath, targetPath string) error {
	// 检查备份文件是否存在
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("备份文件不存在: %s", backupPath)
	}

	// 读取备份文件
	content, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("读取备份文件失败: %v", err)
	}

	// 写入目标文件
	if err := os.WriteFile(targetPath, content, 0644); err != nil {
		return fmt.Errorf("写入目标文件失败: %v", err)
	}

	return nil
}

// cleanupOldBackups 清理旧备份
func (bm *BackupManager) cleanupOldBackups(originalFileName string) error {
	// 列出所有备份
	backups, err := bm.ListBackups(filepath.Join(".", originalFileName))
	if err != nil {
		return err
	}

	// 如果备份数量超过限制，删除最旧的
	if len(backups) > bm.maxBackups {
		backupsToDelete := backups[bm.maxBackups:]
		for _, backup := range backupsToDelete {
			if err := os.Remove(backup.FilePath); err != nil {
				fmt.Printf("删除旧备份失败: %v\n", err)
			}
		}
	}

	return nil
}
