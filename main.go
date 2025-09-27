package main

import (
	"github.com/jmservic/gator/internal/config"
	"fmt"
	"log"
)

func main(){ 
	cfg, err := config.Read()
	if err != nil {
		log.Fatalf("error reading config: %v", err)	
	}
	//fmt.Println(cfg)
	err = cfg.SetUser("jonathan")
	if err != nil {
		log.Fatalf("couldn't set current user: %v", err)
	}
	cfg, err = config.Read()
	if err != nil {
		log.Fatalf("error reading config: %v", err)	
	}
	fmt.Println(cfg)
}
