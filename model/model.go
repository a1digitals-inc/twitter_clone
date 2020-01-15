package model

import (
	"log"
	"fmt"
	"database/sql"
	_ "github.com/lib/pq"
)

type TweetRequest struct {
	UserId int64
	Text string
}

type Tweet struct {
	Id int64
	Username string
	Text string
	Date string
	Liked bool
	Retweeted bool
}

type User struct {
	Username string
	Password string
	CreatedAt string
	Id int64
}

var db *sql.DB

func InitDB() {
	connStr := "postgres://postgres:postgres@localhost:5432/twitter?sslmode=disable"
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Println("Could not connect to database.\n", err)
		return
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS users(
		id serial PRIMARY KEY,
		username VARCHAR (50) UNIQUE NOT NULL,
		password VARCHAR (50) NOT NULL,
		created_at timestamptz NOT NULL DEFAULT now()
		)`)
	if err != nil {
		log.Println("Could not create users table.\n", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS tweets(
		id serial PRIMARY KEY,
		text VARCHAR (140) NOT NULL,
		user_id integer REFERENCES users (id),
		created_at timestamptz NOT NULL DEFAULT now()
		)`)
	if err != nil {
		log.Println("Could not create tweets table.\n", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS follows(
		followed integer REFERENCES users ON DELETE CASCADE,
		follower integer REFERENCES users,
		created_at timestamptz NOT NULL DEFAULT now(),
		PRIMARY KEY (followed, follower)
		)`)
	if err != nil {
		log.Println("Could not create follows table.\n", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS retweets(
		tweet_id integer REFERENCES tweets ON DELETE CASCADE,
		user_id integer REFERENCES users ON DELETE CASCADE,
		created_at timestamp NOT NULL DEFAULT now()
		)`)
	if err != nil {
		log.Println("Could not create retweets table.\n", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS likes(
		tweet_id integer REFERENCES tweets ON DELETE CASCADE,
		user_id integer REFERENCES users ON DELETE CASCADE,
		created_at timestamp NOT NULL DEFAULT now()
		)`)
	if err != nil {
		log.Println("Could not create likes table.\n", err)
	}
}

func CreateUser(username, password string) (int64, error) {
	var id int64
	err := db.QueryRow(`INSERT INTO users (username, password) VALUES ($1, $2) RETURNING id`, username, password).Scan(&id)
	if err != nil {
		log.Println("Query Error: ", err)
		return 0, err
	}
	return id, nil
}

func GetUserFromUsername(username string) (User, error) {
	var password, createdAt string
	var id int64
	err := db.QueryRow(`SELECT id, password, created_at FROM users WHERE username = $1`, username).Scan(&id, &password, &createdAt)
	if err != nil {
		log.Println("Query Error: ", err)
		return User{}, err
	}
	user := User{
		Username: username,
		Password: password,
		CreatedAt: createdAt,
		Id: id,
	}
	return user, nil
}

func GetUserIdFromUsername(username string) (int64, error) {
	var id int64
	err := db.QueryRow(`SELECT id FROM users WHERE username = $1`, username).Scan(&id)
	if err != nil {
		log.Println("Query Error: ", err)
		return 0, err
	}
	return id, nil
}

func CreateTweet(request TweetRequest) (int64, error) {
	var id int64
	err := db.QueryRow(`INSERT INTO tweets (text, user_id) VALUES ($1, $2) RETURNING id`, request.Text, request.UserId).Scan(&id)
	if err != nil {
		log.Println("Query Error: ", err)
		return 0, err
	}
	return id, nil
}

func GetTweet(tweetId, userId int64) (Tweet, error) {
	var text, date, username string
	var liked, retweeted bool
	err := db.QueryRow(`SELECT t.text, t.created_at, u.username,
		(l.user_id IS NOT NULL) AS liked, 
		(r.user_id IS NOT NULL) AS retweeted
		FROM tweets t
		INNER JOIN users u
		ON t.user_id = u.id AND t.id = $1
		LEFT JOIN likes l
        ON l.user_id = $2 AND l.tweet_id = $1
        LEFT JOIN retweets r
        ON r.user_id = $2 AND r.tweet_id = $1`, tweetId, userId).Scan(&text, &date, &username, &liked, &retweeted)
	if err != nil {
		log.Println("Query Error: ", err)
		return Tweet{}, err
	}
	tweet := Tweet{
		Id: tweetId,
		Username: username,
		Text: text,
		Date: date,
		Liked: liked,
		Retweeted: retweeted,
	}
	return tweet, nil
}

func CreateFollow(followed, follower int64) (bool, error) {
	_, err := db.Exec(`INSERT INTO follows (followed, follower) VALUES($1, $2)`, followed, follower)
	if err != nil {
		log.Println("Query Error: ", err)
		return false, err
	}
	return true, nil
}

func CreateRetweet(userId, tweetId int64) (bool, error) {
	_, err := db.Exec(`INSERT INTO retweets (user_id, tweet_id) VALUES($1, $2)`, userId, tweetId)
	if err != nil {
		log.Println("Query Error: ", err)
		return false, err
	}
	return true, nil
}

func CreateLike(userId, tweetId int64) (bool, error) {
	_, err := db.Exec(`INSERT INTO likes (user_id, tweet_id) VALUES($1, $2)`, userId, tweetId)
	if err != nil {
		log.Println("Query Error: ", err)
		return false, err
	}
	return true, nil
}

func GetFeed(userId int64) ([]Tweet, error) {
	result, err := db.Query(`SELECT t.id, t.text, t.created_at, u.username, 
		(l.user_id IS NOT NULL) AS liked, 
		(r.user_id IS NOT NULL) AS retweeted
		FROM tweets t
		INNER JOIN follows f 
		ON t.user_id = f.followed AND f.follower = $1
		INNER JOIN users u
		ON u.id = t.user_id
        FULL JOIN likes l
		ON l.tweet_id = t.id AND l.user_id = $1
		FULL JOIN retweets r
        ON r.tweet_id = t.id AND r.user_id = $1
		ORDER BY t.created_at DESC;`, userId)
	if err != nil {
		return nil, err
	}
	fmt.Println(result)
	defer result.Close()

	var tweets []Tweet
	for result.Next() {
		var id int64
		var text string
		var createdAt string
		var username string
		var liked bool
		var retweeted bool
		err := result.Scan(&id, &text, &createdAt, &username, &liked, &retweeted)
		if err != nil {
			log.Println("Scanning error: ", err)
			break
		}
		tweet := Tweet{
			Id: id,
			Text: text,
			Username: username,
			Date: createdAt,
			Liked: liked,
			Retweeted: retweeted,
		}
		tweets = append(tweets, tweet)
	}
	return tweets, nil
}

func GetHistory(userId int64) ([]Tweet, error) {
	result, err := db.Query(`SELECT t.id, t.text, r.created_at, u.username
		FROM tweets t
		INNER JOIN retweets r
		ON ((r.user_id = $1 AND r.tweet_id = t.id)
		OR (t.user_id = $1))
		INNER JOIN users u
		ON u.id = t.user_id`, userId)
		if err != nil {
		return nil, err
	}
	fmt.Println(result)
	defer result.Close()

	var tweets []Tweet
	for result.Next() {
		var id int64
		var text string
		var createdAt string
		var username string
		err := result.Scan(&id, &text, &createdAt, &username)
		if err != nil {
			log.Println("Scanning error: ", err)
			break
		}
		tweet := Tweet{
			Id: id,
			Text: text,
			Username: username,
			Date: createdAt,
		}
		tweets = append(tweets, tweet)
	}
	return tweets, nil
}