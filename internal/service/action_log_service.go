package service

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"

	"be-remote-device/internal/models"
	"be-remote-device/internal/repository"
)

type ActionLogService interface {
	LogAction(log *models.ActionLog)
	GetLogs(filter models.ActionLogFilter) ([]*models.ActionLog, int64, error)
	GetLogByID(id string) (*models.ActionLog, error)
	Stop()
}

type actionLogService struct {
	repo    repository.ActionLogRepository
	logChan chan *models.ActionLog
	done    chan struct{}
}

func NewActionLogService(repo repository.ActionLogRepository) ActionLogService {
	s := &actionLogService{
		repo:    repo,
		logChan: make(chan *models.ActionLog, 1000), // Buffer hingga 1000 log
		done:    make(chan struct{}),
	}

	// Worker goroutine untuk menyimpan log ke DB secara asynchronous
	go s.worker()

	return s
}

func (s *actionLogService) worker() {
	for {
		select {
		case <-s.done:
			// Drain remaining logs
			for len(s.logChan) > 0 {
				log := <-s.logChan
				_ = s.repo.Save(log)
			}
			return
		case log := <-s.logChan:
			if err := s.repo.Save(log); err != nil {
				slog.Error("Failed to save action log to DB", "error", err, "action", log.Action, "path", log.Path)
			}
		}
	}
}

func (s *actionLogService) LogAction(log *models.ActionLog) {
	if log.ID == "" {
		b := make([]byte, 8)
		_, _ = rand.Read(b)
		log.ID = "log-" + hex.EncodeToString(b)
	}
	if log.Timestamp.IsZero() {
		log.Timestamp = time.Now().UTC()
	}

	select {
	case s.logChan <- log:
	default:
		slog.Warn("Action log channel full, saving synchronously to prevent log loss")
		go func(l *models.ActionLog) {
			_ = s.repo.Save(l)
		}(log)
	}
}

func (s *actionLogService) GetLogs(filter models.ActionLogFilter) ([]*models.ActionLog, int64, error) {
	return s.repo.GetAll(filter)
}

func (s *actionLogService) GetLogByID(id string) (*models.ActionLog, error) {
	return s.repo.GetByID(id)
}

func (s *actionLogService) Stop() {
	close(s.done)
}
