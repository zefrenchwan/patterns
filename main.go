package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/zefrenchwan/patterns.git/serving"
	"github.com/zefrenchwan/patterns.git/storage"
	"go.uber.org/zap"
)

func main() {
	rawLogger, err := zap.NewProduction()
	if err != nil {
		panic(err.Error())
	}

	logger := rawLogger.Sugar()
	if logger == nil {
		panic("failed to start zap sugared logger")
	}
	defer rawLogger.Sync()

	dburl := os.Getenv("PATTERNS_DB_URL")
	if dburl == "" {
		panic("Error: no database set")
	}

	currentContext := context.Background()
	dao, errDao := storage.NewDao(currentContext, dburl)
	if errDao != nil {
		errorMessage := fmt.Sprintf("failed to build dao: %s", errDao.Error())
		panic(errorMessage)
	} else {
		defer dao.Close()
	}

	servingPort := os.Getenv("PATTERNS_PORT")
	if !strings.HasPrefix(servingPort, ":") {
		panic(fmt.Errorf("invalid port %s : it should be a : and a valid number", servingPort))
	}

	mux := serving.InitService(dao, currentContext, logger)
	http.ListenAndServe(servingPort, mux)
}
