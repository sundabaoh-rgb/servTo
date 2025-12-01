package auth

type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) bool
}
