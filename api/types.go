package api

type Vacancy struct {
	ID string `json:"id"`
}

type VacancyResponse struct {
	Items []Vacancy `json:"items"`
	Page  int       `json:"page"`
	Pages int       `json:"pages"`
	Found int       `json:"found"`
}

type Detail struct {
	Name        string     `json:"name"`
	Skills      []KeySkill `json:"key_skills"`
	Experiences Experience `json:"experience"`
	Description string     `json:"description"`
	Salary      Salary     `json:"salary"`
	Date        string     `json:"created_at"`
	Archived    bool       `json:"archived"`
	URL         string     `json:"alternate_url"`
}

type Salary struct {
	From interface{} `json:"from"`
	To   interface{} `json:"to"`
}

type Area struct {
	Name string `json:"name"`
}

type KeySkill struct {
	Name string `json:"name"`
}

type Experience struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

type Resume struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}
