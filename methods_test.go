package quark

import (
	"fmt"
	"testing"
)

var pus = "fdf89feb5dbef29454b84ca2256debf2AARhpdr0ONCtimUEhZno2l6IIgBD4HrVyGGS1i2BbwNJjWeXVkfbBwYNz0ZRr6lxKhhSLxtFBk+wDfESZ5phOuhw"
var puus = "6ac83c56fddbba90ae37704105df2632AARWvkP/31+FOgQBMcXvCQljDFlF8jjIO3XCmdH8w/jOIeatUUiBXerdxuAWWSOx4iIria1faShkgPE5BuZlg31VFpes/SZ8G0/sNXad7P7iBWzoMXmL8AyI34gO7hxI5LWoO8J+zJPF+PAyvKqHh6ZsWVzeO8D1WWHwnppXjqIldglOnrv6uk18hwyNJNFbk1QEOYgEbaTt24gQ99znxORl"

func beforeClient() *QuarkClient {
	return NewClient(pus, puus)
}

func TestConfig(t *testing.T) {
	client := beforeClient()
	resp, err := client.Config()
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	fmt.Println(resp)
}

func TestFileSort(t *testing.T) {
	client := beforeClient()
	files, err := client.FileSort("0")
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	for _, file := range files {
		fmt.Println(file.FileName)
	}
}

func TestDir(t *testing.T) {
	client := beforeClient()
	resp, err := client.MakeDir("夸克上传文件", "0")
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	fmt.Println(resp)
}

func TestDownloadFile(t *testing.T) {
	client := beforeClient()
	files, err := client.FileSort("d6d5e2be0d4e4beea02a99a8cbb3f527")
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	for _, file := range files {
		if file.File && file.FileName == "01.4k.mp4" {
			err = client.DownloadFile(file, "./target", nil)
			if err != nil {
				fmt.Println(err)
				panic(err)
			}
		}
	}

}

func TestOneStepUploadPath(t *testing.T) {
	client := beforeClient()
	err := client.UploadPath(OneStepUploadPathReq{
		LocalPath:  "D:/download/170",
		RemotePath: "/170",
		Resumable:  true,
		RemoteTransfer: func(remoteName, remotePath string) (string, string) {
			return remoteName, remotePath
		},
	})
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
}

func TestShareList(t *testing.T) {
	client := beforeClient()
	shareList, err := client.ShareList()
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	fmt.Println(shareList)
}

func TestOneStepShareFile(t *testing.T) {
	client := beforeClient()
	resp, err := client.ShareFile(ShareReq{
		FidList:     []string{"294442718ee844f4b22c4d67cc6fb418", "5e5f9877681245cc9fd20b66127059cb", "c1ca3e068fb64421a5fd8755f4b4c7b2"},
		Title:       "我的分享",
		UrlType:     2,
		ExpiredType: 1,
	})
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	fmt.Println(resp)
}

func TestShareDelete(t *testing.T) {
	client := beforeClient()
	shareList, err := client.ShareList()
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	shareIds := make([]string, 0)
	for _, list := range shareList {
		shareIds = append(shareIds, list.ShareId)
	}
	err = client.ShareDelete(shareIds)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
}

func TestDirId(t *testing.T) {
	client := beforeClient()
	dirId, err := client.FileId("/", false, false)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	fmt.Println(dirId)
}
