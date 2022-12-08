package ttDownload

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

const (
	BaseParams string = "iid=7173886861033883398&device_id=7109589686008153606&ac=wifi&channel=googleplay&aid=1233&app_name=musical_ly&version_code=270204&version_name=27.2.4&device_platform=android&ab_version=27.2.4&ssmix=a&device_type=SM-N975F&device_brand=samsung&language=en&os_api=25&os_version=7.1.2&openudid=e424163b4fff5720&manifest_version_code=2022702040&resolution=800*1200&dpi=266&update_version_code=2022702040&_rticket=1670305079987&app_type=normal&sys_region=US&mcc_mnc=214214&timezone_name=America%2FChicago&ts=1670305079&timezone_offset=-21600&build_number=27.2.4&region=US&uoo=0&app_language=en&carrier_region=ES&locale=en&op_region=ES&ac2=wifi&host_abi=armeabi-v7a&cdid=2133c73a-770c-4aca-afe8-2daefb9e6946"
)

// DownloadFromSearch downloads a specified number of videos from the top TikToks for a given keyword. Returns true if the download was successful.
func DownloadFromSearch(keyword string, totalCount int) bool {
	client := &http.Client{}

	// create our working directory, if it exists, count the files in the directory and set the offset so we don't download the same videos again
	offset := 0
	if exists(keyword) {
		files, _ := ioutil.ReadDir(keyword)
		offset = len(files)
	} else {
		err := os.Mkdir(keyword, os.ModePerm)
		if err != nil {
			fmt.Println(err)
			return false
		}

		err = os.Mkdir(keyword+"/metadata", os.ModePerm)
		if err != nil {
			fmt.Println(err)
			return false
		}
	}

	baseOffset := offset
	videoIndex := 1

	for i := 0; i < int(math.Ceil(float64(totalCount)/30.0)); i++ { // looks scary, but the upper bound of this is simply the ceiling of the count divided by 30.0, the maximum amount of results the API can get us in a request
		// update our offset
		offset += i * 30

		// the count parameter tells the server how many videos to send back. there's a hard cap of 30 per request.
		count := 30

		// if this is the last run, we might not need to get a whole 30 videos
		if i == int(math.Ceil(float64(totalCount)/30.0))-1 && totalCount%30 != 0 {
			count = totalCount % 30
		}

		// generate the necessary signature headers
		var xGorgon, xKhronos = generateSignature(BaseParams + "&count=" + strconv.Itoa(count) + "&offset=" + strconv.Itoa(offset) + "&keyword=" + url.QueryEscape(keyword))

		// set up and send our request with our search term
		request, _ := http.NewRequest("POST", "https://search19-normal-c-useast1a.tiktokv.com/aweme/v1/search/item/?"+BaseParams+"&count="+strconv.Itoa(count)+"&offset="+strconv.Itoa(offset)+"&keyword="+url.QueryEscape(keyword), nil)
		request.Header.Set("User-Agent", "com.zhiliaoapp.musically/2022702040 (Linux; U; Android 7.1.2; en; SM-N975F; Build/N2G48H;tt-ok/3.12.13.1)")
		request.Header.Add("X-Khronos", strconv.FormatInt(xKhronos, 10))
		request.Header.Add("X-Gorgon", xGorgon)

		// send the search request
		response, err := client.Do(request)
		if err != nil {
			fmt.Println(err)
			return false
		}
		defer func(Body io.ReadCloser) {
			err = Body.Close()
			if err != nil {
				fmt.Println(err)
				return
			}
		}(response.Body)

		// get the response body
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Println(err)
			return false
		}

		// unmarshal the big JSON response to search, an instance of a struct used to store this JSON response
		var search SearchData
		err = json.Unmarshal(body, &search)
		if err != nil {
			fmt.Println(err)
			return false
		}

		// loop through all the returned videos, print some metadata and download each one
		for _, aweme := range search.AwemeList {
			downloadURL := aweme.Video.PlayAddr.URLList[0]

			video, _ := os.Create(keyword + "/" + strconv.Itoa(baseOffset+videoIndex) + ".mp4")

			resp, _ := http.Get(downloadURL)
			_, err := io.Copy(video, resp.Body)
			if err != nil {
				fmt.Println(err)
				return false
			}

			err = resp.Body.Close()
			if err != nil {
				fmt.Println(err)
				return false
			}
			err = video.Close()
			if err != nil {
				fmt.Println(err)
				return false
			}

			var ttVideoMetadata TikTokVideoMetadata

			ttVideoMetadata.Author = aweme.Author.UniqueID
			ttVideoMetadata.Description = aweme.Desc

			file, _ := json.MarshalIndent(ttVideoMetadata, "", " ")

			_ = ioutil.WriteFile(keyword+"/metadata/"+strconv.Itoa(baseOffset+videoIndex)+".json", file, 0644)

			videoIndex++
		}
	}
	return true
}
