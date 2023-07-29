package routes

import (
	"fmt"
	"main/util"
	"net/http"
	"os"

	"github.com/labstack/echo"
)

type DownloadBody struct {
	Url string `json:"videoUri"`
}

func DownloadVideo(c echo.Context) error {
	var downloadBody DownloadBody

	if err := c.Bind(&downloadBody); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"status": false,
			"path":   "not_found.html",
		})
	}

	err, path := util.DownloadRedditVideo(downloadBody.Url)

	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"status": false,
			"path":   "not_found.html",
		})
	}

	err, post := util.GetPost(path)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"status": false,
			"path":   "not_found.html",
		})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"status":  true,
		"message": "Video downloaded successfully.",
		"level":   "success",
		"path":    path,
		"post": echo.Map{
			"title":     post.PostTitle,
			"date":      post.Date,
			"url":       post.PostURL,
			"thumbnail": post.Thumbnail,
		},
	})
}

func GetVideo(c echo.Context) error {
	videoId := c.Param("videoId")
	if videoId == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"status":  false,
			"message": "You need to use 'videoId' param.",
			"level":   "error",
		})
	}

	_, err := os.Stat(fmt.Sprintf("./tmp/%s.mp4", videoId))
	if err != nil {
		return c.JSON(http.StatusOK, echo.Map{
			"status":  false,
			"message": "Video not found.",
			"level":   "error",
		})
	}

	return c.Attachment(fmt.Sprintf("./tmp/%s.mp4", videoId), fmt.Sprintf("%s.mp4", videoId))
}

func RetrieveLatestVideos(c echo.Context) error {
	err, list := util.RetrieveLatestVideos()
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"status":  false,
			"message": err.Error(),
			"level":   "error",
		})
	}

	err, count := util.RetrieveCount()
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"status":  false,
			"message": err.Error(),
			"level":   "error",
		})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"status":          true,
		"videos":          list,
		"totalDownloaded": count,
	})
}
