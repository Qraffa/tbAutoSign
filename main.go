package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/go-resty/resty/v2"
)

const (
	TEST_URL = "http://wttr.in"

	TBS_URL  = "http://tieba.baidu.com/dc/common/tbs"
	LIKE_URL = "https://tieba.baidu.com/mo/q/newmoindex"
	SIGN_URL = "http://c.tieba.baidu.com/c/c/forum/sign" // Post
)

// for TBS_URL
type TbsStruct struct {
	Tbs string `json:"tbs"`
}

// for LIKE_URL
type LikeStruct struct {
	Data struct {
		Tbs       string `json:"tbs"`
		LikeForum []struct {
			ForumName string `json:"forum_name"`
			ForumID   int    `json:"forum_id"`
			IsSign    int    `json:"is_sign"`
		} `json:"like_forum"`
	} `json:"data"`
}

type SignRespStruct struct {
	ErrorCode string `json:"error_code"`
	ErrorMsg  string `json:"error_msg"`
}

func main() {
	// 获取 BDUSS
	bduss := os.Getenv("BDUSS")
	if bduss == "" {
		fmt.Printf("BDUSS is nil")
		return
	}

	client := resty.New()

	tbs := getTBS(client, bduss)
	like := getLike(client, bduss)

	// do sign
	for i := 0; i < len(like.Data.LikeForum); i++ {
		forum := like.Data.LikeForum[i]
		if forum.IsSign == 0 {
			fmt.Printf("do sign. ForumName: %s ...\n", forum.ForumName)
			doSign(client, bduss, tbs, forum.ForumName, forum.ForumID)
		}
	}
}

func getTBS(client *resty.Client, bduss string) string {
	resp, err := client.R().SetCookie(&http.Cookie{Name: "BDUSS", Value: bduss}).Get(TBS_URL)
	tbs := &TbsStruct{}
	err = json.Unmarshal(resp.Body(), tbs)
	if err != nil {
		panic(err)
	}

	return tbs.Tbs
}

func getLike(client *resty.Client, bduss string) *LikeStruct {
	resp, err := client.R().SetCookie(&http.Cookie{Name: "BDUSS", Value: bduss}).Get(LIKE_URL)
	if err != nil {
		panic(err)
	}

	// fmt.Println(string(resp.Body()))

	data := &LikeStruct{}
	err = json.Unmarshal(resp.Body(), data)
	if err != nil {
		panic(err)
	}

	return data
}

func doSign(client *resty.Client, bduss string, tbs string, forumName string, forumID int) {
	bodyMap := map[string]string{
		"_client_id":      "03-00-DA-59-05-00-72-96-06-00-01-00-04-00-4C-43-01-00-34-F4-02-00-BC-25-09-00-4E-36",
		"_client_type":    "4",
		"_client_version": "1.2.1.17",
		"_phone_imei":     "540b43b59d21b7a4824e1fd31b08e9a6",
		"fid":             strconv.Itoa(forumID),
		"kw":              forumName,
		"net_type":        "3",
		"tbs":             tbs,
	}

	sign := getSignMD5(bodyMap)
	bodyMap["sign"] = sign

	// fmt.Println(bytesBuf.String())
	// fmt.Println(bodyMap)

	resp, err := client.R().
		SetFormData(bodyMap).
		SetCookie(&http.Cookie{Name: "BDUSS", Value: bduss}).
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		Post(SIGN_URL)
	if err != nil {
		panic(err)
	}

	signResp := &SignRespStruct{}
	if err := json.Unmarshal(resp.Body(), signResp); err != nil {
		panic(err)
	}

	if signResp.ErrorCode != "0" {
		fmt.Printf("sign failed. resp: %+v\n", signResp)
	}
}

func getSignMD5(m map[string]string) string {
	keys := make([]string, 0)
	for key := range m {
		keys = append(keys, key)
	}

	sort.Sort(sort.StringSlice(keys))

	str := strings.Builder{}
	for i := 0; i < len(keys); i++ {
		str.WriteString(fmt.Sprintf("%s=%s", keys[i], m[keys[i]]))
	}
	str.WriteString("tiebaclient!!!")

	h := md5.New()
	h.Write([]byte(str.String()))
	ent := hex.EncodeToString(h.Sum(nil))

	return strings.ToUpper(ent)
}
