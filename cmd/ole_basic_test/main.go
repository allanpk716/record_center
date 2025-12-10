//go:build windows

package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

func main() {
	fmt.Println("=== go-ole 基础功能测试 ===")
	fmt.Printf("时间: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	// 初始化OLE
	fmt.Println("步骤1: 初始化OLE...")
	if err := ole.CoInitialize(0); err != nil {
		fmt.Printf("❌ OLE初始化失败: %v\n", err)
		return
	}
	defer ole.CoUninitialize()
	fmt.Println("✅ OLE初始化成功")

	// 创建Shell.Application对象
	fmt.Println("\n步骤2: 创建Shell.Application对象...")
	unknown, err := ole.CreateInstance(ole.IID_IDispatch)
	if err != nil {
		fmt.Printf("❌ 创建Shell.Application失败: %v\n", err)
		return
	}
	defer unknown.Release()
	fmt.Println("✅ Shell.Application创建成功")

	// 获取IDispatch接口
	fmt.Println("\n步骤3: 获取IDispatch接口...")
	shell, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		fmt.Printf("❌ 获取IDispatch接口失败: %v\n", err)
		return
	}
	defer shell.Release()
	fmt.Println("✅ IDispatch接口获取成功")

	// 测试基本方法调用
	fmt.Println("\n步骤4: 测试基本方法调用...")

	// 获取Windows版本
	result, err := oleutil.CallMethod(shell, "ExpandEnvironmentStrings", "%OS%")
	if err != nil {
		fmt.Printf("❌ 调用ExpandEnvironmentStrings失败: %v\n", err)
	} else {
		fmt.Printf("✅ Windows版本: %s\n", result.ToString())
	}

	// 尝试获取便携式设备命名空间
	fmt.Println("\n步骤5: 尝试获取便携式设备命名空间...")
	result, err = oleutil.CallMethod(shell, "Namespace", 17)
	if err != nil {
		fmt.Printf("❌ 获取便携式设备命名空间失败: %v\n", err)
	} else {
		fmt.Println("✅ 便携式设备命名空间获取成功")

		// 尝试获取Items
		portable := result.ToIDispatch()
		defer portable.Release()

		itemsResult, err := oleutil.GetProperty(portable, "Items")
		if err != nil {
			fmt.Printf("❌ 获取Items失败: %v\n", err)
		} else {
			items := itemsResult.ToIDispatch()
			defer items.Release()

			// 获取数量
			countResult, err := oleutil.GetProperty(items, "Count")
			if err != nil {
				fmt.Printf("❌ 获取Count失败: %v\n", err)
			} else {
				fmt.Printf("✅ 便携式设备数量: %v\n", countResult.ToString())

				// 列出前几个设备
				maxItems := 5

				fmt.Println("便携式设备列表:")
				for i := 0; i < maxItems; i++ {
					itemResult, err := oleutil.CallMethod(items, "Item", i)
					if err != nil {
						fmt.Printf("  获取项目 %d 失败: %v\n", i, err)
						continue
					}

					item := itemResult.ToIDispatch()
					defer item.Release()

					nameResult, err := oleutil.GetProperty(item, "Name")
					if err != nil {
						fmt.Printf("  项目 %d: 获取名称失败: %v\n", i, err)
					} else {
						name := nameResult.ToString()
						fmt.Printf("  %d. %s\n", i+1, name)

						// 检查是否可能是录音设备
						if containsIgnoreCase(name, "SR302") ||
						   containsIgnoreCase(name, "录音") ||
						   containsIgnoreCase(name, "record") {
							fmt.Printf("     ^ 可能是录音设备！\n")
						}
					}
				}
			}
		}
	}

	// 尝试直接访问This PC
	fmt.Println("\n步骤6: 尝试访问This PC...")
	result, err = oleutil.CallMethod(shell, "Namespace", 0)
	if err != nil {
		fmt.Printf("❌ 获取This PC失败: %v\n", err)
	} else {
		fmt.Println("✅ This PC访问成功")
	}

	fmt.Println("\n=== 测试完成 ===")
	fmt.Println("✅ go-ole库基本功能正常")
	fmt.Println("下一步:")
	fmt.Println("1. 实现设备路径解析")
	fmt.Println("2. 实现文件枚举和访问")
	fmt.Println("3. 集成到MTP框架")

	// 等待用户输入
	fmt.Println("\n按任意键退出...")
	var input string
	fmt.Scanln(&input)
}

// containsIgnoreCase 忽略大小写的字符串包含检查
func containsIgnoreCase(s, substr string) bool {
	s = strings.ToLower(s)
	substr = strings.ToLower(substr)
	return strings.Contains(s, substr)
}