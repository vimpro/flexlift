package main

//data models used in the database and frontend

type User struct {
	Name string
	Handle string `gorm:"unique"`
	Location string
	Bio string

	UUID string `gorm:"unique"`
}

type Like struct {
	PostUUID string
	UserUUID string
	UserName string
}

type Post struct {
	Title string
	Description string
	
	IsPr bool
	Weight int
	Lift string
	UUID string `gorm:"unique"`
	Likes int
	
	UserUUID string
	UserName string
}

type ApplicationState struct {
	SignedIn bool
	UUID string //uuid that is signed in right now
	Cookie string
}

type FrontPageTmpl struct {
	TopPosts []Post
	ApplicationState
}

type Auth struct {
	UserUUID string
	Password string //TODO: change this

	CurrentCookie string
}