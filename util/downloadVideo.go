package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"main/db"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

type RedditVideo struct {
	BitrateKbps int    `json:"bitrate_kbps,omitempty"`
	FallbackUrl string `json:"fallback_url,omitempty"`
	Height      int    `json:"height,omitempty"`
	Width       int    `json:"width,omitempty"`
	DashUrl     string `json:"dash_url,omitempty"`
	IsGift      bool   `json:"is_gif,omitempty"`
	HlsUrl      string `json:"hls_url,omitempty"`
}

type Media struct {
	RedditVideo RedditVideo `json:"reddit_video,omitempty"`
}

type DataPost struct {
	Selftext  string `json:"selftext"`
	Subreddit string `json:"subreddit"`
	Saved     bool   `json:"saved"`
	Downs     int    `json:"downs"`
	Name      string `json:"name"`
	Domain    string `json:"domain"`
	Score     int    `json:"score"`
	Media     Media  `json:"media"`
	Id        string `json:"id"`
	Thumbnail string `json:"thumbnail,omitempty"`
	Title     string `json:"title"`
}

type Children struct {
	Kind string   `json:"kind"`
	Data DataPost `json:"data"`
}

type Data struct {
	After     bool       `json:"after"`
	Modhash   string     `json:"modhash"`
	GeoFilter string     `json:"geo_filter"`
	Before    bool       `json:"before"`
	Children  []Children `json:"children"`
}

type List struct {
	Data Data `json:"data,omitempty"`
}

func ReturnJson(url string) (error, []List) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil) // create a new GET request

	if err != nil { // check for errors
		return err, nil
	}

	req.Header = http.Header{ // set the request headers
		"User-Agent":   {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/113.0.0.0 Safari/537.36"},
		"Content-Type": {"application/json"},
	}

	resp, err := client.Do(req) // make a GET request to the url

	if err != nil {
		return err, nil // return the error and an empty string
	}

	var res []List
	if err != nil {
		return err, nil // return the error and an empty string
	}

	defer resp.Body.Close() // close the response body when the function returns

	b, _ := io.ReadAll(resp.Body) // read the response body

	if err := json.Unmarshal([]byte(b), &res); err != nil {
		return err, nil // return the error and an empty string
	}

	return nil, res
}

func DownloadFile(path string, url string) error {
	resp, err := http.Get(url) // make a GET request to the url
	if err != nil {
		return err // return the error and an empty string
	}

	if resp.StatusCode == http.StatusForbidden {
		return errors.New("403: Status Forbidden")
	}

	defer resp.Body.Close() // close the response body when the function returns

	out, err := os.Create(path) // create the file
	if err != nil {
		return err // return the error and an empty string
	}

	defer out.Close() // close the file when the function returns

	_, err = io.Copy(out, resp.Body) // copy the response body to the file
	return err
}

func ParseUri(uri string) (string, error) {
	parsedUri, err := url.Parse(uri)

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("https://%s%s", parsedUri.Host, parsedUri.Path), nil
}

