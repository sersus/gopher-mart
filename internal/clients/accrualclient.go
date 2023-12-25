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
