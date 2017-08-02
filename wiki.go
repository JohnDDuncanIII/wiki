// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// https://godoc.org/github.com/gorilla/websocket
// https://godoc.org/github.com/dgrijalva/jwt-go

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
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/johndduncaniii/faces"
	"github.com/johndduncaniii/wikidown"
	"github.com/nytimes/gziphandler"

	//"golang.org/x/crypto/bcrypt"
)

var fns = template.FuncMap{
	"plus1": func(x int) int {
		return x + 1
	},
}

var (
	templates = template.Must(template.New("").Funcs(fns).ParseFiles("tmpl/edit.html", "tmpl/view.html", "tmpl/entries.html"))
	titleValidator = regexp.MustCompile("^[a-zA-Z0-9]+$")
	cacheSince = time.Now().Format(http.TimeFormat)
	cacheUntil = time.Now().AddDate(60, 0, 0).Format(http.TimeFormat)
	date_format = "Monday, January 2 2006 at 3:04pm"
)

type Entry struct {
	Title string
	Body template.HTML
	Comments map[int]*Comment
	Toc [][]string
}

type Comment struct {
	Name string
	Email string
	XFace string
	Face string
	Homepage string
	Ip string
	Epoch string
	Comment template.HTML
	EmailMD5 string
	Favatar string
	Picons []template.HTML
}

func (p *Entry) save() error {
	path :=	 "entries/" + p.Title + "/"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, os.ModePerm)
	}
	filename := path + p.Title + ".txt"

	return ioutil.WriteFile(filename, []byte(p.Body), 0600)
}

func (p *Entry) saveComment(outStr string) error {
	path :=	 "entries/" + p.Title + "/comments/"
	numCommentsPath := "entries/" + p.Title + "/comments/num.txt"
	readNumComments, _ := ioutil.ReadFile(numCommentsPath)
	numComments, _ := strconv.Atoi(string(readNumComments))

	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, os.ModePerm)
	}

	var filename string
	if _, err := os.Stat(numCommentsPath); os.IsNotExist(err) {
		ioutil.WriteFile(numCommentsPath, []byte("0"), 0600)
		filename = path + "0.txt"

		return ioutil.WriteFile(filename, []byte(outStr), 0600)
	}

	numComments++
	ioutil.WriteFile(numCommentsPath, []byte(strconv.Itoa(numComments)), 0600)
	filename = path + strconv.Itoa(numComments) + ".txt"

	return ioutil.WriteFile(filename, []byte(outStr), 0600)
}

func (p *Entry) remove() error {
	path :=	 "entries/" + p.Title + "/"
	return os.RemoveAll(path)
}

