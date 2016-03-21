package main

import (
	"net/http"
	"fmt"
	"io/ioutil"
	"github.com/PuerkitoBio/goquery"
	"bufio"
	"os"
	"strconv"
	"strings"
	"path"
	"io"
	"github.com/mitchellh/ioprogress"
	"os/exec"
	"runtime"
)



type DeviceImageUrls struct {
	name string
	infos []UrlInfo
}

type UrlInfo struct {
	version, url, md5, sha1 string
}

func get() {
	response, err := http.Get("https://developers.google.com/android/nexus/images")
	if err != nil {
		fmt.Println(err)
		return
	}

	defer response.Body.Close()
	byteArray, _ := ioutil.ReadAll(response.Body)
	//	fmt.Println(string(byteArray))
	ioutil.WriteFile("sample.html", byteArray, 0644)
}


func download(url string) (string, bool){
	response, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
		return "", false
	}
	defer response.Body.Close()
	s, _ := strconv.Atoi(response.Header.Get("Content-Length"))

	_, filename := path.Split(strings.Replace(url, "https://", "", 1))
	file, err := os.Create(filename + ".tmp")
	if err != nil {
		fmt.Println(err)
		return "", false
	}

	bar := ioprogress.DrawTextFormatBar(20)
	progress := &ioprogress.Reader{
		Reader: response.Body,
		Size:   int64(s),
		DrawFunc: ioprogress.DrawTerminalf(os.Stdout, func(progress, total int64) string {
			return fmt.Sprintf("%s %s %20s", filename, bar(progress, total), ioprogress.DrawTextFormatBytes(progress, total))
		}),
	}
	_, e := io.Copy(file, progress)
	if e != nil {
		fmt.Println(e)
		return "", false
	}
	os.Rename(filename + ".tmp", filename)

	return filename, true
}


func createUrlInfo(s *goquery.Selection) UrlInfo {
	info := UrlInfo{}
	s.Find("td").Each(func(cnt int, s *goquery.Selection) {
		switch cnt {
		case 0:
			info.version = s.Text()
		case 1:
			info.url, _ = s.Children().Attr("href")
		case 2:
			info.md5 = s.Text()
		case 3:
			info.sha1 = s.Text()
		default:
		}
	})
	return info
}


func getUrls() ([]DeviceImageUrls) {
	ret := make([]DeviceImageUrls, 0)
	doc, _ := goquery.NewDocument("https://developers.google.com/android/nexus/images")
	doc.Find("h2 + table").Each(func(i int, s *goquery.Selection) {
		devName, _ := s.Prev().Attr("id")
		deviceImages := DeviceImageUrls{name: devName}
		s.Find("tr").Each(func(j int, ss *goquery.Selection) {
			_id, _ := ss.Attr("id")
			if _id != "" {
				deviceImages.infos = append(deviceImages.infos, createUrlInfo(ss))
			}
		})
		ret = append(ret, deviceImages)
	})
	return ret
}


func getInput(text string, output []string) (string, int) {
	reader := bufio.NewReader(os.Stdin)

START:
	fmt.Println(text)
	for i, o := range output {
		fmt.Println(i+1, strings.Trim(o, "\n"))
	}
	input, _ := reader.ReadString('\n')
	cur, _ := strconv.Atoi(strings.Trim(input, "\n"))

	if(cur > len(output)){
		fmt.Println("Please input showing number")
		goto START
	}

	return output[cur-1], cur-1
}

func output(r io.Reader){
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}
}

func execCmd(cmdString string, params ...string) {
	cmd := exec.Command(cmdString, params...)
	outReader, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Print(err)
		return
	}

	errReader, err := cmd.StderrPipe()
	if err != nil {
		fmt.Print(err)
		return
	}

	go output(outReader)
	go output(errReader)
	if err = cmd.Run(); err != nil {
		fmt.Print(err)
		return
	}
}



func main() {
	devInfos := getUrls()

	// input model name
	models := make([]string, 0)
	for _, dev := range devInfos {
		models = append(models, dev.name)
	}
	_, devIndex := getInput("Select your device", models)

	// input version
	versions := make([]string, 0)
	for _, info := range devInfos[devIndex].infos {
		versions = append(versions, info.version)
	}
	_, versionIndex := getInput("Select the system image version for your device", versions)

	// download target image
	file, ok := download(devInfos[devIndex].infos[versionIndex].url);
	if !ok {
		fmt.Print("download error")
	}

	// uncompress & exec flash-all.sh
	dirName := "temp"
	execCmd("mkdir", dirName)
	execCmd("tar", "xvf", file, "-C", dirName, "--strip-components", "1")
	os.Chdir(dirName)
	switch runtime.GOOS {
	case "windows":
		// TODO:
	case "linux":
		fallthrough
	case "darwin":
		execCmd("./flash-all.sh")
	default:
	}
	os.Chdir("../")
	execCmd("rm", file)
	execCmd("rm", "-fr", dirName)
}
