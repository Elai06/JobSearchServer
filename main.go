package main

import (
	"jobSearchServer/api"
	"jobSearchServer/internal/env"
	"log"
	"sync"
)

func main() {
	cfg, err := env.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	client := api.NewClient(cfg)
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	wg.Add(1)
	go func() {
		defer wg.Done()
		handler := api.NewHTTPHandler(*client, cfg)
		err := handler.StartServer()
		if err != nil {
			log.Fatal(err)
		}
	}()

}
