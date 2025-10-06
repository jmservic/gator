package main

import (
	"github.com/jmservic/gator/internal/config"
	"fmt"
	"github.com/jmservic/gator/internal/database"
	"context"
	"time"
	"github.com/google/uuid"
	"github.com/jmservic/gator/internal/rss"
	"log"
	"database/sql"
	"github.com/lib/pq"
	"strconv"
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
	if len(cmd.args) < 1 {
		return fmt.Errorf("requires a time between requests")
	}
	timeBetweenReqs, err := time.ParseDuration(cmd.args[0])
	if err != nil {
		return fmt.Errorf("error parsing \"%v\": %w", cmd.args[0], err)
	}
	
	fmt.Printf("Collecting feeds every %s...\n", timeBetweenReqs)

	ticker := time.NewTicker(timeBetweenReqs)
	for ; ; <-ticker.C {
		scrapeFeeds(s)
	}
	/*
	rssFeed, err := rss.FetchFeed(context.Background(), "https://www.wagslane.dev/index.xml")
	if err != nil {
		return err
	}

	fmt.Println(*rssFeed)*/
	return nil
}

func scrapeFeeds(s *state) {
	feed, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		fmt.Printf("error getting the next feed: %v", err)
		return
	}

	err = s.db.MarkFeedFetched(context.Background(), feed.ID)
	if err != nil {
		fmt.Printf("error marking feed as fetched: %v", err)
		return
	}

	rssFeed, err := rss.FetchFeed(context.Background(), feed.Url)
	if err != nil {
		fmt.Printf("error fetching feed: %v", err)
		return
	}

	fmt.Printf("Posts for %v:\n", feed.Name)
	for _, post := range rssFeed.Channel.Item {
	//	fmt.Printf(" * %s\n", post.Title)
		currTime := time.Now()
		var title, description sql.NullString
		var publishedAt sql.NullTime
		if len(post.Title) > 0 {
			title = sql.NullString{
				String: post.Title,
				Valid: true,
			}
		} else {
			title = sql.NullString{
				String: "",
				Valid: false,
			}
		}

		if len(post.Description) > 0 {
			description = sql.NullString{
				String: post.Description,
				Valid: true,
			}
		} else {
			description = sql.NullString{
				String: "",
				Valid: false,
			}
		}
		
		if len(post.PubDate) > 0 {
			pubDatetime, err := parsePublicationDate(post.PubDate)
			if err != nil {
				log.Printf("error parsing publication date: %v", err)
				publishedAt = sql.NullTime{
					Valid: false,
				}
			} else {
				publishedAt = sql.NullTime{
					Time: pubDatetime,
					Valid: true,
				}
			}
		} else {
			publishedAt = sql.NullTime{
				Valid: false,
			}

		}

		params := database.CreatePostParams{
			ID: uuid.New(),
			CreatedAt: currTime,
			UpdatedAt: currTime,
			Title: title,
			Url: post.Link,
			Description: description,
			PublishedAt: publishedAt,
			FeedID: feed.ID,
		}
	//	fmt.Println(params)
		_, err := s.db.CreatePost(context.Background(), params)
		switch t := err.(type) {
		case *pq.Error:
			if t.Code != "23505" {
				log.Printf("error creating post: %v, %v", t, t.Code)
			}
		case nil:
		default:
			log.Printf("error creating post: %v", t)
		}
		/*
		if err != nil {
			log.Printf("error creating post: %v", err)
		}
		if err, ok := err.(*pq.Error); ok {
			fmt.Println("pq error:", err.Code)
		}*/
	}
	fmt.Println("")
}

func parsePublicationDate(pubDate string) (time.Time, error) {
	pubDatetime, err := time.Parse(time.Layout, pubDate)
	if err == nil {
		return pubDatetime, nil
	}
	pubDatetime, err = time.Parse(time.ANSIC, pubDate)
	if err == nil {
		return pubDatetime, nil
	}
	pubDatetime, err = time.Parse(time.UnixDate, pubDate)
	if err == nil {
		return pubDatetime, nil
	}
	pubDatetime, err = time.Parse(time.RubyDate, pubDate)
	if err == nil {
		return pubDatetime, nil
	}
	pubDatetime, err = time.Parse(time.RFC822, pubDate)
	if err == nil {
		return pubDatetime, nil
	}
	pubDatetime, err = time.Parse(time.RFC822Z, pubDate)
	if err == nil {
		return pubDatetime, nil
	}
	pubDatetime, err = time.Parse(time.RFC850, pubDate)
	if err == nil {
		return pubDatetime, nil
	}
	pubDatetime, err = time.Parse(time.RFC1123, pubDate)
	if err == nil {
		return pubDatetime, nil
	}
	pubDatetime, err = time.Parse(time.RFC1123Z, pubDate)
	if err == nil {
		return pubDatetime, nil
	}
	pubDatetime, err = time.Parse(time.RFC3339, pubDate)
	if err == nil {
		return pubDatetime, nil
	}
	pubDatetime, err = time.Parse(time.RFC3339Nano, pubDate)
	if err == nil {
		return pubDatetime, nil
	}

	return pubDatetime, nil
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

func handlerUnfollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 1 {
		return fmt.Errorf("requires a feed url.")
	}

	feed, err := s.db.GetFeedByURL(context.Background(), cmd.args[0])
	if err != nil {
		return fmt.Errorf("error getting feed information: %w", err)
	}

	deleteFeedFollowParams := database.DeleteFeedFollowParams{
		UserID: user.ID,
		FeedID: feed.ID,
	}
	err = s.db.DeleteFeedFollow(context.Background(), deleteFeedFollowParams)
	if err != nil {
		return fmt.Errorf("error unfollowing feed: %w", err)
	}
	fmt.Printf("%v unfollowed %v feed\n", user.Name, feed.Name)
	return nil
}

func handlerBrowse( s *state, cmd command, user database.User) error {
	limit := 2
	if len(cmd.args) > 0 {
		if userLimit, err := strconv.Atoi(cmd.args[0]); err == nil {
			limit = userLimit
		} else {
			return fmt.Errorf("invalid limit: %w", err)
		}
	}

	params := database.GetPostsForUserParams{
		UserID: user.ID,
		Limit: int32(limit),
	}
	
	posts, err := s.db.GetPostsForUser(context.Background(), params)
	if err != nil {
		return fmt.Errorf("error getting posts: %v", err)
	}

	for _, post := range posts {
		fmt.Printf("%s from %s\n", post.PublishedAt.Time.Format("Mon Jan 2"), post.FeedName)
		fmt.Printf("--- %s ---\n", post.Title.String)
		fmt.Printf("    %v\n", post.Description.String)
		fmt.Printf("Link: %s\n", post.Url)
		fmt.Println("======================================")
	}
	return nil
}
