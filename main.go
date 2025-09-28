package main

import (
	"github.com/jmservic/gator/internal/config"
	//"fmt"
	"log"
	"os"
	_ "github.com/lib/pq"
	"github.com/jmservic/gator/internal/database"
	"database/sql"
)

func main(){ 
	args := os.Args
	if len(args) < 2 {
		log.Fatalf("no command specified")
	}

	cfg, err := config.Read()
	if err != nil {
		log.Fatalf("error reading config: %v", err)	
	}

	db, err := sql.Open("postgres", cfg.Db_url)
	if err != nil {
		log.Fatalf("error opening the postgres databse: %v", err)
	}

	dbQueries := database.New(db)

	s := state{
		db: dbQueries,
		cfg: &cfg,	
	}

	cmds := commands{
		cmds: map[string]func(*state, command) error{},
	}

	cmds.register("login", handlerLogin)
	cmds.register("register", handlerRegister)
	cmds.register("reset", handlerReset)
	cmds.register("users", handlerUsers)
	cmds.register("agg", handlerAgg)

	//fmt.Println(s.cfg)
	cmd := command{
		name: args[1],
		args: args[2:],
	}
	err = cmds.run(&s, cmd)
	if err != nil {
		log.Fatalf("Problem running %s command: %v", cmd.name, err)
	}
	/*err = cfg.SetUser("jonathan")
	if err != nil {
		log.Fatalf("couldn't set current user: %v", err)
	}
	cfg, err = config.Read()
	if err != nil {
		log.Fatalf("error reading config: %v", err)	
	}
	fmt.Println(cfg)*/
}
