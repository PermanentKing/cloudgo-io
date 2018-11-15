package service

import (
	"fmt"
	"net/http"

	"github.com/unrolled/render"
)

func loginHandler(formatter *render.Render) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		// user parseform to show in backend
		r.ParseForm()
		fmt.Println("username:", r.Form["username"])
		fmt.Println("password:", r.Form["password"])

		// return the message
		formatter.HTML(w, http.StatusOK, "result", struct {
			Username []string `json:"username"`
			Password []string `json:"password"`
		}{Username: r.Form["username"], Password: r.Form["password"]})
	}
}
