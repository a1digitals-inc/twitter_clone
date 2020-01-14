package main

import (
	"net/http"
	"html/template"
	"log"
	"fmt"
	"strconv"
	mux "github.com/gorilla/mux"
	sessions "github.com/gorilla/sessions"
	model "github.com/dustinnewman98/twitter_clone/model"
)

type LoginCreds struct {
	Username string
	Password string
}

type LoginPage struct {
	PasswordFail bool
}

type IndexPage struct {
	Tweets []model.Tweet
	Username string
	UserId int64
}

type UserPage struct {
	Username string
	Tweets []model.Tweet
}

const (
	LOGIN_COOKIE_NAME = "login"
)

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
		user, err := model.GetUserFromUsername(login.Username)
		var uid int64
		if err != nil {
			// New user
			uid, err = model.CreateUser(login.Username, login.Password)
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
	
	tweets, err := model.GetFeed(uid.(int64))
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

		tweet := model.TweetRequest{
			UserId: uid.(int64),
			Text: r.FormValue("tweet"),
		}

		_, err := model.CreateTweet(tweet)
		if err != nil {
			log.Println("Could not create tweet.\n", err)
			return
		}

		http.Redirect(w, r, "/", http.StatusMovedPermanently)
		return
	} else {
		// Render tweet.html with tweet replies
	}
}

func UserHandler(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	uid, err := model.GetUserIdFromUsername(username)
	if err != nil {
		log.Println("Could not get user ID.\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tweets, err := model.GetHistory(uid)
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
	followed, err := model.GetUserIdFromUsername(username)
	if err != nil {
		log.Println("Could not get user ID.\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Println("Follower: ", follower, "; Followed: ", followed)
	_, err = model.CreateFollow(followed, follower.(int64))
	if err != nil {
		log.Println("Could not follow user.\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/%s", username), http.StatusMovedPermanently)
	return
}

func RetweetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		session, _ := store.Get(r, LOGIN_COOKIE_NAME)

		// Check if user is authenticated
		uid, ok := session.Values["uid"]
		if ok == false {
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		tweetId, err := strconv.ParseInt(r.FormValue("tweet_id"), 10, 64)
		if err != nil {
			log.Println("Invalid tweet ID: ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Println("UID: ", uid, "; TweetId: ", tweetId)
		_, err = model.CreateRetweet(uid.(int64), tweetId)
		if err != nil {
			log.Println("Could not retweet.\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusMovedPermanently)
		return
	}
}

func LikeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		session, _ := store.Get(r, LOGIN_COOKIE_NAME)

		// Check if user is authenticated
		uid, ok := session.Values["uid"]
		if ok == false {
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		tweetId, err := strconv.ParseInt(r.FormValue("tweet_id"), 10, 64)
		if err != nil {
			log.Println("Invalid tweet ID: ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Println("UID: ", uid, "; TweetId: ", tweetId)
		_, err = model.CreateLike(uid.(int64), tweetId)
		if err != nil {
			log.Println("Could not like.\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusMovedPermanently)
		return
	}
}

func main() {
	model.InitDB()

	r := mux.NewRouter()
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	r.HandleFunc("/login", LoginHandler)
	r.HandleFunc("/logout", LogoutHandler)
	r.HandleFunc("/tweet", TweetHandler)
	r.HandleFunc("/follow", FollowHandler)
	r.HandleFunc("/retweet", RetweetHandler)
	r.HandleFunc("/like", LikeHandler)
	r.HandleFunc("/{username}", UserHandler)
	r.HandleFunc("/", IndexHandler)
	log.Fatal(http.ListenAndServe(":8000", r))
}