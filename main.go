package main

import (
	"net/http"
	"html/template"
	"log"
	"fmt"
	"os"
	"strconv"
	"database/sql"
	mux "github.com/gorilla/mux"
	sessions "github.com/gorilla/sessions"
	model "github.com/dustinnewman98/twitter_clone/model"
	api "github.com/dustinnewman98/twitter_clone/api"
	session "github.com/dustinnewman98/twitter_clone/session"
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
	CurrentUsername string
	CurrentUserId int64
	Title string
}

type UserPage struct {
	Username string
	UserId int64
	Tweets []model.Tweet
	CrossUsers model.CrossUsers
	Bio string
	Website string
	Location string
	DisplayName string
	CurrentUsername string
	CurrentUserId int64
	Title string
}

type TweetPage struct {
	Tweet model.Tweet
	Replies []model.Tweet
	CurrentUsername string
	CurrentUserId int64
	Title string
}

type UserEditPage struct {
	DisplayName string
	Bio string
	Website string
	Location string
	CurrentUsername string
	CurrentUserId int64
	Title string
}

type MessagesPage struct {
	Conversations []model.Conversation
	CurrentUsername string
	CurrentUserId int64
	Title string
}

type MessagePage struct {
	Messages []model.Message
	ConversationId int64
	CurrentUsername string
	CurrentUserId int64
	Title string
}

type NotificationsPage struct {
	Notifications []model.Notification
	CurrentUsername string
	CurrentUserId int64
	Title string
}

const (
	LOGIN_COOKIE_NAME = "login"
)

var templates = template.Must(template.ParseGlob("templates/*.html"))

func stringToNullString(maybeString string) sql.NullString {
	nullString := sql.NullString{String: "", Valid: false}
	if maybeString != "" {
		nullString = sql.NullString{String: maybeString, Valid: true}
	}
	return nullString
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// Check if user is authenticated
		session, _ := session.Store.Get(r, LOGIN_COOKIE_NAME)
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

		session, err := session.Store.Get(r, LOGIN_COOKIE_NAME)
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
	session, _ := session.Store.Get(r, LOGIN_COOKIE_NAME)
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
	session, _ := session.Store.Get(r, LOGIN_COOKIE_NAME)

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
		CurrentUsername: username.(string),
		CurrentUserId: uid.(int64),
		Title: "Home",
	}
	fmt.Println(data)

	templates.ExecuteTemplate(w, "index.html", data)
}

func TweetHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := session.Store.Get(r, LOGIN_COOKIE_NAME)

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
	replies, err := model.GetReplies(tweetId, uid.(int64))
	if err != nil {
		log.Println("Could not get replies.")
	}

	title := fmt.Sprintf("%v on Gwitter: %q", tweet.Username, tweet.Text)
	data := TweetPage{
		Tweet: tweet,
		Replies: replies,
		CurrentUsername: username.(string),
		CurrentUserId: uid.(int64),
		Title: title,
	}
	templates.ExecuteTemplate(w, "tweet.html", data)
}

