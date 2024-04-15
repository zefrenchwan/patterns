package main

import (
	"context"
	"fmt"
	"os"

	"github.com/zefrenchwan/patterns.git/patterns"
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

	init := patterns.NewEntity([]string{"Person"})
	init.SetValue("last name", "Meee")
	init.SetValue("first name", "Me")

	if err := dao.UpsertEntity(context.Background(), init); err != nil {
		panic(err)
	}
}
