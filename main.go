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
	Title string
}

type UserPage struct {
	ThisUsername string
	ThisUserId int64
	Tweets []model.Tweet
	CrossUsers model.CrossUsers
	Username string
	UserId int64
	Title string
}

type TweetPage struct {
	Tweet model.Tweet
	Replies []model.Tweet
	Username string
	UserId int64
	Title string
}

type UserEditPage struct {
	Bio string
	Website string
	Location string
	Username string
	UserId int64
	Title string
}

const (
	LOGIN_COOKIE_NAME = "login"
)

var store = sessions.NewCookieStore([]byte("SECRET"))
var templates = template.Must(template.ParseGlob("templates/*.html"))

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// Check if user is authenticated
		session, _ := store.Get(r, LOGIN_COOKIE_NAME)
		_, ok := session.Values["uid"]
		if ok == true {
			http.Redirect(w, r, "/", http.StatusMovedPermanently)
			return
		}

		templates.ExecuteTemplate(w, "login.html", nil)
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
				templates.ExecuteTemplate(w, "login.html", data)
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
		Title: "Home",
	}
	fmt.Println(data)

	templates.ExecuteTemplate(w, "index.html", data)
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
		session, _ := store.Get(r, LOGIN_COOKIE_NAME)

		// Check if user is authenticated
		uid, ok := session.Values["uid"]
		if ok == false {
			http.Redirect(w, r, "/login", http.StatusMovedPermanently)
			return
		}
		username, _ := session.Values["username"]

		// Render tweet.html with tweet replies
		tweetId, err := strconv.ParseInt(mux.Vars(r)["tweet_id"], 10, 64)
		if err != nil {
			log.Println("Invalid tweet ID: ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tweet, err := model.GetTweet(tweetId, uid.(int64))
		if err != nil {
			log.Println("Could not get tweet.\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		title := fmt.Sprintf("%v on Gwitter: %q", tweet.Username, tweet.Text)
		data := TweetPage{
			Tweet: tweet,
			Replies: nil,
			Username: username.(string),
			UserId: uid.(int64),
			Title: title,
		}
		templates.ExecuteTemplate(w, "tweet.html", data)
	}
}

func UserHandler(w http.ResponseWriter, r *http.Request) {
	thisUsername := mux.Vars(r)["username"]
	thisUid, err := model.GetUserIdFromUsername(thisUsername)
	if err != nil {
		log.Println("Could not get user ID.\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	session, _ := store.Get(r, LOGIN_COOKIE_NAME)

	// Check if user is authenticated
	uid, ok := session.Values["uid"]
	if ok == false {
		http.Redirect(w, r, "/login", http.StatusMovedPermanently)
		return
	}
	username, _ := session.Values["username"]

	tweets, err := model.GetHistory(thisUid, uid.(int64))
	if err != nil {
		log.Println("Could not get tweets.\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	crossUsers, err := model.GetUsersRelationship(thisUid, uid.(int64))
		if err != nil {
		log.Println("Could not get user relationship.\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := UserPage{
		ThisUsername: thisUsername,
		ThisUserId: thisUid,
		Tweets: tweets,
		CrossUsers: crossUsers,
		Username: username.(string),
		UserId: uid.(int64),
		Title: thisUsername,
	}

	templates.ExecuteTemplate(w, "user.html", data)
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

		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
}

func UserEditHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, LOGIN_COOKIE_NAME)
	// Check if user is authenticated
	uid, ok := session.Values["uid"]
	if ok == false {
		http.Redirect(w, r, "/login", http.StatusMovedPermanently)
		return
	}
	username, _ := session.Values["username"]

	if mux.Vars(r)["username"] != username {
		http.Redirect(w, r, fmt.Sprintf("/%s", mux.Vars(r)["username"]), http.StatusMovedPermanently)
		return
	}

	data := UserEditPage{
		Bio: "This is my bio.",
		Website: "https://dustinnewman.io",
		Username: username.(string),
		UserId: uid.(int64),
		Title: "Edit your profile",
	}

	templates.ExecuteTemplate(w, "user_edit.html", data)
}

func main() {
	model.InitDB()

	r := mux.NewRouter()
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	r.HandleFunc("/login", LoginHandler)
	r.HandleFunc("/logout", LogoutHandler)
	r.HandleFunc("/tweet", TweetHandler)
	r.HandleFunc("/tweet/{tweet_id}", TweetHandler)
	r.HandleFunc("/follow", FollowHandler)
	r.HandleFunc("/retweet", RetweetHandler)
	r.HandleFunc("/like", LikeHandler)
	r.HandleFunc("/{username}", UserHandler)
	r.HandleFunc("/{username}/edit", UserEditHandler)
	r.HandleFunc("/", IndexHandler)
	log.Fatal(http.ListenAndServe(":8000", r))
}