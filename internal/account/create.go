package account

import (
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"os"
	"strings"
)

func GenerateAccounts(filename string, count int) ([]string, error) {

	if checkFileExists(filename) {
		keys, err := readKeysFromFile(filename)
		if err != nil {
			return nil, fmt.Errorf("读取文件失败: %v", err)
		}

		existingKeyCount := len(keys)
		if existingKeyCount < count {
			missingCount := count - existingKeyCount
			err := appendKeysToFile(filename, missingCount)
			if err != nil {
				return nil, fmt.Errorf("补充私钥失败: %v", err)
			}
		}
		return keys, nil
	} else {
		return generateAndSaveKeys(filename, count)
	}
}

// checkFileExists 检查文件是否存在
func checkFileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

// readKeysFromFile 从文件中读取私钥和地址
func readKeysFromFile(filename string) ([]string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	var keys []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			keys = append(keys, line)
		}
	}
	return keys, nil
}

// appendKeysToFile 向文件追加新的私钥
func appendKeysToFile(filename string, count int) error {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("打开文件失败: %v", err)
	}
	defer func() {
		_ = file.Close()
	}()

	for i := 0; i < count; i++ {
		privateKey, _ := crypto.GenerateKey()
		privateKeyHex := hex.EncodeToString(privateKey.D.Bytes())
		_, err = file.WriteString(fmt.Sprintf("%s\n", privateKeyHex))
		if err != nil {
			return fmt.Errorf("写入文件失败: %v", err)
		}
	}
	return nil
}

// generateAndSaveKeys 生成并保存新的私钥（如果文件不存在）
func generateAndSaveKeys(filename string, count int) ([]string, error) {
	file, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("无法创建文件: %v", err)
	}
	defer func() {
		_ = file.Close()
	}()

	keys := []string{}
	for i := 0; i < count; i++ {
		privateKey, _ := crypto.GenerateKey()
		privateKeyHex := hex.EncodeToString(privateKey.D.Bytes())

		keys = append(keys, privateKeyHex)
		_, err = file.WriteString(fmt.Sprintf("%s\n", privateKeyHex))
		if err != nil {
			return nil, fmt.Errorf("写入文件失败: %v", err)
		}
	}
	return keys, nil
}
