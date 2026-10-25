package server

import (
	"context"
	"encoding/json/v2"
	"errors"
	"log"
	"net/http"
	"onibi/application"
	"strings"

	app_reg "onibi/applications_registry"
	"onibi/checker"
	user_reg "onibi/users_registry"
)

type API struct {
	Applications       app_reg.ApplicationStore
	Users              *user_reg.UsersRegistry
	LegalInfoCollector checker.FaceChecker
	MaxReadBytes       int64
}

func (a *API) GetMux() http.Handler {
	authNeeded := http.NewServeMux()
	authNeeded.HandleFunc("GET /applications", a.ShowApplications)
	authNeeded.HandleFunc("POST /applications", a.AddApplication)
	authNeeded.HandleFunc("GET /applications/{id}", a.ShowApplication)
	authNeeded.HandleFunc("PATCH /applications/{id}", a.ChangeAplication)
	authNeeded.HandleFunc("POST /applications/{id}/submit", a.Submit)

	root := http.NewServeMux()
	root.HandleFunc("POST /auth/login", a.Login)
	root.HandleFunc("POST /auth/registration", a.Registration)
	root.HandleFunc("POST /ergul", checker.Mock{}.ErgulEndpoint)
	root.HandleFunc("POST /bki", checker.Mock{}.BkiEndpoint)
	root.Handle("/applic/", a.auth(authNeeded))
	return root
}

func (a *API) ShowApplications(w http.ResponseWriter, r *http.Request) {
	l, err := a.Applications.List(r.Context(), getUserID(r))
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	err = json.MarshalWrite(w, map[string][]application.Application{"applications": l})
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}
func (a *API) ShowApplication(w http.ResponseWriter, r *http.Request) {
	app, err := a.Applications.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	err = json.MarshalWrite(w, app)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}

type Application struct {
	Inn  string `json:"inn"`
	Loan int64  `json:"loan"`
	Term uint64 `json:"term"`
	Goal string `json:"goal"`
}

func (a *API) AddApplication(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, a.MaxReadBytes)
	data := Application{}
	err := json.UnmarshalRead(r.Body, &data, json.RejectUnknownMembers(true))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	} // more likely something wrong with data user passed that the server
	app, err := application.NewApplicaton(data.Inn, data.Loan, data.Term, data.Goal, getUserID(r))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := a.Applications.Add(r.Context(), app); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (a *API) Submit(w http.ResponseWriter, r *http.Request) {
	app, err := a.Applications.Transition(r.Context(), r.PathValue("id"), getUserID(r), application.Submitted)
	if err != nil {
		if errors.Is(err, app_reg.NotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		} else {
			http.Error(w, "conflict", http.StatusConflict)
		}
	}
	// may return something ?

	a.process(r.Context(), app)
}

func (a *API) ChangeAplication(w http.ResponseWriter, r *http.Request) {}

// func (a *API) DeleteApplication(w http.ResponseWriter, r *http.Request) {}

type UserRegJSON struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a *API) Registration(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, a.MaxReadBytes)
	data := UserRegJSON{}
	err := json.UnmarshalRead(r.Body, &data, json.RejectUnknownMembers(true))
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if err := a.Users.AddUser(data.Email, data.Name, data.Password); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

type UserLogJSON struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a *API) Login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, a.MaxReadBytes)
	data := UserLogJSON{}
	err := json.UnmarshalRead(r.Body, &data, json.RejectUnknownMembers(true))
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		log.Printf("while unmarshaling request: %v", err)
		return
	}

	user, err := a.Users.Authenticate(data.Email, data.Password)
	if err != nil {
		http.Error(w, "bad email or password", http.StatusUnauthorized)
		return
	}

	tok, err := a.Users.IssueToken(user.ID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := json.MarshalWrite(w, map[string]string{"token": tok}); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}

type ctxKey struct{}

func (a *API) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		userID, err := a.Users.ParseToken(token)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), ctxKey{}, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getUserID(r *http.Request) string {
	id, _ := r.Context().Value(ctxKey{}).(string)
	return id
}

// idk what to do with errors
// warn admin
// maybe before try a couple of times
func (a *API) process(ctx context.Context, app application.Application) {
	// curent stage submitted
	app, err := a.submitStage(ctx, app)
	if err != nil {
		return
	}
	// current stage validating
	app, err = a.validatingStage(ctx, app)
	if err != nil || app.State == application.Rejected {
		return
	}

	app, err = a.validatingStage(ctx, app)
	if err != nil || app.State != application.Approved {
		return
	}
	// next there manual steps
}

func (a *API) scoringStage(ctx context.Context, app application.Application) (application.Application, error) {
	// scoring magic
	return a.Applications.Transition(ctx, app.ID, app.OwnerID, application.Approved)
}

var (
	NotReal error = errors.New("not real")
)

func (a *API) validatingStage(ctx context.Context, app application.Application) (application.Application, error) {
	if !app.LegalInfo.Ergul.Found {
		return a.Applications.Transition(ctx, app.ID, app.OwnerID, application.Rejected)
	} // and so on, this needs consideration

	return a.Applications.Transition(ctx, app.ID, app.OwnerID, application.Scoring)
}

func (a *API) submitStage(ctx context.Context, app application.Application) (application.Application, error) {
	legalInfo, err := a.LegalInfoCollector.Check(ctx, app.Inn)
	if err != nil {
		return application.Application{}, nil
	}
	a.Applications.Update(ctx, &application.Application{LegalInfo: &legalInfo})

	return a.Applications.Transition(ctx, app.ID, app.OwnerID, application.Validating)
}
