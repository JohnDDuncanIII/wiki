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
	//"image"
	//"image/gif"
	"image/jpeg"
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

	"github.com/JohnDDuncanIII/faces"
	"github.com/unixpickle/resize"
	//"github.com/nfnt/resize"
	//"github.com/bamiaux/rez"
	"github.com/NYTimes/gziphandler"
)

var date_format = "Monday, January 2 2006 at 3:04pm"

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

	var toc [][]string
	body := string(b)

	var re = regexp.MustCompile(`(?m)^\*(.*)`)
	body = re.ReplaceAllString(body, `%$1`)

	re = regexp.MustCompile(`\'\'\'\'\'(.*?)\'\'\'\'\'`)
	body = re.ReplaceAllString(body, `***$1***`)

	re = regexp.MustCompile(`\'\'\'(.*?)\'\'\'`)
	body = re.ReplaceAllString(body, `**$1**`)

	re = regexp.MustCompile(`\'\'(.*?)\'\'`)
	body = re.ReplaceAllString(body, `*$1*`)

	re = regexp.MustCompile(`<blockquote>(?:\n|\r\n)?((?:.)*)(?:\n|\r\n)?<\/blockquote>`)
	body = re.ReplaceAllString(body, `> $1`)

	re = regexp.MustCompile(`(?m)^> (.*)`)
	body = re.ReplaceAllString(body, `$ $1`)

	re = regexp.MustCompile(`<!--.*?-->`)
	body = re.ReplaceAllString(body, ``)

	body = template.HTMLEscapeString(body)

	body, toc = parseToEntry(body)
	body = breakToPara(body)

	/*for _, element := range t {
		if element[1] != "" {
			toc = append(toc, element[1])
		}
	}*/

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
			face := dat_arr[5]
			xface := dat_arr[6]
			md5 := dat_arr[7]
			favatar := dat_arr[8]
			comment := dat_arr[9]
			for i := 10; i < len(dat_arr); i++ {
				comment += "\n" + dat_arr[i]
			}
			comment = template.HTMLEscapeString(comment)
			comment = breakToPara(comment)
			picons := faces.SearchPicons(email)
			c := &Comment{Name: name, Email: email, XFace: xface, Face: face, Homepage: homepage, Ip: ip, Epoch: epoch, Comment: template.HTML(ParseEmoticons(comment)), EmailMD5: md5, Favatar: favatar, Picons: picons}
			m[i] = c;
		}
	}

	return &Entry{Title: title, Body: template.HTML(ParseEmoticons(body)), Comments: m, Toc: toc}, nil
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
				panic("Cannot decode b64")
			}

			r := bytes.NewReader(unbased)
			im, err := png.Decode(r)
			if err != nil {
				panic(err)
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
			//	return
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

var fns = template.FuncMap{
	"plus1": func(x int) int {
		return x + 1
	},
}

var templates = template.Must(template.New("tmpls").Funcs(fns).ParseFiles("tmpl/edit.html", "tmpl/view.html", "tmpl/entries.html"))
var titleValidator = regexp.MustCompile("^[a-zA-Z0-9]+$")

// cache control
var (
	cacheSince = time.Now().Format(http.TimeFormat)
	cacheUntil = time.Now().AddDate(60, 0, 0).Format(http.TimeFormat)
)

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

func main() {
	// start the server
	fmt.Println("starting")
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
	http.Handle("/css/", gziphandler.GzipHandler(StaticHandler(http.StripPrefix("/css/", http.FileServer(http.Dir("css"))))))
	http.Handle("/img/", gziphandler.GzipHandler(StaticHandler(http.StripPrefix("/img/", http.FileServer(http.Dir("img"))))))
	http.Handle("/face/", gziphandler.GzipHandler(StaticHandler(http.StripPrefix("/face/", http.FileServer(http.Dir("face"))))))
	http.Handle("/js/", gziphandler.GzipHandler(StaticHandler(http.StripPrefix("/js/", http.FileServer(http.Dir("js"))))))
	http.HandleFunc("/favicon.ico", faviconHandler)
	//http.Handle("/files/", http.StripPrefix("/files/", http.FileServer(http.Dir("files"))))
	//http.Handle("/files/", http.StripPrefix("/files/", http.FileServer(http.Dir("D:\\Music\\Conor Oberst"))))
	http.ListenAndServe(":8080", nil)
}

// basic helper handlers

func rootHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/entries/main", http.StatusFound)
}

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "img/favicon.ico")
}


