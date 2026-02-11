package outbox_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rafaelleal24/challenge/internal/adapters/config"
	"github.com/rafaelleal24/challenge/internal/adapters/outbox"
	outboxmock "github.com/rafaelleal24/challenge/internal/adapters/outbox/mock"
	portmock "github.com/rafaelleal24/challenge/internal/core/port/mock"
	"go.uber.org/mock/gomock"
)

func TestHandler_ProcessesAndDeletesEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	broker := portmock.NewMockBrokerPort(ctrl)
	repo := outboxmock.NewMockRepository(ctrl)

	entries := []outbox.Entry{
		{ID: "1", EventName: "order.created", EntityName: "order", EventData: []byte(`{"id":"1"}`)},
		{ID: "2", EventName: "order.updated", EntityName: "order", EventData: []byte(`{"id":"2"}`)},
	}

	repo.EXPECT().FetchPending(gomock.Any(), 10).Return(entries, nil).Times(1)
	repo.EXPECT().FetchPending(gomock.Any(), 10).Return(nil, nil).AnyTimes()

	broker.EXPECT().PublishRaw(gomock.Any(), "order.created", "order", []byte(`{"id":"1"}`)).Return(nil)
	broker.EXPECT().PublishRaw(gomock.Any(), "order.updated", "order", []byte(`{"id":"2"}`)).Return(nil)

	repo.EXPECT().Delete(gomock.Any(), "1").Return(nil)
	repo.EXPECT().Delete(gomock.Any(), "2").Return(nil)

	handler := outbox.NewHandler(repo, broker, config.OutboxConfig{
		Interval:  50 * time.Millisecond,
		BatchSize: 10,
	})

	ctx, cancel := context.WithCancel(context.Background())
	go handler.Start(ctx)

	time.Sleep(200 * time.Millisecond)
	cancel()
}

func TestHandler_SkipsEventOnPublishFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	broker := portmock.NewMockBrokerPort(ctrl)
	repo := outboxmock.NewMockRepository(ctrl)

	entries := []outbox.Entry{
		{ID: "1", EventName: "order.fail", EntityName: "order", EventData: []byte(`{"id":"1"}`)},
		{ID: "2", EventName: "order.success", EntityName: "order", EventData: []byte(`{"id":"2"}`)},
	}

	repo.EXPECT().FetchPending(gomock.Any(), 10).Return(entries, nil).Times(1)
	repo.EXPECT().FetchPending(gomock.Any(), 10).Return(nil, nil).AnyTimes()

	// First event fails publish → no Delete called for it
	broker.EXPECT().PublishRaw(gomock.Any(), "order.fail", "order", []byte(`{"id":"1"}`)).Return(errors.New("publish failed"))
	// Second event succeeds
	broker.EXPECT().PublishRaw(gomock.Any(), "order.success", "order", []byte(`{"id":"2"}`)).Return(nil)
	repo.EXPECT().Delete(gomock.Any(), "2").Return(nil)

	handler := outbox.NewHandler(repo, broker, config.OutboxConfig{
		Interval:  50 * time.Millisecond,
		BatchSize: 10,
	})

	ctx, cancel := context.WithCancel(context.Background())
	go handler.Start(ctx)

	time.Sleep(200 * time.Millisecond)
	cancel()
}

func TestHandler_HandlesEmptyOutbox(t *testing.T) {
	ctrl := gomock.NewController(t)
	broker := portmock.NewMockBrokerPort(ctrl)
	repo := outboxmock.NewMockRepository(ctrl)

	repo.EXPECT().FetchPending(gomock.Any(), 10).Return(nil, nil).AnyTimes()

	handler := outbox.NewHandler(repo, broker, config.OutboxConfig{
		Interval:  50 * time.Millisecond,
		BatchSize: 10,
	})

	ctx, cancel := context.WithCancel(context.Background())
	go handler.Start(ctx)

	time.Sleep(200 * time.Millisecond)
	cancel()
}

func TestHandler_HandlesFetchError(t *testing.T) {
	ctrl := gomock.NewController(t)
	broker := portmock.NewMockBrokerPort(ctrl)
	repo := outboxmock.NewMockRepository(ctrl)

	repo.EXPECT().FetchPending(gomock.Any(), 10).Return(nil, errors.New("db down")).AnyTimes()

	handler := outbox.NewHandler(repo, broker, config.OutboxConfig{
		Interval:  50 * time.Millisecond,
		BatchSize: 10,
	})

	ctx, cancel := context.WithCancel(context.Background())
	go handler.Start(ctx)

	time.Sleep(200 * time.Millisecond)
	cancel()
}

func TestHandler_DeleteFailureDoesNotPanic(t *testing.T) {
	ctrl := gomock.NewController(t)
	broker := portmock.NewMockBrokerPort(ctrl)
	repo := outboxmock.NewMockRepository(ctrl)

	entries := []outbox.Entry{
		{ID: "1", EventName: "order.created", EntityName: "order", EventData: []byte(`{"id":"1"}`)},
	}

	repo.EXPECT().FetchPending(gomock.Any(), 10).Return(entries, nil).Times(1)
	repo.EXPECT().FetchPending(gomock.Any(), 10).Return(nil, nil).AnyTimes()

	broker.EXPECT().PublishRaw(gomock.Any(), "order.created", "order", []byte(`{"id":"1"}`)).Return(nil)
	repo.EXPECT().Delete(gomock.Any(), "1").Return(errors.New("delete failed"))

	handler := outbox.NewHandler(repo, broker, config.OutboxConfig{
		Interval:  50 * time.Millisecond,
		BatchSize: 10,
	})

	ctx, cancel := context.WithCancel(context.Background())
	go handler.Start(ctx)

	time.Sleep(200 * time.Millisecond)
	cancel()
}

func TestHandler_StopsOnContextCancel(t *testing.T) {
	ctrl := gomock.NewController(t)
	broker := portmock.NewMockBrokerPort(ctrl)
	repo := outboxmock.NewMockRepository(ctrl)

	handler := outbox.NewHandler(repo, broker, config.OutboxConfig{
		Interval:  1 * time.Hour,
		BatchSize: 10,
	})

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		handler.Start(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not stop after context cancellation")
	}
}

func TestHandler_RespectsBatchSize(t *testing.T) {
	ctrl := gomock.NewController(t)
	broker := portmock.NewMockBrokerPort(ctrl)
	repo := outboxmock.NewMockRepository(ctrl)

	repo.EXPECT().FetchPending(gomock.Any(), 5).Return(nil, nil).AnyTimes()

	handler := outbox.NewHandler(repo, broker, config.OutboxConfig{
		Interval:  50 * time.Millisecond,
		BatchSize: 5,
	})

	ctx, cancel := context.WithCancel(context.Background())
	go handler.Start(ctx)

	time.Sleep(200 * time.Millisecond)
	cancel()
}
