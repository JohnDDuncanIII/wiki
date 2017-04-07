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
)

type Page struct {
	Title string
	Body  []byte
	Comment []byte
	Comments int
	CmtArr []string
}

func (p *Page) save() error {
	path :=  "data/" + p.Title + "/"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, os.ModePerm)
	}
	filename := path + "entry.txt"
	return ioutil.WriteFile(filename, p.Body, 0600)
}

func (p *Page) saveComment() error {
	path :=  "data/" + p.Title + "/comments/"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, os.ModePerm)
	}
	filename := path + strconv.Itoa(p.Comments) + ".txt"

	m_filename := path + "num.txt"
	if _, err := os.Stat(m_filename); os.IsNotExist(err) {
		ioutil.WriteFile(m_filename, []byte("0"), 0600)
		filename = path + "0.txt"
		return ioutil.WriteFile(filename, p.Comment, 0600)
	}
	p.Comments++
	ioutil.WriteFile(m_filename, []byte(strconv.Itoa(p.Comments)), 0600)
	fmt.Println(p.Comments)
	filename = path + strconv.Itoa(p.Comments) + ".txt"
	return ioutil.WriteFile(filename, p.Comment, 0600)
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
	filename := "data/" + title + "/entry.txt"
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	cn_filename := "data/" + title + "/comments/num.txt"
	num_comments, err := ioutil.ReadFile(cn_filename)
	int_comments, _ := strconv.Atoi(string(num_comments))
	var read_comments []string

	for i := 0; i <= int_comments; i++ {
		dat, err := ioutil.ReadFile("data/"+title+"/comments/"+strconv.Itoa(i)+".txt")
		if(err == nil) {
			//read_comments = append(read_comments, strings.Replace(string(dat), "\n","<br>",-1))
			read_comments = append(read_comments, string(dat))
		} else {
			read_comments = append(read_comments, "")
		}
	}

	c_filename := "data/" + title + "/comments/" + string(num_comments) + ".txt"
	comments, err := ioutil.ReadFile(c_filename)

	return &Page{Title: title, Body: body, Comment: comments, Comments: int_comments, CmtArr: read_comments}, nil
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
	p := &Page{Title: title, Body: []byte(body)}
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

func commentHandler(w http.ResponseWriter, r *http.Request, title string) {
	comment := r.FormValue("comment")
	filename := "data/" + title + "/comments/num.txt"
	num_comments, _ := ioutil.ReadFile(filename)
	comments, _ := strconv.Atoi(string(num_comments))
	p := &Page{Title: title, Comment: []byte(comment), Comments: comments}
	err := p.saveComment()
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
	fmt.Println("starting")
	http.HandleFunc("/", rootHandler)
	handleFunc("/view/", viewHandler)
	handleFunc("/edit/", editHandler)
	handleFunc("/save/", saveHandler)
	handleFunc("/remove/", removeHandler)
	handleFunc("/comment/", commentHandler)
	handleFunc("/removecomment/", removeCommentHandler)
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("css"))))
	http.ListenAndServe(":8080", nil)
}
