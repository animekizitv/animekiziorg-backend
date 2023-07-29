package util

import (
	"context"
	"main/db"
)

var database = db.NewClient()
var ctx = context.Background()

func init() {
	database.Connect()
}
