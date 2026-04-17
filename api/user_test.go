package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	mockdb "github.com/simplebank/db/mock"
	db "github.com/simplebank/db/sqlc"
	"github.com/simplebank/token"
	"github.com/simplebank/util"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type eqCreateUserParamsMatcher struct {
	arg      db.CreateUserParams
	password string
}

func (e eqCreateUserParamsMatcher) Matches(x any) bool {
	arg, ok := x.(db.CreateUserParams)
	if !ok {
		return false
	}

	err := util.CheckPassword(e.password, arg.HashedPassword)
	if err != nil {
		return false
	}

	e.arg.HashedPassword = arg.HashedPassword

	return reflect.DeepEqual(e.arg, arg)
}

func (e eqCreateUserParamsMatcher) String() string {
	return fmt.Sprintf("matches arg %v and password %v", e.arg, e.password)
}

func EqCreateUserParams(arg db.CreateUserParams, password string) gomock.Matcher {
	return eqCreateUserParamsMatcher{arg, password}
}

func TestCreateUser(t *testing.T) {
	user, password := createRandomUser(t)

	arg := db.CreateUserParams{
		Username: user.Username,
		FullName: user.FullName,
		Email:    user.Email,
	}

	testCases := []struct {
		name          string
		body          gin.H
		buildstubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: gin.H{
				"username":  arg.Username,
				"full_name": arg.FullName,
				"email":     arg.Email,
				"password":  password,
			},
			buildstubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), EqCreateUserParams(arg, password)).Times(1).Return(user, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchUser(t, recorder.Body, user)
			},
		},
		{
			name: "BadJSON",
			body: gin.H{},
			buildstubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "HashPasswordIssue",
			body: gin.H{
				"username":  arg.Username,
				"full_name": arg.FullName,
				"email":     arg.Email,
				"password":  "trigger_error",
			},

			buildstubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "PostgresForbidden",
			body: gin.H{
				"username":  arg.Username,
				"full_name": arg.FullName,
				"email":     arg.Email,
				"password":  password,
			},

			buildstubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), EqCreateUserParams(arg, password)).Times(1).Return(db.User{}, &pq.Error{Code: "23505"})
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name: "PostgresInternalError",
			body: gin.H{
				"username":  arg.Username,
				"full_name": arg.FullName,
				"email":     arg.Email,
				"password":  password,
			},

			buildstubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), EqCreateUserParams(arg, password)).Times(1).Return(db.User{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)

			// build stubs
			tc.buildstubs(store)

			// start test server and send request
			server := newTestServer(t, store)
			server.hashPassword = func(pw string) (string, error) {
				if pw == "trigger_error" {
					return "", fmt.Errorf("forced error")
				}

				return util.HashPassword(pw)
			}

			recorder := httptest.NewRecorder()

			url := "/users"

			data, err := json.Marshal(tc.body)
			if err != nil {
				log.Fatal(err)
			}
			body := bytes.NewReader(data)

			request, err := http.NewRequest(http.MethodPost, url, body)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			// check response
			tc.checkResponse(t, recorder)
		})

	}
}

type mockTokenMaker struct{}

func (m *mockTokenMaker) CreateToken(username string, duration time.Duration) (string, error) {
	return "", fmt.Errorf("forced token error")
}

func (m *mockTokenMaker) VerifyToken(token string) (*token.Payload, error) {
	return nil, fmt.Errorf("forced token error")
}

func TestLoginUser(t *testing.T) {
	user, password := createRandomUser(t)

	hashedPassword, err := util.HashPassword(password)
	require.NoError(t, err)

	user.HashedPassword = hashedPassword

	testCases := []struct {
		name          string
		body          gin.H
		buildstubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: gin.H{
				"username": user.Username,
				"password": password,
			},
			buildstubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.Username)).Times(1).Return(user, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "JSONError",
			body: gin.H{},
			buildstubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "SQLNoRows",
			body: gin.H{
				"username": user.Username,
				"password": password,
			},
			buildstubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.Username)).Times(1).Return(db.User{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "SQLConnDone",
			body: gin.H{
				"username": user.Username,
				"password": password,
			},
			buildstubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.Username)).Times(1).Return(db.User{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "WrongPassword",
			body: gin.H{
				"username": user.Username,
				"password": "abcdef",
			},
			buildstubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.Username)).Times(1).Return(user, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "TokenError",
			body: gin.H{
				"username": user.Username,
				"password": password,
			},
			buildstubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.Username)).Times(1).Return(user, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)

			// build stubs
			tc.buildstubs(store)

			// start test server and send request
			server := newTestServer(t, store)

			if tc.name == "TokenError" {
				server.tokenMaker = &mockTokenMaker{}
			}

			recorder := httptest.NewRecorder()

			url := "/users/login"

			data, err := json.Marshal(tc.body)
			if err != nil {
				log.Fatal(err)
			}
			body := bytes.NewReader(data)

			request, err := http.NewRequest(http.MethodPost, url, body)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			// check response
			tc.checkResponse(t, recorder)
		})

	}
}

func createRandomUser(t *testing.T) (db.User, string) {
	username := util.RandomString(5)
	fullName := util.RandomString(10)
	email := util.RandomString(5) + "@gmail.com"
	password := util.RandomString(6)

	user := db.User{
		Username:          username,
		FullName:          fullName,
		Email:             email,
		PasswordChangedAt: time.Time{},
		CreatedAt:         time.Now().Round(0),
	}

	return user, password
}

func requireBodyMatchUser(t *testing.T, body *bytes.Buffer, user db.User) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotUser db.User
	err = json.Unmarshal(data, &gotUser)
	require.NoError(t, err)
	require.Equal(t, user.Username, gotUser.Username)
	require.Equal(t, user.FullName, gotUser.FullName)
	require.Equal(t, user.Email, gotUser.Email)
	require.WithinDuration(t, user.CreatedAt, gotUser.CreatedAt, time.Second)
}
