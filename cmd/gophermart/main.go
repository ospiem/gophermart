package main

import (
	"fmt"
	"log"

	"github.com/ospiem/gophermart/internal/config"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		log.Fatal("cannot get config")
	}
	fmt.Println(cfg)
}