func StaticHandler(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age:290304000, public")
		w.Header().Set("Last-Modified", cacheSince)
		w.Header().Set("Expires", cacheUntil)
		h.ServeHTTP(w, r)
    }

    return http.HandlerFunc(fn)
}

// gzip
// https://play.golang.org/p/80HukFxfs4
// https://github.com/NYTimes/gziphandler

func getSize(a, b, c int) int {
	d := a * b / c
	return (d + 1) & -1
}

// https://en.wikipedia.org/wiki/Wikipedia:Tutorial/Formatting
// https://upload.wikimedia.org/wikipedia/commons/b/b3/Wiki_markup_cheatsheet_EN.pdf
// https://en.wikipedia.org/wiki/Help:Wiki_markup
// https://github.com/adam-p/markdown-here/wiki/Markdown-Cheatsheet
func breakToPara(s string) string {
	// internal wiki image
	var re = regexp.MustCompile(`\[\[File:(.*?)\|(thumb(?:nail)?)\|(.*?)\]\]`)
	s = re.ReplaceAllString(s, `<img class="right" src="/img/$1" alt="$3">`)

	re = regexp.MustCompile(`(?m)^%(.*)`)
	var ree = regexp.MustCompile(`(?m)^#(.*)`)
	var reList = regexp.MustCompile(`(?m)^;(.*)`)
	var reSubList = regexp.MustCompile(`(?m)^:(.*)`)

	var reImgSpace = regexp.MustCompile(`src="\/img\/(.*?)"`)

	s = strings.Replace(s, "\r", "" , -1)

	strArr := strings.Split(s,"\n")
	counter := 0
	match := false

	counterOrd := 0
	matchOrd := false

	counterList := 0
	matchList := false
	for i, v := range strArr {
		if(reImgSpace.MatchString(v)) {
			first := v[0:strings.Index(v, "src")]
			second := reImgSpace.FindString(v)
			second =  strings.Replace(second, " ", "_", -1)
			third := v[strings.Index(v, "alt"):]
			imgFileName := reImgSpace.ReplaceAllString(second, `$1`)

			if _, err := os.Stat("img/"+imgFileName); !os.IsNotExist(err) {
				// open "test.jpg"
				file, err := os.Open("img/"+imgFileName)
				if err != nil {
					fmt.Println("file does not exist")
				}

				// decode jpeg into image.Image
				img, err := jpeg.Decode(file)
				if err != nil {
					fmt.Println("file does not exist")
				}

				file.Close()

				g := img.Bounds().Size()

				srcW := g.X
				srcH := g.Y

				w, h := 220, getSize(220, g.Y, g.X)
				if g.X < g.Y {
					w, h = getSize(220, g.X, g.Y), 220
				}
				_, err = os.Stat("img/220px-"+imgFileName[0:strings.Index(imgFileName, ".")]+".jpg");
				if (g.X > 220) && os.IsNotExist(err) {
					fmt.Println("resizing")
					/*src, ok := img.(*image.YCbCr)
					if !ok {
						fmt.Println("input picture is not ycbcr")
					}
					var resized image.Image
					resized = image.NewYCbCr(image.Rect(0, 0, w, h), src.SubsampleRatio)

					rez.Convert(resized, img, rez. NewBicubicFilter())
					//rez.Convert(resized, img, filter{})

					out, err := os.Create("img/220px-"+imgFileName[0:strings.Index(imgFileName, ".")]+".jpg")
					if err != nil {
						fmt.Println(err)
					}
					defer out.Close()*/


					resized := resize.Thumbnail(uint(w), uint(h), img, 2)

					out, err := os.Create("img/220px-"+imgFileName[0:strings.Index(imgFileName, ".")]+".jpg")
					if err != nil {
						fmt.Println(err)
					}
					defer out.Close()

					// https://meta.wikimedia.org/wiki/Thumbnails
					// https://www.mediawiki.org/wiki/Manual:Image_administration#Image_thumbnailing
					// https://golang.org/pkg/image/jpeg/
					// https://en.wikipedia.org/wiki/Image_scaling
					// unfortunately, image/jpeg does not support 4:4:4 chroma subsampling, so even a jpeg of quality of 100 will have washed out colors
					//jpeg.Encode(out, resized, &jpeg.Options{Quality: 100})

					png.Encode(out, resized)
				}
				_, err = os.Stat("img/220px-"+imgFileName[0:strings.Index(imgFileName, ".")]+".jpg");
				if !os.IsNotExist(err) {
					second = strings.Replace(second, `src="/img/`, `src="/img/220px-`, -1)
					second = second + ` data-file-width="` + strconv.Itoa(srcW) + `" data-file-height="` + strconv.Itoa(srcH) + `" `
				}
			}
			strArr[i] = `<a href="/img/`+ imgFileName +`">` + first + " " + second + " " + third + "</a>"
		}
		if re.MatchString(v) {
			strArr[i] = re.ReplaceAllString(v, `<li>$1</li>`)
			if counter == 0 {
				strArr[i] = "<ul>" + strArr[i]
			}
			match = true
			counter++
		} else if match == true {
			strArr[i-1] = strArr[i-1] + "</ul>"
			match = false
			counter = 0
		}

		if ree.MatchString(v) {
			strArr[i] = ree.ReplaceAllString(v, `<li>$1</li>`)
			if counterOrd == 0 {
				strArr[i] = "<ol>" + strArr[i]
			}
			matchOrd = true
			counterOrd++
		} else if matchOrd == true {
			strArr[i-1] = strArr[i-1] + "</ol>"
			matchOrd = false
			counterOrd = 0
		}

		if reList.MatchString(v) {
			var reSingleLine = regexp.MustCompile(`^;([^:]*):(.*)+`)
			if reSingleLine.MatchString(v) {
				strArr[i] = reSingleLine.ReplaceAllString(v, `<dl><dt>$1</dt><dd>$2</dd></dl>`)
			} else {
				strArr[i] = reList.ReplaceAllString(v, `<dt>$1</dt>`)


				if counterList == 0 {
					strArr[i] = "<dl>" + strArr[i]
				}
				matchList = true
				counterList++
			}
		} else if matchList == true && reSubList.MatchString(v) {
			strArr[i] = reSubList.ReplaceAllString(v, `<dd>$1</dd>`)
		} else if matchList == true {
			strArr[i-1] = strArr[i-1] + "</dl>"
			matchList = false
			counterList = 0
		}
	}

	if(match == true) {
		strArr[len(strArr)-1] = strArr[len(strArr)-1] + "</ul>"
	}

	if(matchOrd == true) {
		strArr[len(strArr)-1] = strArr[len(strArr)-1] + "</ol>"
	}

	s = strings.Join(strArr, "\n")

	re = regexp.MustCompile(`----`)
	s = re.ReplaceAllString(s, `<hr>`)

	re = regexp.MustCompile(`(?m)^\$ (.*)`)
	s = re.ReplaceAllString(s, `<blockquote>$1</blockquote>`)

	re = regexp.MustCompile(`\*\*\*(.*?)\*\*\*`)
	s = re.ReplaceAllString(s, `<b><i>$1</i></b>`)

	re = regexp.MustCompile(`\*\*(.*?)\*\*`)
	s = re.ReplaceAllString(s, `<b>$1</b>`)

	re = regexp.MustCompile(`\*(.*?)\*`)
	s = re.ReplaceAllString(s, `<i>$1</i>`)

	// embed youtube urls
	re = regexp.MustCompile(`https[:]\/\/www.youtube.com\/watch\?v=([a-zA-Z0-9_]*)`)
	s = re.ReplaceAllString(s, `<object style="width:100%;height:100%;width:420px;height:315px;float:none;clear:both;margin:2px auto;" data="http://www.youtube.com/embed/$1"></object>`)

	// replace email with mailto
	re = regexp.MustCompile(`\w+@\w+\.\w+(\.\w+)?`)
	s = re.ReplaceAllString(s, `<a href="mailto:$0">$0</a>`)

	// markdown force unwrap image
	re = regexp.MustCompile(`!\[(.*?)\]\((.*?)\)`)
	s = re.ReplaceAllString(s, `<img src="/img/$2" alt="$1">`)

	re = regexp.MustCompile(`\[\[([^\]\[:]+)\|([^\]\[:]+)\]\]`)
	s = re.ReplaceAllString(s, `<a href="/entries/$1">$2</a>`)

	urlReg := `(https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=;!]{2,256}\.[a-z]{2,6}\b([-a-zA-Z0-9@:%_\+.~#?&//=;!]*))`

	re = regexp.MustCompile(`\[`+urlReg+`\s([^\]]+)\]`)
	s = re.ReplaceAllString(s, `<a class="external" href="$1">$4</a>`)

	re = regexp.MustCompile(`\[?`+urlReg+`\]?( |\n)`)
	s = re.ReplaceAllString(s, `<a class="external" href="$1">$1</a>$4`)

	re = regexp.MustCompile(`(?m)`+urlReg+`$`)
	s = re.ReplaceAllString(s, `<a class="external" href="$1">$1</a>`)

	re = regexp.MustCompile(`\[([a-z]+)\]\(([a-zA-Z0-9_\/:.-;!]+)\)`)
	s = re.ReplaceAllString(s, `<a class="external" href="$2">$1</a>`)

	re = regexp.MustCompile(`\[\[(.*?)\]\]`)
	s = re.ReplaceAllString(s, `<a href="/entries/$1">$1</a>`)

	//re = regexp.MustCompile(`([^"])(https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=;!]{2,256}\.[a-z]{2,6}\b([-a-zA-Z0-9@:%_\+.~#?&//=;!]*))`)
	//s = re.ReplaceAllString(s, `$1<a class="external" href="$2">$2</a>`)

	s = strings.Replace(s, "\n", "\n<p>" , -1)

	re = regexp.MustCompile(`(?m)^<p>$`)
	s = re.ReplaceAllString(s, ``)

	re = regexp.MustCompile(`<p>(<h[1-5].*)`)
	s = re.ReplaceAllString(s, `$1`)


	/*
	words := strings.Fields(s)
	for i, word := range words {
		fmt.Println(word)
		_, err := url.ParseRequestURI(word)
		if err == nil {
			fmt.Println(i, " => ", word)
		}
	}*/

	return s
}

