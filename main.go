package main

import (
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

	postcard := "layout/templates/postcard.html"
	topbar := "layout/templates/topbar.html"

    tmplFrontPage := template.Must(template.ParseFiles("layout/frontpage/index.html", postcard, topbar))
    tmplPost := template.Must(template.ParseFiles("layout/post/post.html", postcard, topbar))
    tmplUser := template.Must(template.ParseFiles("layout/user/user.html", postcard, topbar))
    tmplSubmit := template.Must(template.ParseFiles("layout/upload/submit.html", postcard, topbar))
    tmplLogin := template.Must(template.ParseFiles("layout/upload/login.html", postcard, topbar))
    tmplNotFound := template.Must(template.ParseFiles("layout/404.html", postcard, topbar))
	tmplSignUp := template.Must(template.ParseFiles("layout/upload/signup.html", postcard, topbar))

	r := mux.NewRouter()
	app.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tmplNotFound.Execute(w, r.URL.String())
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

		isLoggedIn := false // just for now

		if isLoggedIn {
			// handle this later
		} else {
			data := FrontPageTmpl{
				TopPosts: best,
			}

			tmplFrontPage.Execute(w, data)
		}

    })

	r.HandleFunc("/post/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		isLoggedIn := false // just for now

		if isLoggedIn {
			// handle this later
		} else {
			data, err := app.getPostByUUID(vars["uuid"])
			if err != nil {
				app.NotFoundHandler(w, r)
				return
			}
			tmplPost.Execute(w, data)
		}

    })

	r.HandleFunc("/user/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		isLoggedIn := false // just for now

		if isLoggedIn {
			// handle this later
		} else {
			data, err := app.getUserByUUID(vars["uuid"])
			if err != nil {
				app.NotFoundHandler(w, r)
				return
			}
			tmplUser.Execute(w, data)
		}

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
		tmplSubmit.Execute(w, nil)
	})

	r.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		tmplLogin.Execute(w, nil)
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
		tmplSignUp.Execute(w, nil)
	})

	r.HandleFunc("/signupForm", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		handle := r.FormValue("handle")
		name := r.FormValue("name")
		password := r.FormValue("password")
		location := r.FormValue("location")
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

		if location != "" {
			user.Location = location
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
		if r.FormValue("ispr") != "" {
			post.IsPr = true
		} else {
			post.IsPr = false
		}

		temp_weight, _ := strconv.Atoi(r.FormValue("weight"))
		post.Weight = int(temp_weight)
		post.Lift= r.FormValue("lift")

		post.UserUUID = user.UUID
		post.UserName = user.Name

		post_uuid, err := app.createPost(post)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Failed to submit post due to internal error"))
			return
		}

		file, _, err := r.FormFile("thumbnail")
		if err != nil {
			panic(err)
		}
		defer file.Close()

		f, err := os.OpenFile("upload/post/" + post_uuid, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		_, _ = io.Copy(f, file)

		http.Redirect(w, r, "/post/" + post_uuid, http.StatusSeeOther)

	})
    http.ListenAndServe(":8080", nil)
}