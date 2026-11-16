package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"squad/commons/auth"
	"squad/commons/llm"
	"squad/commons/queue"
	"squad/commons/session"
	"squad/commons/sqlite"
	"squad/commons/utils"
	"squad/features/agent"
	"squad/features/bots"
	"squad/features/files"
	"squad/features/gateways"
	"squad/features/memories"
	"squad/features/schedules"
	"squad/features/search"
	"squad/features/threads"
	"squad/features/tools"
	"squad/features/usage"
	"squad/features/users"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

//go:embed assets/*
var assets embed.FS

//go:embed migrations/*.sql
var migrations embed.FS

func main() {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("killing server", "error", r)
			os.Exit(1)
		}
	}()

	if err := godotenv.Load(); err != nil {
		slog.Warn("no .env file found, using env vars only")
	}

	sqlite.NewDB()
	defer sqlite.DB.Close()

	osSignalChan := make(chan os.Signal, 1)
	signal.Notify(osSignalChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-osSignalChan
		slog.Info("received shutdown signal, closing database connection")
		if err := sqlite.DB.Close(); err != nil {
			slog.Error("error closing database", "error", err)
		}
		os.Exit(0)
	}()

	sqlite.Migrate(migrations)

	llm.Init()
	if len(llm.GetGatewayNames()) == 0 {
		slog.Warn("no LLM gateway configured, set OPENROUTER_API_KEY")
	}

	tools.Init()
	agent.Init()
	queue.Start(2)
	schedules.StartScheduler()
	go runBackgroundTasks()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	slog.Info("starting server", "port", port)
	if err := http.ListenAndServe(":"+port, newRouter()); err != nil {
		panic(err)
	}
}

func newRouter() *http.ServeMux {
	mux := http.NewServeMux()

	addPublicRoute(mux, "GET /api/users/me/", users.HandleGetCurrentUser)
	addPublicRoute(mux, "POST /api/users/login/", users.HandleLogin)
	addPublicRoute(mux, "POST /api/users/", users.HandleCreateUser)
	addSessionRoute(mux, "POST /api/users/logout/", users.HandleLogout)
	addSessionRoute(mux, "PUT /api/users/me/password/", users.HandleUpdatePassword)
	addSessionRoute(mux, "PUT /api/users/me/about/", users.HandleUpdateAboutMe)

	addSessionRoute(mux, "GET /api/gateways/", gateways.HandleGetGateways)
	addSessionRoute(mux, "GET /api/gateways/{gateway}/models/", gateways.HandleGetModels)
	addSessionRoute(mux, "GET /api/gateways/{gateway}/zdr/", gateways.HandleGetZDREndpoints)
	addSessionRoute(mux, "GET /api/gateways/{gateway}/providers/", gateways.HandleGetProviders)

	addSessionRoute(mux, "GET /api/bots/", bots.HandleGetBots)
	addSessionRoute(mux, "POST /api/bots/", bots.HandleCreateBot)
	addSessionRoute(mux, "PUT /api/bots/{botId}/", bots.HandleUpdateBot)
	addSessionRoute(mux, "DELETE /api/bots/{botId}/", bots.HandleDeleteBot)
	addSessionRoute(mux, "PUT /api/bots/{botId}/tools/", bots.HandleUpdateBotTools)
	addSessionRoute(mux, "PUT /api/bots/{botId}/webhook/", bots.HandleRegenerateBotWebhook)
	addSessionRoute(mux, "DELETE /api/bots/{botId}/webhook/", bots.HandleDeleteBotWebhook)
	addPublicRoute(mux, "POST /api/hooks/{token}/", threads.HandleReceiveWebhook)
	addSessionRoute(mux, "GET /api/tools/", tools.HandleGetToolGroups)

	addSessionRoute(mux, "GET /api/bots/{botId}/memories/", memories.HandleGetMemories)
	addSessionRoute(mux, "DELETE /api/bots/{botId}/memories/{memoryId}/", memories.HandleDeleteMemory)

	addSessionRoute(mux, "GET /api/bots/{botId}/cost/", usage.HandleGetBotCost)

	addSessionRoute(mux, "GET /api/bots/{botId}/files/", files.HandleGetFiles)
	addSessionRoute(mux, "GET /api/bots/{botId}/files/{name}/", files.HandleGetFile)
	addSessionRoute(mux, "PUT /api/bots/{botId}/files/{name}/", files.HandleWriteFile)
	addSessionRoute(mux, "DELETE /api/bots/{botId}/files/{name}/", files.HandleDeleteFile)

	addSessionRoute(mux, "GET /api/bots/{botId}/schedules/", schedules.HandleGetSchedules)
	addSessionRoute(mux, "POST /api/bots/{botId}/schedules/", schedules.HandleCreateSchedule)
	addSessionRoute(mux, "PUT /api/bots/{botId}/schedules/{scheduleId}/", schedules.HandleUpdateSchedule)
	addSessionRoute(mux, "DELETE /api/bots/{botId}/schedules/{scheduleId}/", schedules.HandleDeleteSchedule)
	addSessionRoute(mux, "POST /api/bots/{botId}/schedules/{scheduleId}/run/", schedules.HandleRunSchedule)
	addSessionRoute(mux, "GET /api/bots/{botId}/schedules/{scheduleId}/threads/", schedules.HandleGetScheduleThreads)

	addSessionRoute(mux, "GET /api/bots/{botId}/threads/", threads.HandleGetThreads)
	addSessionRoute(mux, "POST /api/bots/{botId}/threads/", threads.HandleCreateThread)
	addSessionRoute(mux, "POST /api/bots/{botId}/attachments/", threads.HandleUploadAttachment)
	addSessionRoute(mux, "GET /api/attachments/{attachmentId}/", threads.HandleGetAttachment)
	addSessionRoute(mux, "GET /api/threads/{threadId}/", threads.HandleGetThread)
	addSessionRoute(mux, "PUT /api/threads/{threadId}/", threads.HandleRenameThread)
	addSessionRoute(mux, "DELETE /api/threads/{threadId}/", threads.HandleDeleteThread)
	addSessionRoute(mux, "GET /api/threads/{threadId}/usage/", threads.HandleGetThreadUsage)
	addSessionRoute(mux, "GET /api/threads/{threadId}/messages/", threads.HandleGetMessages)
	addSessionRoute(mux, "POST /api/threads/{threadId}/messages/", threads.HandleSendMessage)
	addSessionRoute(mux, "POST /api/threads/{threadId}/retry/", threads.HandleRetryThread)
	addSessionRoute(mux, "POST /api/threads/{threadId}/cancel/", threads.HandleCancelThread)
	addSessionRoute(mux, "GET /api/threads/{threadId}/events/", threads.HandleThreadEvents)

	addSessionRoute(mux, "GET /api/search/", search.HandleSearch)

	// Without this, an unknown API path would fall through to the index page and confuse the client with HTML.
	addPublicRoute(mux, "GET /api/", handleUnknownAPIRoute)

	addPublicRoute(mux, "GET /assets/", handleStaticAssets)
	addPublicRoute(mux, "GET /sw.js", handleServiceWorker)
	addPublicRoute(mux, "GET /", handleRoot)

	return mux
}

