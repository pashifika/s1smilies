/*
Version: 0.1
@author: Pashifika
*/
package main

import (
	"bufio"
	"fmt"
	"go.uber.org/atomic"
	"os"
	"path/filepath"
	"regexp"
	"s1smilies/libs"
	"sync"
)

const (
	dlDirPath         = "./download"
	dlThreads         = 5
	smiliesJS_cache   = "https://bbs.saraba1st.com/2b/data/cache/common_smilies_var.js"
	smiliesURL_root   = "https://static.saraba1st.com/image/smiley/"
	smiliesType_re    = `smilies_type\['_([0-9]+)'] = \['(.+)', '(.+)']` // Submatch the [TypeID, TypeName, DirPath]
	smiliesArray_re   = `smilies_array\[([0-9]+)][\[\]0-9]{3,} = (.+)`   // Submatch the [TypeID, smiliesData]
	smiliesFile_index = 2                                                // Find the smilies file name in json data index
)

func main() {
	fmt.Println("加载s1论坛缓存数据中...")
	sdl, err := libs.LoadJStoMemory(
		smiliesJS_cache,
		regexp.MustCompile(smiliesType_re), regexp.MustCompile(smiliesArray_re),
		smiliesFile_index,
	)
	if err != nil {
		panic(err)
	}

	fmt.Print("数据加载完毕！\n\n")
	fmt.Println("请选择需要下载的麻将脸类型")
	var dlList, allList []string
	for stID, v := range sdl.Stype {
		fmt.Printf("ID:%s\t\tName:%s\t\tTotal:%d\n", stID, v.Name, len(sdl.DLlist[stID]))
		allList = append(allList, stID)
	}
	fmt.Print("提示: 输入 'all' 下载全部类型，输入 'list' 列出所有麻将脸类型，输入 'exit' 退出程序\n\n")

	// Check download dir
	os_info := os.Args[0]
	run_dir, err := filepath.Abs(filepath.Dir(os_info))
	if err != nil {
		panic(err)
	}
	dlDir := filepath.Join(run_dir, dlDirPath)
	if err = makeDirs(dlDir, 0755); err != nil {
		panic(err)
	}

	// Interactive cli for download smiliesType
	fmt.Print("请输入需要下载的麻将脸类型ID: ")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input := scanner.Text()
		switch input {
		case "exit": // Exit program
			return
		case "list": // List all smilies type
			fmt.Println("请选择需要下载的麻将脸类型")
			for stID, v := range sdl.Stype {
				fmt.Printf("ID:%s\t\tName:%s\t\tTotal:%d\n", stID, v.Name, len(sdl.DLlist[stID]))
			}
			fmt.Print("提示: 输入 'all' 下载全部类型，输入 'list' 列出所有麻将脸类型，输入 'exit' 退出程序\n\n")
			fmt.Print("请输入需要下载的麻将脸类型ID: ")
		default: // Download process
			if input != "all" {
				if _, ok := sdl.Stype[input]; !ok {
					fmt.Println("输入错误, 请再次输入！")
					fmt.Print("请输入需要下载的麻将脸类型ID: ")
					continue
				}
				dlList = []string{input}
			} else {
				dlList = allList
			}

			fmt.Println("下载程序干活中，请勿关闭本窗口...")
			// multi limit setting start
			var wg sync.WaitGroup
			semaphore := make(chan int, dlThreads)
			// multi limit setting end
			for _, stID := range dlList {
				var count_err, count atomic.Int64
				for _, fdl := range sdl.DLlist[stID] {
					// Make download thread
					wg.Add(1)
					semaphore <- 1
					go func(url, saveName string) {
						defer func() {
							<-semaphore
							wg.Done()
						}()
						filePath := filepath.Join(dlDir, saveName)

						// Check local smilies is exists
						if exists(filePath) {
							count.Add(1)
							return
						}

						// Download remote smilies to local
						err = libs.DownloadFile(filePath, url)
						if err != nil {
							fmt.Printf("file %s download error, %s", url, err.Error())
							count_err.Add(1)
						} else {
							count.Add(1)
						}
					}(smiliesURL_root+sdl.Stype[stID].DirPath+"/"+fdl.FileName, fdl.Name)
				}

				// Wait all download thread complete
				wg.Wait()
				fmt.Printf("类型：%s\t下载成功：%d\t下载失败：%d\n\n", sdl.Stype[stID].Name, count.Load(), count_err.Load())
			}

			fmt.Print("提示: 输入 'all' 下载全部类型，输入 'list' 列出所有麻将脸类型，输入 'exit' 退出程序\n")
			fmt.Print("请输入需要下载的麻将脸类型ID: ")
		}
	}
}

// Exists returns whether the given file or directory exists or not
func exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true
}

// dir_perm 0755
func makeDirs(dir_path string, dir_perm os.FileMode) error {
	if !exists(dir_path) {
		err := os.MkdirAll(dir_path, dir_perm)
		if err != nil {
			return err
		}
	}

	return nil
}
