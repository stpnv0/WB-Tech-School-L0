package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"order-service/internal/models"
	"order-service/internal/repository"
	"order-service/internal/service/mocks"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestOrderService_ProcessNewOrder_Success(t *testing.T) {
	t.Parallel()

	repo := new(mocks.OrderRepository)
	cache := new(mocks.OrderCache)
	defer repo.AssertExpectations(t)
	defer cache.AssertExpectations(t)

	svc := NewOrderService(repo, cache, testLogger())
	ctx := context.Background()
	order := &models.Order{OrderUID: "uid-1"}

	repo.On("SaveOrder", mock.Anything, order).Return(nil).Once()
	cache.On("Set", order).Once()

	err := svc.ProcessNewOrder(ctx, order)
	require.NoError(t, err)
}

func TestOrderService_ProcessNewOrder_SaveError(t *testing.T) {
	t.Parallel()

	repo := new(mocks.OrderRepository)
	cache := new(mocks.OrderCache)
	defer repo.AssertExpectations(t)
	defer cache.AssertExpectations(t)

	svc := NewOrderService(repo, cache, testLogger())
	ctx := context.Background()
	order := &models.Order{OrderUID: "uid-err"}

	repo.On("SaveOrder", mock.Anything, order).Return(errors.New("db failed")).Once()

	err := svc.ProcessNewOrder(ctx, order)
	require.Error(t, err)

	cache.AssertNotCalled(t, "Set", mock.Anything)
}

func TestOrderService_GetOrderByUID_CacheHit(t *testing.T) {
	t.Parallel()

	repo := new(mocks.OrderRepository)
	cache := new(mocks.OrderCache)
	defer repo.AssertExpectations(t)
	defer cache.AssertExpectations(t)

	svc := NewOrderService(repo, cache, testLogger())
	ctx := context.Background()
	order := &models.Order{OrderUID: "uid-2"}

	cache.On("Get", "uid-2").Return(order, true).Once()

	got, err := svc.GetOrderByUID(ctx, "uid-2")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, order, got)

	repo.AssertNotCalled(t, "GetOrderByUID", mock.Anything, mock.Anything)
}

func TestOrderService_GetOrderByUID_CacheMiss_FoundInRepo(t *testing.T) {
	t.Parallel()

	repo := new(mocks.OrderRepository)
	cache := new(mocks.OrderCache)
	defer repo.AssertExpectations(t)
	defer cache.AssertExpectations(t)

	svc := NewOrderService(repo, cache, testLogger())
	ctx := context.Background()
	order := &models.Order{OrderUID: "uid-3"}

	cache.On("Get", "uid-3").Return((*models.Order)(nil), false).Once()
	repo.On("GetOrderByUID", mock.Anything, "uid-3").Return(order, nil).Once()
	cache.On("Set", order).Once()

	got, err := svc.GetOrderByUID(ctx, "uid-3")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, order, got)
}

func TestOrderService_GetOrderByUID_CacheMiss_NotFound(t *testing.T) {
	t.Parallel()

	repo := new(mocks.OrderRepository)
	cache := new(mocks.OrderCache)
	defer repo.AssertExpectations(t)
	defer cache.AssertExpectations(t)

	svc := NewOrderService(repo, cache, testLogger())
	ctx := context.Background()

	cache.On("Get", "uid-404").Return((*models.Order)(nil), false).Once()
	repo.On("GetOrderByUID", mock.Anything, "uid-404").Return(nil, repository.ErrNotFound).Once()

	got, err := svc.GetOrderByUID(ctx, "uid-404")
	require.Error(t, err)
	require.Nil(t, got)
	// В методе GetOrderByUID ошибка оборачивается через %w — проверяем, что ErrNotFound сохраняется
	assert.ErrorIs(t, err, repository.ErrNotFound)

	cache.AssertNotCalled(t, "Set", mock.Anything)
}

func TestOrderService_GetOrderByUID_CacheMiss_RepoError(t *testing.T) {
	t.Parallel()

	repo := new(mocks.OrderRepository)
	cache := new(mocks.OrderCache)
	defer repo.AssertExpectations(t)
	defer cache.AssertExpectations(t)

	svc := NewOrderService(repo, cache, testLogger())
	ctx := context.Background()
	someErr := errors.New("db explosion")

	cache.On("Get", "uid-err").Return((*models.Order)(nil), false).Once()
	repo.On("GetOrderByUID", mock.Anything, "uid-err").Return(nil, someErr).Once()

	got, err := svc.GetOrderByUID(ctx, "uid-err")
	require.Error(t, err)
	require.Nil(t, got)
	assert.ErrorIs(t, err, someErr)

	cache.AssertNotCalled(t, "Set", mock.Anything)
}

func TestOrderService_PreloadCache_Success(t *testing.T) {
	t.Parallel()

	repo := new(mocks.OrderRepository)
	cache := new(mocks.OrderCache)
	defer repo.AssertExpectations(t)
	defer cache.AssertExpectations(t)

	svc := NewOrderService(repo, cache, testLogger())
	ctx := context.Background()
	orders := []*models.Order{
		{OrderUID: "o1"},
		{OrderUID: "o2"},
	}

	repo.On("GetLastNOrders", mock.Anything, 2).Return(orders, nil).Once()
	// Проверяем, что в кэш передаётся тот же слайс (можно и mock.Anything, но так строже)
	cache.On("LoadBatch", orders).Once()

	err := svc.PreloadCache(ctx, 2)
	require.NoError(t, err)
}

func TestOrderService_PreloadCache_RepoError(t *testing.T) {
	t.Parallel()

	repo := new(mocks.OrderRepository)
	cache := new(mocks.OrderCache)
	defer repo.AssertExpectations(t)
	defer cache.AssertExpectations(t)

	svc := NewOrderService(repo, cache, testLogger())
	ctx := context.Background()

	repo.On("GetLastNOrders", mock.Anything, 5).Return(nil, errors.New("db fail")).Once()

	err := svc.PreloadCache(ctx, 5)
	require.Error(t, err)

	cache.AssertNotCalled(t, "LoadBatch", mock.Anything)
}