func parseToEntry(s string) (string, [][]string) {
	var h5 = regexp.MustCompile(`=====(.*?)=====`)
	var h4 = regexp.MustCompile(`====(.*?)====`)
	var h3 = regexp.MustCompile(`===(.*?)===`)
	var h2 = regexp.MustCompile(`==(.*?)==`)

	s = h5.ReplaceAllString(s, `<h5 id="$1">$1</h5>`)
	s = h4.ReplaceAllString(s, `<h4 id="$1">$1</h4>`)

	toc := [][]string{}
	words := strings.Fields(s)
	counter := -1

	for i, word := range words {
		if len(word) > 4 {
			if word[0:2] == "==" && word[len(word)-2:len(word)] != "==" {
				for k:=i; word[len(word)-2:len(word)] != "=="; k++ {
					word += " " + words[k+1]
				}
			}
		}
		if h3.MatchString(word) {
			if counter > -1 {
				toc[counter] = append(toc[counter], h3.FindStringSubmatch(word)[1])
			}
		} else if h2.MatchString(word) {
			toc = append(toc, []string{h2.FindStringSubmatch(word)[1]})
			counter++
		}
	}

	s = h3.ReplaceAllString(s, `<h3 id="$1">$1</h3>`)
	s = h2.ReplaceAllString(s, `<h2 id="$1">$1</h2>`)

	return s, toc
}

