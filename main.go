package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/zefrenchwan/patterns.git/serving"
	"github.com/zefrenchwan/patterns.git/storage"
)

func main() {
	dburl := os.Getenv("PATTERNS_DB_URL")
	if dburl == "" {
		panic("Error: no database set")
	}

	dao, errDao := storage.NewDao(context.Background(), dburl)
	if errDao != nil {
		errorMessage := fmt.Sprintf("failed to build dao: %s", errDao.Error())
		panic(errorMessage)
	} else {
		defer dao.Close()
	}

	servingPort := os.Getenv("PATTERNS_PORT")
	mux := http.NewServeMux()
	serving.InitService(mux, dao)

	http.ListenAndServe(servingPort, mux)
}
