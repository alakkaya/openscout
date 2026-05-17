package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alakkaya/openscout/internal/adapter/github"
	analyzerhttp "github.com/alakkaya/openscout/internal/adapter/http"
	"github.com/alakkaya/openscout/internal/adapter/notification"
	postgresadapter "github.com/alakkaya/openscout/internal/adapter/postgres"
	"github.com/alakkaya/openscout/internal/infrastructure"
	"github.com/alakkaya/openscout/internal/usecase"
	"github.com/alakkaya/openscout/internal/web"
)

func main() {
	cfg, err := infrastructure.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	log := infrastructure.NewLogger(cfg.Logger.Level)
	log.Info("config loaded", "db", "postgres")

	db, err := infrastructure.NewDatabase(cfg.Database)
	if err != nil {
		log.Error("database setup failed", "error", err)
		os.Exit(1)
	}
	log.Info("database ready")

	githubClient := github.NewClient(cfg.GitHub.Token, 20)
	analyzerClient := analyzerhttp.NewAnalyzerHTTPClient(cfg.Analyzer.URL, 30*time.Second)

	var emailSender *notification.EmailSender
	if cfg.Email.Host != "" && cfg.Email.Username != "" && cfg.Email.Password != "" && cfg.Email.From != "" {
		emailSender = notification.NewEmailSender(notification.EmailConfig{
			Host:     cfg.Email.Host,
			Port:     cfg.Email.Port,
			Username: cfg.Email.Username,
			Password: cfg.Email.Password,
			From:     cfg.Email.From,
		})
		log.Info("email notifications enabled")
	} else {
		log.Info("email notifications disabled")
	}

	var telegramSender *notification.TelegramSender
	if cfg.Telegram.BotToken != "" {
		telegramSender = notification.NewTelegramSender(cfg.Telegram.BotToken)
		log.Info("telegram notifications enabled")
	} else {
		log.Info("telegram notifications disabled")
	}
	notifier := notification.NewCompositeNotifier(emailSender, telegramSender)

	userRepo := postgresadapter.NewUserRepository(db)
	prefRepo := postgresadapter.NewUserPreferenceRepository(db)
	feedbackRepo := postgresadapter.NewFeedbackRepository(db)
	notificationRepo := postgresadapter.NewNotificationRepository(db)

	collectUC := usecase.NewCollectIssuesUseCase(githubClient)
	analyzeUC := usecase.NewAnalyzeIssuesUseCase(analyzerClient)
	sendUC := usecase.NewSendNotificationsUseCase(userRepo, prefRepo, feedbackRepo, notificationRepo, notifier)

	pipelineJob := func(jobCtx context.Context) error {
		languages := []string{"Go", "Python", "TypeScript"}
		labels := []string{"good first issue", "help wanted"}

		issues, err := collectUC.Execute(jobCtx, languages, labels)
		if err != nil {
			return err
		}
		if len(issues) == 0 {
			log.Info("no issues fetched")
			return nil
		}
		log.Info("collected issues", "count", len(issues))

		analyses, err := analyzeUC.Execute(jobCtx, issues)
		if err != nil {
			return err
		}
		log.Info("analyzed issues", "count", len(analyses))

		if err := sendUC.Execute(jobCtx, issues, analyses); err != nil {
			return err
		}
		log.Info("sent notifications")
		return nil
	}

	scheduler := infrastructure.NewScheduler(log)
	if err := scheduler.Schedule("08:00", pipelineJob); err != nil {
		log.Error("schedule job failed", "error", err)
		os.Exit(1)
	}
	log.Info("scheduler scheduled for 08:00 daily")

	// Setup HTTP handlers
	handler := web.NewHandler(userRepo, prefRepo, feedbackRepo, log)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /subscribe", handler.Subscribe)
	mux.HandleFunc("POST /preferences", handler.Preferences)
	mux.HandleFunc("POST /feedback", handler.Feedback)

	port := os.Getenv("HOST_HTTP_PORT")
	if port == "" {
		port = "8080"
	}
	addr := fmt.Sprintf(":%s", port)
	server := &http.Server{Addr: addr, Handler: mux}
	go func() {
		log.Info("http server starting", "addr", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http server failed", "error", err)
		}
	}()

	// Start scheduler (blocks)
	log.Info("starting scheduler")
	scheduler.Start()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
	scheduler.Stop()
}
