package handlers

type credentialsBody struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}
