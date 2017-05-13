// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import ( // https://gowebexamples.github.io/password-hashing/
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var date_format = "Monday, January 2 2006 at 3:04pm"

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
}

type Entry struct {
	Title string
	Body string
	Comments map[int]*Comment
}

func (p *Entry) save() error {
	path :=  "entries/" + p.Title + "/"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, os.ModePerm)
	}
	filename := path + p.Title + ".txt"
	return ioutil.WriteFile(filename, []byte(p.Body), 0600)
}

func (p *Entry) saveComment(outStr string) error {
	path :=  "entries/" + p.Title + "/comments/"
	filename_n := "entries/" + p.Title + "/comments/num.txt"
	num_comments, _ := ioutil.ReadFile(filename_n)
	comments, _ := strconv.Atoi(string(num_comments))

	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, os.ModePerm)
	}

	var filename string
	if _, err := os.Stat(filename_n); os.IsNotExist(err) {
		ioutil.WriteFile(filename_n, []byte("0"), 0600)
		filename = path + "0.txt"
		return ioutil.WriteFile(filename, []byte(outStr), 0600)
	}

	comments++
	ioutil.WriteFile(filename_n, []byte(strconv.Itoa(comments)), 0600)
	filename = path + strconv.Itoa(comments) + ".txt"
	return ioutil.WriteFile(filename, []byte(outStr), 0600)
}

func (p *Entry) remove() error {
	path :=  "entries/" + p.Title + "/"
	return os.RemoveAll(path)
}

func (p *Entry) removeComment(cmt_num string) error {
	path :=  "entries/" + p.Title + "/comments/" + cmt_num + ".txt"
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
	/*body_src := template.HTMLEscapeString(string(b_body))
	body := template.HTML(ParseEmoticons(body_src))*/
	body := string(b)

	cn_filename := "entries/" + title + "/comments/num.txt"
	num_comments, err := ioutil.ReadFile(cn_filename)
	int_comments, _ := strconv.Atoi(string(num_comments))

	m := make(map[int]*Comment)

	for i := 0; i <= int_comments; i++ {
		dat, err := ioutil.ReadFile("entries/"+title+"/comments/"+strconv.Itoa(i)+".txt")
		if(err == nil) {
			dat_arr := strings.Split(string(dat), "\n")

			name := dat_arr[0]
			ip := dat_arr[1]
			email := dat_arr[2]
			homepage := dat_arr[3]
			epoch := dat_arr[4]
			epoch_i, _ := strconv.ParseInt(epoch, 10, 64)
			epoch = time.Unix(epoch_i, 0).Format(date_format)
			comment_src := dat_arr[5]
			comment_src = template.HTMLEscapeString(comment_src)
			comment := template.HTML(ParseEmoticons(comment_src))
			face := dat_arr[6]
			xface := dat_arr[7]
			md5 := dat_arr[8]
			favatar := dat_arr[9]
			c := &Comment{Name: name, Email: email, XFace: xface, Face: face, Homepage: homepage, Ip: ip, Epoch: epoch, Comment: comment, EmailMD5: md5, Favatar: favatar}
			m[i] = c;
		}
	}

	return &Entry{Title: title, Body: body, Comments: m}, nil
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	var output bytes.Buffer
	if title != "" {
		//fmt.Println(r.URL.Path[len("/entries/"):])
		p, err := loadEntry(title)
		if err != nil {
			http.Redirect(w, r, "/edit/"+title, http.StatusFound)
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
		files, _ := ioutil.ReadDir("./"+path)
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
	p, err := loadEntry(title)
	if err != nil {
		p = &Entry{Title: title}
	}
	if title != "" {
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
		p := &Entry{Title: title, Body: body}
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

			//fmt.Println(favicon)
			//fmt.Println(homepage)

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
		// convert newlines to separate paragraphs
		comment = strings.Replace(comment, "\r\n", "</p><p>" , -1)
		comment = strings.Replace(comment, "\n", "</p><p>" , -1)
		comment = strings.Replace(comment, "<p></p>", "" , -1)

		face := r.FormValue("face")
		// face (base64 png) validation
		_, f_err := base64.StdEncoding.DecodeString(face)
		if f_err != nil {
			http.Error(w, f_err.Error() + "\n" + "malformed base64 encoded png face image", http.StatusInternalServerError)
			return
		}

		// no way to validate xfaces on the back-end (yet)
		// see: use cgo to run compface (looking for a better solution)
		xface := r.FormValue("xface")
		if xface != "" {
			xface = doXFace(xface)
		}

		// md5 validation
		md5 := md5.Sum([]byte(email))
		emailMD5 := hex.EncodeToString(md5[:])
		emailMD5URL := "http://www.gravatar.com/avatar.php?gravatar_id="+emailMD5+"&size=48&d=404"
		emailRes, _ := http.Get(emailMD5URL)
		imgBody, imgErr := ioutil.ReadAll(emailRes.Body)
		emailRes.Body.Close()
		if imgErr != nil {
			fmt.Println(imgErr)
		}
		imgContent := http.DetectContentType(imgBody)
		var extantMD5 string
		if imgContent == "image/jpeg" {
			extantMD5 = emailMD5
		}

		outStr := name + "\n" + ip + "\n" + email + "\n" + homepage + "\n" + epoch + "\n" + comment + "\n" + face + "\n" + xface + "\n" + extantMD5 +  "\n" + favicon
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

	if(title != "") {
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
	}
}

var templates = template.Must(template.ParseFiles("tmpl/edit.html", "tmpl/view.html", "tmpl/entries.html"))
var titleValidator = regexp.MustCompile("^[a-zA-Z0-9]+$")

func handleFunc (path string, fn func(http.ResponseWriter, *http.Request, string)) {
	lenPath := len(path)
	handler := func(w http.ResponseWriter, r *http.Request) {
		title := r.URL.Path[lenPath:]
		/*if !titleValidator.MatchString(title) {
			http.NotFound(w, r)
			return
		}*/
		fn(w, r, title)
	}

	http.HandleFunc(path, handler)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/entries/main", http.StatusFound)
}

func main() {
	// start the server
	fmt.Println("starting")
	// dynamic content
	http.HandleFunc("/", rootHandler)
	handleFunc("/entries/", viewHandler)
	handleFunc("/edit/", editHandler)
	handleFunc("/save/", saveHandler)
	handleFunc("/encode/", encodeHandler)
	handleFunc("/remove/", removeHandler)
	handleFunc("/comment/", commentHandler)
	handleFunc("/removecomment/", removeCommentHandler)
	// static content
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("css"))))
	http.Handle("/img/", http.StripPrefix("/img/", http.FileServer(http.Dir("img"))))
	http.Handle("/face/", http.StripPrefix("/face/", http.FileServer(http.Dir("face"))))
	http.ListenAndServe(":8080", nil)
}