func (p *Entry) removeComment(cmt_num string) error {
	path :=	 "entries/" + p.Title + "/comments/" + cmt_num + ".txt"
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

	body, toc := wikidown.ParseAll(string(b))

	numCommentsPath := "entries/" + title + "/comments/num.txt"
	readNumComments, err := ioutil.ReadFile(numCommentsPath)
	numComments, _ := strconv.Atoi(string(readNumComments))

	m := make(map[int]*Comment)

	for i := 0; i <= numComments; i++ {
		dat, err := ioutil.ReadFile("entries/"+title+"/comments/"+strconv.Itoa(i)+".txt")
		if(err == nil) {
			dat_arr := strings.Split(string(dat), "\n")

			name := dat_arr[0]
			ip := dat_arr[1]
			email := dat_arr[2]
			homepage := dat_arr[3]
			epoch := dat_arr[4]
			intEpoch, _ := strconv.ParseInt(epoch, 10, 64)
			epoch = time.Unix(intEpoch, 0).Format(date_format)
			face := dat_arr[5]
			xface := dat_arr[6]
			md5 := dat_arr[7]
			favatar := dat_arr[8]
			comment := dat_arr[9]
			for i := 10; i < len(dat_arr); i++ {
				comment += "\n" + dat_arr[i]
			}
			comment = template.HTMLEscapeString(comment)
			comment = wikidown.Parse(comment)
			picons := faces.SearchPicons(email)
			c := &Comment{Name: name, Email: email, XFace: xface, Face: face, Homepage: homepage, Ip: ip, Epoch: epoch, Comment: template.HTML(wikidown.Emoticons(comment)), EmailMD5: md5, Favatar: favatar, Picons: picons}
			m[i] = c;
		}
	}

	return &Entry{Title: title, Body: template.HTML(wikidown.Emoticons(body)), Comments: m, Toc: toc}, nil
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	var output bytes.Buffer
	if title != "" {
		//fmt.Println(r.URL.Path[len("/entries/"):])
		p, err := loadEntry(title)
		if err != nil {
			http.Redirect(w, r, "/edit/" + title, http.StatusFound)
			return
		}
		err = templates.ExecuteTemplate(&output, "view.html", p)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		var entries []string
		var path = "entries/"
		files, _ := ioutil.ReadDir("./" + path)

		for _, f := range files {
			entries = append(entries, f.Name())
		}

		err := templates.ExecuteTemplate(&output, "entries.html", entries)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Write(output.Bytes())
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	if title != "" {
		filename := "entries/" + title + "/" + title + ".txt"
		b, err := ioutil.ReadFile(filename)
		p := &Entry{Title: title, Body: template.HTML(b)}
		err = templates.ExecuteTemplate(w, "edit.html", p)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		http.Redirect(w, r, "/edit/main", http.StatusFound)
	}
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	if title != "" {
		body := r.FormValue("body")
		p := &Entry{Title: title, Body: template.HTML(body)}
		err := p.save()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/entries/"+title, http.StatusFound)
	} else {
		http.Redirect(w, r, "/entries/main", http.StatusFound)
	}
}

func removeHandler(w http.ResponseWriter, r *http.Request, title string) {
	if title != "" {
		p := &Entry{Title: title}
		err := p.remove()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/entries/main", http.StatusFound)
}

// name ip email homepage unixtime comment face xface emailMD5 favatar
func commentHandler(w http.ResponseWriter, r *http.Request, title string) {
	if _, err := os.Stat("entries/" + title); os.IsNotExist(err) {
		http.Redirect(w, r, "/entries/main", http.StatusFound)
		return
	}
	if title != "" {
		name := r.FormValue("name")
		// default name
		if name == "" {
			name = "Anonymous"
		}

		ip := r.RemoteAddr

		email := r.FormValue("email")
		// email validation
		e, err := mail.ParseAddress(email)
		if err != nil {
			if(err.Error() != "mail: no address") {
				http.Error(w, err.Error() + "\n" + "malformed email address", http.StatusInternalServerError)
				return
			}
		} else {
			email = e.Address
		}

		homepage := r.FormValue("homepage")
		// homepage URL validation & favatar parsing
		_, err = url.ParseRequestURI(homepage)
		if err != nil {
			if err.Error() != "parse : empty url" {
				http.Error(w, err.Error() + "\n" + "malformed homepage url", http.StatusInternalServerError)
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
				http.Error(w, getErr.Error() + "\n" + "could not parse homepage url: " + homepage, http.StatusInternalServerError)
				return
			}

			respBody, readErr := ioutil.ReadAll(res.Body)
			res.Body.Close()
			html = string(respBody)
			if readErr != nil {
				http.Error(w, readErr.Error() + "\n" + "could not parse page", http.StatusInternalServerError)
				return
			}

			regex1 := regexp.MustCompile(`<link (?:[^>\s]*)rel="(?:[S|s]hortcut|[I|i]con|[S|s]hortcut [I|i]con|mask-icon|apple-touch-icon-precomposed)"(?:[^>]*)href="*([^"\s]+)"*\s*(?:[^>]*)>`)
			regex2 := regexp.MustCompile(`<link (?:[^>\s]*)href="*([^"\s]+)"*\s*(?:[^>\s]*)rel="(?:[S|s]hortcut|[I|i]con|[S|s]hortcut [I|i]con|mask-icon|apple-touch-icon-precomposed)"(?:[^>\s]*)>`)

			result_slice1 := regex1.FindStringSubmatch(html)
			result_slice2 := regex2.FindStringSubmatch(html)
			if len(result_slice1) > 0 {
				favicon = result_slice1[1]
			} else if len(result_slice2) > 0 {
				favicon = result_slice2[1]
			}

			if strings.Contains(favicon, "~") {
				s := strings.LastIndex(favicon, "/")
				favicon = favicon[s+1:len(favicon)]
			}
			_, err = url.ParseRequestURI(favicon)
			if !strings.Contains(favicon, "://") || err != nil {
				if favicon != "" && !strings.Contains(favicon, "data:image/png;base64,") {
					if favicon[0:2] == "//" { // cnn uses this strange syntax
						favicon = "http://" + favicon[2:len(favicon)]
					} else if favicon[0:1] == "/" && homepage[len(homepage)-1:] == "/" { // double backslash
						favicon = homepage + favicon[1:len(favicon)]
					} else { // if the favicon itself is not a url, try it with the homepage
						favicon = homepage + favicon
					}
				}
			}

			if favicon == "" {
				res, err = http.Get(homepage + "favicon.ico")
				if err != nil {
					http.Error(w, err.Error() + "\n" + "could not parse homepage url: " + homepage, http.StatusInternalServerError)
					return
				}
				imgBody, imgErr := ioutil.ReadAll(res.Body)
				res.Body.Close()
				if imgErr != nil {
					http.Error(w, imgErr.Error() + "\n" + "could not parse page", http.StatusInternalServerError)
					return
				}
				imgContent := http.DetectContentType(imgBody)

				if imgContent == "image/jpeg" || imgContent == "image/png" ||
					imgContent == "image/gif" || imgContent == "image/x-icon" ||
					imgContent == "image/vnd.microsoft.icon" {
					favicon = homepage + "favicon.ico"
				}
			}
		}

		epoch := strconv.Itoa(int(time.Now().Unix()))

		comment := r.FormValue("comment")

		face := r.FormValue("face")
		// face (base64 png) validation
		_, err = base64.StdEncoding.DecodeString(face)
		if err != nil {
			http.Error(w, err.Error() + "\n" + "malformed base64 encoded png face image", http.StatusInternalServerError)
			return
		}

		// use faces package for X-Face decode
		xface := r.FormValue("xface")
		if xface != "" {
			xface = faces.DoXFace(xface) // xface package call

			unbased, err := base64.StdEncoding.DecodeString(xface)
			if err != nil {
				http.Error(w, err.Error() + "\n" + "cannot decode b64", http.StatusInternalServerError)
				return
			}

			r := bytes.NewReader(unbased)
			im, err := png.Decode(r)
			if err != nil {
				http.Error(w, err.Error() + "\n" + "invalid png", http.StatusInternalServerError)
				return
			}

			buf := new(bytes.Buffer)
			png.Encode(buf, im)
			xface = base64.StdEncoding.EncodeToString([]byte(buf.String()))
		}

		var extantMD5 string
		// md5 validation
		if email != "" {
			md5 := md5.Sum([]byte(email))
			emailMD5 := hex.EncodeToString(md5[:])
			emailMD5URL := "http://www.gravatar.com/avatar.php?gravatar_id="+emailMD5+"&size=48&d=404"
			emailRes, err := http.Get(emailMD5URL)
			if err != nil {
				http.Error(w, err.Error() + "\n" + "cannot contact gravatar service", http.StatusInternalServerError)
				//return
			} else {
				imgBody, err := ioutil.ReadAll(emailRes.Body)
				if err != nil {
					http.Error(w, err.Error() + "\n" + "failed to decode gravatar image", http.StatusInternalServerError)
					return
				}
				emailRes.Body.Close()
				imgContent := http.DetectContentType(imgBody)
				if imgContent == "image/jpeg" {
					extantMD5 = emailMD5
				}
			}
		}


		outStr := name + "\n" + ip + "\n" + email + "\n" + homepage + "\n" + epoch + "\n" + face + "\n" + xface + "\n" + extantMD5 +  "\n" + favicon + "\n" + comment
		p := &Entry{Title: title}
		err = p.saveComment(outStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/entries/"+title, http.StatusFound)
	} else {
		http.Redirect(w, r, "/entries/main", http.StatusFound)
	}
}

func removeCommentHandler(w http.ResponseWriter, r *http.Request, title string) {
	comment_num := r.FormValue("comment_num")

	if(title != "" && comment_num != "") {
		p := &Entry{Title: title}
		err := p.removeComment(comment_num)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/entries/"+title, http.StatusFound)
	} else {
		http.Redirect(w, r, "/entries/main", http.StatusFound)
	}
}

func encodeHandler(w http.ResponseWriter, r *http.Request, title string) {
	if title != "" {
		p, err := loadEntry(title)
		if err != nil {
			http.Redirect(w, r, "/edit/"+title, http.StatusFound)
			return
		}
		json.NewEncoder(w).Encode(p)
	} else {
		http.Redirect(w, r, "/entries/main", http.StatusFound)
	}
}

/*func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}*/

func main() {
	// start the server
	fmt.Println("starting")
	/*password := "secret"
	hash, _ := HashPassword(password) // ignore error for the sake of simplicity

	fmt.Println("Password:", password)
	fmt.Println("Hash:    ", hash)

	match := CheckPasswordHash(password, hash)
	fmt.Println("Match:   ", match)*/
	// dynamic content
	http.Handle("/", gziphandler.GzipHandler(http.HandlerFunc(rootHandler)))
	handleFunc("/entries/", viewHandler)
	handleFunc("/edit/", editHandler)
	handleFunc("/save/", saveHandler)
	handleFunc("/encode/", encodeHandler)
	handleFunc("/remove/", removeHandler)
	handleFunc("/comment/", commentHandler)
	handleFunc("/removecomment/", removeCommentHandler)
	// static content
	http.Handle("/css/", gziphandler.GzipHandler(staticHandler(http.StripPrefix("/css/", http.FileServer(http.Dir("css"))))))
	http.Handle("/img/", gziphandler.GzipHandler(staticHandler(http.StripPrefix("/img/", http.FileServer(http.Dir("img"))))))
	http.Handle("/face/", gziphandler.GzipHandler(staticHandler(http.StripPrefix("/face/", http.FileServer(http.Dir("face"))))))
	http.Handle("/js/", gziphandler.GzipHandler(staticHandler(http.StripPrefix("/js/", http.FileServer(http.Dir("js"))))))
	http.HandleFunc("/favicon.ico", faviconHandler)
	//http.Handle("/files/", http.StripPrefix("/files/", http.FileServer(http.Dir("D:\\Music\\"))))
	http.ListenAndServe(":8080", nil)
}

// handle specific page types
func handleFunc (path string, fn func(http.ResponseWriter, *http.Request, string)) {
	lenPath := len(path)
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age:290304000, public")
		w.Header().Set("Last-Modified", cacheSince)
		w.Header().Set("Expires", cacheUntil)

		/*key := ""
		e := `"` + key + `"`
		w.Header().Set("Etag", e)
		if match := r.Header.Get("If-None-Match"); match != "" {
			if strings.Contains(match, e) {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}*/

		title := r.URL.Path[lenPath:]

		/*if !titleValidator.MatchString(title) {
			http.NotFound(w, r)
			return
		}*/
		/*if title == "" {
			http.Redirect(w, r, "/entries/main", http.StatusFound)
			fmt.Println("no title!")
			return
		}*/

		fn(w, r, title)
	}

	h := gziphandler.GzipHandler(http.HandlerFunc(handler))
	http.Handle(path, h)
}

// basic helper handlers
func rootHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/entries/main", http.StatusFound)
}

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "img/favicon.ico")
}


