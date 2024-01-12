package clients

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/sersus/gopher-mart/internal/config"
)

const (
	NEW        string = "NEW"
	REGISTERED string = "REGISTERED"
	INVALID    string = "INVALID"
	PROCESSING string = "PROCESSING"
	PROCESSED  string = "PROCESSED"
)

type OrderAccrual struct {
	OrderID      string  `json:"order"`
	Status       string  `json:"status"`
	AccrualValue float64 `json:"accrual"`
}

type OrderAccrualResponse struct {
	OrderAccrual *OrderAccrual
	Code         int
	Error        error
}
type AccrualClient struct {
	Address  string
	Client   *http.Client
	Interval time.Duration
}

// NewAccrualClient создает новый экземпляр AccrualClient
func NewAccrualClient(address string, interval time.Duration) *AccrualClient {
	transport := &http.Transport{
		MaxIdleConns:          10,               // Максимальное количество постоянных соединений
		IdleConnTimeout:       30 * time.Second, // Время простоя соединения
		TLSHandshakeTimeout:   10 * time.Second, // Таймаут рукопожатия TLS
		ExpectContinueTimeout: 1 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
	}

	return &AccrualClient{
		Address:  address,
		Client:   client,
		Interval: interval,
	}
}

// GetOrdersAccruals запускает горутины для получения данных о начислениях для каждого заказа
func (c *AccrualClient) GetOrdersAccruals(orderIDs []int64, accrualResponses *[]OrderAccrualResponse) {
	var wg sync.WaitGroup

	for _, orderID := range orderIDs {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			response, err := c.GetOrderAccrual(id)
			if err != nil {
				*accrualResponses = append(*accrualResponses, OrderAccrualResponse{Error: err})
				return
			}
			*accrualResponses = append(*accrualResponses, *response)
		}(orderID)
	}

	wg.Wait()
}

// GetOrderAccrual выполняет запрос на получение данных о начислении для конкретного заказа
func (c *AccrualClient) GetOrderAccrual(orderID int64) (*OrderAccrualResponse, error) {
	reqURL := fmt.Sprintf("%s/api/orders/%s", c.Address, strconv.FormatInt(orderID, 10))
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &OrderAccrualResponse{Code: resp.StatusCode, Error: errors.New("non-ok status code received")}, nil
	}

	var accrual OrderAccrual
	if err := json.NewDecoder(resp.Body).Decode(&accrual); err != nil {
		return nil, err
	}

	return &OrderAccrualResponse{OrderAccrual: &accrual, Code: resp.StatusCode}, nil
}

// StartAccrualRoutine запускает рутину для периодических запросов к системе начислений
func (c *AccrualClient) StartAccrualRoutine(orderIDs []int64) {
	ticker := time.NewTicker(c.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			var accrualResponses []OrderAccrualResponse
			c.GetOrdersAccruals(orderIDs, &accrualResponses)
			// Обработка полученных данных о начислениях
			// ...
		}
	}
}

func GetOrdersAccruals(orderIDs []int64, accrualResponses *[]OrderAccrualResponse, wg *sync.WaitGroup) {
	for i := 0; i < len(orderIDs); i++ {
		go GetOrderAccrual(orderIDs[i], accrualResponses, wg)
	}
}

func GetOrderAccrual(orderID int64, orderAccrualResponses *[]OrderAccrualResponse, wg *sync.WaitGroup) {
	defer wg.Done()

	var m sync.Mutex

	orderIDString := strconv.FormatInt(orderID, 10)

	fmt.Println(config.AccrualSystemAddress + "/api/orders/" + orderIDString)

	response, responseErr := http.Get(config.AccrualSystemAddress + "/api/orders/" + orderIDString)

	if responseErr != nil {
		m.Lock()
		*orderAccrualResponses = append(*orderAccrualResponses, OrderAccrualResponse{Error: responseErr})
		m.Unlock()
		return
	}

	if response.StatusCode == http.StatusInternalServerError {
		m.Lock()
		*orderAccrualResponses = append(*orderAccrualResponses, OrderAccrualResponse{Code: http.StatusInternalServerError, Error: errors.New("error in the points calculation system")})
		m.Unlock()
		return
	}

	if response.StatusCode == http.StatusNoContent {
		stringOrderID := strconv.FormatInt(orderID, 10)

		m.Lock()
		*orderAccrualResponses = append(*orderAccrualResponses, OrderAccrualResponse{Code: http.StatusNoContent, OrderAccrual: &OrderAccrual{OrderID: stringOrderID, AccrualValue: 0, Status: NEW}})
		m.Unlock()
		return
	}

	if response.StatusCode == http.StatusTooManyRequests {
		var repeatWg sync.WaitGroup

		delay, err := strconv.ParseInt(response.Header.Get("Retry-After"), 10, 64)
		if err != nil {
			m.Lock()
			*orderAccrualResponses = append(*orderAccrualResponses, OrderAccrualResponse{Error: err, Code: http.StatusTooManyRequests})
			m.Unlock()
			return
		}

		repeatWg.Add(1)

		time.AfterFunc(time.Duration(delay)*time.Second, func() { go GetOrderAccrual(orderID, orderAccrualResponses, &repeatWg) })

		repeatWg.Wait()

		return
	}

	bodyBytes, readAllError := io.ReadAll(response.Body)
	response.Body.Close()
	if readAllError != nil {
		fmt.Println(readAllError)
		m.Lock()
		*orderAccrualResponses = append(*orderAccrualResponses, OrderAccrualResponse{Error: readAllError})
		m.Unlock()
		return
	}

	var accrual OrderAccrual

	if unmarshalErr := json.Unmarshal(bodyBytes, &accrual); unmarshalErr != nil {
		m.Lock()
		*orderAccrualResponses = append(*orderAccrualResponses, OrderAccrualResponse{Error: unmarshalErr})
		m.Unlock()
		return
	}

	m.Lock()
	*orderAccrualResponses = append(*orderAccrualResponses, OrderAccrualResponse{OrderAccrual: &accrual, Code: response.StatusCode})
	m.Unlock()
}