// replace emoticon markup with html
func ParseEmoticons(s string) string {
	e_path := "<img src=" + path + "img/emoticons/"
	s = strings.Replace(s,":angry:",e_path + "angry.gif>",-1)
	s = strings.Replace(s,">:(",e_path + "angry.gif>",-1)
	s = strings.Replace(s,":laugh:",e_path + "laugh.gif>",-1)
	s = strings.Replace(s,":DD",e_path + "laugh.gif>",-1)
	s = strings.Replace(s,":yell:",e_path + "yell.gif>",-1)
	s = strings.Replace(s,">:O",e_path + "yell.gif>",-1)
	s = strings.Replace(s,":innocent:",e_path + "innocent.gif>",-1)
	s = strings.Replace(s,"O:)",e_path + "innocent.gif>",-1)
	s = strings.Replace(s,":satisfied:",e_path + "satisfied.gif>",-1)
	s = strings.Replace(s,"/:D",e_path + "satisfied.gif>",-1)
	s = strings.Replace(s,":)",e_path + "smile.gif>",-1)
	s = strings.Replace(s,":O",e_path + "shocked.gif>",-1)
	s = strings.Replace(s,":(",e_path + "sad.gif>",-1)
	s = strings.Replace(s,":D",e_path + "biggrin.gif>",-1)
	s = strings.Replace(s,":P",e_path + "tongue.gif>",-1)
	s = strings.Replace(s,";)",e_path + "wink.gif>",-1)
	s = strings.Replace(s,":blush:",e_path + "blush.gif>",-1)
	s = strings.Replace(s,":\")",e_path + "blush.gif>",-1)
	s = strings.Replace(s,":confused:",e_path + "confused.gif>",-1)
	s = strings.Replace(s,":S",e_path + "confused.gif>",-1)
	s = strings.Replace(s,":cool:",e_path + "cool.gif>",-1)
	s = strings.Replace(s,"B)",e_path + "cool.gif>",-1)
	s = strings.Replace(s,":crazy:",e_path + "crazy.gif>",-1)
	s = strings.Replace(s,":cry:",e_path + "cry.gif>",-1)
	s = strings.Replace(s,":~(",e_path + "cry.gif>",-1)
	s = strings.Replace(s,":doze",e_path + "doze.gif>",-1)
	s = strings.Replace(s,":?",e_path + "doze.gif>",-1)
	s = strings.Replace(s,":hehe:",e_path + "hehe.gif>",-1)
	s = strings.Replace(s,"XD",e_path + "hehe.gif>",-1)
	s = strings.Replace(s,":plain:",e_path + "plain.gif>",-1)
	s = strings.Replace(s,":|",e_path + "plain.gif>",-1)
	s = strings.Replace(s,":rolleyes:",e_path + "rolleyes.gif>",-1)
	s = strings.Replace(s,"9_9",e_path + "rolleyes.gif>",-1)
	s = strings.Replace(s,":dizzy:",e_path + "crazy.gif>",-1)
	s = strings.Replace(s,"o_O",e_path + "crazy.gif>",-1)
	s = strings.Replace(s,":money:",e_path + "money.gif>",-1)
	s = strings.Replace(s,":$",e_path + "money.gif>",-1)
	s = strings.Replace(s,":sealed:",e_path + "sealed.gif>",-1)
	s = strings.Replace(s,":X",e_path + "sealed.gif>",-1)
	s = strings.Replace(s,":eek:",e_path + "eek.gif>",-1)
	s = strings.Replace(s,"O_O",e_path + "eek.gif>",-1)
	s = strings.Replace(s,":kiss:",e_path + "kiss.gif>",-1)
	s = strings.Replace(s,":*",e_path + "kiss.gif>",-1)
	//s = strings.Replace(s,"&lt;br&gt;", "<br>", -1)
	s = strings.Replace(s,"&lt;/p&gt;", "</p>", -1)
	s = strings.Replace(s,"&lt;p&gt;", "<p>", -1)

	return s
}

