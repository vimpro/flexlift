package main

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type App struct {
	Templates []template.Template
	DB *gorm.DB

	Routes []http.HandlerFunc
	NotFoundHandler http.HandlerFunc
}

func (a App) genAppState(r *http.Request) ApplicationState {
	var isLoggedIn bool
	var user User

	cookie, err := r.Cookie("auth")
	if err != nil {
		isLoggedIn = false
	} else if user, err = app.validateCookie(cookie.Value); err != nil {
		isLoggedIn = false
	} else {
		isLoggedIn = true
	}

	data := ApplicationState{}

	if isLoggedIn {
		data.SignedIn = true
		data.UUID = user.UUID
		data.Moderator = user.Moderator
	} else {
		data.SignedIn = false
	}

	return data
}

var app App

func main() {
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		panic("couldn't open DB")
	}

	app.DB = db

	app.DB.Table("Users").AutoMigrate(&User{})
	app.DB.Table("Posts").AutoMigrate(&Post{})
	app.DB.Table("Auth").AutoMigrate(&Auth{})
	app.DB.Table("Likes").AutoMigrate(&Like{})

	postcard := "layout/templates/postcard.html"
	topbar := "layout/templates/topbar.html"

    tmplFrontPage := template.Must(template.ParseFiles("layout/frontpage/index.html", postcard, topbar))
    tmplPost := template.Must(template.ParseFiles("layout/post/post.html", postcard, topbar))
    tmplUser := template.Must(template.ParseFiles("layout/user/user.html", postcard, topbar))
    tmplSubmit := template.Must(template.ParseFiles("layout/upload/submit.html", postcard, topbar))
    tmplLogin := template.Must(template.ParseFiles("layout/upload/login.html", postcard, topbar))
    tmplNotFound := template.Must(template.ParseFiles("layout/404.html", postcard, topbar))
	tmplSignUp := template.Must(template.ParseFiles("layout/upload/signup.html", postcard, topbar))
	tmplAdmin := template.Must(template.ParseFiles("layout/admin/admin.html", postcard, topbar))

	r := mux.NewRouter()
	app.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tmplNotFound.Execute(w, map[string]interface{}{
			"Url": r.URL.String(),
			"ApplicationState": app.genAppState(r),
		})
	})
	r.NotFoundHandler = app.NotFoundHandler

	fs := http.FileServer(http.Dir("public"))
    http.Handle("/public/", http.StripPrefix("/public/", fs))

	http.Handle("/", r)

    r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		best, err := app.getTopPosts(10, 0)
		
		if err != nil {
			panic(err)
		}

		appstate := app.genAppState(r)

		if appstate.SignedIn {
			user, err := app.getUserByUUID(appstate.UUID)
			if err != nil {
				fmt.Println("Failed to create user object from UUID")

			}
			for i, post := range best {
				liked, err := app.getLike(post, user)
				if err != nil {
					fmt.Println("Failed to get like count")
				}
				post.Liked = liked
				if post.UserUUID == appstate.UUID || appstate.Moderator {
					post.Owner = true
				} else {
					post.Owner = false
				}
				best[i] = post
			}
		}

		data := map[string]interface{}{
			"TopPosts": best,
			"ApplicationState": app.genAppState(r),
		}


		tmplFrontPage.Execute(w, data)

    })

	r.HandleFunc("/post/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		post, err := app.getPostByUUID(vars["uuid"])
		if err != nil {
			app.NotFoundHandler(w, r)
			return
		}

		appstate := app.genAppState(r)

		if appstate.SignedIn {
			user, err := app.getUserByUUID(appstate.UUID)
			if err != nil {
				fmt.Println("Failed to create user object from UUID")
			}
			liked, err := app.getLike(post, user)
			if err != nil {
				fmt.Println("Failed to get like count")
			}
			post.Liked = liked
			if post.UserUUID == appstate.UUID || appstate.Moderator {
				post.Owner = true
			} else {
				post.Owner = false
			}
		}

		data := map[string]interface{}{
			"Post": post,
			"ApplicationState": app.genAppState(r),
		}

		tmplPost.Execute(w, data)

    })

	r.HandleFunc("/user/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		page_user, err := app.getUserByUUID(vars["uuid"])
		if err != nil {
			app.NotFoundHandler(w, r)
			return
		}

		posts, err := app.getPostsByUser(page_user, 10, 0)

		if err != nil {
			posts = make([]Post, 0)
		}

		data := map[string]interface{}{
			"User": page_user,
			"Posts": posts,
			"ApplicationState": app.genAppState(r),
		}

		tmplUser.Execute(w, data)
    })

	r.HandleFunc("/upload/post/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		fileBytes, err := ioutil.ReadFile("upload/post/" + vars["uuid"])
		if err != nil {
			app.NotFoundHandler(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(fileBytes)
    })

	r.HandleFunc("/submit", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("auth")
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(("Sign in to post")))
			return
		}
		if _, err := app.validateCookie(cookie.Value); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(("Invalid cookie")))
			return
		}

		data := map[string]interface{}{
			"ApplicationState": app.genAppState(r),
		}

		tmplSubmit.Execute(w, data)
	})

	r.HandleFunc("/likePost/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("auth")
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(("Sign in to like")))
			return
		}
		user, err := app.validateCookie(cookie.Value)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(("Invalid cookie")))
			return
		}

		vars := mux.Vars(r)
		post, err := app.getPostByUUID(vars["uuid"])

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Provide a valid post to like"))
			return
		}

		err = app.likePost(post, user)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Failed to like post"))
			return
		}

		w.WriteHeader(http.StatusCreated)
	}).Methods("POST")

	r.HandleFunc("/removeLike/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("auth")
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(("Sign in to remove like")))
			return
		}
		user, err := app.validateCookie(cookie.Value)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(("Invalid cookie")))
			return
		}

		vars := mux.Vars(r)
		post, err := app.getPostByUUID(vars["uuid"])

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Provide a valid post to remove like"))
			return
		}

		err = app.removeLike(post, user)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Failed to remove like"))
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}).Methods("POST")

	r.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		data := map[string]interface{}{
			"ApplicationState": app.genAppState(r),
		}
		tmplLogin.Execute(w, data)
	})

	r.HandleFunc("/loginSubmit", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		handle := r.FormValue("handle")
		password := r.FormValue("password")

		user, err := app.getUserByHandle(handle)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("User " + handle + " not found"))
			return
		}

		cookie, err := app.signIn(user.UUID, password)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Invalid credentials"))
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name: "auth",
			Value: cookie,
		})
		
		http.Redirect(w, r, "/user/" + user.UUID, http.StatusSeeOther)
	}).Methods("POST")

	r.HandleFunc("/handleExists/{handle}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		if vars["handle"] == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("invalid handle"))
			return
		}

		_, err := app.getUserByHandle(vars["handle"])

		if err == gorm.ErrRecordNotFound {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("false"))
		} else if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("error"))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("true"))
		}
	})

	r.HandleFunc("/signup", func(w http.ResponseWriter, r *http.Request) {
		data := map[string]interface{}{
			"ApplicationState": app.genAppState(r),
		}
		tmplSignUp.Execute(w, data)
	})

	r.HandleFunc("/signupForm", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		handle := r.FormValue("handle")
		name := r.FormValue("name")
		password := r.FormValue("password")
		bio := r.FormValue("bio")

		if handle == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("missing handle"))
			return
		}

		if name == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("missing name"))
			return
		}

		if password == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("missing password"))
			return
		}

		user := User {
			Name: name,
			Handle: handle,
		}

		if bio != "" {
			user.Bio = bio
		}

		uuid, err := app.createUser(user, password)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("something went wrong"))
			return
		}

		cookie, err := app.signIn(uuid, password)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("something went wrong"))
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name: "auth",
			Value: cookie,
		})
		
		http.Redirect(w, r, "/user/" + uuid, http.StatusSeeOther)
	}).Methods("POST")

	r.HandleFunc("/submitPost", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("auth")
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Sign in to post"))
			return
		}
		user, err := app.validateCookie(cookie.Value)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid cookie"))
			return
		}

		var post Post

		post.Title = r.FormValue("title")
		post.Description = r.FormValue("description")

		temp_weight, _ := strconv.Atoi(r.FormValue("weight"))
		post.Weight = int(temp_weight)
		post.Lift= r.FormValue("lift")

		post.UserUUID = user.UUID
		post.UserName = user.Name
		
		file, _, err := r.FormFile("thumbnail")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Insert an image"))
			return
		}
		defer file.Close()

		post_uuid, err := app.createPost(post)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Failed to submit post due to internal error"))
			return
		}


		f, err := os.OpenFile("upload/post/" + post_uuid, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		_, _ = io.Copy(f, file)

		http.Redirect(w, r, "/post/" + post_uuid, http.StatusSeeOther)

	})

	r.HandleFunc("/deleteUser/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		appstate := app.genAppState(r)
		vars := mux.Vars(r)

		if !appstate.SignedIn {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(("Sign in to delete users")))
			return
		}

		user, err := app.getUserByUUID(vars["uuid"])

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Provide a valid user to delete"))
			return
		}

		if user.UUID != appstate.UUID && !appstate.Moderator {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Must be owner to delete"))
			return
		}

		err = app.deleteUser(user)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Failed to delete post"))
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}).Methods("POST")

	r.HandleFunc("/deletePost/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		appstate := app.genAppState(r)
		vars := mux.Vars(r)

		if !appstate.SignedIn {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(("Sign in to delete posts")))
			return
		}

		post, err := app.getPostByUUID(vars["uuid"])

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Provide a valid post to delete"))
			return
		}

		if post.UserUUID != appstate.UUID && !appstate.Moderator {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Must be owner to delete"))
			return
		}

		err = app.deletePost(post)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Failed to delete post"))
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}).Methods("POST")

	r.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		appstate := app.genAppState(r)

		if !appstate.SignedIn || !appstate.Moderator {
			app.NotFoundHandler(w, r)
			return
		}

		posts, err := app.getTopPosts(10, 0)
		if err != nil {
			posts = make([]Post, 0)
		}

		users, err := app.getUsers(10, 0)
		if err != nil {
			users = make([]User, 0)
		}

		data := map[string]interface{}{
			"ApplicationState": appstate,
			"Posts": posts,
			"Users": users,
		}

		tmplAdmin.Execute(w, data)
	})

	http.ListenAndServe(":8080", nil)
}