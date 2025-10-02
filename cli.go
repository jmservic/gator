package main

import (
	"github.com/jmservic/gator/internal/config"
	"fmt"
	"github.com/jmservic/gator/internal/database"
	"context"
	"time"
	"github.com/google/uuid"
	"github.com/jmservic/gator/internal/rss"
)

type state struct {
	db *database.Queries
	cfg *config.Config
}

type command struct {
	name string
	args []string
}

type commands struct {
	cmds map[string]func(*state, command) error
}

func middlewareLoggedIn(handler func(*state, command, database.User) error) func(*state, command) error {
	return func (s *state, cmd command) error {
		user, err := s.db.GetUser(context.Background(), s.cfg.Current_user_name)

		if err != nil {
			return fmt.Errorf("error logging into user: %w", err)
		}

		return handler(s, cmd, user) 
	}
}

func (c *commands) run(s *state, cmd command) error {
	cmdFunc, ok := c.cmds[cmd.name]
	if ok != true {
		return fmt.Errorf("%s is not a recognized command.", cmd.name)
	}
	return cmdFunc(s, cmd)	
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.cmds[name] = f
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("A Username is required.")
	}
	
	user, err := s.db.GetUser(context.Background(), cmd.args[0])

	if err != nil {
		return fmt.Errorf("error logging into user: %w", err)
	}

	err = s.cfg.SetUser(user.Name)
	if err != nil {
		return fmt.Errorf("couldn't set current user: %w", err)
	}

	fmt.Printf("%s has been set as the current user.\n", user.Name)
	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("A Username is required.")
	}

	curTime := time.Now()
	createUserParams := database.CreateUserParams{
		ID: uuid.New(),
		CreatedAt: curTime,
		UpdatedAt: curTime,
		Name: cmd.args[0],
	}

	user, err := s.db.CreateUser(context.Background(), createUserParams)
	if err != nil {
		return fmt.Errorf("error creating user: %w", err)
	}

	err = s.cfg.SetUser(cmd.args[0])
	if err != nil {
		return fmt.Errorf("couldn't set current user: %w", err)
	}

	fmt.Printf("user %s was created\n", cmd.args[0])
	fmt.Println(user)
	return nil
}

func handlerReset(s *state, cmd command) error {
	if err := s.db.DeleteUsers(context.Background()); err != nil {
		return fmt.Errorf("couldn't empty the users table: %w", err)
	}
	fmt.Println("successfully cleared the user table")
	return nil
}

func handlerUsers(s *state, cmd command) error {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("couldn't get users: %w", err)
	}

	for _, user := range users {
		if user.Name == s.cfg.Current_user_name {
			fmt.Printf("* %s (current)\n", user.Name)
		} else {
			fmt.Printf("* %s\n", user.Name)
		}
	}

	return nil
}

func handlerAgg(s *state, cmd command) error {
	rssFeed, err := rss.FetchFeed(context.Background(), "https://www.wagslane.dev/index.xml")
	if err != nil {
		return err
	}

	fmt.Println(*rssFeed)
	return nil
}

func handlerAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 2 {
		return fmt.Errorf("not enough arguments passed into command")
	}

	currTime := time.Now()
	createFeedParams := database.CreateFeedParams{
		ID: uuid.New(),
		CreatedAt: currTime,
		UpdatedAt: currTime,
		Name: cmd.args[0],
		Url: cmd.args[1],
		UserID: user.ID,
	}

	feed, err := s.db.CreateFeed(context.Background(), createFeedParams)
	if err != nil {
		return fmt.Errorf("error creating feed: %w", err)
	}
	fmt.Println(feed)
	return handlerFollow(s, command{
		name: "follow",
		args: []string{feed.Url},
	}, user)
}

func handlerFeeds(s *state, cmd command) error {
	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		return fmt.Errorf("couldn't get feeds: %w", err)
	}
	
	for _, feed := range feeds {
		fmt.Printf("Feed: %s\n", feed.Name)
		fmt.Printf("  URL: %s\n", feed.Url)
		user, err := s.db.GetUserByID(context.Background(), feed.UserID)
		if err != nil {
			fmt.Printf("error getting user from ID: %v", err)
			continue
		}
		fmt.Printf("  Username: %s\n", user.Name)
	}
	return nil

}

func handlerFollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 1 {
		return fmt.Errorf("requires a feed url.")
	}

	feed, err := s.db.GetFeedByURL(context.Background(), cmd.args[0])
	if err != nil {
		return fmt.Errorf("error getting feed information: %w", err)
	}

	currTime := time.Now()
	createFeedFollowParams := database.CreateFeedFollowParams{
		ID: uuid.New(),
		CreatedAt: currTime,
		UpdatedAt: currTime,
		UserID: user.ID,
		FeedID: feed.ID,
	}

	rtnRow , err := s.db.CreateFeedFollow(context.Background(), createFeedFollowParams)
	if err != nil {
		return fmt.Errorf("error following the feed: %w", err)
	}
	fmt.Printf("%v followed \"%v\"\n", rtnRow.UserName, rtnRow.FeedName)
	
	return nil
}

func handlerFollowing(s *state, cmd command, user database.User) error {
	feeds, err := s.db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("error getting followed feeds: %w", err)
	}
	
	fmt.Println("Followed Feeds:", len(feeds))
	for _, feed := range feeds {
		fmt.Printf("  - %s\n", feed.FeedName)
	}

	return nil
}
