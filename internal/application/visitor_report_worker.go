package application

import (
	"context"
	"log/slog"
	"time"

	"github.com/DextaAfrica/Backend/internal/service"
)

// VisitorReportWorker runs as a resilient background worker checking for due visitor check-in reports.
type VisitorReportWorker struct {
	reportService *service.VisitorReportService
	stopCh        chan struct{}
}

// NewVisitorReportWorker constructs a VisitorReportWorker.
func NewVisitorReportWorker(reportService *service.VisitorReportService) *VisitorReportWorker {
	return &VisitorReportWorker{
		reportService: reportService,
		stopCh:        make(chan struct{}),
	}
}

// Start initiates the recurring 1-minute ticker for dispatching due scheduled reports.
func (w *VisitorReportWorker) Start(ctx context.Context) {
	slog.Info("automated visitor report worker started (cron schedule active)")

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	// Perform initial check on boot
	if err := w.reportService.ProcessDueSchedules(ctx); err != nil {
		slog.Warn("error in initial visitor report schedule scan", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			slog.Info("visitor report worker shutting down...")
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			if err := w.reportService.ProcessDueSchedules(ctx); err != nil {
				slog.Error("error processing due visitor reports", "error", err)
			}
		}
	}
}

// Stop cleanly terminates the worker loop.
func (w *VisitorReportWorker) Stop() {
	close(w.stopCh)
}
