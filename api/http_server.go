package api

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"jobSearchServer/internal/env"
	"log"
	"net/http"
	"runtime"
	"sync"
	"time"
)

type HTTPHandler struct {
	hClient   HHClient
	cfg       env.Config
	Vacancies []Vacancy
	IdResume  string
}

func NewHTTPHandler(client HHClient, cfg env.Config) *HTTPHandler {
	return &HTTPHandler{hClient: client, cfg: cfg}
}

func (h *HTTPHandler) StartServer() error {
	r := gin.Default()
	r.POST("/auth/authorization", h.authorization)
	r.GET("/auth/callback", h.callback)
	r.POST("/vacancy/response", h.response)
	r.GET("/vacancies/", h.searchVacancies)

	server := &http.Server{
		Addr:         h.cfg.HttpPort,
		Handler:      r,
		ReadTimeout:  h.cfg.ReadTimeout * time.Second,
		WriteTimeout: h.cfg.WriteTimeout * time.Second,
	}
	err := server.ListenAndServe()
	if err != nil {
		return fmt.Errorf("error starting server on port 8080: %w", err)
	}

	return nil
}

func (h *HTTPHandler) callback(c *gin.Context) {
	authCode := c.Query("code")
	err := h.hClient.Authorization(authCode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.hClient.SendResumeToBot()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"auth": "Authorization success"})
}

func (h *HTTPHandler) authorization(c *gin.Context) {
	chatId := c.Query("chat_id")
	authURL := fmt.Sprintf("https://hh.ru/oauth/authorize?response_type=code&client_id=%s&redirect_uri=%s",
		h.hClient.clientId, "http://localhost:8081/auth/callback")

	h.hClient.ChatId = chatId
	c.JSON(http.StatusOK, gin.H{"message": authURL})

}

func (h *HTTPHandler) refreshAccess(c *gin.Context) {
	resp, err := h.hClient.client.R().SetQueryParams(map[string]string{
		"grant_type":     "authorization_code",
		"refresh_token ": h.hClient.userData.RefreshToken,
		"client_id":      h.hClient.clientId,
		"client_secret":  h.hClient.secretKey,
	}).Post("https://api.hh.ru/token")

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	info := map[string]interface{}{}
	err = json.Unmarshal(resp.Body(), &info)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
	c.JSON(http.StatusOK, info)
}

func (h *HTTPHandler) response(c *gin.Context) {
	resumeId := c.Query("resume_id")

	if !h.hClient.IsValidResume(resumeId) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid resume_id"})
		return
	}

	for _, vacancy := range h.Vacancies {
		go func() {
			err := h.hClient.ResponceVacancy(resumeId, vacancy.ID)
			if err != nil {
				err = fmt.Errorf("error responding to vacancy %s : %s", vacancy.ID, err)
				c.JSON(http.StatusBadRequest, gin.H{"error": err})
				return
			}
		}()
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("successfully fetched %d Vacancy", len(h.Vacancies))})
}

func (h *HTTPHandler) searchVacancies(c *gin.Context) {
	title := c.Query("title")

	vacancies, err := h.hClient.SearchVacancies(title)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err})
		return
	}

	if h.hClient.userData.RefreshToken != "" && h.hClient.userData.AccessToken != "" {
		sortedVacancies, err := h.checkIfResponded(vacancies)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": err})
			return
		}
		h.Vacancies = sortedVacancies
		log.Printf("vacancies have been sent %v", len(h.Vacancies))
		c.JSON(http.StatusOK, gin.H{"result": sortedVacancies})
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": vacancies})
}

func (h *HTTPHandler) optimizeDetailVacancies(vacancies []Vacancy, client *HHClient) {
	workerNums := runtime.NumCPU()
	jobs := make(chan Vacancy, workerNums)
	results := make(chan Detail, workerNums)
	wg := &sync.WaitGroup{}

	go func() {
		for _, vacancy := range vacancies {
			jobs <- vacancy
		}

		close(jobs)
	}()

	for i := 0; i < workerNums; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			getDetail(client, jobs, results)
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	index := 0
	for detail := range results {
		index++
		fmt.Println("index:", index, detail)
	}
}

func getDetail(client *HHClient, jobs <-chan Vacancy, result chan<- Detail) {
	for job := range jobs {
		detail, err := client.GetVacancyDetail(job.ID)
		if err != nil {
			log.Fatalf("Error getting vacancy detail: %v", err)
		}

		result <- detail
	}
}

func (h *HTTPHandler) checkIfResponded(vacancies []Vacancy) ([]Vacancy, error) {
	resp, err := h.hClient.client.R().
		SetHeader("Authorization", "Bearer "+h.hClient.userData.AccessToken).
		Get("https://api.hh.ru/negotiations")

	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to get negotiations: status code %d, response: %s", resp.StatusCode(), resp.String())
	}

	var result VacancyResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	sortedVacancies := make([]Vacancy, 0)
	for _, vacancy := range vacancies {
		if !compareVacancy(vacancy.ID, result.Items) {
			sortedVacancies = append(sortedVacancies, vacancy)
		}
	}

	return sortedVacancies, nil
}

func compareVacancy(vacancyId string, vacancies []Vacancy) bool {
	for _, vacancy := range vacancies {
		if vacancy.ID == vacancyId {
			return true
		}
	}

	return false
}