func UserHandler(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	user, err := model.GetUserFromUsername(username)
	if err != nil {
		log.Println("Could not get user ID.\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	session, _ := session.Store.Get(r, LOGIN_COOKIE_NAME)

	// Check if user is authenticated
	currentUid, ok := session.Values["uid"].(int64)
	if ok == false {
		http.Redirect(w, r, "/login", http.StatusMovedPermanently)
		return
	}
	currentUsername, _ := session.Values["username"].(string)

	tweets, err := model.GetHistory(user.Id, currentUid)
	if err != nil {
		log.Println("Could not get tweets.\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	crossUsers, err := model.GetUsersRelationship(user.Id, currentUid)
	if err != nil {
		log.Println("Could not get user relationship.\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	title := username
	if user.DisplayName != "" {
		title = fmt.Sprintf("%s (@%s)", user.DisplayName, username)
	}

	data := UserPage{
		Username: username,
		UserId: user.Id,
		Tweets: tweets,
		CrossUsers: crossUsers,
		Bio: user.Bio,
		DisplayName: user.DisplayName,
		Location: user.Location,
		Website: user.Website,
		CurrentUsername: currentUsername,
		CurrentUserId: currentUid,
		Title: title,
	}

	templates.ExecuteTemplate(w, "user_tweets.html", data)
}

func UserLikesHandler(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	user, err := model.GetUserFromUsername(username)
	if err != nil {
		log.Println("Could not get user ID.\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	session, _ := session.Store.Get(r, LOGIN_COOKIE_NAME)

	// Check if user is authenticated
	currentUid, ok := session.Values["uid"]
	if ok == false {
		http.Redirect(w, r, "/login", http.StatusMovedPermanently)
		return
	}
	currentUsername, _ := session.Values["username"]

	tweets, err := model.GetLikes(user.Id, currentUid.(int64))
	if err != nil {
		log.Println("Could not get tweets.\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	crossUsers, err := model.GetUsersRelationship(user.Id, currentUid.(int64))
	if err != nil {
		log.Println("Could not get user relationship.\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	title := username
	if user.DisplayName != "" {
		title = fmt.Sprintf("%s (@%s)", user.DisplayName, username)
	}

	data := UserPage{
		Username: username,
		UserId: user.Id,
		Tweets: tweets,
		CrossUsers: crossUsers,
		Bio: user.Bio,
		DisplayName: user.DisplayName,
		Location: user.Location,
		Website: user.Website,
		CurrentUsername: currentUsername.(string),
		CurrentUserId: currentUid.(int64),
		Title: title,
	}

	templates.ExecuteTemplate(w, "user_likes.html", data)
}

func UserEditHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := session.Store.Get(r, LOGIN_COOKIE_NAME)
	// Check if user is authenticated
	username, ok := session.Values["username"]
	if ok == false {
		log.Println("Not valid username")
		http.Redirect(w, r, "/login", http.StatusMovedPermanently)
		return
	}
	fmt.Println(mux.Vars(r)["username"])

	if mux.Vars(r)["username"] != username {
		fmt.Printf("%s not equal to %s\n", mux.Vars(r)["username"], username)
		http.Redirect(w, r, fmt.Sprintf("/%s", username), http.StatusMovedPermanently)
		return
	}

	user, err := model.GetUserFromUsername(username.(string))
	if err != nil {
		log.Println("Could not get user.\n", err)
		http.Error(w, err.Error(), http.StatusMovedPermanently)
		return
	}

	data := UserEditPage{
		DisplayName: user.DisplayName,
		Bio: user.Bio,
		Location: user.Location,
		Website: user.Website,
		CurrentUsername: user.Username,
		CurrentUserId: user.Id,
		Title: "Edit your profile",
	}

	templates.ExecuteTemplate(w, "user_edit.html", data)
}

func MessagesHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := session.Store.Get(r, LOGIN_COOKIE_NAME)

	// Check if user is authenticated
	uid, ok := session.Values["uid"]
	if ok == false {
		http.Redirect(w, r, "/login", http.StatusMovedPermanently)
		return
	}
	username, _ := session.Values["username"]
	
	conversations, err := model.GetConversations(uid.(int64))
	if err != nil {
		log.Println("Could not get conversations.\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := MessagesPage{
		Conversations: conversations,
		CurrentUserId: uid.(int64),
		CurrentUsername: username.(string),
		Title: "Messages",
	}

	templates.ExecuteTemplate(w, "messages.html", data)
	return
}

func DMHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := session.Store.Get(r, LOGIN_COOKIE_NAME)

	// Check if user is authenticated
	currentUid, ok := session.Values["uid"].(int64)
	if ok == false {
		http.Redirect(w, r, "/login", http.StatusMovedPermanently)
		return
	}
	currentUsername, _ := session.Values["username"].(string)

	otherUsername := mux.Vars(r)["user_b"]
	if otherUsername == currentUsername {
		otherUsername = mux.Vars(r)["user_a"]
	}

	if otherUsername == "" {
		http.Redirect(w, r, "/", http.StatusMovedPermanently)
		return
	}

	otherUserId, err := strconv.ParseInt(otherUsername, 10, 64)
	if err != nil {
		log.Println("Invalid user ID ", otherUsername)
		http.Redirect(w, r, "/", http.StatusMovedPermanently)
		return
	}

	conversationId, err := model.GetTwoUsersConversation(otherUserId, currentUid)
	if err != nil {
		log.Println("Could not get conversation.\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if conversationId == 0 {
		conversationId, err = model.CreateTwoUsersConversation(otherUserId, currentUid)
		if err != nil {
			log.Println("Could not create conversation.\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	conversationURL := fmt.Sprintf("/messages/%d", conversationId)
	http.Redirect(w, r, conversationURL, http.StatusMovedPermanently)
	return
}

func ConversationHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := session.Store.Get(r, LOGIN_COOKIE_NAME)

	// Check if user is authenticated
	currentUid, ok := session.Values["uid"].(int64)
	if ok == false {
		http.Redirect(w, r, "/login", http.StatusMovedPermanently)
		return
	}
	currentUsername, _ := session.Values["username"].(string)

	conversationVariable := mux.Vars(r)["conversation_id"]
	if conversationVariable == "" {
		http.Redirect(w, r, "/messages", http.StatusMovedPermanently)
		return
	}

	conversationId, err := strconv.ParseInt(conversationVariable, 10, 64)
	if err != nil {
		log.Println("Invalid conversation ID.")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	messages, err := model.GetConversation(conversationId)
	if err != nil {
		log.Println("Could not get conversation.\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	title := "Conversation"
	for _, message := range messages {
		if message.SenderId != currentUid {
			if message.SenderDisplayName != "" {
				title = message.SenderDisplayName
			} else {
				title = message.SenderUsername
			}
			break
		}
	}

	data := MessagePage{
		Messages: messages,
		ConversationId: conversationId,
		CurrentUserId: currentUid,
		CurrentUsername: currentUsername,
		Title: title,
	}

	templates.ExecuteTemplate(w, "message.html", data)
	return
}

func NotificationsHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := session.Store.Get(r, LOGIN_COOKIE_NAME)

	// Check if user is authenticated
	currentUid, ok := session.Values["uid"].(int64)
	if ok == false {
		http.Redirect(w, r, "/login", http.StatusMovedPermanently)
		return
	}
	currentUsername, _ := session.Values["username"].(string)

	notifications, err := model.GetNotifications(currentUid)
	if err != nil {
		log.Println("Could not get notifications")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	data := NotificationsPage{
		Notifications: notifications,
		CurrentUserId: currentUid,
		CurrentUsername: currentUsername,
		Title: "Notifications",
	}

	templates.ExecuteTemplate(w, "notifications.html", data)
	return
}

func main() {
	model.InitDB()
	api.Init()

	port := ":" + os.Getenv("PORT")

	r := mux.NewRouter()
	s := r.PathPrefix("/api").Subrouter()
	s.HandleFunc("/tweet", api.TweetHandler).Methods("POST")
	s.HandleFunc("/follow", api.FollowHandler).Methods("POST")
	s.HandleFunc("/retweet", api.RetweetHandler).Methods("POST")
	s.HandleFunc("/like", api.LikeHandler).Methods("POST")
	s.HandleFunc("/messages/{conversation_id}", api.MessageHandler).Methods("POST")
	s.HandleFunc("/{username}/edit", api.UserEditHandler).Methods("POST")

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	r.HandleFunc("/login", LoginHandler)
	r.HandleFunc("/logout", LogoutHandler)
	r.HandleFunc("/tweet/{tweet_id}", TweetHandler).Methods("GET")
	r.HandleFunc("/notifications", NotificationsHandler).Methods("GET")
	r.HandleFunc("/messages", MessagesHandler).Methods("GET")
	r.HandleFunc("/messages/{user_a}-{user_b}", DMHandler).Methods("GET")
	r.HandleFunc("/messages/{conversation_id}", ConversationHandler).Methods("GET")
	r.HandleFunc("/{username}", UserHandler).Methods("GET")
	r.HandleFunc("/{username}/likes", UserLikesHandler).Methods("GET")
	r.HandleFunc("/{username}/edit", UserEditHandler).Methods("GET")
	r.HandleFunc("/", IndexHandler).Methods("GET")
	log.Fatal(http.ListenAndServe(port, r))
}