func addPublicRoute(mux *http.ServeMux, pattern string, handlerFunc http.HandlerFunc) {
	mux.HandleFunc(pattern, handlerFunc)
}

func addSessionRoute(mux *http.ServeMux, pattern string, handlerFunc http.HandlerFunc) {
	mux.HandleFunc(pattern, auth.EnsureSession(handlerFunc))
}

func handleUnknownAPIRoute(w http.ResponseWriter, r *http.Request) {
	utils.SendErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "Not found", nil)
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	indexPage, err := readAsset("index.html")
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(indexPage)
}

func handleStaticAssets(w http.ResponseWriter, r *http.Request) {
	var fileSystem http.FileSystem

	if os.Getenv("DEV_MODE") == "true" {
		// Files on disk carry Last-Modified, which lets the browser reuse a stale copy without asking, so dev builds force a check.
		w.Header().Set("Cache-Control", "no-cache")
		fileSystem = http.Dir("./assets")
	} else {
		subtree, err := fs.Sub(assets, "assets")
		if err != nil {
			err = fmt.Errorf("error reading assets subtree: %w", err)
			slog.Error(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fileSystem = http.FS(subtree)
	}

	http.StripPrefix("/assets/", http.FileServer(fileSystem)).ServeHTTP(w, r)
}

// A worker can only control pages under the path it was served from, so it is served from the root and widened with Service-Worker-Allowed.
func handleServiceWorker(w http.ResponseWriter, r *http.Request) {
	content, err := readAsset("sw.js")
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/javascript")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Service-Worker-Allowed", "/")
	w.WriteHeader(http.StatusOK)
	w.Write(content)
}

func readAsset(name string) ([]byte, error) {
	var content []byte
	var err error

	if os.Getenv("DEV_MODE") == "true" {
		content, err = os.ReadFile("./assets/" + name)
	} else {
		content, err = assets.ReadFile("assets/" + name)
	}

	if err != nil {
		return nil, fmt.Errorf("error reading %s: %w", name, err)
	}

	return content, nil
}

func runBackgroundTasks() {
	for range time.Tick(24 * time.Hour) {
		session.DeleteExpiredSessions()
	}
}
