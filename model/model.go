package model

import (
	"log"
	"fmt"
    "os"
	"database/sql"
	_ "github.com/lib/pq"
)

type TweetRequest struct {
	UserId int64
	Text string
	ImageURL string
	ParentId int64
}

type Tweet struct {
	Id int64
	Username string
	Text string
	ImageURL string
	Date string
	Liked bool
	Retweeted bool
	DisplayName string
}

type User struct {
	Username string
	Password string
	CreatedAt string
	Id int64
	DisplayName string
	Bio string
	Website string
	Location string
}

type CrossUsers struct {
	Followers int64
	Follows int64
	SecondFollowsFirst bool
}

type Message struct {
	Id int64
	SenderId int64
	SenderUsername string
	SenderDisplayName string
	Text string
	CreatedAt string
}

type MessageRequest struct {
	SenderId int64
	Text string
	ConversationId int64
}

type Conversation struct {
	Id int64
	Name string
	Text string
	OtherUserDisplayName string
	OtherUserName string
	MostRecentDate string
}

type Notification struct {
	TweetId int64
	Text string
	Username string
	Retweeted bool
	Liked bool
	DisplayName string
}

var db *sql.DB

func nullStringToString(nullString sql.NullString) string {
	var maybeString string
	if nullString.Valid {
		maybeString = nullString.String
	}
	return maybeString
}

func nullInt64ToInt64(nullInt sql.NullInt64) int64 {
	var maybeInt int64
	if nullInt.Valid {
		maybeInt = nullInt.Int64
	}
	return maybeInt
}