// replace emoticon markup with html
func ParseEmoticons(s string) string {
	e_path := "<img src=/img/emoticons/"
	s = strings.Replace(s,":angry:",e_path + "angry.gif>",-1)
	//s = strings.Replace(s,">:(",e_path + "angry.gif>",-1)
	s = strings.Replace(s,":laugh:",e_path + "laugh.gif>",-1)
	s = strings.Replace(s,":DD",e_path + "laugh.gif>",-1)
	s = strings.Replace(s,":yell:",e_path + "yell.gif>",-1)
	//s = strings.Replace(s,">:O",e_path + "yell.gif>",-1)
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
	s = strings.Replace(s,":\\",e_path + "blush.gif>",-1)
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
	s = strings.Replace(s,"&lt;/p&gt;", "</p>", -1)
	s = strings.Replace(s,"&lt;p&gt;", "<p>", -1)

	return s
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

DONE:
s|https[:]\/\/www.youtube.com\/watch\?v=([a-zA-Z0-9_]*)|<object style="width:100%;height:100%;width:420px;height:315px;float:none;clear:both;margin:2px auto;" data="http:\/\/www.youtube.com\/embed\/\1"><\/object>|g;
s|[[:space:]](http[:]//[^ ]*[a-zA-Z])[[:space:]]| <a href=\"\1\">\1</a> |g;  # replace urls  html urls
s|\w+@\w+\.\w+(\.\w+)?|<a href=\"mailto:\0\">\0</a>|g;s/\//\\\//g' $2);  # email mailto
*/
}
