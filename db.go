package main

import (
	"encoding/base64"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

// Returns created user's UUID
func (a App) createUser(user User, Password string) (string, error) {
	id := uuid.New()
	user.UUID = id.String()

	err := a.DB.Table("Users").Create(user).Error
	if err != nil {
		return "", err
	}
	err = nil

	err = a.DB.Table("Auth").Create(Auth{UserUUID: user.UUID, Password: Password}).Error
	if err != nil {
		return "", err
	}

	return user.UUID, nil
}

// Returns a User object
func (a App) getUserByHandle(Handle string) (User, error) {
	var user User

	err := a.DB.Table("Users").First(&user, "handle = ?", Handle).Error

	return user, err

}

// Returns a User object
func (a App) getUserByUUID(UUID string) (User, error) {
	var user User

	err := a.DB.Table("Users").First(&user, "UUID = ?", UUID).Error

	return user, err
}

// Returns the post UUID
func (a App) createPost(post Post) (string, error) {
	id := uuid.New()
	post.UUID = id.String()

	err := a.DB.Table("Posts").Create(post).Error

	return post.UUID, err
}

// Returns a Post object
func (a App) getPostByUUID(UUID string) (Post, error) {
	var post Post

	err := a.DB.Table("Posts").First(&post, "UUID = ?", UUID).Error

	return post, err
}


// Limit for how many top posts to get
//
// offset for pagination
func (a App) getTopPosts(Limit int, Offset int) ([]Post, error) {
	var posts []Post

	err := a.DB.Table("Posts").Offset(Offset).Limit(Limit).Find(&posts).Error

	return posts, err
}

// Auth functions
func (a App) signIn(UserUUID string, Password string) (string, error) {
	var auth Auth

	err := a.DB.Table("Auth").First(&auth, "user_uuid = ? AND password = ?", UserUUID, Password).Error

	if err != nil {
		return "", err
	}

	cookie := make([]byte, 16)
	
	rand.Seed(time.Now().UnixNano())
	rand.Read(cookie)
	cookie_string := string(base64.StdEncoding.EncodeToString(cookie[:]))

	err = a.DB.Table("Auth").Where("user_uuid = ?", UserUUID).Update("current_cookie", cookie_string).Error

	if err != nil {
		return "", err
	}

	return cookie_string, nil
}

func (a App) validateCookie(Cookie string) (User, error) {
	var auth Auth
	err := a.DB.Table("Auth").Select("UserUUID").First(&auth, "current_cookie = ?", Cookie).Error

	if err != nil {
		return User{}, err
	}

	user, err := a.getUserByUUID(auth.UserUUID)

	if err != nil {
		return User{}, err
	}

	return user, nil
}