package json_api

import (
	"encoding/json"
	"github.com/genya0407/confession-server/usecase"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetAccountInfo(t *testing.T) {
	mockname := "Mock name"
	accountID, err := uuid.NewUUID()
	var uc = func(id uuid.UUID) usecase.AccountInfoDTO {
		if id != accountID {
			panic("unexpected id")
		}
		return usecase.AccountInfoDTO{
			AccountID: accountID,
			Name:      mockname,
		}
	}

	if err != nil {
		panic(err.Error())
	}
	handler := GetAccountInfoGenerator(uc)
	router := httprouter.New()
	router.GET("/account/:account_id", handler)

	req := httptest.NewRequest("GET", "http://confession.com/account/"+accountID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Error("Invalid status")
	}

	result := &AccountJSON{}
	err = json.NewDecoder(w.Body).Decode(result)
	if err != nil {
		t.Error(err.Error())
	}

	if result.AccountID != accountID {
		t.Error("Invalid result id")
	}
	if result.Name != mockname {
		t.Error("Invalid result name")
	}
}
