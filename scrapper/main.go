package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/jasonlvhit/gocron"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	twitterscraper "github.com/n0madic/twitter-scraper"
)

func write_in_db(db *sql.DB, tweet_text string) {
	rows, err := db.Query("SELECT * FROM public.notifications ORDER BY id ASC")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var is_gov_active bool
		var is_twitter_subscribe_active bool
		var gov_time_subscription string
		var user_id int

		err := rows.Scan(&id, &is_gov_active, &is_twitter_subscribe_active, &gov_time_subscription, &user_id)
		if err != nil {
			panic(err)
		}

		_, err = db.Exec("INSERT INTO public.feed_item (text, user_id, is_showed) VALUES ($1, $2, $3)", tweet_text, user_id, false)
		if err != nil {
			panic(err)
		}
	}

	if err := rows.Err(); err != nil {
		panic(err)
	}

	err = db.Close()
	if err != nil {
		panic(err)
	}
}

func scrapper() {
	scraper := twitterscraper.New()
	tweet_text := ""

	for tweet := range scraper.GetTweets(context.Background(), "SeiNetwork", 1) {
		if tweet.Error != nil {
			panic(tweet.Error)
		}

		tweet_text = tweet.Text
	}

	envFile, err := godotenv.Read("../.env")

	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	databaseUrl := envFile["DATABASE_URL"]
	fmt.Println(databaseUrl)

	db, err := sql.Open("postgres", databaseUrl)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	var id int
	var text string

	err = db.QueryRow("SELECT id, text FROM public.feed_item ORDER BY id DESC LIMIT 1").Scan(&id, &text)

	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("no rows returned by the query")
			write_in_db(db, tweet_text)

			return
		} else {
			panic(err)
		}
	}

	if text == tweet_text {
		return
	}

	write_in_db(db, tweet_text)
}

func main() {
	gocron.Every(10).Minutes().Do(scrapper)

	<-gocron.Start()
}
