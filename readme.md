# gator
# Description
Gator is a project on the Boot.dev course that reads RSS feeds and displays the feed, date, title, and link to the user. It uses Postgres to store the user, feeds, posts, etc.
# Setup
You will need to have Postgres and Go installed the run the program.
1. Run `go install https://github.com/jmservic/gator` to install the program.
2. Create a .gatorconfig.json in your home directory with the following information:
   ```json
   {
     "db_url": "postgres://{postgresUser}:{postgresPassword}@localhost:5432/gator?sslmode=disable"
   }
   ```
3. In your postgres instance, create a new database named gator.
4. If you haven't installed goose in your go environment, run `go install github.com/pressly/gloose/v3/cmd/goose@latest`.
5. Using the the migrations in sql/schema in this repo, switch the sql/schema folder and run `goose postgres://{postgresUser}:{postgresPassword}@localhost:5432/gator up` to the last migration (5).
# Usage
## login {username}
* Switches the current user to {username}.
## register {username}
* Registers a new user and makes the new user the current user.
## reset
* Clears the postgres database.
## users
* Prints the users in the gator database.
## agg {time_between_requests}
* Aggregates posts from stored feeds. Requires a time between request i.e. 1s, 1m, 1h.
## addfeed {Name} {URL}
* Adds a new RSS feed, requires a name and URL.
## follow {URL}
* Makes the current user follow the RSS feed specified by URL.
## following
* Shows what RSS feeds the current user is following.
## unfollow {URL}
* Unfollows the RSS feed specified by URL for the current user.
## browse {Limit}
* Shows the most recent aggregated RSS feeds. Limits the output to the number specified by the Limit argument.
