package store

import (
	"log"
	"fmt"
	"net/http"
	"strconv"
	mux "github.com/gorilla/mux"
	_ "github.com/lib/pq"
	model "github.com/dustinnewman98/twitter_clone/model"
	session "github.com/dustinnewman98/twitter_clone/session"
)

const (
	LOGIN_COOKIE_NAME = "login"
)

func TweetHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := session.Store.Get(r, LOGIN_COOKIE_NAME)

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
}

func RetweetHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := session.Store.Get(r, LOGIN_COOKIE_NAME)

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

func LikeHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := session.Store.Get(r, LOGIN_COOKIE_NAME)

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

func FollowHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := session.Store.Get(r, LOGIN_COOKIE_NAME)

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

func UserEditHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := session.Store.Get(r, LOGIN_COOKIE_NAME)
	// Check if user is authenticated
	uid, ok := session.Values["uid"]
	if ok == false {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	username, _ := session.Values["username"]

	if mux.Vars(r)["username"] != username {
		log.Println("Could not authenticate user.")
		http.Error(w, "Permission Denied", http.StatusInternalServerError)
		return
	}

	displayName := r.FormValue("display_name")
	bio := r.FormValue("bio")
	location := r.FormValue("location")
	website := r.FormValue("website")

	userWithEdits := model.User{
		DisplayName: displayName,
		Bio: bio,
		Location: location,
		Website: website,
		Id: uid.(int64),
		Username: username.(string),
	}

	err := model.EditUser(userWithEdits)
	if err != nil {
		log.Println("Error editing: \n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/%s", username), http.StatusMovedPermanently)
	return
}