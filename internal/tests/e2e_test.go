package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/Nikita-onlyHD/merchandise-store/internal/dto"
	"github.com/Nikita-onlyHD/merchandise-store/internal/handler"
	coinHandler "github.com/Nikita-onlyHD/merchandise-store/internal/handler/coin_transfer"
	itemHandler "github.com/Nikita-onlyHD/merchandise-store/internal/handler/item"
	userHandler "github.com/Nikita-onlyHD/merchandise-store/internal/handler/user"
	"github.com/Nikita-onlyHD/merchandise-store/internal/repository"
	coinService "github.com/Nikita-onlyHD/merchandise-store/internal/service/coin_transfer"
	itemService "github.com/Nikita-onlyHD/merchandise-store/internal/service/item"
	userService "github.com/Nikita-onlyHD/merchandise-store/internal/service/user"
)

type E2ESuite struct {
	suite.Suite
	pgContainer *postgres.PostgresContainer
	testServer  *httptest.Server
	client      *http.Client
}

func (s *E2ESuite) SetupSuite() {
	ctx := context.Background()

	migrations := filepath.Join("..", "..", "migrations", "000001_create_tables.up.sql")

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("shop_test"),
		postgres.WithUsername("test_user"),
		postgres.WithPassword("test_pass"),
		postgres.WithInitScripts(migrations),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(10*time.Second),
		),
	)
	s.Require().NoError(err)
	s.pgContainer = pgContainer

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	s.Require().NoError(err)

	jwtSecret := "test-jwt-secret"
	pool, err := pgxpool.New(ctx, connStr)
	s.Require().NoError(err)

	txManager := repository.NewTxManager(pool)
	userRepo := repository.NewUserRepository(pool)
	itemRepo := repository.NewItemRepository(pool)
	inventoryRepo := repository.NewInventoryRepository(pool)
	coinTransferRepo := repository.NewCoinTransferRepository(pool)

	userSvc := userService.NewUserService(userRepo, inventoryRepo, coinTransferRepo, jwtSecret)
	coinTransferSvc := coinService.NewCoinTransferService(txManager, userRepo, coinTransferRepo)
	itemSvc := itemService.NewItemService(txManager, itemRepo, inventoryRepo, userRepo)

	userH := userHandler.NewUserHandler(userSvc)
	coinTransferH := coinHandler.NewCoinTransferHandler(coinTransferSvc)
	itemH := itemHandler.NewItemHandler(itemSvc)

	mux := http.NewServeMux()
	appHandler := handler.RegisterRoutes(mux, userH, itemH, coinTransferH, jwtSecret)

	s.testServer = httptest.NewServer(appHandler)
	s.client = s.testServer.Client()
}

func (s *E2ESuite) TearDownSuite() {
	ctx := context.Background()
	if s.testServer != nil {
		s.testServer.Close()
	}
	if s.pgContainer != nil {
		s.Require().NoError(s.pgContainer.Terminate(ctx))
	}
}

func TestE2ESuite(t *testing.T) {
	suite.Run(t, new(E2ESuite))
}

func (s *E2ESuite) TestAuth_Success() {
	reqBody := bytes.NewBufferString(`{
		"username": "test_username",
		"password": "test_password"
	}`)

	resp, err := s.client.Post(s.testServer.URL+"/api/auth", "application/json", reqBody)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Require().Equal(http.StatusOK, resp.StatusCode)

	var authResp struct {
		Token string `json:"token"`
	}
	err = json.NewDecoder(resp.Body).Decode(&authResp)
	s.Require().NoError(err)
	s.Assert().NotEmpty(authResp.Token)
}

func (s *E2ESuite) doAuthRequest(method, path, token string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, s.testServer.URL+path, body)
	if err != nil {
		return nil, err
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return s.client.Do(req)
}

