package main

//data models used in the database and frontend

type User struct {
	Name string
	Handle string `gorm:"unique"`
	Bio string

	UUID string `gorm:"unique"`

	Moderator bool
}

type Like struct {
	PostUUID string
	UserUUID string
	UserName string
}

type Post struct {
	Title string
	Description string
	
	Weight int
	Lift string
	UUID string `gorm:"unique"`
	Likes int
	
	UserUUID string
	UserName string

	Liked bool `gorm:"-"` //shitty hack for passing thru to postcard template
	Owner bool `gorm:"-"` //same shit
}

type ApplicationState struct {
	SignedIn bool
	UUID string //uuid that is signed in right now
	Moderator bool //currently signed in user is moderator?
}

type Auth struct {
	UserUUID string
	Password string //TODO: change this

	CurrentCookie string
}