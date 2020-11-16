// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Expanded 2017-2018 John D. Duncan, III.

package main

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"image/png"
	"io/ioutil"
	"log"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/johndduncaniii/faces"
	// "github.com/johndduncaniii/wikidown"
)

// constants
const (
	chmod = 0600

	dateFormat = "Monday, January 2 2006 at 3:04pm"
)

// variables
var (
	fns = template.FuncMap{
		"plus1": func(x int) int {
			return x + 1
		},
	}

	templates = template.Must(template.
		New("").
		Funcs(fns).
		ParseFiles("tmpl/entries.html", "tmpl/edit.html", "tmpl/view.html"),
	)

	validPath = regexp.MustCompile(
		"^/(comment|encode|edit|entries|remove|removecomment|save)/([a-zA-Z0-9_()]*)$",
	)
)

// types
type Entry struct {
	Title    string
	Body     string
	Comments map[int]*Comment
	Toc      [][]string
}

type Comment struct {
	Name     string
	Ip       string
	Email    string
	Homepage string
	Epoch    string
	Face     string
	XFace    string
	EmailMD5 string
	Favatar  string
	Comment  string
	Picons   []template.HTML
}

// functions
func (p *Entry) save() error {
	path := "entries/" + p.Title + "/"

	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, os.ModePerm)
	}

	filename := path + p.Title + ".txt"

	return ioutil.WriteFile(
		filename,
		[]byte(p.Body),
		chmod,
	)
}

func (p *Entry) saveComment(outStr string) error {
	path := "entries/" + p.Title + "/comments/"
	numCommentsPath := "entries/" + p.Title + "/comments/num.txt"
	readNumComments, _ := ioutil.ReadFile(numCommentsPath)
	numComments, _ := strconv.Atoi(string(readNumComments))

	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, os.ModePerm)
	}

	var filename string

	if _, err := os.Stat(numCommentsPath); os.IsNotExist(err) {
		ioutil.WriteFile(
			numCommentsPath,
			[]byte("0"),
			chmod,
		)
		filename = path + "0.txt"

		return ioutil.WriteFile(
			filename,
			[]byte(outStr),
			chmod,
		)
	}

	numComments++
	ioutil.WriteFile(
		numCommentsPath,
		[]byte(strconv.Itoa(numComments)),
		chmod,
	)
	filename = path + strconv.Itoa(numComments) + ".txt"

	return ioutil.WriteFile(
		filename,
		[]byte(outStr),
		chmod,
	)
}

func (p *Entry) remove() error {
	path := "entries/" + p.Title + "/"

	return os.RemoveAll(path)
}

func (p *Entry) removeComment(commentNum string) error {
	path := "entries/" + p.Title + "/comments/" + commentNum + ".txt"

	return os.Remove(path)
}