func DownloadRedditVideo(uri string) (error, string) {
	parsedUri, err := ParseUri(uri) // Parse url
	if err != nil {
		return err, "" // If there is a error, return.
	}

	uri = parsedUri
	err, response := ReturnJson(fmt.Sprintf("%s.json", uri))
	if err != nil {
		return err, ""
	}

	videoInfo := struct {
		DashUrl     string
		AudioUrl    string
		OldAudioUrl string
	}{
		DashUrl:     response[0].Data.Children[0].Data.Media.RedditVideo.FallbackUrl,
		AudioUrl:    fmt.Sprintf("https://v.redd.it/%s/DASH_AUDIO_128.mp4", strings.Split(response[0].Data.Children[0].Data.Media.RedditVideo.FallbackUrl, "/")[3]),
		OldAudioUrl: fmt.Sprintf("https://v.redd.it/%s/DASH_audio.mp4", strings.Split(response[0].Data.Children[0].Data.Media.RedditVideo.FallbackUrl, "/")[3]),
	} // create a map to store the response body

	dashPath := fmt.Sprintf("./tmp/v/%s.mp4", response[0].Data.Children[0].Data.Id)
	audioPath := fmt.Sprintf("./tmp/a/%s.mp4", response[0].Data.Children[0].Data.Id)
	outputPath := fmt.Sprintf("./tmp/%s.mp4", response[0].Data.Children[0].Data.Id)

	if _, err := os.Stat(outputPath); err == nil {
		return nil, response[0].Data.Children[0].Data.Id
	}

	if err := DownloadFile(dashPath, videoInfo.DashUrl); err != nil {
		return err, "" // return the error and an empty string
	}
	if err := DownloadFile(audioPath, videoInfo.AudioUrl); err != nil {
		if err := DownloadFile(audioPath, videoInfo.OldAudioUrl); err != nil {
			return err, "" // return the error and an empty string
		}
	}
	cmd := exec.Command("ffmpeg", "-i", dashPath, "-i", audioPath, "-c:v", "copy", "-c:a", "copy", outputPath) // create the command
	if err := cmd.Run(); err != nil {
		return err, "" // return the error and an empty string
	}
	time.Sleep(1 * time.Second) // wait for 1 second

	if err := os.Remove(dashPath); err != nil { // Delete the unused file.
		// Ignore the error
		log.Fatal(err)
	}

	if err := os.Remove(audioPath); err != nil { // Delete the unused file.
		// Ignore the error
		log.Fatal(err)
	}

	created, err := database.Post.CreateOne(
		db.Post.ID.Set(response[0].Data.Children[0].Data.Id),
		db.Post.PostTitle.Set(response[0].Data.Children[0].Data.Title),
		db.Post.Thumbnail.Set(strings.Replace(response[0].Data.Children[0].Data.Thumbnail, "amp;", "", -1)),
		db.Post.PostURL.Set(uri),
	).Exec(ctx) // create a new post
	_ = created // ignore the created object

	if err != nil {
		return err, "" // return the error and an empty string
	}
	return nil, response[0].Data.Children[0].Data.Id
}

const VIDEO_PER_PAGE = 50

func RetrieveLatestVideos(page int) (error, []db.PostModel) {
	skip := page * VIDEO_PER_PAGE

	posts, err := database.Post.FindMany().Skip(skip).Take(VIDEO_PER_PAGE).OrderBy(db.Post.Date.Order(db.DESC)).Exec(ctx) // find all posts
	if err != nil {
		return err, nil // return the error and an empty string
	}

	posts = DeleteNsfwPosts(posts)

	return nil, posts // return the error and the posts
}

func DeleteNsfwPosts(posts []db.PostModel) []db.PostModel {
	var tempArray []db.PostModel

	for _, post := range posts {
		a := strings.Contains(post.Thumbnail, "nsfw")
		b := strings.Contains(post.Thumbnail, "default")
		if !(a || b) {
			tempArray = append(tempArray, post)
		}
	}

	if len(tempArray) == 0 {
		return []db.PostModel{}
	}

	return tempArray
}

func RetrieveCount() (error, int) {
	posts, err := database.Post.FindMany().Exec(ctx)
	if err != nil {
		return err, 0 // return the error and an empty string
	}
	return nil, len(posts) // return the error and the posts
}

func GetPost(id string) (error, *db.PostModel) {
	post, err := database.Post.FindFirst(db.Post.ID.Contains(id)).Exec(ctx)
	if err != nil {
		return err, nil
	}

	return nil, post
}

func DeletePost(id string) error {
	post, err := database.Post.FindUnique(db.Post.ID.Equals(id)).Delete().Exec(ctx) // Get post data from database and delete from db.
	if err != nil {
		return err // return error if there is a error while getting data
	}
	_ = os.Remove(fmt.Sprintf("./tmp/%s.mp4", post.ID)) // Delete the file.

	return nil
}
