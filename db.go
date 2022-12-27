package main

import (
	"encoding/base64"
	"errors"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
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

//Returns list of users
func (a App) getUsers(Limit int, Offset int) ([]User, error) {
	var users []User

	err := a.DB.Table("Users").Offset(Offset).Limit(Limit).Find(&users).Error

	return users, err
}

//Delete the user
func (a App) deleteUser(user User) error {
	err := a.DB.Table("Users").Where("uuid = ?", user.UUID).Delete(&User{}).Error
	if err != nil {
		return err
	}
	err = a.DB.Table("Auth").Where("user_uuid = ?", user.UUID).Delete(&Auth{}).Error
	if err != nil {
		return err
	}
	err = a.DB.Table("Posts").Where("user_uuid = ?", user.UUID).Delete(&Post{}).Error
	if err != nil {
		return err
	}

	return nil
}

// Returns the post UUID
func (a App) createPost(post Post) (string, error) {
	id := uuid.New()
	post.UUID = id.String()

	err := a.DB.Table("Posts").Create(post).Error

	return post.UUID, err
}

//Delete the post
func (a App) deletePost(post Post) error {
	err := a.DB.Table("Posts").Where("uuid = ?", post.UUID).Delete(&Post{}).Error
	if err != nil {
		return err
	}
	err = a.DB.Table("Likes").Where("post_uuid = ?", post.UUID).Delete(&Like{}).Error
	if err != nil {
		return err
	}

	return nil
}

func (a App) deleteComment(comment Comment) error {
	err := a.DB.Table("Comments").Where("uuid = ?", comment.UUID).Delete(&Comment{}).Error

	if err != nil {
		return err
	}

	err = a.DB.Table("Posts").Where("uuid = ?", comment.PostUUID).UpdateColumn("Comments", gorm.Expr("Comments - ?", 1)).Error

	return err
}

// Returns a Post object
func (a App) getPostByUUID(UUID string) (Post, error) {
	var post Post

	err := a.DB.Table("Posts").First(&post, "UUID = ?", UUID).Error

	return post, err
}

//gets the most recent posts from a user
func (a App) getPostsByUser(user User, Limit int, Offset int) ([]Post, error) {
	var posts []Post

	err := a.DB.Table("Posts").Where("user_uuid = ?", user.UUID).Offset(Offset).Limit(Limit).Find(&posts).Error

	return posts, err
}

func (a App) getAllComments(Limit int, Offset int) ([]Comment, error) {
	var comments []Comment

	err := a.DB.Table("Comments").Offset(Offset).Limit(Limit).Find(&comments).Error

	return comments, err
}

//Like a post
func (a App) likePost(post Post, user User) error {
	var has_liked Like
	err := a.DB.Table("Likes").First(&has_liked, "post_uuid = ? AND user_uuid = ?", post.UUID, user.UUID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err := a.DB.Table("Likes").Create(&Like{UserUUID: user.UUID, PostUUID: post.UUID}).Error
		if err != nil {
			return err
		}

		err = a.DB.Table("Posts").Where("UUID = ?", post.UUID).Update("likes", post.Likes + 1).Error
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}

//remove a like
func (a App) removeLike(post Post, user User) error {
	var has_liked Like

	err := a.DB.Table("Likes").First(&has_liked, "post_uuid = ? AND user_uuid = ?", post.UUID, user.UUID).Error
	if err == gorm.ErrRecordNotFound {
		return nil
	} else if err != nil {
		return err
	} else {
		err := a.DB.Table("Likes").Where("post_uuid = ? AND user_uuid = ?", post.UUID, user.UUID).Delete(&Like{}).Error
		if err != nil {
			return err
		}

		err = a.DB.Table("Posts").Where("UUID = ?", post.UUID).Update("likes", post.Likes - 1).Error
		if err != nil {
			return err
		}
	}

	return nil
}

func (a App) getLike(post Post, user User) (bool, error) {
	var has_liked Like

	err := a.DB.Table("Likes").First(&has_liked, "post_uuid = ? AND user_uuid = ?", post.UUID, user.UUID).Error
	if err == gorm.ErrRecordNotFound {
		return false, nil
	} else if err != nil {
		return false, err
	} else {
		return true, nil
	}
}

// Limit for how many top posts to get
//
// offset for pagination
func (a App) getTopPosts(Limit int, Offset int) ([]Post, error) {
	var posts []Post

	err := a.DB.Table("Posts").Offset(Offset).Limit(Limit).Order("likes DESC").Find(&posts).Error

	return posts, err
}

func (a App) createComment(comment Comment) (string, error) {
	id := uuid.New()
	comment.UUID = id.String()

	err := a.DB.Table("Comments").Create(&comment).Error

	if err != nil {
		return "", err
	}

	err = a.DB.Table("Posts").Where("UUID = ?", comment.PostUUID).UpdateColumn("Comments", gorm.Expr("Comments + ?", 1)).Error

	if err != nil {
		return "", err
	}

	return comment.UUID, nil
}

func (a App) getCommentsByPost(post Post, Limit int, Offset int) ([]Comment, error) {
	var comments []Comment

	err := a.DB.Table("Comments").Where("post_uuid = ?", post.UUID).Offset(Offset).Limit(Limit).Find(&comments).Error

	return comments, err
}

func (a App) getCommentByUUID(UUID string) (Comment, error) {
	var comment Comment

	err := a.DB.Table("Comments").First(&comment, "uuid = ?", UUID).Error

	return comment, err
}

// Auth functions
func (a App) signIn(UserUUID string, Password string) (string, error) {
	var auth Auth

	err := a.DB.Table("Auth").First(&auth, "user_uuid = ? AND password = ?", UserUUID, Password).Error

	if err != nil {
		return "", err
	}

	cookie := make([]byte, 64)
	
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

func (a App) logOut(Cookie string) error {
	err := a.DB.Table("Auth").Where("current_cookie = ?", Cookie).Update("current_cookie", gorm.Expr("NULL")).Error

	return err
}