func loadEntry(title string) (*Entry, error) {
	if _, err := os.Stat("entries/"); os.IsNotExist(err) {
		os.Mkdir("entries", os.ModePerm)
	}

	filename := "entries/" + title + "/" + title + ".txt"
	b, err := ioutil.ReadFile(filename)

	if err != nil {
		return nil, err
	}

	body := string(b)

	numCommentsPath := "entries/" + title + "/comments/num.txt"
	readNumComments, _ := ioutil.ReadFile(numCommentsPath)
	numComments, _ := strconv.Atoi(string(readNumComments))

	m := make(map[int]*Comment)

	for i := 0; i <= numComments; i++ {
		readComment, err := ioutil.ReadFile("entries/" + title + "/comments/" + strconv.Itoa(i) + ".txt")

		if err == nil {
			commentArray := strings.Split(string(readComment), "\n")

			name := 		commentArray[0]
			ip := 			commentArray[1]
			email := 		commentArray[2]
			homepage := 	commentArray[3]
			epoch := 		commentArray[4]
			intEpoch, _ := 	strconv.ParseInt(epoch, 10, 64)
			epoch = 		time.Unix(intEpoch, 0).Format(dateFormat)
			face := 		commentArray[5]
			xface := 		commentArray[6]
			md5 := 			commentArray[7]
			favatar := 		commentArray[8]
			comment := 		commentArray[9]

			for i := 10; i < len(commentArray); i++ {
				comment += "\n" + commentArray[i]
			}

			comment = template.HTMLEscapeString(comment)

			// faces package call
			picons := faces.SearchPicons(email)

			// create comment struct that will be passed to an Entry
			c := &Comment{
				Name:     name,
				Ip:       ip,
				Email:    email,
				Homepage: homepage,
				Epoch:    epoch,
				Face:     face,
				XFace:    xface,
				EmailMD5: md5,
				Favatar:  favatar,
				Comment:  comment,
				Picons:   picons,
			}

			m[i] = c
		}
	}

	return &Entry {
		Title:    title,
		Body:     body,
		Comments: m,
		Toc:      nil,
	}, nil
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	if title != "" {
		p, err := loadEntry(title)

		if err != nil {
			http.Redirect(
				w,
				r,
				"/edit/" + title,
				http.StatusFound,
			)
			return
		}

		err = templates.ExecuteTemplate(w, "view.html", p)

		if err != nil {
			http.Error(
				w,
				err.Error(),
				http.StatusInternalServerError,
			)
			return
		}
	} else {
		var entries []string
		var path = "entries/"
		files, _ := ioutil.ReadDir("./" + path)

		for i := 0; i < len(files); i++ {
			entries = append(entries, files[i].Name())
		}

		err := templates.ExecuteTemplate(w, "entries.html", entries)

		if err != nil {
			http.Error(
				w,
				err.Error(),
				http.StatusInternalServerError,
			)
			return
		}
	}
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	if title != "" {
		filename := "entries/" + title + "/" + title + ".txt"
		b, err := ioutil.ReadFile(filename)
		p := &Entry{Title: title, Body: string(b)}
		err = templates.ExecuteTemplate(w, "edit.html", p)

		if err != nil {
			http.Error(
				w,
				err.Error(),
				http.StatusInternalServerError,
			)
			return
		}
	} else {
		http.Error(
			w,
			"error: you must provide a wiki page to edit",
			http.StatusInternalServerError,
		)
	}
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	if title != "" {
		body := r.FormValue("body")
		p := &Entry{Title: title, Body: body}
		err := p.save()

		if err != nil {
			http.Error(
				w,
				err.Error(),
				http.StatusInternalServerError,
			)
			return
		}

		http.Redirect(
			w,
			r,
			"/entries/" + title,
			http.StatusFound,
		)
	} else {
		http.Error(
			w,
			"error: you must provide a wiki page to save",
			http.StatusInternalServerError,
		)
	}
}

func removeHandler(w http.ResponseWriter, r *http.Request, title string) {
	if title != "" {
		p := &Entry{Title: title}
		err := p.remove()

		if err != nil {
			http.Error(
				w,
				err.Error(),
				http.StatusInternalServerError,
			)
			return
		}

		http.Redirect(
			w,
			r,
			"/entries/main",
			http.StatusFound,
		)
	} else {
		http.Error(
			w,
			"error: you must provide a wiki page to remove",
			http.StatusInternalServerError,
		)
	}
}