func (s *E2ESuite) registerAndLogin(username, password string) string {
	reqBody := fmt.Sprintf(`{"username":"%s","password":"%s"}`, username, password)
	resp, err := s.client.Post(s.testServer.URL+"/api/auth", "application/json", strings.NewReader(reqBody))
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Require().Equal(http.StatusOK, resp.StatusCode)

	var authResp struct {
		Token string `json:"token"`
	}
	err = json.NewDecoder(resp.Body).Decode(&authResp)
	s.Require().NoError(err)

	return authResp.Token
}

func (s *E2ESuite) getUserInfo(token string) dto.UserInfo {
	resp, err := s.doAuthRequest("GET", "/api/info", token, nil)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Require().Equal(http.StatusOK, resp.StatusCode)

	var info dto.UserInfo
	err = json.NewDecoder(resp.Body).Decode(&info)
	s.Require().NoError(err)

	return info
}

func (s *E2ESuite) TestBuyItem_SuccessAndError() {
	token := s.registerAndLogin("test_username", "test_password")

	resp, err := s.doAuthRequest("GET", "/api/buy/cup", token, nil)
	s.Require().NoError(err)
	s.Assert().Equal(http.StatusOK, resp.StatusCode)

	info := s.getUserInfo(token)
	s.Assert().Equal(980, info.Coins)
	s.Assert().Len(info.Inventory, 1)
	s.Assert().Equal("cup", info.Inventory[0].Type)
	s.Assert().Equal(1, info.Inventory[0].Quantity)

	badResp, err := s.doAuthRequest("GET", "/api/buy/unknown_item", token, nil)
	s.Require().NoError(err)
	s.Assert().Equal(http.StatusBadRequest, badResp.StatusCode)
}

func (s *E2ESuite) TestSendCoin_SuccessAndErrors() {
	testSenderToken := s.registerAndLogin("test_sender", "test_password")
	testReceiverToken := s.registerAndLogin("test_receiver", "test_password")

	sendBody := `{"toUser": "test_receiver", "amount": 100}`
	resp, err := s.doAuthRequest("POST", "/api/sendCoin", testSenderToken, strings.NewReader(sendBody))
	s.Require().NoError(err)
	s.Assert().Equal(http.StatusOK, resp.StatusCode)

	testSenderInfo := s.getUserInfo(testSenderToken)
	s.Assert().Equal(900, testSenderInfo.Coins)
	s.Require().Len(testSenderInfo.CoinHistory.Sent, 1)
	s.Assert().Equal("test_receiver", testSenderInfo.CoinHistory.Sent[0].ToUser)
	s.Assert().Equal(100, testSenderInfo.CoinHistory.Sent[0].Amount)

	testReceiverInfo := s.getUserInfo(testReceiverToken)
	s.Assert().Equal(1100, testReceiverInfo.Coins)
	s.Require().Len(testReceiverInfo.CoinHistory.Received, 1)
	s.Assert().Equal("test_sender", testReceiverInfo.CoinHistory.Received[0].FromUser)
	s.Assert().Equal(100, testReceiverInfo.CoinHistory.Received[0].Amount)

	testCases := []struct {
		name   string
		body   string
		status int
	}{
		{
			name:   "Перевод самому себе",
			body:   `{"toUser": "test_sender", "amount": 10}`,
			status: http.StatusBadRequest,
		},
		{
			name:   "Перевод несуществующему пользователю",
			body:   `{"toUser": "unknown_user", "amount": 10}`,
			status: http.StatusBadRequest,
		},
		{
			name:   "Отрицательная сумма",
			body:   `{"toUser": "test_receiver", "amount": -50}`,
			status: http.StatusBadRequest,
		},
		{
			name:   "Сумма превышает баланс",
			body:   `{"toUser": "test_receiver", "amount": 5000}`,
			status: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			resp, err := s.doAuthRequest("POST", "/api/sendCoin", testSenderToken, strings.NewReader(tc.body))
			s.Require().NoError(err)
			s.Assert().Equal(tc.status, resp.StatusCode)
		})
	}
}
