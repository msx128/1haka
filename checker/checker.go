package checker

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"io"
	"log"
	"net/http"
	"onibi/application"
	"time"
)

// Сюда ергюл и бки checkers
type FaceChecker interface {
	Check(ctx context.Context, inn string) (application.LegalInfo, error)
}

// should add stuct on every vendor change

type Mock struct {
	ergulUrl, bkiUrl string
}

// show cases

type InnJSON struct {
	Inn string `json:"inn"`
}

func (Mock) ErgulEndpoint(w http.ResponseWriter, r *http.Request) {
	inn := InnJSON{}
	err := json.UnmarshalRead(r.Body, &inn)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
	if inn.Inn == "123456789010" {
	}
}
func (Mock) BkiEndpoint(w http.ResponseWriter, r *http.Request) {
	inn := InnJSON{}
	err := json.UnmarshalRead(r.Body, &inn)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
	if inn.Inn == "123456789010" {
	}
}

func (m Mock) Check(ctx context.Context, inn string) (application.LegalInfo, error) {
	client := http.Client{Timeout: 15 * time.Second}

	ij, err := json.Marshal(inn)
	if err != nil {
		return application.LegalInfo{}, err
	}
	innJson := bytes.NewBuffer(ij)

	ergul, err := m.getErgul(&client, innJson, m.ergulUrl)
	if err != nil {
		return application.LegalInfo{}, err
	}
	bki, err := m.getBki(&client, innJson, m.bkiUrl)
	if err != nil {
		return application.LegalInfo{}, err
	}

	return application.LegalInfo{Ergul: ergul, Bki: bki}, nil
}

func (Mock) getErgul(client *http.Client, inn io.Reader, url string) (*application.ErgulJSON, error) {
	ergulReq, err := http.NewRequest(http.MethodPost, url, inn)
	if err != nil {
		return nil, err
	}
	ergulResp, err := client.Do(ergulReq)
	if err != nil {
		log.Printf("failed to get from legal source: %v", err)
		return nil, err
	}
	ergulOut := application.ErgulJSON{}
	if err := json.UnmarshalRead(ergulResp.Body, &ergulOut); err != nil {
		return nil, err
	}
	return &ergulOut, nil
}
func (Mock) getBki(client *http.Client, inn io.Reader, url string) (*application.BkiJSON, error) {
	bkiReq, err := http.NewRequest(http.MethodPost, url, inn)
	if err != nil {
		return nil, err
	}
	bkiResp, err := client.Do(bkiReq)
	if err != nil {
		log.Printf("failed to get from legal source: %v", err)
		return nil, err
	}
	bkiOut := application.BkiJSON{}
	if err := json.UnmarshalRead(bkiResp.Body, &bkiOut); err != nil {
		return nil, err
	}
	return &bkiOut, nil
}
