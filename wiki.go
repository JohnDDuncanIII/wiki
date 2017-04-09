// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var date_format = "Monday, January 2 2006 at 3:04pm"

type CommentType struct {
	Name string
	Email string
	Xface string
	Face string
	Homepage string
	Ip string
	Epoch string
	Comment template.HTML
}

type Page struct {
	Title string
	Body  string
	Comments int
	CmtArr []string
	CommentsArr []CommentType
}

func (p *Page) save() error {
	path :=  "data/" + p.Title + "/"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, os.ModePerm)
	}
	filename := path + p.Title + ".txt"
	return ioutil.WriteFile(filename, []byte(p.Body), 0600)
}

func (p *Page) saveComment(outStr string) error {
	path :=  "data/" + p.Title + "/comments/"
	filename_n := "data/" + p.Title + "/comments/num.txt"
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
	fmt.Println(outStr)
	ioutil.WriteFile(filename_n, []byte(strconv.Itoa(comments)), 0600)
	filename = path + strconv.Itoa(comments) + ".txt"
	return ioutil.WriteFile(filename, []byte(outStr), 0600)
}

func (p *Page) remove() error {
	path :=  "data/" + p.Title + "/"
	return os.RemoveAll(path)
}

func (p *Page) removeComment(cmt_num string) error {
	path :=  "data/" + p.Title + "/comments/" + cmt_num + ".txt"
	return os.Remove(path)
}

func loadPage(title string) (*Page, error) {
	if _, err := os.Stat("data/"); os.IsNotExist(err) {
		os.Mkdir("data", os.ModePerm)
	}
	filename := "data/" + title + "/" + title + ".txt"
	b_body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	body := string(b_body)

	cn_filename := "data/" + title + "/comments/num.txt"
	num_comments, err := ioutil.ReadFile(cn_filename)
	int_comments, _ := strconv.Atoi(string(num_comments))
	var commentsArr []CommentType

	for i := 0; i <= int_comments; i++ {
		dat, err := ioutil.ReadFile("data/"+title+"/comments/"+strconv.Itoa(i)+".txt")
		if(err == nil) {
			dat_arr := strings.Split(string(dat), "¦")

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

			c := &CommentType{Name: name, Email: email, Xface: xface, Face: face, Homepage: homepage, Ip: ip, Epoch: epoch, Comment: comment}
			commentsArr = append(commentsArr, *c)
		} else {
			commentsArr = append(commentsArr, CommentType{})
		}
	}

	return &Page{Title: title, Body: body, CommentsArr: commentsArr}, nil
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	var output bytes.Buffer
	err = templates.ExecuteTemplate(&output, "view.html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(output.Bytes())
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	err = templates.ExecuteTemplate(w, "edit.html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	//p := &Page{Title: title, Body: []byte(body)}
	p := &Page{Title: title, Body: body}
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func removeHandler(w http.ResponseWriter, r *http.Request, title string) {
	p := &Page{Title: title}
	err := p.remove()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/main", http.StatusFound)
}

// name¦ip¦email¦homepage¦unixtime¦comment¦face¦xface¦
func commentHandler(w http.ResponseWriter, r *http.Request, title string) {
	name := r.FormValue("name")
	ip := r.RemoteAddr
	email := r.FormValue("email")
	homepage := r.FormValue("homepage")
	epoch := strconv.Itoa(int(time.Now().Unix()))
	comment := r.FormValue("comment")
	face := r.FormValue("face")
	xface := r.FormValue("xface")

	if name == "" {
		name = "Anonymous"
	}

	outStr := name + "¦" + ip + "¦" + email + "¦" + homepage + "¦" + epoch + "¦" + comment + "¦" + face + "¦" + xface + "¦"

	p := &Page{Title: title}
	err := p.saveComment(outStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func removeCommentHandler(w http.ResponseWriter, r *http.Request, title string) {
	comment_num := r.FormValue("comment_num")

	p := &Page{Title: title}
	err := p.removeComment(comment_num)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

var templates = template.Must(template.ParseFiles("tmpl/edit.html", "tmpl/view.html"))
var titleValidator = regexp.MustCompile("^[a-zA-Z0-9]+$")

func handleFunc (path string, fn func(http.ResponseWriter, *http.Request, string)) {
	lenPath := len(path)
	handler := func(w http.ResponseWriter, r *http.Request) {
		title := r.URL.Path[lenPath:]
		if !titleValidator.MatchString(title) {
			http.NotFound(w, r)
			return
		}
		fn(w, r, title)
	}

	http.HandleFunc(path, handler)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/view/main", http.StatusFound)
}

func main() {
	// start the server
	fmt.Println("starting")
	// dynamic content
	http.HandleFunc("/", rootHandler)
	handleFunc("/view/", viewHandler)
	handleFunc("/edit/", editHandler)
	handleFunc("/save/", saveHandler)
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

	return s
}

var path = ""
// script that parses through a picon db w/ a given email
// TODO: move this to faces package
func (c *CommentType) SearchPicons(s string) []template.HTML {
	var pBox []template.HTML
	if(s=="") {
		pImg := `<img class="face" src="face/picons/misc/MISC/noface/face.gif" title="picon">`
		pBox = append(pBox, template.HTML(pImg))
	} else {
		atSign := strings.Index(s, "@");
		mfPiconDatabases := [4]string{"domains/", "users/", "misc", "usenix/"}
		count := 0
		// if we have a valid email address
		if (atSign != -1) {
			host := s[atSign + 1:len(s)]
			user := s[0:atSign]
			host_pieces := strings.Split(host, ".")

			pDef := `<img class="face" src="` + path + `face/picons/unknown/` + host_pieces[len(host_pieces)-1] + `/unknown/face.gif" title="` + host_pieces[len(host_pieces)-1] + `">`
			pBox = append(pBox, template.HTML(pDef))

			for i := range mfPiconDatabases {
				p_path := "face/picons/" + mfPiconDatabases[i]; // they are stored in $PROFILEPATH$/messagefaces/picons/ by default
				if mfPiconDatabases[i] == "misc/" {
					p_path += "MISC/"
				} // special case MISC

				// get number of database folders (probably six, but could theoretically change)
				var l = len(host_pieces)-1
				// we will check to see if we have a match at EACH depth,
				//     so keep a cloned version w/o the 'unknown/face.gif' portion
				for l >= 0 { // loop through however many pieces we have of the host
					p_path += host_pieces[l] + "/" // add that portion of the host (ex: 'edu' or 'gettysburg' or 'cs')
					clonedLocal := p_path
					if mfPiconDatabases[i] == "users/" {
						p_path += user + "/"
					} else {
						p_path += "unknown/"
					}
					p_path += "face.gif"
					if _, err := os.Stat(p_path); err == nil {
						if(count==0) {
							pBox[0] = template.HTML(`<img class="face" src="` + path + p_path + `"`)
							if strings.Contains(p_path,"users") {
								pBox[0] += template.HTML(` title="` + host_pieces[len(host_pieces)-1] + `">`)
							} else {
								pBox[0] += template.HTML(` title="` + host_pieces[l] + `">`)
							}
						} else {
							pImg := `<img class="face" src="` + path + p_path + `"`
							if strings.Contains(p_path, "users") {
								pImg += ` title="` + user + `">`
							} else {
								pImg += ` title="` + host_pieces[l] + `">`
							}
							pBox = append(pBox, template.HTML(pImg))
						}
						count++;
					}
					p_path = clonedLocal;
					l--;
				}
			}
		}
	}
	return pBox
}
