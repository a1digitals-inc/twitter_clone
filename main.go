package main

import (
	"net/http"
	"html/template"
	"log"
	"fmt"
	"database/sql"
	mux "github.com/gorilla/mux"
	sessions "github.com/gorilla/sessions"
	_ "github.com/lib/pq"
)

type Tweet struct {
	Id int64
	Username string
	Text string
	Date string
}

type TweetRequest struct {
	UserId int64
	Text string
}

type LoginCreds struct {
	Username string
	Password string
}

type User struct {
	Username string
	Password string
	CreatedAt string
	Id int64
}

type LoginPage struct {
	PasswordFail bool
}

type IndexPage struct {
	Tweets []Tweet
	Username string
	UserId int64
}

type UserPage struct {
	Username string
	Tweets []Tweet
}

const (
	LOGIN_COOKIE_NAME = "login"
)

var db *sql.DB
var store = sessions.NewCookieStore([]byte("SECRET"))

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// Check if user is authenticated
		session, _ := store.Get(r, LOGIN_COOKIE_NAME)
		_, ok := session.Values["uid"]
		if ok == true {
			http.Redirect(w, r, "/", http.StatusMovedPermanently)
			return
		}

		t, _ := template.ParseFiles("login.html")
		t.Execute(w, nil)
	} else {
		login := LoginCreds{
			Username: r.FormValue("username"),
			Password: r.FormValue("password"),
		}
		fmt.Println(login)
		user, err := getUserFromUsername(login.Username)
		var uid int64
		if err != nil {
			// New user
			uid, err = createUser(login.Username, login.Password)
		} else {
			// Existing user
			// Check password
			if login.Password != user.Password {
				data := LoginPage{
					PasswordFail: true,
				}
				t, _ := template.ParseFiles("login.html")
				t.Execute(w, data)
			} else {
				uid = user.Id
			}
		}

		log.Println("User id: ", uid, "user.Id: ", user.Id)

		session, err := store.Get(r, LOGIN_COOKIE_NAME)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		session.Values["uid"] = uid
		session.Values["username"] = login.Username
		session.Options = &sessions.Options{
			Path:     "/",
			MaxAge:   600,
			HttpOnly: true,
		}
		
		err = session.Save(r, w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusMovedPermanently)
		return
	}
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, LOGIN_COOKIE_NAME)
	session.Values["uid"] = 0
	session.Values["username"] = ""
	session.Options.MaxAge = -1
	err := session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/login", http.StatusFound)
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, LOGIN_COOKIE_NAME)

	// Check if user is authenticated
	uid, ok := session.Values["uid"]
    if ok == false {
        http.Redirect(w, r, "/login", http.StatusMovedPermanently)
        return
	}
	username, _ := session.Values["username"]
	
	tweets, err := getFeed(uid.(int64))
	if err != nil {
		log.Println("Could not get feed.\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := IndexPage{
		Tweets: tweets,
		Username: username.(string),
		UserId: uid.(int64),
	}
	fmt.Println(data)

	t, _ := template.ParseFiles("index.html")
	t.Execute(w, data)
}

func TweetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		session, _ := store.Get(r, LOGIN_COOKIE_NAME)

		// Check if user is authenticated
		uid, ok := session.Values["uid"]
		if ok == false {
			http.Redirect(w, r, "/login", http.StatusMovedPermanently)
			return
		}
		fmt.Println("Tweet: ", r.FormValue("tweet"))

		tweet := TweetRequest{
			UserId: uid.(int64),
			Text: r.FormValue("tweet"),
		}

		_, err := createTweet(tweet)
		if err != nil {
			log.Println("Could not create tweet.\n", err)
			return
		}

		http.Redirect(w, r, "/", http.StatusMovedPermanently)
		return
	}
}

func UserHandler(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	uid, err := getUserIdFromUsername(username)
	if err != nil {
		log.Println("Could not get user ID.\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tweets, err := getHistory(uid)
	if err != nil {
		log.Println("Could not get tweets.\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := UserPage{
		Username: username,
		Tweets: tweets,
	}
	t, _ := template.ParseFiles("user.html")
	t.Execute(w, data)
}

func FollowHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, LOGIN_COOKIE_NAME)

	// Check if user is authenticated
	follower, ok := session.Values["uid"]
	if ok == false {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	username := r.FormValue("username")
	followed, err := getUserIdFromUsername(username)
	if err != nil {
		log.Println("Could not get user ID.\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Println("Follower: ", follower, "; Followed: ", followed)
	_, err = createFollow(followed, follower.(int64))
	if err != nil {
		log.Println("Could not follow user.\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/%s", username), http.StatusMovedPermanently)
	return
}

func main() {
	initDB()

	r := mux.NewRouter()
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	r.HandleFunc("/login", LoginHandler)
	r.HandleFunc("/logout", LogoutHandler)
	r.HandleFunc("/tweet", TweetHandler)
	r.HandleFunc("/follow", FollowHandler)
	r.HandleFunc("/{username}", UserHandler)
	r.HandleFunc("/", IndexHandler)
	log.Fatal(http.ListenAndServe(":8000", r))
}

func initDB() {
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
}

func createUser(username, password string) (int64, error) {
	var id int64
	err := db.QueryRow(`INSERT INTO users (username, password) VALUES ($1, $2) RETURNING id`, username, password).Scan(&id)
	if err != nil {
		log.Println("Query Error: ", err)
		return 0, err
	}
	return id, nil
}

func getUserFromUsername(username string) (User, error) {
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

func getUserIdFromUsername(username string) (int64, error) {
	var id int64
	err := db.QueryRow(`SELECT id FROM users WHERE username = $1`, username).Scan(&id)
	if err != nil {
		log.Println("Query Error: ", err)
		return 0, err
	}
	return id, nil
}

func createTweet(request TweetRequest) (int64, error) {
	var id int64
	err := db.QueryRow(`INSERT INTO tweets (text, user_id) VALUES ($1, $2) RETURNING id`, request.Text, request.UserId).Scan(&id)
	if err != nil {
		log.Println("Query Error: ", err)
		return 0, err
	}
	return id, nil
}

func createFollow(followed, follower int64) (bool, error) {
	_, err := db.Exec(`INSERT INTO follows (followed, follower) VALUES($1, $2)`, followed, follower)
	if err != nil {
		log.Println("Query Error: ", err)
		return false, err
	}
	return true, nil
}

func getFeed(userId int64) ([]Tweet, error) {
	result, err := db.Query(`SELECT t.id, t.text, t.created_at, u.username 
		FROM tweets t
		INNER JOIN follows f 
		ON t.user_id = f.followed AND f.follower = $1
		INNER JOIN users u
		ON u.id = t.user_id
		ORDER BY created_at DESC`, userId)
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

func getHistory(userId int64) ([]Tweet, error) {
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