var path = ""
// script that parses through a picon db w/ a given email
// TODO: move this to faces package
func (c *Comment) SearchPicons(s string) []template.HTML {
	var pBox []template.HTML
	if s == "" {
		pImg := `<img class="face" src="face/picons/misc/MISC/noface/face.gif" title="noface">`
		pBox = append(pBox, template.HTML(pImg))
	} else {
		atSign := strings.Index(s, "@")
		mfPiconDatabases := [4]string{"domains/", "users/", "misc", "usenix/"}
		count := 0
		// if we have a valid email address
		if atSign != -1 {
			host := s[atSign + 1:len(s)]
			user := s[0:atSign]
			host_pieces := strings.Split(host, ".")

			pDef := `<img class="face" src="` + path + `face/picons/unknown/` + host_pieces[len(host_pieces)-1] + `/unknown/face.gif" title="` + host_pieces[len(host_pieces)-1] + `">`
			pBox = append(pBox, template.HTML(pDef))

			for i := range mfPiconDatabases {
				piconPath := "face/picons/" + mfPiconDatabases[i] // they are stored in $PROFILEPATH$/messagefaces/picons/ by default
				if mfPiconDatabases[i] == "misc/" {
					piconPath += "MISC/"
				} // special case MISC

				// get number of database folders (probably six, but could theoretically change)
				var l = len(host_pieces)-1
				// we will check to see if we have a match at EACH depth,
				//     so keep a cloned version w/o the 'unknown/face.gif' portion
				for l >= 0 { // loop through however many pieces we have of the host
					piconPath += host_pieces[l] + "/" // add that portion of the host (ex: 'edu' or 'gettysburg' or 'cs')
					clonedLocal := piconPath
					if mfPiconDatabases[i] == "users/" {
						piconPath += user + "/"
					} else {
						piconPath += "unknown/"
					}
					piconPath += "face.gif"
					if _, err := os.Stat(piconPath); err == nil {
						if count == 0 {
							pBox[0] = template.HTML(`<img class="face" src="` + path + piconPath + `"`)
							if strings.Contains(piconPath, "users") {
								pBox[0] += template.HTML(` title="` + host_pieces[len(host_pieces)-1] + `">`)
							} else {
								pBox[0] += template.HTML(` title="` + host_pieces[l] + `">`)
							}
						} else {
							pImg := `<img class="face" src="` + path + piconPath + `"`
							if strings.Contains(piconPath, "users") {
								pImg += ` title="` + user + `">`
							} else {
								pImg += ` title="` + host_pieces[l] + `">`
							}
							pBox = append(pBox, template.HTML(pImg))
						}
						count++
					}
					piconPath = clonedLocal
					l--
				}
			}
		}
	}
	return pBox
}

func markdown(in string) {
/* markdown
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
s|[[:space:]](http[:]//[^ ]*[a-zA-Z])[[:space:]]| <a href=\"\1\">\1</a> |g;  # replace urls  html urls
s|https[:]\/\/www.youtube.com\/watch\?v=([a-zA-Z0-9_]*)|<object style="width:100%;height:100%;width:420px;height:315px;float:none;clear:both;margin:2px auto;" data="http:\/\/www.youtube.com\/embed\/\1"><\/object>|g;
s|\w+@\w+\.\w+(\.\w+)?|<a href=\"mailto:\0\">\0</a>|g;s/\//\\\//g' $2);  # email mailto
*/
}

/* gravatar
http.DetectContentType
https://github.com/eefret/gravatar/blob/master/gravatar.go
*/