func staticHandler(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age:290304000, public")
		w.Header().Set("Last-Modified", cacheSince)
		w.Header().Set("Expires", cacheUntil)
		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}



func markdown(in string) {
/* markdown
https://en.wikipedia.org/wiki/Wikipedia:Tutorial/Formatting
https://upload.wikimedia.org/wikipedia/commons/b/b3/Wiki_markup_cheatsheet_EN.pdf
https://en.wikipedia.org/wiki/Help:Wiki_markup
https://github.com/adam-p/markdown-here/wiki/Markdown-Cheatsheet
\n </p>
** <b>
__ <b>
_ <i>
* <i>
~~ <s>
> blockquote
``` <code> (block)
` <code> (inline)
--- <hr>
*** <hr>
___ <hr>
-_  _- <small>
## <u>
%% <mark>
_{} <sub>
^{} <sup>

###### <h6>
##### <h5>
#### <h4>
### <h3>
## <h2>
# <h1>

Alt-H1
======

Alt-H2
------

[]() url
![]() img

Bullet list:
  * apples
  * oranges
  * pears

Numbered list:
  1. apples
  2. oranges
  3. pears

table:
Markdown | Less | Pretty
--- | --- | ---
*Still* | `renders` | **nicely**
1 | 2 | 3

raw html
s/\!\[(.*)\]\((.*)\)/<img src=\2 alt=\1>/g;  # markdown url (force unwrap image)
s/\[(.*)\]\((.*)\)/<a href=\2>\1<\/a>/g;  # markdown url
s/\[(.*)\]<([a-zA-Z0-9[:space:]_,]*)>/<abbr title="\2">\1<\/abbr>/g;  # replace []<> <abbr>

DONE:
s|https[:]\/\/www.youtube.com\/watch\?v=([a-zA-Z0-9_]*)|<object style="width:100%;height:100%;width:420px;height:315px;float:none;clear:both;margin:2px auto;" data="http:\/\/www.youtube.com\/embed\/\1"><\/object>|g;
s|[[:space:]](http[:]//[^ ]*[a-zA-Z])[[:space:]]| <a href=\"\1\">\1</a> |g;  # replace urls  html urls
s|\w+@\w+\.\w+(\.\w+)?|<a href=\"mailto:\0\">\0</a>|g;s/\//\\\//g' $2);	 # email mailto
*/
}
