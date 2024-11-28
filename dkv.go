package quark

import (
	"encoding/json"
	"fmt"
	"github.com/peterbourgon/diskv/v3"
	"os"
	"reflect"
	"strconv"
)

var dkv *diskv.Diskv

func init() {
	// 定义一个简单的转换函数，将所有数据文件放入基础目录。
	// 使用提供的选项初始化一个新的diskv存储，根目录为从配置读出，缓存大小为10MB。
	dkv = diskv.New(diskv.Options{
		BasePath:     os.TempDir() + "/quark-cache",
		CacheSizeMax: 10 * 1024 * 1024, // 10MB
	})
}

func GetCache[V any](key string, val V) error {
	if dkv.Has(key) {
		raw := dkv.ReadString(key)

		// 反射获取val的指针
		valValue := reflect.ValueOf(val)
		if valValue.Kind() != reflect.Ptr {
			return fmt.Errorf("expected a pointer, got: %s", valValue.Kind())
		}

		// 将string转换为目标类型
		switch valValue.Elem().Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64,
			reflect.String:
			// 基础类型直接转换
			if valValue.Elem().Kind() == reflect.String {
				valValue.Elem().SetString(raw)
			} else if valValue.Elem().Kind() == reflect.Int {
				// 尝试将string转换为数值
				num, err := strconv.ParseInt(raw, 10, 64)
				if err != nil {
					return err
				}
				valValue.Elem().SetInt(num)
			} else {
				// 尝试将string转换为数值
				num, err := strconv.ParseFloat(raw, 64)
				if err != nil {
					return err
				}
				valValue.Elem().SetFloat(num)
			}
		default:
			// 对于非基础类型，使用JSON转换
			if valValue.Elem().Kind() == reflect.Struct {
				err := json.Unmarshal([]byte(raw), val)
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("unsupported type: %s", valValue.Elem().Kind())
			}
		}
		return nil
	}
	return fmt.Errorf("key:%s not exist", key)
}

func SetCache(key string, value interface{}) error {
	var val string
	if v, ok := value.(string); ok {
		val = v
	} else if v, ok := value.(int64); ok {
		val = strconv.FormatInt(v, 10)
	} else {
		marshal, err := json.Marshal(value)
		if err != nil {
			return err
		}
		val = string(marshal)
	}
	return dkv.WriteString(key, val)
}

func DelCache(key string) error {
	return dkv.Erase(key)
}
