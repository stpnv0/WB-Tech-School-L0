package handlers_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"order-service/internal/handlers"
	"order-service/internal/handlers/mocks"
	"order-service/internal/models"
	"order-service/internal/repository"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func setupRouter(t *testing.T) (*gin.Engine, *mocks.OrderService) {
	t.Helper()
	mockSvc := new(mocks.OrderService)
	h := handlers.NewHandler(mockSvc, testLogger())

	r := gin.New()
	r.GET("/order/:order_uid", h.GetOrderByUID)
	return r, mockSvc
}

func TestGetOrderByUID_Success(t *testing.T) {
	r, svc := setupRouter(t)
	defer svc.AssertExpectations(t)

	order := &models.Order{
		OrderUID:    "uid-123",
		TrackNumber: "TRK-1",
	}
	svc.On("GetOrderByUID", mock.Anything, "uid-123").
		Return(order, nil).Once()

	req := httptest.NewRequest(http.MethodGet, "/order/uid-123", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

	var got models.Order
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, order.OrderUID, got.OrderUID)
	assert.Equal(t, order.TrackNumber, got.TrackNumber)
}

func TestGetOrderByUID_NotFound(t *testing.T) {
	r, svc := setupRouter(t)
	defer svc.AssertExpectations(t)

	svc.On("GetOrderByUID", mock.Anything, "missing").
		Return((*models.Order)(nil), repository.ErrNotFound).Once()

	req := httptest.NewRequest(http.MethodGet, "/order/missing", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "Order not found", body["error"])
}

func TestGetOrderByUID_ServiceError(t *testing.T) {
	r, svc := setupRouter(t)
	defer svc.AssertExpectations(t)

	svc.On("GetOrderByUID", mock.Anything, "boom").
		Return((*models.Order)(nil), assert.AnError).Once()

	req := httptest.NewRequest(http.MethodGet, "/order/boom", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "Internal server error", body["error"])
}
