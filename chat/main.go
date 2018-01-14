package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"text/template"

	"github.com/stretchr/objx"

	"github.com/stretchr/gomniauth"
	"github.com/stretchr/gomniauth/providers/google"

	"github.com/takatori/go-programming-blue-prints/trace"
)

// 現在のアクティブなAvatarの実装
var avatars Avatar = TryAvatars{
	UseFileSystemAvatar,
	UseAuthAvatar,
	UseGravatar,
}

// templは一つのテンプレートを表す
type templateHandler struct {
	once     sync.Once
	filename string
	templ    *template.Template
}

// ServerHTTPはHTTPリクエストを処理する
func (t *templateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.once.Do(func() {
		t.templ = template.Must(template.ParseFiles(filepath.Join("templates", t.filename)))
	})
	data := map[string]interface{}{
		"Host": r.Host,
	}
	if authCookie, err := r.Cookie("auth"); err == nil {
		data["UserData"] = objx.MustFromBase64(authCookie.Value)
	}

	t.templ.Execute(w, data)
}

func main() {

	var addr = flag.String("addr", ":8080", "アプリケーションのアドレス")
	flag.Parse() // フラグを解釈します

	// Gomniauthのセットアップ
	gomniauth.SetSecurityKey("agaetwqpr3242413b")
	gomniauth.WithProviders(
		google.New("547548936277-k36evd5m8tsl4g3ed89ioo6nbpmfu3pi.apps.googleusercontent.com", "Ll-LAQfzRmpphWMxOpM3GLlh", "http://localhost:8080/auth/callback/google"),
	)

	r := newRoom()
	r.tracer = trace.New(os.Stdout)

	http.Handle("/chat", MustAuth(&templateHandler{filename: "chat.html"}))

	http.Handle("/login", &templateHandler{filename: "login.html"})
	http.Handle("/room", r)
	http.Handle("/upload", &templateHandler{filename: "upload.html"})
	http.HandleFunc("/uploader", uploadHandler)
	http.Handle("/avatars/", http.StripPrefix("/avatars/", http.FileServer(http.Dir("./avatars"))))
	http.HandleFunc("/auth/", loginHandler)
	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:   "auth",
			Value:  "",
			Path:   "/",
			MaxAge: -1, // クッキーを即座に削除する
		})
		w.Header()["Location"] = []string{"/chat"}
		w.WriteHeader(http.StatusTemporaryRedirect)
	})

	// チャットルームを開始します
	go r.run()

	// Webサーバーを開始する
	log.Println("Webサーバーを開始します。ポート: ", *addr)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
