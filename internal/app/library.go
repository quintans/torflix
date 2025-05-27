package app

type LibraryAuth interface {
	PollForToken(deviceCodeResponse DeviceCodeResponse) (TokenResponse, error)
	GetDeviceCode() (DeviceCodeResponse, error)
}

type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURL string `json:"verification_url"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	CreatedAt    int    `json:"created_at"`
}

type Library interface {
	Search(query string) ([]SearchResult, error)
}

type SearchResult struct {
	Type  string  `json:"type"`
	Score float64 `json:"score"`
	Movie *Movie  `json:"movie,omitempty"`
	Show  *Show   `json:"show,omitempty"`
}

type Movie struct {
	Title string `json:"title"`
	Year  int    `json:"year"`
	IDs   IDs    `json:"ids"`
}

type Show struct {
	Title string `json:"title"`
	Year  int    `json:"year"`
	IDs   IDs    `json:"ids"`
}

type IDs struct {
	Trakt int    `json:"trakt"`
	Slug  string `json:"slug"`
	IMDB  string `json:"imdb"`
	TMDB  int    `json:"tmdb"`
}
