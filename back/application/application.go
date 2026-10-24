package application

import (
	"fmt"
	"slices"

	"github.com/google/uuid"
)

// when proccesing we need to populate LegalInfo
// if not exit abort
type LegalInfo struct {
	Ergul *ErgulJSON
	Bki   *BkiJSON
}
type ErgulJSON struct {
	RequestId string     `json:"requestId"`
	Found     bool       `json:"found"`
	Data      *ErgulData `json:"data"`
}
type ErgulData struct {
	Inn              string   `json:"inn"`
	Ogrn             string   `json:"ogrn"`
	Kpp              string   `json:"kpp"`
	Name             string   `json:"name"`
	Status           string   `json:"status"`
	RegistrationDate string   `json:"registrationDate"`
	Address          string   `json:"address"`
	Okved            []string `json:"okved"`
}

type BkiJSON struct {
	RequestId string     `json:"requestId"`
	Status    string     `json:"status"`
	Report    *BkiReport `json:"report"`
}
type BkiReport struct {
	CreditHistoryFound bool           `json:"creditHistoryFound"`
	CreditAccounts     []*BkiAccaunts `json:"creditAccounts"`
	ReportDate         string         `json:"reportDate"`
}
type BkiAccaunts struct {
	Type        string `json:"type"`
	Status      string `json:"status"`
	OverdueDays int    `json:"overdueDays"`
}

// func turning all info into credit score

type Application struct {
	// This should not be changed by hand
	ID      string
	OwnerID string
	Inn     string
	State   Status
	Loan    int64
	Term    uint64 // in days I suppose
	Goal    string
	// this can and should be changing
	LegalInfo *LegalInfo
	// why make it no exportable, if it nil marshal whould just make empty field
}

func NewApplicaton(inn string,
	loan int64,
	term uint64,
	goal string,
	ownerID string) (*Application, error) {
	if err := validateInn(inn); err != nil {
		return nil, err
	}
	return &Application{ID: uuid.NewString(),
			OwnerID: ownerID,
			Inn:     inn, State: Draft,
			Loan: loan, Term: term,
			Goal: goal, LegalInfo: nil},
		nil
}

func (a *Application) Transition(next Status) error {
	if !slices.Contains(StateRuleMap[a.State], next) {
		return fmt.Errorf("cannot switch from %v to %v", a.State, next)
	}
	a.State = next
	return nil
}

type Status string

const (
	Draft      Status = "Draft"
	Submitted  Status = "Submitted"
	Validating Status = "Validating"
	Scoring    Status = "Scoring"

	Approved     Status = "Approved"
	Rejected     Status = "Rejected"
	NeedsInfo    Status = "NeedsInfo"
	ManualReview Status = "ManualReview"
	CounterOffer Status = "CounterOffer"

	Accepted  Status = "Accepted"
	Signed    Status = "Signed"
	Disbursed Status = "Disbursed"
)

var StateRuleMap = map[Status][]Status{
	Draft:      {Submitted},
	Submitted:  {Validating},
	Validating: {Scoring, Rejected},
	Scoring:    {Approved, CounterOffer, Rejected, NeedsInfo, ManualReview},
	Approved:   {Accepted},
	Accepted:   {Signed},
	Signed:     {Disbursed},
}

func validateInn(inn string) error {
	// I forgot we only on legal or also on phys
	l := len(inn)
	if !(l == 10 || l == 12) {
		return fmt.Errorf("bad inn")
	}
	return nil
}