func commentHandler(w http.ResponseWriter, r *http.Request, title string) {
	if _, err := os.Stat("entries/" + title); os.IsNotExist(err) {
		http.Redirect(
			w,
			r,
			"/entries/main",
			http.StatusFound,
		)

		return
	}

	if title != "" {
		name := r.FormValue("name")

		// default name
		if name == "" {
			name = "Anonymous"
		}

		ip := r.RemoteAddr

		// email validation
		email := r.FormValue("email")
		e, err := mail.ParseAddress(email)

		if err != nil {
			if err.Error() != "mail: no address" {
				http.Error(
					w,
					err.Error() + "\n" + "malformed email address",
					http.StatusInternalServerError,
				)
				return
			}
		} else {
			email = e.Address
		}

		// homepage URL validation & favatar parsing
		homepage := r.FormValue("homepage")
		_, err = url.ParseRequestURI(homepage)

		if err != nil {
			if err.Error() != "parse : empty url" {
				http.Error(
					w,
					err.Error() + "\n" + "malformed homepage url",
					http.StatusInternalServerError,
				)
				return
			}
		}

		var favicon string

		if len(homepage) > 0 {
			if homepage[len(homepage)-1:] != "/" {
				homepage += "/"
			}

			res, getErr := http.Get(homepage)
			var html string

			if getErr != nil {
				http.Error(
					w,
					getErr.Error() + "\n" + "could not parse homepage url: " + homepage,
					http.StatusInternalServerError,
				)
				return
			}

			respBody, readErr := ioutil.ReadAll(res.Body)
			res.Body.Close()
			html = string(respBody)

			if readErr != nil {
				http.Error(
					w,
					readErr.Error() + "\n" + "could not parse page",
					http.StatusInternalServerError,
				)
				return
			}

			regex1 := regexp.MustCompile(`<link (?:[^>\s]*)rel="(?:[S|s]hortcut|[I|i]con|[S|s]hortcut [I|i]con|mask-icon|apple-touch-icon-precomposed)"(?:[^>]*)href="*([^"\s]+)"*\s*(?:[^>]*)>`)
			regex2 := regexp.MustCompile(`<link (?:[^>\s]*)href="*([^"\s]+)"*\s*(?:[^>\s]*)rel="(?:[S|s]hortcut|[I|i]con|[S|s]hortcut [I|i]con|mask-icon|apple-touch-icon-precomposed)"(?:[^>\s]*)>`)
			favicon1 := regex1.FindStringSubmatch(html)
			favicon2 := regex2.FindStringSubmatch(html)

			if len(favicon1) > 0 {
				favicon = favicon1[1]
			} else if len(favicon2) > 0 {
				favicon = favicon2[1]
			}

			if strings.Contains(favicon, "~") {
				s := strings.LastIndex(favicon, "/")
				favicon = favicon[s+1 : len(favicon)]
			}

			_, err = url.ParseRequestURI(favicon)

			if !strings.Contains(favicon, "://") || err != nil {
				if favicon != "" && !strings.Contains(favicon, "data:image/png;base64,") {
					if favicon[0:2] == "//" {
						favicon = "http://" + favicon[2:len(favicon)]
					} else if favicon[0:1] == "/" && homepage[len(homepage)-1:] == "/" {
						favicon = homepage + favicon[1:len(favicon)]
					} else {
						favicon = homepage + favicon
					}
				}
			}

			if favicon == "" {
				res, err = http.Get(homepage + "favicon.ico")

				if err != nil {
					http.Error(
						w,
						err.Error() + "\n" + "could not parse homepage url: " + homepage,
						http.StatusInternalServerError,
					)
					return
				}

				imgBody, imgErr := ioutil.ReadAll(res.Body)
				res.Body.Close()

				if imgErr != nil {
					http.Error(
						w,
						imgErr.Error() + "\n" + "could not parse page",
						http.StatusInternalServerError,
					)
					return
				}

				imgContent := http.DetectContentType(imgBody)

				if imgContent == "image/jpeg" ||
					imgContent == "image/png" ||
					imgContent == "image/gif" ||
					imgContent == "image/x-icon" ||
					imgContent == "image/vnd.microsoft.icon" {
					favicon = homepage + "favicon.ico"
				}
			}
		}

		epoch := strconv.Itoa(int(time.Now().Unix()))

		comment := r.FormValue("comment")

		// face (base64 png) validation
		face := r.FormValue("face")
		_, err = base64.StdEncoding.DecodeString(face)

		if err != nil {
			http.Error(
				w,
				err.Error() + "\n" + "malformed base64 encoded png face image",
				http.StatusInternalServerError,
			)
			return
		}

		// use faces package for X-Face decode
		xface := r.FormValue("xface")

		if xface != "" {
			// faces package call
			xface = faces.DoXFace(xface)
			unbased, err := base64.StdEncoding.DecodeString(xface)

			if err != nil {
				http.Error(
					w,
					err.Error() + "\n" + "cannot decode b64",
					http.StatusInternalServerError,
				)
				return
			}

			r := bytes.NewReader(unbased)
			im, err := png.Decode(r)

			if err != nil {
				http.Error(
					w,
					err.Error() + "\n" + "invalid png",
					http.StatusInternalServerError,
				)
				return
			}

			buf := new(bytes.Buffer)
			png.Encode(buf, im)
			xface = base64.StdEncoding.EncodeToString([]byte(buf.String()))
		}

		// md5 validation
		var extantMD5 string

		if email != "" {
			md5 := md5.Sum([]byte(email))
			emailMD5 := hex.EncodeToString(md5[:])
			emailMD5URL := "http://www.gravatar.com/avatar.php?gravatar_id=" + emailMD5 + "&size=48&d=404"
			emailRes, err := http.Get(emailMD5URL)

			if err != nil {
				http.Error(
					w,
					err.Error() + "\n" + "cannot contact gravatar service",
					http.StatusInternalServerError,
				)
			} else {
				imgBody, err := ioutil.ReadAll(emailRes.Body)

				if err != nil {
					http.Error(
						w,
						err.Error() + "\n" + "failed to decode gravatar image",
						http.StatusInternalServerError,
					)
					return
				}

				emailRes.Body.Close()
				imgContent := http.DetectContentType(imgBody)

				if imgContent == "image/jpeg" {
					extantMD5 = emailMD5
				}
			}
		}

		outStr := name + "\n" + 
			ip + "\n" + 
			email + "\n" + 
			homepage + "\n" + 
			epoch + "\n" + 
			face + "\n" + 
			xface + "\n" + 
			extantMD5 + "\n" + 
			favicon + "\n" + 
			comment

		p := &Entry{Title: title}
		err = p.saveComment(outStr)

		if err != nil {
			http.Error(
				w,
				err.Error(),
				http.StatusInternalServerError,
			)
			return
		}

		http.Redirect(
			w,
			r,
			"/entries/" + title,
			http.StatusFound,
		)
	} else {
		http.Error(
			w,
			"error: you must provide a wiki page to add a comment to",
			http.StatusInternalServerError,
		)
	}
}

