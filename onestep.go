package quark

import (
	"fmt"
	"github.com/Xhofe/go-cache"
	"github.com/imroc/req/v3"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var dirCache = cache.NewMemCache(cache.WithShards[[]File](100))

type ProgressReader struct {
	io.ReadCloser
	totalSize int64
	uploaded  int64
	startTime time.Time
}

func (pr *ProgressReader) Read(p []byte) (n int, err error) {
	n, err = pr.ReadCloser.Read(p)
	if n > 0 {
		pr.uploaded += int64(n)
		elapsed := time.Since(pr.startTime).Seconds()
		var speed float64
		if elapsed == 0 {
			speed = float64(pr.uploaded) / 1024
		} else {
			speed = float64(pr.uploaded) / 1024 / elapsed // KB/s
		}

		// 计算进度百分比
		percent := float64(pr.uploaded) / float64(pr.totalSize) * 100
		fmt.Printf("\ruploading: %.2f%% (%d/%d bytes, %.2f KB/s)", percent, pr.uploaded, pr.totalSize, speed)
		// 相等即已经处理完毕
		if pr.uploaded == pr.totalSize {
			fmt.Println()
		}
	}
	return n, err
}

func isEmpty(dirPath string) (bool, error) {
	dir, err := os.Open(dirPath)
	if err != nil {
		return false, err
	}
	defer dir.Close()
	//如果目录不为空，Readdirnames 会返回至少一个文件名
	_, err = dir.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

// UploadPath 一键上传路径
func (c *QuarkClient) UploadPath(req OneStepUploadPathReq) error {
	dirCache.Clear()
	// 遍历目录
	err := filepath.Walk(req.LocalPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			for _, ignorePath := range req.IgnorePaths {
				if filepath.Base(path) == ignorePath {
					return filepath.SkipDir
				}
			}
		} else {
			// 获取相对于root的相对路径
			relPath, _ := filepath.Rel(req.LocalPath, path)
			relPath = strings.Replace(relPath, "\\", "/", -1)
			relPath = strings.Replace(relPath, info.Name(), "", 1)
			NotUpload := false
			for _, ignoreFile := range req.IgnoreFiles {
				if info.Name() == ignoreFile {
					NotUpload = true
					break
				}
			}
			for _, extension := range req.IgnoreExtensions {
				if strings.HasSuffix(info.Name(), extension) {
					NotUpload = true
					break
				}
			}
			for _, extension := range req.Extensions {
				if strings.HasSuffix(info.Name(), extension) {
					NotUpload = false
					break
				}
				NotUpload = true
			}
			if !NotUpload {
				err = c.UploadFile(OneStepUploadFileReq{
					LocalFile:      path,
					RemotePath:     strings.TrimRight(req.RemotePath, "/") + "/" + relPath,
					Resumable:      req.Resumable,
					SuccessDel:     req.SuccessDel,
					RemoteTransfer: req.RemoteTransfer,
				})
				if err == nil {
					if req.SuccessDel {
						dir := filepath.Dir(path)
						if dir != "." {
							empty, _ := isEmpty(dir)
							if empty {
								_ = os.Remove(dir)
								fmt.Println("uploaded success and delete", dir)
							}
						}
					}
				} else {
					if !req.SkipFileErr {
						return err
					} else {
						fmt.Println("upload err", err)
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

// UploadFile 一键上传文件
func (c *QuarkClient) UploadFile(req OneStepUploadFileReq) error {

	md5Str, err := getFileMd5(req.LocalFile)
	if err != nil {
		return err
	}
	sha1Str, err := getFileSha1(req.LocalFile)
	if err != nil {
		return err
	}

	mimeType := getMimeType(req.LocalFile)

	file, err := os.Open(req.LocalFile)
	if err != nil {
		return err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return err
	}
	remoteName := stat.Name()
	remotePath := req.RemotePath
	if req.RemoteTransfer != nil {
		remoteName, remotePath = req.RemoteTransfer(remoteName, remotePath)
	}
	dirId, err := c.FileId(remotePath, true)
	if err != nil {
		return err
	}
	md5Key := md5Hash(req.LocalFile + remotePath + dirId)
	var pre RespDataWithMeta[FileUpPre, FileUpPreMeta]
	if req.Resumable {
		cacheErr := GetCache("session_"+md5Key, &pre)
		if cacheErr != nil {
			fmt.Println("cache err:", cacheErr)
		}
	}
	if pre.Data.TaskId == "" {
		// pre
		resp, err := c.FileUploadPre(FileUpPreReq{
			ParentId: dirId,
			FileName: remoteName,
			FileSize: stat.Size(),
			MimeType: mimeType,
		})
		if err != nil {
			return err
		}
		pre = *resp
		cacheErr := SetCache("session_"+md5Key, pre)
		if cacheErr != nil {
			fmt.Println("cache err:", cacheErr)
		}
	}

	// hash
	finish, err := c.FileUploadHash(FileUpHashReq{
		Md5:    md5Str,
		Sha1:   sha1Str,
		TaskId: pre.Data.TaskId,
	})
	if err != nil {
		return err
	}
	if finish.Data.Finish {
		return nil
	}

	uploadedSize := 0
	md5s := make([]string, 0)
	if req.Resumable {
		cacheErr := GetCache("chunk_"+md5Key, &uploadedSize)
		if cacheErr != nil {
			fmt.Println("cache err:", cacheErr)
		}
		var md5Strs string
		cacheErr = GetCache("md5s_"+md5Key, &md5Strs)
		if cacheErr != nil {
			fmt.Println("cache err:", cacheErr)
		}
		if md5Strs != "" {
			md5s = strings.Split(md5Strs, ",")
		}
	}
	// part up
	partSize := pre.Metadata.PartSize
	total := stat.Size()
	left := total - int64(uploadedSize)
	if uploadedSize > 0 {
		// 将文件指针移动到指定的分片位置
		ret, _ := file.Seek(int64(uploadedSize), 0)
		if ret == 0 {
			return fmt.Errorf("seek file failed")
		}
	}
	partNumber := (uploadedSize / partSize) + 1
	pr := &ProgressReader{
		startTime: time.Now(),
		totalSize: total,
		uploaded:  int64(uploadedSize),
	}
	for left > 0 {
		pr.ReadCloser = io.NopCloser(&io.LimitedReader{
			R: file,
			N: int64(partSize),
		})
		chunkUploadSize := min(total-int64((partNumber-1)*partSize), int64(partSize))
		left -= chunkUploadSize
		m, err := c.FileUpPart(FileUpPartReq{
			ObjKey:     pre.Data.ObjKey,
			Bucket:     pre.Data.Bucket,
			UploadId:   pre.Data.UploadId,
			AuthInfo:   pre.Data.AuthInfo,
			UploadUrl:  pre.Data.UploadUrl,
			MineType:   mimeType,
			PartNumber: partNumber,
			TaskId:     pre.Data.TaskId,
			Reader:     pr,
		})
		if err != nil {
			return err
		}
		if m == "finish" {
			return nil
		}
		md5s = append(md5s, m)
		if req.Resumable {
			cacheErr := SetCache("chunk_"+md5Key, int64(uploadedSize)+chunkUploadSize)
			if cacheErr != nil {
				fmt.Println("cache err:", cacheErr)
			}
			cacheErr = SetCache("md5s_"+md5Key, strings.Join(md5s, ","))
			if cacheErr != nil {
				fmt.Println("cache err:", cacheErr)
			}
		}
		partNumber++
	}
	err = c.FileUpCommit(FileUpCommitReq{
		ObjKey:    pre.Data.ObjKey,
		Bucket:    pre.Data.Bucket,
		UploadId:  pre.Data.UploadId,
		AuthInfo:  pre.Data.AuthInfo,
		UploadUrl: pre.Data.UploadUrl,
		MineType:  mimeType,
		TaskId:    pre.Data.TaskId,
		Callback:  pre.Data.Callback,
	}, md5s)
	if err != nil {
		return err
	}
	_, err = c.FileUpFinish(FileUpFinishReq{
		ObjKey: pre.Data.ObjKey,
		TaskId: pre.Data.TaskId,
	})
	if err != nil {
		return err
	}
	if req.Resumable {
		_ = DelCache("session_" + md5Key)
		_ = DelCache("chunk_" + md5Key)
		_ = DelCache("md5s_" + md5Key)
	}
	// 上传成功则移除文件了
	if req.SuccessDel {
		_ = os.Remove(req.LocalFile)
		fmt.Println("uploaded success and delete", req.LocalFile)
	}
	return nil
}

func (c *QuarkClient) DownloadFile(object File, localPath string, downloadCallback DownloadCallback) error {
	fmt.Println("start download file", object.FileName)
	outputFile := localPath + "/" + object.FileName
	resp, err := c.FileDownload(object.Fid)
	if err != nil {
		return err
	}
	downloadUrl := resp.Data[0].DownloadUrl
	err = os.MkdirAll(localPath, os.ModePerm)
	if err != nil {
		return err
	}

	startTime := time.Now()
	callback := func(info req.DownloadInfo) {
		if info.Response.Response != nil {
			totalSize := info.Response.ContentLength
			downloaded := info.DownloadedSize
			elapsed := time.Since(startTime).Seconds()
			var speed float64
			if elapsed == 0 {
				speed = float64(downloaded) / 1024
			} else {
				speed = float64(downloaded) / 1024 / elapsed // KB/s
			}

			// 计算进度百分比
			percent := float64(downloaded) / float64(totalSize) * 100
			fmt.Printf("\rdownloaded: %.2f%% (%d/%d bytes, %.2f KB/s)", percent, downloaded, totalSize, speed)
			// 相等即已经处理完毕
			if downloaded == totalSize {
				fmt.Println()
			}
		}
	}

	//TODO 改成分片下载就能把带宽拉满
	_, err = c.downloadClient.R().
		//SetHeader("Range", fmt.Sprintf("bytes=%d-%d", 250*1024*1024, 500*1024*1024-1)).
		SetOutputFile(outputFile).
		SetDownloadCallback(callback).
		Get(downloadUrl)
	if err != nil {
		return err
	}

	fmt.Println("end download file", object.FileName)
	if downloadCallback != nil {
		abs, _ := filepath.Abs(outputFile)
		downloadCallback(filepath.Dir(abs), abs)
	}
	return nil
}

// ShareFile 一键创建分享
func (c *QuarkClient) ShareFile(req ShareReq) (*RespData[SharePasswordData], error) {
	shareId, err := c.Share(req)
	if err != nil {
		return nil, err
	}
	return c.SharePassword(shareId)
}

// FileId 此方法会判断目录是否存在，不存在会直接创建
func (c *QuarkClient) FileId(path string, usingCache bool) (string, error) {
	truePath := strings.Trim(path, "/")
	paths := strings.Split(truePath, "/")

	fileId := "0"
	lastParentId := "0"
	var search []File
	for index, pathStr := range paths {
		if pathStr == "" {
			continue
		}
		if usingCache {
			cacheSearch, found := dirCache.Get(lastParentId)
			if found {
				search = cacheSearch
			} else {
				remoteSearch, err := c.FileSort(lastParentId)
				if err != nil {
					return "", err
				}
				if len(remoteSearch) > 0 {
					dirCache.Set(lastParentId, remoteSearch)
				}
				search = remoteSearch
			}
		} else {
			remoteSearch, err := c.FileSort(lastParentId)
			if err != nil {
				return "", err
			}
			search = remoteSearch
		}

		exist := false
		for _, file := range search {
			if file.FileName == pathStr {
				lastParentId = file.Fid
				exist = true
				break
			}
		}
		if !exist {
			if filepath.Ext(pathStr) == "" {
				dir, err := c.MakeDir(pathStr, lastParentId)
				if err != nil {
					return "", err
				}
				dirCache.Del(lastParentId)
				lastParentId = dir.Data.Fid
			} else {
				return "", fmt.Errorf("file not found:%s", path)
			}
		}
		if index == len(paths)-1 {
			fileId = lastParentId
		}
	}
	return fileId, nil
}
