package main

import (
	"main/routes"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

func main() {
	app := echo.New()

	//app.Use(middleware.Logger())
	app.Use(middleware.Recover())
	app.Use(middleware.CORS())

	app.POST("/api/download", routes.DownloadVideo)
	app.GET("/api/videos/:videoId", routes.GetVideo)
	app.GET("/api/retrieveLatest", routes.RetrieveLatestVideos)
	app.GET("/api/deleteVideo", routes.DeleteEntry)

	app.Logger.Fatal(app.Start(":1337"))
}
