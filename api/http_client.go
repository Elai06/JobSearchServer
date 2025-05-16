package api

import (
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"jobSearchServer/internal/env"
)

type HHClient struct {
	client      *resty.Client
	clientId    string
	secretKey   string
	authData    TokenResponse
	botUserData BotUserData
}

type BotUserData struct {
	ChatId   string `json:"chat_id"`
	UserName string `json:"user_id"`
}

type AuthorizationData struct {
}

func NewClient(cfg env.Config) *HHClient {
	return &HHClient{
		client:    resty.New(),
		clientId:  cfg.ClientId,
		secretKey: cfg.SecretKey,
	}
}

func (c *HHClient) SearchVacancies(text string) ([]Vacancy, error) {
	result, err := c.getPageVacancies(text, 0)
	if err != nil {
		return nil, err
	}
	allVacancies := make([]Vacancy, result.Found)
	allVacancies = append(allVacancies, result.Items...)
	totalPages := result.Found / 100

	if totalPages > 1 {
		for i := 1; i < totalPages; i++ {
			vacancies, err := c.getPageVacancies(text, i)
			if err != nil {
				fmt.Println("Error getting vacancies")
				continue
			}

			allVacancies = append(allVacancies, vacancies.Items...)
		}
	}

	return allVacancies, nil
}

func (c *HHClient) getPageVacancies(text string, page int) (*VacancyResponse, error) {
	resp, err := c.client.R().
		SetQueryParams(map[string]string{
			"text":       "name:" + text,
			"archived":   "false",
			"page":       fmt.Sprint(page),
			"per_page":   "100",
			"experience": "between1And3",
		}).Get("https://api.hh.ru/vacancies")

	if err != nil {
		return nil, fmt.Errorf("api call failed: %w", err)
	}

	result := VacancyResponse{}

	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("api call failed: %w", err)
	}
	return &result, nil
}

func (c *HHClient) GetVacancyDetail(vacancyId string) (Detail, error) {
	resp, err := c.client.R().Get(fmt.Sprintf("https://api.hh.ru/vacancies/%s", vacancyId))
	if err != nil {
		return Detail{}, fmt.Errorf("api call failed: %w", err)
	}

	detail := Detail{}

	if err := json.Unmarshal(resp.Body(), &detail); err != nil {
		return Detail{}, fmt.Errorf("api call failed: %w", err)
	}

	return detail, nil
}

func (c *HHClient) Authorization(authCode string) error {
	resp, err := c.client.R().SetQueryParams(map[string]string{
		"grant_type":    "authorization_code",
		"code":          authCode,
		"client_id":     c.clientId,
		"client_secret": c.secretKey,
		"redirect_uri":  "http://localhost:8081/auth/callback",
	}).Post("https://api.hh.ru/token")

	if err != nil {
		return fmt.Errorf("api call failed: %w", err)
	}

	detail := TokenResponse{}

	if err := json.Unmarshal(resp.Body(), &detail); err != nil {
		return fmt.Errorf("api call failed: %w", err)
	}

	if detail.AccessToken == "" {
		return fmt.Errorf("api call failed: %w", fmt.Errorf("access_token is empty"))
	}

	c.authData = detail
	return nil
}

func (c *HHClient) ResponseVacancy(idResume string, idVacancy string) error {
	detail, err := c.GetVacancyDetail(idVacancy)

	if err != nil {
		return fmt.Errorf("api call failed: %w", err)
	}
	fmt.Println(detail)

	c.client.SetHeader("User-Agent", "JobSearch/0.1 (dof312@gmail.com)")
	resp, err := c.client.R().SetHeader("Authorization", "Bearer "+c.authData.AccessToken).
		SetQueryParams(map[string]string{
			"vacancy_id": idVacancy,
			"resume_id":  idResume,
		}).Post("https://api.hh.ru/negotiations")

	if err != nil {
		return fmt.Errorf("api call failed: %w", err)
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("api call failed: %w", fmt.Errorf("status code %d", resp.StatusCode()))
	}

	return nil
}

func (c *HHClient) SendResumeToBot() error {
	resumes, err := c.GetResumes()

	resumesBody, err := json.Marshal(resumes)
	if err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}

	_, err = c.client.R().SetQueryParams(map[string]string{
		"resumes": string(resumesBody),
		"chat_id": c.botUserData.ChatId,
	}).Post("http://localhost:8080/bot/callback")

	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}

	return nil
}

func (c *HHClient) GetResumes() ([]Resume, error) {
	resp, err := c.client.R().
		SetHeader("Authorization", "Bearer "+c.authData.AccessToken).
		Get("https://api.hh.ru/resumes/mine")

	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}

	var result struct {
		Resumes []Resume `json:"items"`
	}
	if err = json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return result.Resumes, nil
}

func (c *HHClient) IsValidResume(resumeId string) bool {
	resumes, err := c.GetResumes()
	if err != nil {
		return false
	}

	for _, resume := range resumes {
		if resumeId == resume.ID {
			return true
		}
	}

	return false
}