func removeCommentHandler(w http.ResponseWriter, r *http.Request, title string) {
	commentNum := r.FormValue("commentNum")

	if title != "" && commentNum != "" {
		p := &Entry{Title: title}
		err := p.removeComment(commentNum)

		if err != nil {
			http.Error(
				w,
				err.Error(),
				http.StatusInternalServerError,
			)
			return
		}

		http.Redirect(
			w,
			r,
			"/entries/" + title,
			http.StatusFound,
		)
	} else if title == "" {
		http.Error(
			w,
			"error: you must provide a wiki page to remove a comment from ",
			http.StatusInternalServerError,
		)
	} else {
		http.Error(
			w,
			"error: you must provide a comment number to delete for the entry " + title,
			http.StatusInternalServerError,
		)
	}
}

func encodeHandler(w http.ResponseWriter, r *http.Request, title string) {
	if title != "" {
		p, err := loadEntry(title)

		if err != nil {
			http.Redirect(
				w,
				r,
				"/edit/" + title,
				http.StatusFound,
			)
			return
		}

		json.NewEncoder(w).Encode(p)
	} else {
		http.Error(
			w,
			"error: you must provide a a wiki page to encode",
			http.StatusInternalServerError,
		)
	}
}

// handlers
func rootHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(
		w,
		r,
		"/entries/",
		http.StatusFound,
	)
}

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(
		w,
		r,
		"img/favicon.ico",
	)
}

func staticHandler(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)

		if m == nil {
			http.NotFound(w, r)
			return
		}

		fn(w, r, m[2])
	}
}

func main() {
	fmt.Println("starting")

	http.HandleFunc("/", http.HandlerFunc(rootHandler))

	// dynamic content
	http.HandleFunc("/entries/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))
	http.HandleFunc("/encode/", makeHandler(encodeHandler))
	http.HandleFunc("/remove/", makeHandler(removeHandler))
	http.HandleFunc("/comment/", makeHandler(commentHandler))
	http.HandleFunc("/removecomment/", makeHandler(removeCommentHandler))

	// static content
	http.Handle(
		"/css/",
		staticHandler(
			http.StripPrefix(
				"/css/",
				http.FileServer(http.Dir("css")),
			),
		),
	)

	http.Handle(
		"/img/",
		staticHandler(
			http.StripPrefix(
				"/img/",
				http.FileServer(http.Dir("img")),
			),
		),
	)

	http.Handle("/face/",
		staticHandler(
			http.StripPrefix(
				"/face/",
				http.FileServer(http.Dir("face")),
			),
		),
	)

	http.Handle("/js/",
		staticHandler(
			http.StripPrefix(
				"/js/",
				http.FileServer(http.Dir("js")),
			),
		),
	)

	http.HandleFunc("/favicon.ico", faviconHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