func InitDB() {
    postgresUsername := os.Getenv("POSTGRES_USER")
	postgresPassword := os.Getenv("POSTGRES_PASSWORD")
	
	var connStr string
	if postgresUsername == "" {
		connStr = "postgres://postgres@db_postgres:5432/twitter?sslmode=disable"
	} else {
		connStr = fmt.Sprintf("postgres://%s:%s@postgres:5432/twitter?sslmode=disable", postgresUsername, postgresPassword)
	}
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
		created_at timestamptz NOT NULL DEFAULT now(),
		display_name VARCHAR(50),
		bio VARCHAR(160),
		location VARCHAR(30),
		website VARCHAR(100)
		)`)
	if err != nil {
		log.Println("Could not create users table.\n", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS tweets(
		id serial PRIMARY KEY,
		text VARCHAR (140) NOT NULL,
		image_url TEXT,
		user_id integer REFERENCES users (id),
		parent_id integer REFERENCES tweets,
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
		created_at timestamptz NOT NULL DEFAULT now()
		)`)
	if err != nil {
		log.Println("Could not create retweets table.\n", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS likes(
		tweet_id integer REFERENCES tweets ON DELETE CASCADE,
		user_id integer REFERENCES users ON DELETE CASCADE,
		created_at timestamptz NOT NULL DEFAULT now()
		)`)
	if err != nil {
		log.Println("Could not create likes table.\n", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS conversations(
		id serial PRIMARY KEY,
		name VARCHAR(30),
		created_at timestamptz NOT NULL DEFAULT now()
		)`)
	if err != nil {
		log.Println("Could not create conversations table.\n", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS conversations_users(
		conversation_id integer REFERENCES conversations ON DELETE CASCADE,
		user_id integer REFERENCES users ON DELETE CASCADE,
		created_at timestamptz NOT NULL DEFAULT now()
		)`)
	if err != nil {
		log.Println("Could not create conversations_users table.\n", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS messages(
		id serial PRIMARY KEY,
		text TEXT,
		conversation_id integer REFERENCES conversations ON DELETE CASCADE,
		sender_id integer REFERENCES users ON DELETE CASCADE,
		created_at timestamptz NOT NULL DEFAULT now()
		)`)
	if err != nil {
		log.Println("Could not create messages table.\n", err)
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
	var displayName, bio, website, location sql.NullString
	var id int64
	err := db.QueryRow(`SELECT 
		id, password, created_at, display_name, bio, website, location 
		FROM users WHERE username = $1`, username).Scan(&id, &password, &createdAt, &displayName, &bio, &website, &location)
	if err != nil {
		log.Println("Query Error: ", err)
		return User{}, err
	}

	user := User{
		Username: username,
		Password: password,
		CreatedAt: createdAt,
		Id: id,
		DisplayName: nullStringToString(displayName),
		Bio: nullStringToString(bio),
		Website: nullStringToString(website),
		Location: nullStringToString(location),
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

func GetUsersRelationship(userId, currentUserId int64) (CrossUsers, error) {
	var followers, follows int64
	var secondFollowsFirst bool
	err := db.QueryRow(`SELECT 
		COUNT(*) FILTER (WHERE f.followed = $1) as followers,
		COUNT(*) FILTER (WHERE f.follower = $1) as follows,
		COUNT(*) FILTER (WHERE f.follower = $2 AND f.followed = $1) = 1 as dnf
		FROM follows f
		WHERE $1 IN (f.followed, f.follower)`, userId, currentUserId).Scan(&followers, &follows, &secondFollowsFirst)
	if err != nil {
		log.Println("Query Error: ", err)
		return CrossUsers{}, err
	}

	crossUsers := CrossUsers{
		Followers: followers,
		Follows: follows,
		SecondFollowsFirst: secondFollowsFirst,
	}
	return crossUsers, nil
}

func GetTwoUsersConversation(userId, currentUserId int64) (int64, error) {
	var id sql.NullInt64
	err := db.QueryRow(`SELECT cu.conversation_id
		FROM conversations_users cu
		INNER JOIN conversations_users cus
		ON cus.conversation_id = cu.conversation_id AND cus.user_id = $2
		WHERE cu.user_id = $1 AND NOT EXISTS (
		SELECT conversation_id
		FROM conversations_users
		WHERE conversation_id = cus.conversation_id AND user_id != $1 AND user_id != $2
		)`, userId, currentUserId).Scan(&id)
	if err != nil && err != sql.ErrNoRows {
		log.Println("Query Error: ", err)
		return 0, err
	}
	return nullInt64ToInt64(id), nil
}

func CreateTwoUsersConversation(userId, currentUserId int64) (int64, error) {
	var conversationId int64
	err := db.QueryRow(`INSERT INTO conversations DEFAULT VALUES RETURNING id`).Scan(&conversationId)
	if err != nil {
		log.Println("Query Error: ", err)
		return 0, err
	}

	_, err = db.Exec(`INSERT INTO 
		conversations_users(conversation_id, user_id) VALUES($1, $2)`,
		conversationId, userId)
	if err != nil {
		log.Println("Query Error: ", err)
		return 0, err
	}

	_, err = db.Exec(`INSERT INTO 
		conversations_users(conversation_id, user_id) VALUES($1, $2)`,
		conversationId, currentUserId)
	if err != nil {
		log.Println("Query Error: ", err)
		return 0, err
	}
	return conversationId, nil
}

func SmartCreateUser(request MessageRequest) (int64, error) {
	var id int64
	err := db.QueryRow(`INSERT INTO messages(sender_id, text, conversation_id)
		SELECT $1, $2, $3
		WHERE EXISTS (
			SELECT c.conversation_id
			FROM conversations_users c
			WHERE c.user_id = $1 AND c.conversation_id = $3
		) RETURNING id`, request.SenderId, request.Text, request.ConversationId).Scan(&id)
	if err != nil {
		log.Println("Query Error: ", err)
		return 0, err
	}
	return id, nil
}

func EditUser(edits User) error {
	result, err := db.Exec(`UPDATE users
		SET display_name = $1, bio = $2, location = $3, website = $4 WHERE id = $5`,
	edits.DisplayName, edits.Bio, edits.Location, edits.Website, edits.Id)
	if err != nil {
		log.Println("Query Error: ", err)
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		log.Println("Query Error: ", err)
		return err
	}
	if rows != 1 {
		log.Fatalf("expected to affect 1 row, affected %d", rows)
		log.Println("Query Error: ", err)
		return err
	}
	return nil
}

func CreateTweet(request TweetRequest) (int64, error) {
	var id int64
	var err error
	if request.ParentId != 0 {
		if request.ImageURL != "" {
			err = db.QueryRow(`INSERT INTO tweets (text, user_id, image_url, parent_id) 
				VALUES ($1, $2, $3, $4) RETURNING id`, 
				request.Text, request.UserId, request.ImageURL, request.ParentId).Scan(&id)
		} else {
			err = db.QueryRow(`INSERT INTO tweets (text, user_id, parent_id) 
				VALUES ($1, $2, $3) RETURNING id`, 
				request.Text, request.UserId, request.ParentId).Scan(&id)
		}
	} else {
		if request.ImageURL != "" {
			err = db.QueryRow(`INSERT INTO tweets (text, user_id, image_url) 
				VALUES ($1, $2, $3) RETURNING id`, 
				request.Text, request.UserId, request.ImageURL).Scan(&id)
		} else {
			err = db.QueryRow(`INSERT INTO tweets (text, user_id) 
				VALUES ($1, $2) RETURNING id`, 
				request.Text, request.UserId).Scan(&id)
		}
	}
	if err != nil {
		log.Println("Query Error: ", err)
		return 0, err
	}
	return id, nil
}

func GetTweet(tweetId, userId int64) (Tweet, error) {
	var text, date, username string
	var imageURL, displayName sql.NullString
	var liked, retweeted bool
	err := db.QueryRow(`SELECT t.text, t.created_at, t.image_url, u.username,
		(l.user_id IS NOT NULL) AS liked, 
		(r.user_id IS NOT NULL) AS retweeted,
		u.display_name
		FROM tweets t
		INNER JOIN users u
		ON t.user_id = u.id AND t.id = $1
		LEFT JOIN likes l
        ON l.user_id = $2 AND l.tweet_id = $1
        LEFT JOIN retweets r
		ON r.user_id = $2 AND r.tweet_id = $1`, 
	tweetId, userId).Scan(&text, &date, &imageURL, &username, &liked, &retweeted, &displayName)
	if err != nil {
		log.Println("Query Error: ", err)
		return Tweet{}, err
	}
	tweet := Tweet{
		Id: tweetId,
		Username: username,
		Text: text,
		ImageURL: nullStringToString(imageURL),
		Date: date,
		Liked: liked,
		Retweeted: retweeted,
		DisplayName: nullStringToString(displayName),
	}
	return tweet, nil
}

func GetReplies(tweetId, userId int64) ([]Tweet, error) {
	result, err := db.Query(`SELECT t.id, t.text, t.image_url, 
		t.created_at, u.username, u.display_name,
		(l.user_id IS NOT NULL) as user_liked,
		(r.user_id IS NOT NULL) as user_retweeted
		FROM tweets t
		INNER JOIN users u
		ON u.id = t.user_id
		LEFT JOIN likes l
		ON l.user_id = $2 AND l.tweet_id = t.id
		LEFT JOIN retweets r
		ON r.user_id = $2 AND r.tweet_id = t.id
		WHERE t.parent_id = $1`, tweetId, userId)
	if err != nil {
		log.Println("Query Error: ", err)
		return nil, err
	}
	defer result.Close()
	
	var replies []Tweet
	for result.Next() {
		var id int64
		var text, imageURL, createdAt, username string
		var displayName sql.NullString
		var liked, retweeted bool
		err := result.Scan(&id, &text, &imageURL, &createdAt, &username, &displayName, &liked, &retweeted)
		if err != nil {
			log.Println("Scanning error: ", err)
			break
		}
		reply := Tweet{
			Id: id,
			Text: text,
			ImageURL: imageURL,
			Username: username,
			DisplayName: nullStringToString(displayName),
			Date: createdAt,
			Liked: liked,
			Retweeted: retweeted,
		}
		replies = append(replies, reply)
	}
	return replies, nil
}

func GetConversation(conversationId int64) ([]Message, error) {
	result, err := db.Query(`SELECT m.id, m.text, m.created_at,
		u.id, u.username, u.display_name
		FROM messages m
		LEFT JOIN users u
		ON u.id = m.sender_id
		WHERE m.conversation_id = $1
		ORDER BY m.created_at ASC`, conversationId)
	if err != nil {
		log.Println("Query error: ", err)
		return nil, err
	}
	defer result.Close()

	var messages []Message
	for result.Next() {
		var id, senderId int64
		var text, senderUsername, createdAt string
		var senderDisplayName sql.NullString
		err = result.Scan(&id, &text, &createdAt, &senderId, &senderUsername, &senderDisplayName)
		if err != nil {
			log.Println("Scanning Error: ", err)
			break
		}
		message := Message{
			Id: id,
			Text: text,
			CreatedAt: createdAt,
			SenderId: senderId,
			SenderUsername: senderUsername,
			SenderDisplayName: nullStringToString(senderDisplayName),
		}
		messages = append(messages, message)
	}
	return messages, nil
}

func GetConversations(userId int64) ([]Conversation, error) {
	result, err := db.Query(`SELECT DISTINCT ON (m.conversation_id)
		m.conversation_id, m.text, m.created_at,
		c.name, u.username, u.display_name
		FROM messages m
		LEFT JOIN conversations c
		ON c.id = m.conversation_id
		LEFT JOIN users u
		ON u.id = m.sender_id
		WHERE m.conversation_id IN (
			SELECT c.id
			FROM conversations_users cu
			INNER JOIN conversations c
			ON c.id = cu.conversation_id
			WHERE cu.user_id = $1
		)
		ORDER BY m.conversation_id, m.created_at DESC`, userId)
	if err != nil {
		log.Println("Query Error: ", err)
		return nil, err
	}
	defer result.Close()
	
	var conversations []Conversation
	for result.Next() {
		var id int64
		var text, createdAt, otherUsername string
		var name, otherUserDisplayName sql.NullString
		err := result.Scan(&id, &text, &createdAt, &name, &otherUsername, &otherUserDisplayName)
		if err != nil {
			log.Println("Scanning error: ", err)
			break
		}
		conversation := Conversation{
			Id: id,
			Text: text,
			Name: nullStringToString(name),
			OtherUserName: otherUsername,
			OtherUserDisplayName: nullStringToString(otherUserDisplayName),
			MostRecentDate: createdAt,
		}
		
		conversations = append(conversations, conversation)
	}
	return conversations, nil
}

func GetNotifications(userId int64) ([]Notification, error) {
	result, err := db.Query(`SELECT t.id, t.text,
		r.user_id IS NOT NULL as retweeted, 
		l.user_id IS NOT NULL as liked,
		u.username, u.display_name
		FROM tweets t
		LEFT JOIN retweets r
		ON r.tweet_id = t.id AND r.user_id != $1
		LEFT JOIN likes l
		ON l.tweet_id = t.id AND l.user_id != $1
		INNER JOIN users u
		ON u.id = l.user_id OR u.id = r.user_id
		WHERE t.user_id = $1
		ORDER BY l.created_at DESC, r.created_at DESC`, userId)
	if err != nil && err != sql.ErrNoRows {
		log.Println("Query Error: ", err)
		return nil, err
	}
	defer result.Close()

	var notifications []Notification
	for result.Next() {
		var id int64
		var text, username string
		var retweeted, liked bool
		var displayName sql.NullString

		err := result.Scan(&id, &text, &retweeted, &liked, &username, &displayName)
		if err != nil {
			log.Println("Scanning error: ", err)
			break
		}

		notification := Notification{
			TweetId: id,
			Text: text,
			Username: username,
			Retweeted: retweeted,
			Liked: liked,
			DisplayName: nullStringToString(displayName),
		}

		notifications = append(notifications, notification)
	}
	return notifications, nil
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
		WHERE t.id IS NOT NULL
		ORDER BY t.created_at DESC`, userId)
	if err != nil {
		return nil, err
	}
	fmt.Println(result)
	defer result.Close()

	var tweets []Tweet
	for result.Next() {
		var id int64
		var text, createdAt, username string
		var liked, retweeted bool
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

func GetHistory(userId, currentUserId int64) ([]Tweet, error) {
	result, err := db.Query(`SELECT t.id, t.text, u.username, t.created_at,
		(l.tweet_id IS NOT NULL) AS liked,
		(e.tweet_id IS NOT NULL) AS retweeted
		FROM tweets t
		LEFT JOIN retweets r
		ON r.tweet_id = t.id
		LEFT JOIN users u 
		ON t.user_id = u.id 
		LEFT JOIN likes l
		ON l.tweet_id = t.id AND l.user_id = $2
		LEFT JOIN retweets e
		ON e.user_id = $2 AND e.tweet_id = t.id
		WHERE r.user_id = $1 OR t.user_id = $1 
		ORDER BY t.created_at DESC`, userId, currentUserId)
		if err != nil {
		return nil, err
	}
	fmt.Println(result)
	defer result.Close()

	var tweets []Tweet
	for result.Next() {
		var id int64
		var text, username, createdAt string
		var liked, retweeted bool
		err := result.Scan(&id, &text, &username, &createdAt, &liked, &retweeted)
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

func GetLikes(userId, currentUserId int64) ([]Tweet, error) {
	result, err := db.Query(`SELECT t.id, t.text, u.username, t.created_at,
		(l.tweet_id IS NOT NULL) AS liked,
		(e.tweet_id IS NOT NULL) AS retweeted
		FROM likes k
			LEFT JOIN tweets t
			ON k.tweet_id = t.id
			LEFT JOIN users u 
			ON t.user_id = u.id 
			LEFT JOIN likes l
			ON l.tweet_id = t.id AND l.user_id = $2
			LEFT JOIN retweets e
			ON e.user_id = $2 AND e.tweet_id = t.id
		WHERE k.user_id = $1
		ORDER BY t.created_at DESC`, userId, currentUserId)
		if err != nil {
		return nil, err
	}
	fmt.Println(result)
	defer result.Close()

	var tweets []Tweet
	for result.Next() {
		var id int64
		var text, username, createdAt string
		var liked, retweeted bool
		err := result.Scan(&id, &text, &username, &createdAt, &liked, &retweeted)
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
