// Copyright 2024-2026 GAGOS Project
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gaga951/gagos/internal/auth"
	"github.com/gaga951/gagos/internal/cicd"
	"github.com/gaga951/gagos/internal/database"
	"github.com/gaga951/gagos/internal/k8s"
	"github.com/gaga951/gagos/internal/monitoring"
	"github.com/gaga951/gagos/internal/network"
	"github.com/gaga951/gagos/internal/storage"
	"github.com/gaga951/gagos/internal/terminal"
	"github.com/gaga951/gagos/internal/tools"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	// Setup logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	log.Info().
		Str("version", version).
		Str("build_time", buildTime).
		Msg("Starting GAGOS")

	// Initialize Kubernetes client
	if err := k8s.InitClient(); err != nil {
		log.Warn().Err(err).Msg("Failed to initialize Kubernetes client - K8s features will be unavailable")
	} else {
		log.Info().Msg("Kubernetes client initialized successfully")
	}

	// Initialize authentication
	auth.Init()

	// Initialize storage
	if err := storage.Init(); err != nil {
		log.Warn().Err(err).Msg("Failed to initialize storage - notepad will be unavailable")
	} else {
		log.Info().Msg("Storage initialized successfully")
	}

	// Initialize CI/CD scheduler
	scheduler := cicd.InitScheduler()
	if err := scheduler.Start(); err != nil {
		log.Warn().Err(err).Msg("Failed to start CI/CD scheduler")
	} else {
		log.Info().Msg("CI/CD scheduler started")
	}

	// Load notification configurations
	if err := cicd.LoadNotificationConfigs(); err != nil {
		log.Warn().Err(err).Msg("Failed to load notification configs")
	} else {
		log.Info().Msg("Notification configs loaded")
	}

	// Initialize monitoring
	if err := monitoring.Init(); err != nil {
		log.Warn().Err(err).Msg("Failed to initialize monitoring")
	} else {
		log.Info().Msg("Monitoring initialized")
	}

	// Get configuration from environment
	// Use GAGOS_SERVER_* to avoid conflict with K8s service-injected GAGOS_PORT
	host := getEnv("GAGOS_SERVER_HOST", getEnv("GAGOS_HOST", "0.0.0.0"))
	port := getEnv("GAGOS_SERVER_PORT", getEnv("GAGOS_PORT", "8080"))

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:               "GAGOS",
		DisableStartupMessage: false,
		ReadTimeout:           30 * time.Second,
		WriteTimeout:          30 * time.Second,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format:     "${time} | ${status} | ${latency} | ${ip} | ${method} | ${path}\n",
		TimeFormat: "2006-01-02 15:04:05",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))

	// Authentication middleware
	app.Use(auth.Middleware())

	// Routes
	setupRoutes(app)

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Info().Msg("Shutting down server...")
		if err := app.ShutdownWithTimeout(30 * time.Second); err != nil {
			log.Error().Err(err).Msg("Server shutdown error")
		}
	}()

	// Start server
	addr := fmt.Sprintf("%s:%s", host, port)
	log.Info().Str("address", addr).Msg("Server listening")
	if err := app.Listen(addr); err != nil {
		log.Fatal().Err(err).Msg("Server failed to start")
	}
}

func setupRoutes(app *fiber.App) {
	// Health check (public)
	app.Get("/api/health", healthHandler)

	// API info
	app.Get("/api", apiInfoHandler)

	// Version (public)
	app.Get("/api/version", versionHandler)

	// Auth routes (public)
	app.Get("/login", loginPageHandler)
	app.Post("/api/auth/login", loginHandler)
	app.Post("/api/auth/logout", logoutHandler)

	// Runtime info (public - for login page hint)
	app.Get("/api/runtime", runtimeHandler)

	// Static files with no-cache for JS files
	app.Use("/js/", func(c *fiber.Ctx) error {
		c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Set("Pragma", "no-cache")
		c.Set("Expires", "0")
		return c.Next()
	})
	app.Static("/", "/app/web/static")

	// API v1 group
	v1 := app.Group("/api/v1")

	// Network tools endpoints
	net := v1.Group("/network")
	net.Post("/ping", pingHandler)
	net.Post("/dns", dnsHandler)
	net.Post("/port-check", portCheckHandler)
	net.Post("/traceroute", tracerouteHandler)
	net.Post("/telnet", telnetHandler)
	net.Post("/whois", whoisHandler)
	net.Post("/ssl-check", sslCheckHandler)
	net.Post("/curl", curlHandler)
	net.Get("/interfaces", interfacesHandler)

	// Kubernetes endpoints
	k8sGroup := v1.Group("/k8s")
	// List endpoints
	k8sGroup.Get("/namespaces", namespacesHandler)
	k8sGroup.Get("/nodes", nodesHandler)
	k8sGroup.Get("/pods", podsHandler)
	k8sGroup.Get("/pods/:namespace", podsHandler)
	k8sGroup.Get("/services", servicesHandler)
	k8sGroup.Get("/services/:namespace", servicesHandler)
	k8sGroup.Get("/deployments", deploymentsHandler)
	k8sGroup.Get("/deployments/:namespace", deploymentsHandler)
	k8sGroup.Get("/configmaps", configMapsHandler)
	k8sGroup.Get("/configmaps/:namespace", configMapsHandler)
	k8sGroup.Get("/secrets", secretsHandler)
	k8sGroup.Get("/secrets/:namespace", secretsHandler)
	k8sGroup.Get("/serviceaccounts", serviceAccountsHandler)
	k8sGroup.Get("/serviceaccounts/:namespace", serviceAccountsHandler)
	k8sGroup.Get("/pvs", pvsHandler)
	k8sGroup.Get("/pvcs", pvcsHandler)
	k8sGroup.Get("/pvcs/:namespace", pvcsHandler)
	k8sGroup.Get("/ingresses", ingressesHandler)
	k8sGroup.Get("/ingresses/:namespace", ingressesHandler)
	k8sGroup.Get("/daemonsets", daemonSetsHandler)
	k8sGroup.Get("/daemonsets/:namespace", daemonSetsHandler)
	k8sGroup.Get("/statefulsets", statefulSetsHandler)
	k8sGroup.Get("/statefulsets/:namespace", statefulSetsHandler)
	k8sGroup.Get("/jobs", jobsHandler)
	k8sGroup.Get("/jobs/:namespace", jobsHandler)
	k8sGroup.Get("/cronjobs", cronJobsHandler)
	k8sGroup.Get("/cronjobs/:namespace", cronJobsHandler)
	k8sGroup.Get("/events", eventsHandler)
	k8sGroup.Get("/events/:namespace", eventsHandler)
	k8sGroup.Get("/replicasets", replicaSetsHandler)
	k8sGroup.Get("/replicasets/:namespace", replicaSetsHandler)

	// Single resource operations (describe/edit/delete)
	// Pods
	k8sGroup.Get("/pod/:namespace/:name", getPodHandler)
	k8sGroup.Get("/pod/:namespace/:name/logs", getPodLogsHandler)
	k8sGroup.Patch("/pod/:namespace/:name", patchPodHandler)
	k8sGroup.Delete("/pod/:namespace/:name", deletePodHandler)
	// Services
	k8sGroup.Get("/service/:namespace/:name", getServiceHandler)
	k8sGroup.Patch("/service/:namespace/:name", patchServiceHandler)
	k8sGroup.Delete("/service/:namespace/:name", deleteServiceHandler)
	// Deployments
	k8sGroup.Get("/deployment/:namespace/:name", getDeploymentHandler)
	k8sGroup.Patch("/deployment/:namespace/:name", patchDeploymentHandler)
	k8sGroup.Delete("/deployment/:namespace/:name", deleteDeploymentHandler)
	k8sGroup.Post("/deployment/:namespace/:name/scale", scaleDeploymentHandler)
	k8sGroup.Post("/deployment/:namespace/:name/restart", restartDeploymentHandler)
	// ConfigMaps
	k8sGroup.Get("/configmap/:namespace/:name", getConfigMapHandler)
	k8sGroup.Patch("/configmap/:namespace/:name", patchConfigMapHandler)
	k8sGroup.Delete("/configmap/:namespace/:name", deleteConfigMapHandler)
	// Secrets
	k8sGroup.Get("/secret/:namespace/:name", getSecretHandler)
	k8sGroup.Patch("/secret/:namespace/:name", patchSecretHandler)
	k8sGroup.Delete("/secret/:namespace/:name", deleteSecretHandler)
	// Namespaces
	k8sGroup.Get("/namespace/:name", getNamespaceHandler)
	k8sGroup.Delete("/namespace/:name", deleteNamespaceHandler)
	// Nodes
	k8sGroup.Get("/node/:name", getNodeHandler)
	// ServiceAccounts
	k8sGroup.Get("/serviceaccount/:namespace/:name", getServiceAccountHandler)
	k8sGroup.Delete("/serviceaccount/:namespace/:name", deleteServiceAccountHandler)
	// PersistentVolumes
	k8sGroup.Get("/pv/:name", getPVHandler)
	k8sGroup.Delete("/pv/:name", deletePVHandler)
	// PersistentVolumeClaims
	k8sGroup.Get("/pvc/:namespace/:name", getPVCHandler)
	k8sGroup.Patch("/pvc/:namespace/:name", patchPVCHandler)
	k8sGroup.Delete("/pvc/:namespace/:name", deletePVCHandler)
	// Ingresses
	k8sGroup.Get("/ingress/:namespace/:name", getIngressHandler)
	k8sGroup.Patch("/ingress/:namespace/:name", patchIngressHandler)
	k8sGroup.Delete("/ingress/:namespace/:name", deleteIngressHandler)
	// DaemonSets
	k8sGroup.Get("/daemonset/:namespace/:name", getDaemonSetHandler)
	k8sGroup.Patch("/daemonset/:namespace/:name", patchDaemonSetHandler)
	k8sGroup.Delete("/daemonset/:namespace/:name", deleteDaemonSetHandler)
	k8sGroup.Post("/daemonset/:namespace/:name/restart", restartDaemonSetHandler)
	// StatefulSets
	k8sGroup.Get("/statefulset/:namespace/:name", getStatefulSetHandler)
	k8sGroup.Patch("/statefulset/:namespace/:name", patchStatefulSetHandler)
	k8sGroup.Delete("/statefulset/:namespace/:name", deleteStatefulSetHandler)
	k8sGroup.Post("/statefulset/:namespace/:name/scale", scaleStatefulSetHandler)
	k8sGroup.Post("/statefulset/:namespace/:name/restart", restartStatefulSetHandler)
	// Jobs
	k8sGroup.Get("/job/:namespace/:name", getJobHandler)
	k8sGroup.Delete("/job/:namespace/:name", deleteJobHandler)
	// CronJobs
	k8sGroup.Get("/cronjob/:namespace/:name", getCronJobHandler)
	k8sGroup.Patch("/cronjob/:namespace/:name", patchCronJobHandler)
	k8sGroup.Delete("/cronjob/:namespace/:name", deleteCronJobHandler)
	// ReplicaSets
	k8sGroup.Get("/replicaset/:namespace/:name", getReplicaSetHandler)
	k8sGroup.Delete("/replicaset/:namespace/:name", deleteReplicaSetHandler)
	// Events
	k8sGroup.Get("/event/:namespace/:name", getEventHandler)
	// Create resource
	k8sGroup.Post("/create", createResourceHandler)

	// Docker endpoints (placeholder for future)
	docker := v1.Group("/docker")
	docker.Get("/containers", containersHandler)
	docker.Get("/images", imagesHandler)

	// Notepad endpoints
	notepad := v1.Group("/notepad")
	notepad.Get("/", listNotepadsHandler)
	notepad.Get("/:key", getNotepadHandler)
	notepad.Post("/:key", saveNotepadHandler)
	notepad.Delete("/:key", deleteNotepadHandler)

	// Desktop preferences endpoints
	prefs := v1.Group("/preferences")
	prefs.Get("/desktop", getDesktopPrefsHandler)
	prefs.Post("/desktop", saveDesktopPrefsHandler)
	prefs.Delete("/desktop", resetDesktopPrefsHandler)

	// CI/CD endpoints
	cicdGroup := v1.Group("/cicd")
	cicdGroup.Get("/stats", cicdStatsHandler)
	cicdGroup.Get("/sample", cicdSampleHandler)
	cicdGroup.Get("/pipelines", listPipelinesHandler)
	cicdGroup.Post("/pipelines", createPipelineHandler)
	cicdGroup.Get("/pipelines/:id", getPipelineHandler)
	cicdGroup.Put("/pipelines/:id", updatePipelineHandler)
	cicdGroup.Delete("/pipelines/:id", deletePipelineHandler)
	cicdGroup.Post("/pipelines/:id/trigger", triggerPipelineHandler)
	cicdGroup.Get("/pipelines/:id/runs", listPipelineRunsHandler)
	cicdGroup.Get("/pipelines/:id/badge", pipelineBadgeHandler)
	cicdGroup.Get("/runs", listAllRunsHandler)
	cicdGroup.Get("/runs/:runId", getRunHandler)
	cicdGroup.Post("/runs/:runId/cancel", cancelRunHandler)
	cicdGroup.Delete("/runs/:runId", deleteRunHandler)
	cicdGroup.Get("/runs/:runId/jobs/:job/logs", getJobLogsHandler)
	cicdGroup.Get("/artifacts", listArtifactsHandler)
	cicdGroup.Get("/artifacts/:id/download", downloadArtifactHandler)
	cicdGroup.Delete("/artifacts/:id", deleteArtifactHandler)

	// Notification configuration endpoints
	notifGroup := cicdGroup.Group("/notifications")
	notifGroup.Get("/", listNotificationsHandler)
	notifGroup.Post("/", createNotificationHandler)
	notifGroup.Get("/:id", getNotificationHandler)
	notifGroup.Put("/:id", updateNotificationHandler)
	notifGroup.Delete("/:id", deleteNotificationHandler)
	notifGroup.Post("/test", testNotificationHandler)

	// SSH Hosts endpoints
	sshGroup := cicdGroup.Group("/ssh")
	sshGroup.Get("/hosts", listSSHHostsHandler)
	sshGroup.Post("/hosts", createSSHHostHandler)
	sshGroup.Get("/hosts/:id", getSSHHostHandler)
	sshGroup.Put("/hosts/:id", updateSSHHostHandler)
	sshGroup.Delete("/hosts/:id", deleteSSHHostHandler)
	sshGroup.Post("/hosts/:id/test", testSSHHostHandler)
	sshGroup.Get("/groups", getSSHHostGroupsHandler)
	sshGroup.Post("/hostkey", getSSHHostKeyHandler)

	// Git Credentials endpoints
	gitGroup := cicdGroup.Group("/git")
	gitGroup.Get("/credentials", listGitCredentialsHandler)
	gitGroup.Post("/credentials", createGitCredentialHandler)
	gitGroup.Get("/credentials/:id", getGitCredentialHandler)
	gitGroup.Put("/credentials/:id", updateGitCredentialHandler)
	gitGroup.Delete("/credentials/:id", deleteGitCredentialHandler)
	gitGroup.Post("/credentials/:id/test", testGitCredentialHandler)

	// Freestyle Jobs endpoints
	freestyleGroup := cicdGroup.Group("/freestyle")
	freestyleGroup.Get("/jobs", listFreestyleJobsHandler)
	freestyleGroup.Post("/jobs", createFreestyleJobHandler)
	freestyleGroup.Get("/jobs/:id", getFreestyleJobHandler)
	freestyleGroup.Put("/jobs/:id", updateFreestyleJobHandler)
	freestyleGroup.Delete("/jobs/:id", deleteFreestyleJobHandler)
	freestyleGroup.Post("/jobs/:id/build", triggerFreestyleBuildHandler)
	freestyleGroup.Get("/jobs/:id/builds", listJobBuildsHandler)
	freestyleGroup.Get("/jobs/:id/badge", freestyleJobBadgeHandler)

	// Freestyle Builds endpoints
	freestyleGroup.Get("/builds", listFreestyleBuildsHandler)
	freestyleGroup.Get("/builds/:id", getFreestyleBuildHandler)
	freestyleGroup.Post("/builds/:id/cancel", cancelFreestyleBuildHandler)
	freestyleGroup.Delete("/builds/:id", deleteFreestyleBuildHandler)
	freestyleGroup.Get("/builds/:id/logs", getFreestyleBuildLogsHandler)

	// CI/CD Webhook endpoint (public - no auth for external triggers)
	app.Post("/api/v1/cicd/webhooks/:pipelineId/:token", cicdWebhookHandler)

	// Freestyle Webhook endpoint (public - no auth for external triggers)
	app.Post("/api/v1/cicd/freestyle/webhook/:token", freestyleWebhookHandler)

	// CI/CD Log stream WebSocket
	app.Use("/api/v1/cicd/runs/:runId/jobs/:job/logs/stream", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/api/v1/cicd/runs/:runId/jobs/:job/logs/stream", websocket.New(cicdLogStreamHandler))

	// Freestyle Build Log stream WebSocket
	app.Use("/api/v1/cicd/freestyle/builds/:id/logs/stream", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/api/v1/cicd/freestyle/builds/:id/logs/stream", websocket.New(freestyleLogStreamHandler))

	// Monitoring endpoints
	mon := v1.Group("/monitoring")
	mon.Get("/summary", monitoringSummaryHandler)
	mon.Get("/nodes", monitoringNodesHandler)
	mon.Get("/pods", monitoringPodsHandler)
	mon.Get("/pods/:namespace", monitoringPodsHandler)
	mon.Get("/quotas", monitoringQuotasHandler)
	mon.Get("/quotas/:namespace", monitoringQuotasHandler)
	mon.Get("/limitranges", monitoringLimitRangesHandler)
	mon.Get("/limitranges/:namespace", monitoringLimitRangesHandler)
	mon.Get("/hpa", monitoringHPAHandler)
	mon.Get("/hpa/:namespace", monitoringHPAHandler)

	// Tools endpoints
	toolsGroup := v1.Group("/tools")
	// Base64
	toolsGroup.Post("/base64/encode", base64EncodeHandler)
	toolsGroup.Post("/base64/decode", base64DecodeHandler)
	toolsGroup.Post("/base64/k8s-secret", k8sSecretDecodeHandler)
	// Hash
	toolsGroup.Post("/hash", hashHandler)
	toolsGroup.Post("/hash/compare", hashCompareHandler)
	// Certificate
	toolsGroup.Post("/cert/check", certCheckHandler)
	toolsGroup.Post("/cert/parse", certParseHandler)
	// SSH Keys
	toolsGroup.Post("/ssh/generate", sshGenerateHandler)
	toolsGroup.Post("/ssh/info", sshInfoHandler)
	// Converters
	toolsGroup.Post("/convert", convertHandler)
	// Diff
	toolsGroup.Post("/diff", diffHandler)

	// Database Tools - PostgreSQL
	pgGroup := v1.Group("/db/postgres")
	pgGroup.Post("/connect", postgresConnectHandler)
	pgGroup.Post("/info", postgresInfoHandler)
	pgGroup.Post("/query", postgresQueryHandler)
	pgGroup.Post("/dump", postgresDumpHandler)
	pgGroup.Post("/databases", postgresDatabasesHandler)

	// Database Tools - Redis
	redisGroup := v1.Group("/db/redis")
	redisGroup.Post("/connect", redisConnectHandler)
	redisGroup.Post("/info", redisInfoHandler)
	redisGroup.Post("/cluster", redisClusterHandler)
	redisGroup.Post("/scan", redisScanHandler)
	redisGroup.Post("/key", redisKeyHandler)
	redisGroup.Post("/command", redisCommandHandler)

	// Database Tools - MySQL/MariaDB
	mysqlGroup := v1.Group("/db/mysql")
	mysqlGroup.Post("/connect", mysqlConnectHandler)
	mysqlGroup.Post("/info", mysqlInfoHandler)
	mysqlGroup.Post("/query", mysqlQueryHandler)
	mysqlGroup.Post("/dump", mysqlDumpHandler)
	mysqlGroup.Post("/databases", mysqlDatabasesHandler)

	// S3 Storage
	s3Group := v1.Group("/storage/s3")
	s3Group.Post("/connect", s3ConnectHandler)
	s3Group.Post("/buckets", s3ListBucketsHandler)
	s3Group.Post("/bucket/create", s3CreateBucketHandler)
	s3Group.Post("/bucket/delete", s3DeleteBucketHandler)
	s3Group.Post("/objects", s3ListObjectsHandler)
	s3Group.Post("/object/info", s3ObjectInfoHandler)
	s3Group.Post("/object/upload", s3UploadHandler)
	s3Group.Post("/object/download", s3DownloadHandler)
	s3Group.Post("/object/delete", s3DeleteHandler)
	s3Group.Post("/object/presign", s3PresignHandler)

	// Elasticsearch
	esGroup := v1.Group("/elasticsearch")
	esGroup.Post("/connect", esConnectHandler)
	esGroup.Post("/health", esHealthHandler)
	esGroup.Post("/stats", esStatsHandler)
	esGroup.Post("/nodes", esNodesHandler)
	esGroup.Post("/indices", esIndicesHandler)
	esGroup.Post("/index/create", esCreateIndexHandler)
	esGroup.Post("/index/delete", esDeleteIndexHandler)
	esGroup.Post("/index/mapping", esIndexMappingHandler)
	esGroup.Post("/index/settings", esIndexSettingsHandler)
	esGroup.Post("/index/refresh", esRefreshIndexHandler)
	esGroup.Post("/search", esSearchHandler)
	esGroup.Post("/document", esGetDocumentHandler)
	esGroup.Post("/document/delete", esDeleteDocumentHandler)
	esGroup.Post("/query", esQueryHandler)

	// WebSocket terminal endpoint
	app.Use("/api/v1/terminal/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/api/v1/terminal/ws", websocket.New(terminal.HandleWebSocket))
}

// Basic handlers

func healthHandler(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func apiInfoHandler(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"name":        "GAGOS API",
		"description": "Go-based Administration & GitOps System - Network Multi-Tool",
		"version":     version,
		"endpoints": fiber.Map{
			"health":  "/api/health",
			"version": "/api/version",
			"network": fiber.Map{
				"ping":       "POST /api/v1/network/ping",
				"dns":        "POST /api/v1/network/dns",
				"port-check": "POST /api/v1/network/port-check",
				"traceroute": "POST /api/v1/network/traceroute",
			},
			"kubernetes": fiber.Map{
				"namespaces":  "GET /api/v1/k8s/namespaces",
				"nodes":       "GET /api/v1/k8s/nodes",
				"pods":        "GET /api/v1/k8s/pods[/:namespace]",
				"services":    "GET /api/v1/k8s/services[/:namespace]",
				"deployments": "GET /api/v1/k8s/deployments[/:namespace]",
			},
		},
	})
}

func versionHandler(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"version":    version,
		"build_time": buildTime,
	})
}

// Network handlers

type PingRequest struct {
	Host    string `json:"host"`
	Count   int    `json:"count"`
	Timeout int    `json:"timeout"` // seconds
}

func pingHandler(c *fiber.Ctx) error {
	var req PingRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Host == "" {
		return c.Status(400).JSON(fiber.Map{"error": "host is required"})
	}

	if req.Count <= 0 || req.Count > 10 {
		req.Count = 4
	}
	if req.Timeout <= 0 || req.Timeout > 30 {
		req.Timeout = 10
	}

	result := network.Ping(req.Host, req.Count, time.Duration(req.Timeout)*time.Second)
	return c.JSON(result)
}

type DNSRequest struct {
	Host       string `json:"host"`
	RecordType string `json:"record_type"` // A, AAAA, CNAME, MX, NS, TXT
}

func dnsHandler(c *fiber.Ctx) error {
	var req DNSRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Host == "" {
		return c.Status(400).JSON(fiber.Map{"error": "host is required"})
	}

	if req.RecordType == "" {
		req.RecordType = "A"
	}

	result := network.DNSLookup(req.Host, req.RecordType)
	return c.JSON(result)
}

type PortCheckRequest struct {
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Timeout int    `json:"timeout"` // seconds
}

func portCheckHandler(c *fiber.Ctx) error {
	var req PortCheckRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Host == "" {
		return c.Status(400).JSON(fiber.Map{"error": "host is required"})
	}
	if req.Port <= 0 || req.Port > 65535 {
		return c.Status(400).JSON(fiber.Map{"error": "valid port (1-65535) is required"})
	}
	if req.Timeout <= 0 || req.Timeout > 30 {
		req.Timeout = 5
	}

	result := network.CheckPort(req.Host, req.Port, time.Duration(req.Timeout)*time.Second)
	return c.JSON(result)
}

type TracerouteRequest struct {
	Host    string `json:"host"`
	MaxHops int    `json:"max_hops"`
	Timeout int    `json:"timeout"` // seconds
}

func tracerouteHandler(c *fiber.Ctx) error {
	var req TracerouteRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Host == "" {
		return c.Status(400).JSON(fiber.Map{"error": "host is required"})
	}

	if req.MaxHops <= 0 || req.MaxHops > 30 {
		req.MaxHops = 15
	}
	if req.Timeout <= 0 || req.Timeout > 60 {
		req.Timeout = 30
	}

	result := network.Traceroute(req.Host, req.MaxHops, time.Duration(req.Timeout)*time.Second)
	return c.JSON(result)
}

// Kubernetes handlers

func namespacesHandler(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	namespaces, err := k8s.ListNamespaces(ctx)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"count":      len(namespaces),
		"namespaces": namespaces,
	})
}

func nodesHandler(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	nodes, err := k8s.ListNodes(ctx)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"count": len(nodes),
		"nodes": nodes,
	})
}

func podsHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace", "")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pods, err := k8s.ListPods(ctx, namespace)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"namespace": namespace,
		"count":     len(pods),
		"pods":      pods,
	})
}

func servicesHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace", "")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	services, err := k8s.ListServices(ctx, namespace)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"namespace": namespace,
		"count":     len(services),
		"services":  services,
	})
}

func deploymentsHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace", "")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	deployments, err := k8s.ListDeployments(ctx, namespace)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"namespace":   namespace,
		"count":       len(deployments),
		"deployments": deployments,
	})
}

// Single resource handlers - Pods

func getPodHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	detail, err := k8s.GetPod(ctx, namespace, name)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(detail)
}

func getPodLogsHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")
	container := c.Query("container", "")
	tailLines := c.QueryInt("tail", 100)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logs, err := k8s.GetPodLogs(ctx, namespace, name, container, int64(tailLines))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"namespace": namespace,
		"pod":       name,
		"container": container,
		"logs":      logs,
	})
}

func patchPodHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")

	var req struct {
		YAML string `json:"yaml"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := k8s.PatchPod(ctx, namespace, name, req.YAML); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Pod updated"})
}

func deletePodHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := k8s.DeletePod(ctx, namespace, name); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Pod deleted"})
}

// Single resource handlers - Services

func getServiceHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	detail, err := k8s.GetService(ctx, namespace, name)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(detail)
}

func patchServiceHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")

	var req struct {
		YAML string `json:"yaml"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := k8s.PatchService(ctx, namespace, name, req.YAML); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Service updated"})
}

func deleteServiceHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := k8s.DeleteService(ctx, namespace, name); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Service deleted"})
}

// Single resource handlers - Deployments

func getDeploymentHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	detail, err := k8s.GetDeployment(ctx, namespace, name)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(detail)
}

func patchDeploymentHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")

	var req struct {
		YAML string `json:"yaml"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := k8s.PatchDeployment(ctx, namespace, name, req.YAML); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Deployment updated"})
}

func deleteDeploymentHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := k8s.DeleteDeployment(ctx, namespace, name); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Deployment deleted"})
}

func scaleDeploymentHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")

	var req struct {
		Replicas int32 `json:"replicas"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := k8s.ScaleDeployment(ctx, namespace, name, req.Replicas); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": fmt.Sprintf("Deployment scaled to %d replicas", req.Replicas)})
}

func restartDeploymentHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := k8s.RestartDeployment(ctx, namespace, name); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Deployment restart triggered"})
}

// Single resource handlers - ConfigMaps

func getConfigMapHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	detail, err := k8s.GetConfigMap(ctx, namespace, name)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(detail)
}

func patchConfigMapHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")

	var req struct {
		YAML string `json:"yaml"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := k8s.PatchConfigMap(ctx, namespace, name, req.YAML); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "ConfigMap updated"})
}

func deleteConfigMapHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := k8s.DeleteConfigMap(ctx, namespace, name); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "ConfigMap deleted"})
}

// Single resource handlers - Secrets

func getSecretHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	detail, err := k8s.GetSecret(ctx, namespace, name)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(detail)
}

func patchSecretHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")

	var req struct {
		YAML string `json:"yaml"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := k8s.PatchSecret(ctx, namespace, name, req.YAML); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Secret updated"})
}

func deleteSecretHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := k8s.DeleteSecret(ctx, namespace, name); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Secret deleted"})
}

// Single resource handlers - Namespaces

func getNamespaceHandler(c *fiber.Ctx) error {
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	detail, err := k8s.GetNamespace(ctx, name)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(detail)
}

func deleteNamespaceHandler(c *fiber.Ctx) error {
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := k8s.DeleteNamespace(ctx, name); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Namespace deleted"})
}

// Single resource handlers - Nodes

func getNodeHandler(c *fiber.Ctx) error {
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	detail, err := k8s.GetNode(ctx, name)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(detail)
}

// List handlers for additional K8s resources

func configMapsHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace", "")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cms, err := k8s.ListConfigMaps(ctx, namespace)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"namespace":  namespace,
		"count":      len(cms),
		"configmaps": cms,
	})
}

func secretsHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace", "")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	secrets, err := k8s.ListSecrets(ctx, namespace)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"namespace": namespace,
		"count":     len(secrets),
		"secrets":   secrets,
	})
}

func serviceAccountsHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace", "")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sas, err := k8s.ListServiceAccounts(ctx, namespace)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"namespace":       namespace,
		"count":           len(sas),
		"serviceaccounts": sas,
	})
}

func pvsHandler(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pvs, err := k8s.ListPersistentVolumes(ctx)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"count": len(pvs),
		"pvs":   pvs,
	})
}

func pvcsHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace", "")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pvcs, err := k8s.ListPersistentVolumeClaims(ctx, namespace)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"namespace": namespace,
		"count":     len(pvcs),
		"pvcs":      pvcs,
	})
}

func ingressesHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace", "")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ingresses, err := k8s.ListIngresses(ctx, namespace)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"namespace": namespace,
		"count":     len(ingresses),
		"ingresses": ingresses,
	})
}

func daemonSetsHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace", "")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dss, err := k8s.ListDaemonSets(ctx, namespace)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"namespace":  namespace,
		"count":      len(dss),
		"daemonsets": dss,
	})
}

func statefulSetsHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace", "")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sss, err := k8s.ListStatefulSets(ctx, namespace)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"namespace":    namespace,
		"count":        len(sss),
		"statefulsets": sss,
	})
}

func jobsHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace", "")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	jobs, err := k8s.ListJobs(ctx, namespace)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"namespace": namespace,
		"count":     len(jobs),
		"jobs":      jobs,
	})
}

func cronJobsHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace", "")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cjs, err := k8s.ListCronJobs(ctx, namespace)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"namespace": namespace,
		"count":     len(cjs),
		"cronjobs":  cjs,
	})
}

func eventsHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace", "")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	events, err := k8s.ListEvents(ctx, namespace)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"namespace": namespace,
		"count":     len(events),
		"events":    events,
	})
}

func replicaSetsHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace", "")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rss, err := k8s.ListReplicaSets(ctx, namespace)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"namespace":   namespace,
		"count":       len(rss),
		"replicasets": rss,
	})
}

// Single resource handlers for additional K8s resources

func getServiceAccountHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	detail, err := k8s.GetServiceAccount(ctx, namespace, name)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(detail)
}

func deleteServiceAccountHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := k8s.DeleteServiceAccount(ctx, namespace, name); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "ServiceAccount deleted"})
}

func getPVHandler(c *fiber.Ctx) error {
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	detail, err := k8s.GetPersistentVolume(ctx, name)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(detail)
}

func deletePVHandler(c *fiber.Ctx) error {
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := k8s.DeletePersistentVolume(ctx, name); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "PersistentVolume deleted"})
}

func getPVCHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	detail, err := k8s.GetPersistentVolumeClaim(ctx, namespace, name)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(detail)
}

func patchPVCHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")

	var req struct {
		YAML string `json:"yaml"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := k8s.PatchPersistentVolumeClaim(ctx, namespace, name, req.YAML); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "PVC updated"})
}

func deletePVCHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := k8s.DeletePersistentVolumeClaim(ctx, namespace, name); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "PVC deleted"})
}

func getIngressHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	detail, err := k8s.GetIngress(ctx, namespace, name)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(detail)
}

func patchIngressHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")

	var req struct {
		YAML string `json:"yaml"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := k8s.PatchIngress(ctx, namespace, name, req.YAML); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Ingress updated"})
}

func deleteIngressHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := k8s.DeleteIngress(ctx, namespace, name); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Ingress deleted"})
}

func getDaemonSetHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	detail, err := k8s.GetDaemonSet(ctx, namespace, name)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(detail)
}

func patchDaemonSetHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")

	var req struct {
		YAML string `json:"yaml"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := k8s.PatchDaemonSet(ctx, namespace, name, req.YAML); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "DaemonSet updated"})
}

func deleteDaemonSetHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := k8s.DeleteDaemonSet(ctx, namespace, name); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "DaemonSet deleted"})
}

func restartDaemonSetHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := k8s.RestartDaemonSet(ctx, namespace, name); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "DaemonSet restart triggered"})
}

func getStatefulSetHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	detail, err := k8s.GetStatefulSet(ctx, namespace, name)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(detail)
}

func patchStatefulSetHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")

	var req struct {
		YAML string `json:"yaml"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := k8s.PatchStatefulSet(ctx, namespace, name, req.YAML); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "StatefulSet updated"})
}

func deleteStatefulSetHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := k8s.DeleteStatefulSet(ctx, namespace, name); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "StatefulSet deleted"})
}

func scaleStatefulSetHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")

	var req struct {
		Replicas int32 `json:"replicas"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := k8s.ScaleStatefulSet(ctx, namespace, name, req.Replicas); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": fmt.Sprintf("StatefulSet scaled to %d replicas", req.Replicas)})
}

func restartStatefulSetHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := k8s.RestartStatefulSet(ctx, namespace, name); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "StatefulSet restart triggered"})
}

func getJobHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	detail, err := k8s.GetJob(ctx, namespace, name)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(detail)
}

func deleteJobHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := k8s.DeleteJob(ctx, namespace, name); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Job deleted"})
}

func getCronJobHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	detail, err := k8s.GetCronJob(ctx, namespace, name)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(detail)
}

func patchCronJobHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")

	var req struct {
		YAML string `json:"yaml"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := k8s.PatchCronJob(ctx, namespace, name, req.YAML); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "CronJob updated"})
}

func deleteCronJobHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := k8s.DeleteCronJob(ctx, namespace, name); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "CronJob deleted"})
}

func getReplicaSetHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	detail, err := k8s.GetReplicaSet(ctx, namespace, name)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(detail)
}

func deleteReplicaSetHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := k8s.DeleteReplicaSet(ctx, namespace, name); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "ReplicaSet deleted"})
}

func getEventHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	detail, err := k8s.GetEvent(ctx, namespace, name)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(detail)
}

// Create resource handler
func createResourceHandler(c *fiber.Ctx) error {
	var req struct {
		Type      string `json:"type"`
		Namespace string `json:"namespace"`
		YAML      string `json:"yaml"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Type == "" {
		return c.Status(400).JSON(fiber.Map{"error": "resource type is required"})
	}
	if req.YAML == "" {
		return c.Status(400).JSON(fiber.Map{"error": "YAML content is required"})
	}
	if req.Namespace == "" {
		req.Namespace = "default"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var err error
	switch req.Type {
	case "deployment":
		err = k8s.CreateDeployment(ctx, req.Namespace, req.YAML)
	case "service":
		err = k8s.CreateService(ctx, req.Namespace, req.YAML)
	case "configmap":
		err = k8s.CreateConfigMap(ctx, req.Namespace, req.YAML)
	case "secret":
		err = k8s.CreateSecret(ctx, req.Namespace, req.YAML)
	case "ingress":
		err = k8s.CreateIngress(ctx, req.Namespace, req.YAML)
	case "pod":
		err = k8s.CreatePod(ctx, req.Namespace, req.YAML)
	case "cronjob":
		err = k8s.CreateCronJob(ctx, req.Namespace, req.YAML)
	case "job":
		err = k8s.CreateJob(ctx, req.Namespace, req.YAML)
	case "pvc":
		err = k8s.CreatePersistentVolumeClaim(ctx, req.Namespace, req.YAML)
	case "serviceaccount":
		err = k8s.CreateServiceAccount(ctx, req.Namespace, req.YAML)
	case "daemonset":
		err = k8s.CreateDaemonSet(ctx, req.Namespace, req.YAML)
	case "statefulset":
		err = k8s.CreateStatefulSet(ctx, req.Namespace, req.YAML)
	default:
		return c.Status(400).JSON(fiber.Map{"error": "unsupported resource type: " + req.Type})
	}

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": fmt.Sprintf("%s created successfully", req.Type)})
}

// New Network Tool handlers

type TelnetRequest struct {
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Command string `json:"command"`
	Timeout int    `json:"timeout"`
}

func telnetHandler(c *fiber.Ctx) error {
	var req TelnetRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Host == "" {
		return c.Status(400).JSON(fiber.Map{"error": "host is required"})
	}
	if req.Port <= 0 || req.Port > 65535 {
		return c.Status(400).JSON(fiber.Map{"error": "valid port (1-65535) is required"})
	}
	if req.Timeout <= 0 || req.Timeout > 30 {
		req.Timeout = 10
	}

	result := network.TelnetConnect(req.Host, req.Port, req.Command, time.Duration(req.Timeout)*time.Second)
	return c.JSON(result)
}

type WhoisRequest struct {
	Query   string `json:"query"`
	Timeout int    `json:"timeout"`
}

func whoisHandler(c *fiber.Ctx) error {
	var req WhoisRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Query == "" {
		return c.Status(400).JSON(fiber.Map{"error": "query is required"})
	}
	if req.Timeout <= 0 || req.Timeout > 30 {
		req.Timeout = 10
	}

	result := network.Whois(req.Query, time.Duration(req.Timeout)*time.Second)
	return c.JSON(result)
}

type SSLCheckRequest struct {
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Timeout int    `json:"timeout"`
}

func sslCheckHandler(c *fiber.Ctx) error {
	var req SSLCheckRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Host == "" {
		return c.Status(400).JSON(fiber.Map{"error": "host is required"})
	}
	if req.Port <= 0 {
		req.Port = 443
	}
	if req.Timeout <= 0 || req.Timeout > 30 {
		req.Timeout = 10
	}

	result := network.CheckSSL(req.Host, req.Port, time.Duration(req.Timeout)*time.Second)
	return c.JSON(result)
}

type CurlRequest struct {
	URL             string            `json:"url"`
	Method          string            `json:"method"`
	Headers         map[string]string `json:"headers"`
	Body            string            `json:"body"`
	Timeout         int               `json:"timeout"`
	FollowRedirects bool              `json:"follow_redirects"`
	IncludeBody     bool              `json:"include_body"`
}

func curlHandler(c *fiber.Ctx) error {
	var req CurlRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.URL == "" {
		return c.Status(400).JSON(fiber.Map{"error": "url is required"})
	}
	if req.Method == "" {
		req.Method = "GET"
	}
	if req.Timeout <= 0 || req.Timeout > 60 {
		req.Timeout = 30
	}

	result := network.Curl(req.URL, req.Method, req.Headers, req.Body, time.Duration(req.Timeout)*time.Second, req.FollowRedirects, req.IncludeBody)
	return c.JSON(result)
}

func interfacesHandler(c *fiber.Ctx) error {
	result := network.GetNetworkInfo()
	return c.JSON(result)
}

// Docker handlers (placeholders)

func containersHandler(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Docker containers endpoint - coming soon",
		"status":  "placeholder",
	})
}

func imagesHandler(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Docker images endpoint - coming soon",
		"status":  "placeholder",
	})
}

// Notepad handlers

func listNotepadsHandler(c *fiber.Ctx) error {
	keys, err := storage.ListNotepads()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if keys == nil {
		keys = []string{}
	}
	return c.JSON(fiber.Map{
		"notepads": keys,
		"count":    len(keys),
	})
}

func getNotepadHandler(c *fiber.Ctx) error {
	key := c.Params("key")
	if key == "" {
		return c.Status(400).JSON(fiber.Map{"error": "key is required"})
	}

	data, err := storage.GetNotepad(key)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	if data == nil {
		return c.JSON(fiber.Map{
			"key":     key,
			"content": "",
		})
	}

	return c.JSON(fiber.Map{
		"key":       key,
		"content":   data.Content,
		"updatedAt": data.UpdatedAt,
	})
}

func saveNotepadHandler(c *fiber.Ctx) error {
	key := c.Params("key")
	if key == "" {
		return c.Status(400).JSON(fiber.Map{"error": "key is required"})
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	data := &storage.NotepadData{
		Content:   req.Content,
		UpdatedAt: time.Now().Unix(),
	}

	if err := storage.SaveNotepad(key, data); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success":   true,
		"key":       key,
		"updatedAt": data.UpdatedAt,
	})
}

func deleteNotepadHandler(c *fiber.Ctx) error {
	key := c.Params("key")
	if key == "" {
		return c.Status(400).JSON(fiber.Map{"error": "key is required"})
	}

	if err := storage.DeleteNotepad(key); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"key":     key,
	})
}

// Desktop preferences handlers

func getDesktopPrefsHandler(c *fiber.Ctx) error {
	prefs, err := storage.GetDesktopPreferences()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	if prefs == nil {
		return c.JSON(fiber.Map{})
	}

	// Return whichever format is stored
	result := fiber.Map{"updated_at": prefs.UpdatedAt}
	if prefs.Slots != nil {
		result["slots"] = prefs.Slots
	}
	if prefs.IconOrder != nil {
		result["icon_order"] = prefs.IconOrder
	}
	if prefs.Hidden != nil {
		result["hidden"] = prefs.Hidden
	}
	return c.JSON(result)
}

func saveDesktopPrefsHandler(c *fiber.Ctx) error {
	var req struct {
		Slots     []*string `json:"slots"`
		IconOrder []string  `json:"icon_order"`
		Hidden    []string  `json:"hidden"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	prefs := &storage.DesktopPreferences{
		Slots:     req.Slots,
		IconOrder: req.IconOrder,
		Hidden:    req.Hidden,
		UpdatedAt: time.Now().Unix(),
	}

	if err := storage.SaveDesktopPreferences(prefs); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success":    true,
		"updated_at": prefs.UpdatedAt,
	})
}

func resetDesktopPrefsHandler(c *fiber.Ctx) error {
	if err := storage.DeletePreference("desktop"); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
	})
}

// Auth handlers

func loginPageHandler(c *fiber.Ctx) error {
	return c.SendFile("/app/web/static/login.html")
}

func loginHandler(c *fiber.Ctx) error {
	var req struct {
		Password string `json:"password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	if !auth.ValidatePassword(req.Password) {
		return c.Status(401).JSON(fiber.Map{"error": "invalid password"})
	}

	token := auth.CreateSession()
	c.Cookie(&fiber.Cookie{
		Name:     "gagos_session",
		Value:    token,
		HTTPOnly: true,
		Secure:   false, // Set true for HTTPS
		SameSite: "Lax",
		MaxAge:   86400, // 24 hours
	})

	return c.JSON(fiber.Map{"success": true})
}

func logoutHandler(c *fiber.Ctx) error {
	token := c.Cookies("gagos_session")
	if token != "" {
		auth.DeleteSession(token)
	}

	c.Cookie(&fiber.Cookie{
		Name:   "gagos_session",
		Value:  "",
		MaxAge: -1,
	})

	return c.JSON(fiber.Map{"success": true})
}

func runtimeHandler(c *fiber.Ctx) error {
	runtime := getEnv("GAGOS_RUNTIME", "docker")
	return c.JSON(fiber.Map{
		"runtime": runtime,
	})
}

// CI/CD handlers

func cicdStatsHandler(c *fiber.Ctx) error {
	stats, err := cicd.GetStats()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(stats)
}

func cicdSampleHandler(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"yaml": cicd.GetSamplePipelineYAML(),
	})
}

func listPipelinesHandler(c *fiber.Ctx) error {
	pipelines, err := cicd.ListPipelines()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"count":     len(pipelines),
		"pipelines": pipelines,
	})
}

func createPipelineHandler(c *fiber.Ctx) error {
	var req cicd.CreatePipelineRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.YAML == "" {
		return c.Status(400).JSON(fiber.Map{"error": "yaml is required"})
	}

	pipeline, err := cicd.ParsePipelineYAML(req.YAML)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	if err := cicd.SavePipeline(pipeline); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Register with scheduler if has cron triggers
	if scheduler := cicd.GetScheduler(); scheduler != nil {
		scheduler.RegisterPipeline(pipeline)
	}

	return c.Status(201).JSON(fiber.Map{
		"id":          pipeline.ID,
		"name":        pipeline.Name,
		"webhook_url": pipeline.Status.WebhookURL,
		"created_at":  pipeline.CreatedAt.Format(time.RFC3339),
	})
}

func getPipelineHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	pipeline, err := cicd.GetPipeline(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(pipeline)
}

func pipelineBadgeHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	pipeline, err := cicd.GetPipeline(id)
	if err != nil {
		return c.Status(404).SendString(generateBadgeSVG("not found", "#9f9f9f"))
	}

	// Determine status and color
	status := "no builds"
	color := "#9f9f9f"

	// Get last run status if we have a last run
	if pipeline.Status.LastRunID != "" {
		run, err := cicd.GetRun(pipeline.Status.LastRunID)
		if err == nil {
			switch run.Status {
			case cicd.RunStatusSucceeded:
				status = "passing"
				color = "#4ade80"
			case cicd.RunStatusFailed:
				status = "failing"
				color = "#ef4444"
			case cicd.RunStatusRunning:
				status = "running"
				color = "#fbbf24"
			case cicd.RunStatusCancelled:
				status = "cancelled"
				color = "#9f9f9f"
			default:
				status = string(run.Status)
			}
		}
	}

	c.Set("Content-Type", "image/svg+xml")
	c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	return c.SendString(generateBadgeSVG(status, color))
}

func generateBadgeSVG(status, color string) string {
	labelWidth := 50
	statusWidth := len(status)*7 + 10
	totalWidth := labelWidth + statusWidth

	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="20">
  <linearGradient id="b" x2="0" y2="100%%">
    <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/>
  </linearGradient>
  <mask id="a">
    <rect width="%d" height="20" rx="3" fill="#fff"/>
  </mask>
  <g mask="url(#a)">
    <rect width="%d" height="20" fill="#555"/>
    <rect x="%d" width="%d" height="20" fill="%s"/>
    <rect width="%d" height="20" fill="url(#b)"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11">
    <text x="%d" y="15" fill="#010101" fill-opacity=".3">build</text>
    <text x="%d" y="14">build</text>
    <text x="%d" y="15" fill="#010101" fill-opacity=".3">%s</text>
    <text x="%d" y="14">%s</text>
  </g>
</svg>`, totalWidth, totalWidth, labelWidth, labelWidth, statusWidth, color, totalWidth,
		labelWidth/2, labelWidth/2,
		labelWidth+statusWidth/2, status,
		labelWidth+statusWidth/2, status)
}

func updatePipelineHandler(c *fiber.Ctx) error {
	id := c.Params("id")

	// Get existing pipeline
	existing, err := cicd.GetPipeline(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}

	var req cicd.CreatePipelineRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.YAML == "" {
		return c.Status(400).JSON(fiber.Map{"error": "yaml is required"})
	}

	// Parse the new YAML
	newPipeline, err := cicd.ParsePipelineYAML(req.YAML)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	// Preserve ID and status
	newPipeline.ID = existing.ID
	newPipeline.Status.TotalRuns = existing.Status.TotalRuns
	newPipeline.Status.LastRunID = existing.Status.LastRunID
	newPipeline.Status.LastRunAt = existing.Status.LastRunAt
	newPipeline.Status.WebhookToken = existing.Status.WebhookToken
	newPipeline.Status.WebhookURL = existing.Status.WebhookURL
	newPipeline.CreatedAt = existing.CreatedAt
	newPipeline.UpdatedAt = time.Now()

	if err := cicd.SavePipeline(newPipeline); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Update scheduler
	if scheduler := cicd.GetScheduler(); scheduler != nil {
		scheduler.RegisterPipeline(newPipeline)
	}

	return c.JSON(fiber.Map{
		"id":         newPipeline.ID,
		"name":       newPipeline.Name,
		"updated_at": newPipeline.UpdatedAt.Format(time.RFC3339),
	})
}

func deletePipelineHandler(c *fiber.Ctx) error {
	id := c.Params("id")

	// Unregister from scheduler
	if scheduler := cicd.GetScheduler(); scheduler != nil {
		scheduler.UnregisterPipeline(id)
	}

	if err := cicd.DeletePipeline(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true})
}

func triggerPipelineHandler(c *fiber.Ctx) error {
	id := c.Params("id")

	pipeline, err := cicd.GetPipeline(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}

	var req cicd.TriggerPipelineRequest
	c.BodyParser(&req) // Optional body

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	run, err := cicd.TriggerPipeline(ctx, pipeline, "manual", "", req.Variables)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"run_id":     run.ID,
		"run_number": run.RunNumber,
		"status":     run.Status,
	})
}

func listPipelineRunsHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	limit := c.QueryInt("limit", 50)

	runs, err := cicd.ListRuns(id, limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"count": len(runs),
		"runs":  runs,
	})
}

func listAllRunsHandler(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 50)

	runs, err := cicd.ListRuns("", limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"count": len(runs),
		"runs":  runs,
	})
}

func getRunHandler(c *fiber.Ctx) error {
	runId := c.Params("runId")

	run, err := cicd.GetRun(runId)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(run)
}

func cancelRunHandler(c *fiber.Ctx) error {
	runId := c.Params("runId")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := cicd.CancelRun(ctx, runId); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true})
}

func deleteRunHandler(c *fiber.Ctx) error {
	runId := c.Params("runId")

	// Clean up artifacts
	cicd.CleanupRunArtifacts(runId)

	if err := cicd.DeleteRun(runId); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true})
}

func getJobLogsHandler(c *fiber.Ctx) error {
	runId := c.Params("runId")
	jobName := c.Params("job")
	tailLines := int64(c.QueryInt("tail", 1000))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logs, err := cicd.GetJobLogs(ctx, runId, jobName, tailLines)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"run_id": runId,
		"job":    jobName,
		"logs":   logs,
	})
}

func cicdLogStreamHandler(c *websocket.Conn) {
	runId := c.Params("runId")
	jobName := c.Params("job")
	cicd.StreamJobLogs(c, runId, jobName)
}

func cicdWebhookHandler(c *fiber.Ctx) error {
	pipelineId := c.Params("pipelineId")
	token := c.Params("token")

	var payload cicd.WebhookPayload
	c.BodyParser(&payload) // Optional

	signature := c.Get("X-Hub-Signature-256", "")

	run, err := cicd.HandleWebhook(pipelineId, token, &payload, signature)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"run_id":     run.ID,
		"run_number": run.RunNumber,
		"status":     run.Status,
	})
}

func listArtifactsHandler(c *fiber.Ctx) error {
	runId := c.Query("run_id", "")
	pipelineId := c.Query("pipeline_id", "")

	artifacts, err := cicd.ListArtifacts(runId, pipelineId)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"count":     len(artifacts),
		"artifacts": artifacts,
	})
}

func downloadArtifactHandler(c *fiber.Ctx) error {
	id := c.Params("id")

	file, artifact, err := cicd.GetArtifactFile(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}
	defer file.Close()

	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", artifact.Filename))
	c.Set("Content-Type", artifact.MimeType)

	return c.SendStream(file)
}

func deleteArtifactHandler(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := cicd.DeleteArtifact(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true})
}

// SSH Host handlers

func listSSHHostsHandler(c *fiber.Ctx) error {
	hosts, err := cicd.ListSSHHostsSafe()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"count": len(hosts),
		"hosts": hosts,
	})
}

func createSSHHostHandler(c *fiber.Ctx) error {
	var req cicd.CreateSSHHostRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Name == "" || req.Host == "" || req.Username == "" {
		return c.Status(400).JSON(fiber.Map{"error": "name, host, and username are required"})
	}

	if req.AuthMethod == "" {
		req.AuthMethod = cicd.SSHAuthPassword
	}

	host, err := cicd.CreateSSHHost(&req)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(host.ToSafe())
}

func getSSHHostHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	host, err := cicd.GetSSHHost(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(host.ToSafe())
}

func updateSSHHostHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	var req cicd.UpdateSSHHostRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	host, err := cicd.UpdateSSHHost(id, &req)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(host.ToSafe())
}

func deleteSSHHostHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := cicd.DeleteSSHHost(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true})
}

func testSSHHostHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := cicd.TestSSHHostConnection(id); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Connection test passed",
	})
}

func getSSHHostGroupsHandler(c *fiber.Ctx) error {
	groups, err := cicd.GetSSHHostGroups()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"groups": groups,
	})
}

func getSSHHostKeyHandler(c *fiber.Ctx) error {
	var req struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Host == "" {
		return c.Status(400).JSON(fiber.Map{"error": "host is required"})
	}
	if req.Port == 0 {
		req.Port = 22
	}

	keyInfo, err := cicd.GetHostFingerprint(req.Host, req.Port)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(keyInfo)
}

// Git Credential handlers

func listGitCredentialsHandler(c *fiber.Ctx) error {
	creds, err := cicd.ListGitCredentialsSafe()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"count":       len(creds),
		"credentials": creds,
	})
}

func createGitCredentialHandler(c *fiber.Ctx) error {
	var req cicd.CreateGitCredentialRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Name == "" {
		return c.Status(400).JSON(fiber.Map{"error": "name is required"})
	}

	if req.AuthMethod == "" {
		return c.Status(400).JSON(fiber.Map{"error": "auth_method is required"})
	}

	// Validate required fields based on auth method
	switch req.AuthMethod {
	case cicd.GitAuthToken:
		if req.Token == "" {
			return c.Status(400).JSON(fiber.Map{"error": "token is required for token authentication"})
		}
	case cicd.GitAuthPassword:
		if req.Username == "" || req.Password == "" {
			return c.Status(400).JSON(fiber.Map{"error": "username and password are required for password authentication"})
		}
	case cicd.GitAuthSSHKey:
		if req.PrivateKey == "" {
			return c.Status(400).JSON(fiber.Map{"error": "private_key is required for SSH key authentication"})
		}
	default:
		return c.Status(400).JSON(fiber.Map{"error": "invalid auth_method"})
	}

	cred, err := cicd.CreateGitCredential(&req)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(cred.ToSafe())
}

func getGitCredentialHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	cred, err := cicd.GetGitCredential(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(cred.ToSafe())
}

func updateGitCredentialHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	var req cicd.UpdateGitCredentialRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	cred, err := cicd.UpdateGitCredential(id, &req)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(cred.ToSafe())
}

func deleteGitCredentialHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := cicd.DeleteGitCredential(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true})
}

func testGitCredentialHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	var req cicd.TestGitCredentialRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.URL == "" {
		return c.Status(400).JSON(fiber.Map{"error": "url is required for testing"})
	}

	if err := cicd.TestGitCredential(id, req.URL); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Credential test passed",
	})
}

// Freestyle Job handlers

func listFreestyleJobsHandler(c *fiber.Ctx) error {
	jobs, err := cicd.ListFreestyleJobs()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"count": len(jobs),
		"jobs":  jobs,
	})
}

func createFreestyleJobHandler(c *fiber.Ctx) error {
	var req cicd.CreateFreestyleJobRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Name == "" {
		return c.Status(400).JSON(fiber.Map{"error": "name is required"})
	}

	job, err := cicd.CreateFreestyleJob(&req)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(job)
}

func getFreestyleJobHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	job, err := cicd.GetFreestyleJob(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(job)
}

func freestyleJobBadgeHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	job, err := cicd.GetFreestyleJob(id)
	if err != nil {
		return c.Status(404).SendString(generateBadgeSVG("not found", "#9f9f9f"))
	}

	// Determine status and color
	status := "unknown"
	color := "#9f9f9f"

	if job.Status.LastStatus != "" {
		switch job.Status.LastStatus {
		case "succeeded":
			status = "passing"
			color = "#4ade80"
		case "failed":
			status = "failing"
			color = "#ef4444"
		case "running":
			status = "running"
			color = "#fbbf24"
		case "cancelled":
			status = "cancelled"
			color = "#9f9f9f"
		default:
			status = job.Status.LastStatus
		}
	} else {
		status = "no builds"
	}

	c.Set("Content-Type", "image/svg+xml")
	c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	return c.SendString(generateBadgeSVG(status, color))
}

func updateFreestyleJobHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	var req cicd.CreateFreestyleJobRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	job, err := cicd.UpdateFreestyleJob(id, &req)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(job)
}

func deleteFreestyleJobHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := cicd.DeleteFreestyleJob(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true})
}

func triggerFreestyleBuildHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	var req cicd.TriggerFreestyleBuildRequest
	c.BodyParser(&req) // Optional params

	build, err := cicd.TriggerFreestyleBuild(id, "manual", "", req.Parameters)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(build)
}

func listJobBuildsHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	builds, err := cicd.ListFreestyleBuildsForJob(id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"count":  len(builds),
		"builds": builds,
	})
}

// Freestyle Build handlers

func listFreestyleBuildsHandler(c *fiber.Ctx) error {
	builds, err := cicd.ListFreestyleBuilds()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"count":  len(builds),
		"builds": builds,
	})
}

func getFreestyleBuildHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	build, err := cicd.GetFreestyleBuild(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(build)
}

func cancelFreestyleBuildHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := cicd.CancelFreestyleBuild(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true})
}

func deleteFreestyleBuildHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := cicd.DeleteFreestyleBuild(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true})
}

func getFreestyleBuildLogsHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	logs, err := cicd.GetBuildLogs(id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"build_id": id,
		"logs":     logs,
	})
}

func freestyleWebhookHandler(c *fiber.Ctx) error {
	token := c.Params("token")

	job, err := cicd.GetFreestyleJobByWebhookToken(token)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "invalid webhook token"})
	}

	// Check if webhook trigger is enabled
	webhookEnabled := false
	for _, t := range job.Triggers {
		if t.Type == "webhook" && t.Enabled {
			webhookEnabled = true
			break
		}
	}

	if !webhookEnabled {
		return c.Status(403).JSON(fiber.Map{"error": "webhook trigger is not enabled for this job"})
	}

	// Verify HMAC signature if secret is configured
	if job.Status.WebhookSecret != "" {
		signature := c.Get("X-GAGOS-Signature")
		if signature == "" {
			// Also check common webhook signature headers
			signature = c.Get("X-Hub-Signature-256")
			if signature == "" {
				signature = c.Get("X-Signature-256")
			}
		}

		if signature != "" {
			body := c.Body()
			if !cicd.VerifyWebhookSignature(body, signature, job.Status.WebhookSecret) {
				log.Warn().
					Str("job_id", job.ID).
					Str("job_name", job.Name).
					Msg("Webhook signature verification failed")
				return c.Status(401).JSON(fiber.Map{"error": "invalid signature"})
			}
		}
		// If no signature provided but secret is set, log a warning but allow
		// This allows for gradual adoption of signature verification
		if signature == "" {
			log.Warn().
				Str("job_id", job.ID).
				Str("job_name", job.Name).
				Msg("Webhook called without signature - consider adding X-GAGOS-Signature header")
		}
	}

	var params map[string]string
	c.BodyParser(&params)

	build, err := cicd.TriggerFreestyleBuild(job.ID, "webhook", "", params)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(fiber.Map{
		"build_id":     build.ID,
		"build_number": build.BuildNumber,
		"job_name":     build.JobName,
	})
}

func freestyleLogStreamHandler(c *websocket.Conn) {
	buildID := c.Params("id")

	stream := cicd.GetBuildOutputStream(buildID)
	if stream == nil {
		// Build not running, send existing logs and close
		logs, err := cicd.GetBuildLogs(buildID)
		if err != nil {
			c.WriteMessage(websocket.TextMessage, []byte("Error: "+err.Error()))
			return
		}
		c.WriteMessage(websocket.TextMessage, []byte(logs))
		return
	}

	// Subscribe to live output
	ch := stream.Subscribe()
	defer stream.Unsubscribe(ch)

	for data := range ch {
		if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
			break
		}
	}
}

// Monitoring handlers

func monitoringSummaryHandler(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	summary, err := monitoring.GetClusterSummary(ctx)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(summary)
}

func monitoringNodesHandler(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	nodes, err := monitoring.GetNodeMetrics(ctx)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"count":            len(nodes),
		"nodes":            nodes,
		"metrics_available": monitoring.IsMetricsAvailable(),
	})
}

func monitoringPodsHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace", "")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pods, err := monitoring.GetPodMetrics(ctx, namespace)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"namespace":         namespace,
		"count":             len(pods),
		"pods":              pods,
		"metrics_available": monitoring.IsMetricsAvailable(),
	})
}

func monitoringQuotasHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace", "")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	quotas, err := monitoring.ListResourceQuotas(ctx, namespace)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"namespace": namespace,
		"count":     len(quotas),
		"quotas":    quotas,
	})
}

func monitoringLimitRangesHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace", "")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	limitRanges, err := monitoring.ListLimitRanges(ctx, namespace)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"namespace":    namespace,
		"count":        len(limitRanges),
		"limit_ranges": limitRanges,
	})
}

func monitoringHPAHandler(c *fiber.Ctx) error {
	namespace := c.Params("namespace", "")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	hpas, err := monitoring.ListHPAs(ctx, namespace)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"namespace": namespace,
		"count":     len(hpas),
		"hpas":      hpas,
	})
}

// Tools handlers

func base64EncodeHandler(c *fiber.Ctx) error {
	var req struct {
		Input   string `json:"input"`
		URLSafe bool   `json:"url_safe"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	result := tools.EncodeBase64(req.Input, req.URLSafe)
	return c.JSON(result)
}

func base64DecodeHandler(c *fiber.Ctx) error {
	var req struct {
		Input   string `json:"input"`
		URLSafe bool   `json:"url_safe"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	result := tools.DecodeBase64(req.Input, req.URLSafe)
	if result.Error != "" {
		return c.Status(400).JSON(result)
	}
	return c.JSON(result)
}

func k8sSecretDecodeHandler(c *fiber.Ctx) error {
	var req struct {
		Data map[string]string `json:"data"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	result := tools.DecodeK8sSecret(req.Data)
	return c.JSON(fiber.Map{"decoded": result})
}

func hashHandler(c *fiber.Ctx) error {
	var req struct {
		Input     string `json:"input"`
		Algorithm string `json:"algorithm"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	if req.Algorithm == "" || req.Algorithm == "all" {
		result := tools.HashAll(req.Input)
		return c.JSON(result)
	}

	var result tools.HashResult
	switch req.Algorithm {
	case "md5":
		result = tools.HashMD5(req.Input)
	case "sha1":
		result = tools.HashSHA1(req.Input)
	case "sha256":
		result = tools.HashSHA256(req.Input)
	case "sha512":
		result = tools.HashSHA512(req.Input)
	default:
		return c.Status(400).JSON(fiber.Map{"error": "unsupported algorithm"})
	}
	return c.JSON(result)
}

func hashCompareHandler(c *fiber.Ctx) error {
	var req struct {
		Hash1 string `json:"hash1"`
		Hash2 string `json:"hash2"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	result := tools.CompareHashes(req.Hash1, req.Hash2)
	return c.JSON(result)
}

func certCheckHandler(c *fiber.Ctx) error {
	var req struct {
		Host    string `json:"host"`
		Port    int    `json:"port"`
		Timeout int    `json:"timeout"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if req.Host == "" {
		return c.Status(400).JSON(fiber.Map{"error": "host is required"})
	}
	if req.Port == 0 {
		req.Port = 443
	}
	if req.Timeout == 0 {
		req.Timeout = 10
	}
	result := tools.GetCertificateInfo(req.Host, req.Port, time.Duration(req.Timeout)*time.Second)
	if result.Error != "" {
		return c.Status(500).JSON(result)
	}
	return c.JSON(result)
}

func certParseHandler(c *fiber.Ctx) error {
	var req struct {
		PEM string `json:"pem"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if req.PEM == "" {
		return c.Status(400).JSON(fiber.Map{"error": "PEM data is required"})
	}
	result := tools.ParsePEMCertificate(req.PEM)
	if result.Error != "" {
		return c.Status(400).JSON(result)
	}
	return c.JSON(result)
}

func sshGenerateHandler(c *fiber.Ctx) error {
	var req struct {
		Algorithm string `json:"algorithm"`
		BitSize   int    `json:"bit_size"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if req.Algorithm == "" {
		req.Algorithm = "ED25519"
	}
	result := tools.GenerateSSHKeyPair(req.Algorithm, req.BitSize)
	if result.Error != "" {
		return c.Status(500).JSON(result)
	}
	return c.JSON(result)
}

func sshInfoHandler(c *fiber.Ctx) error {
	var req struct {
		Key string `json:"key"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if req.Key == "" {
		return c.Status(400).JSON(fiber.Map{"error": "key data is required"})
	}
	result := tools.GetSSHKeyInfo(req.Key)
	return c.JSON(result)
}

func convertHandler(c *fiber.Ctx) error {
	var req struct {
		Input string `json:"input"`
		From  string `json:"from"`
		To    string `json:"to"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if req.Input == "" {
		return c.Status(400).JSON(fiber.Map{"error": "input is required"})
	}

	var result tools.ConvertResult
	conversion := req.From + "->" + req.To

	switch conversion {
	case "csv->json":
		result = tools.CSVToJSON(req.Input)
	case "json->yaml":
		result = tools.JSONToYAML(req.Input)
	case "yaml->json":
		result = tools.YAMLToJSON(req.Input)
	case "xml->json":
		result = tools.XMLToJSON(req.Input)
	case "toml->yaml":
		result = tools.TOMLToYAML(req.Input)
	case "yaml->toml":
		result = tools.YAMLToTOML(req.Input)
	case "properties->yaml":
		result = tools.PropertiesToYAML(req.Input)
	case "json->json":
		if req.To == "minified" {
			result = tools.MinifyJSON(req.Input)
		} else {
			result = tools.FormatJSON(req.Input)
		}
	default:
		return c.Status(400).JSON(fiber.Map{"error": "unsupported conversion: " + conversion})
	}

	if result.Error != "" {
		return c.Status(400).JSON(result)
	}
	return c.JSON(result)
}

func diffHandler(c *fiber.Ctx) error {
	var req struct {
		Text1 string `json:"text1"`
		Text2 string `json:"text2"`
		Type  string `json:"type"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	var result tools.DiffResult
	switch req.Type {
	case "json":
		result = tools.JSONDiff(req.Text1, req.Text2)
	case "yaml":
		result = tools.YAMLDiff(req.Text1, req.Text2)
	default:
		result = tools.TextDiff(req.Text1, req.Text2)
	}

	if result.Error != "" {
		return c.Status(400).JSON(result)
	}
	return c.JSON(result)
}

// PostgreSQL handlers

func postgresConnectHandler(c *fiber.Ctx) error {
	var config database.PostgresConfig
	if err := c.BodyParser(&config); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if config.Port == 0 {
		config.Port = 5432
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := database.TestPostgresConnection(ctx, config)
	return c.JSON(result)
}

func postgresInfoHandler(c *fiber.Ctx) error {
	var config database.PostgresConfig
	if err := c.BodyParser(&config); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if config.Port == 0 {
		config.Port = 5432
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := database.GetPostgresInfo(ctx, config)
	return c.JSON(result)
}

func postgresQueryHandler(c *fiber.Ctx) error {
	var req struct {
		database.PostgresConfig
		Query    string `json:"query"`
		ReadOnly bool   `json:"readonly"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if req.Port == 0 {
		req.Port = 5432
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result := database.ExecutePostgresQuery(ctx, req.PostgresConfig, req.Query, req.ReadOnly)
	if result.Error != "" {
		return c.Status(400).JSON(result)
	}
	return c.JSON(result)
}

func postgresDumpHandler(c *fiber.Ctx) error {
	var req struct {
		database.PostgresConfig
		SchemaOnly bool     `json:"schema_only"`
		DataOnly   bool     `json:"data_only"`
		Tables     []string `json:"tables"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if req.Port == 0 {
		req.Port = 5432
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result := database.DumpPostgres(ctx, req.PostgresConfig, req.SchemaOnly, req.DataOnly, req.Tables)
	return c.JSON(result)
}

func postgresDatabasesHandler(c *fiber.Ctx) error {
	var config database.PostgresConfig
	if err := c.BodyParser(&config); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if config.Port == 0 {
		config.Port = 5432
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	databases, err := database.GetPostgresDatabases(ctx, config)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"databases": databases})
}

// Redis handlers

func redisConnectHandler(c *fiber.Ctx) error {
	var config database.RedisConfig
	if err := c.BodyParser(&config); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if config.Port == 0 {
		config.Port = 6379
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := database.TestRedisConnection(ctx, config)
	return c.JSON(result)
}

func redisInfoHandler(c *fiber.Ctx) error {
	var config database.RedisConfig
	if err := c.BodyParser(&config); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if config.Port == 0 {
		config.Port = 6379
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := database.GetRedisInfo(ctx, config)
	return c.JSON(result)
}

func redisClusterHandler(c *fiber.Ctx) error {
	var config database.RedisConfig
	if err := c.BodyParser(&config); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if config.Port == 0 {
		config.Port = 6379
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := database.GetRedisClusterInfo(ctx, config)
	return c.JSON(result)
}

func redisScanHandler(c *fiber.Ctx) error {
	var req struct {
		database.RedisConfig
		Pattern string `json:"pattern"`
		Cursor  uint64 `json:"cursor"`
		Count   int64  `json:"count"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if req.Port == 0 {
		req.Port = 6379
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := database.ScanRedisKeys(ctx, req.RedisConfig, req.Pattern, req.Cursor, req.Count)
	return c.JSON(result)
}

func redisKeyHandler(c *fiber.Ctx) error {
	var req struct {
		database.RedisConfig
		Key string `json:"key"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if req.Port == 0 {
		req.Port = 6379
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := database.GetRedisKeyValue(ctx, req.RedisConfig, req.Key)
	return c.JSON(result)
}

func redisCommandHandler(c *fiber.Ctx) error {
	var req struct {
		database.RedisConfig
		Command string `json:"command"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if req.Port == 0 {
		req.Port = 6379
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := database.ExecuteRedisCommand(ctx, req.RedisConfig, req.Command)
	return c.JSON(result)
}

// MySQL handlers

func mysqlConnectHandler(c *fiber.Ctx) error {
	var config database.MySQLConfig
	if err := c.BodyParser(&config); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if config.Port == 0 {
		config.Port = 3306
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := database.TestMySQLConnection(ctx, config)
	return c.JSON(result)
}

func mysqlInfoHandler(c *fiber.Ctx) error {
	var config database.MySQLConfig
	if err := c.BodyParser(&config); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if config.Port == 0 {
		config.Port = 3306
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := database.GetMySQLInfo(ctx, config)
	return c.JSON(result)
}

func mysqlQueryHandler(c *fiber.Ctx) error {
	var req struct {
		database.MySQLConfig
		Query    string `json:"query"`
		ReadOnly bool   `json:"readonly"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if req.Port == 0 {
		req.Port = 3306
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result := database.ExecuteMySQLQuery(ctx, req.MySQLConfig, req.Query, req.ReadOnly)
	if result.Error != "" {
		return c.Status(400).JSON(result)
	}
	return c.JSON(result)
}

func mysqlDumpHandler(c *fiber.Ctx) error {
	var req struct {
		database.MySQLConfig
		SchemaOnly bool     `json:"schema_only"`
		DataOnly   bool     `json:"data_only"`
		Tables     []string `json:"tables"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if req.Port == 0 {
		req.Port = 3306
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result := database.DumpMySQL(ctx, req.MySQLConfig, req.SchemaOnly, req.DataOnly, req.Tables)
	return c.JSON(result)
}

func mysqlDatabasesHandler(c *fiber.Ctx) error {
	var config database.MySQLConfig
	if err := c.BodyParser(&config); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if config.Port == 0 {
		config.Port = 3306
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	databases, err := database.GetMySQLDatabases(ctx, config)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"databases": databases})
}

// Notification handlers

func listNotificationsHandler(c *fiber.Ctx) error {
	configs, err := cicd.ListNotificationConfigs()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"count":         len(configs),
		"notifications": configs,
	})
}

func createNotificationHandler(c *fiber.Ctx) error {
	var config cicd.NotificationConfig
	if err := c.BodyParser(&config); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if config.Name == "" {
		return c.Status(400).JSON(fiber.Map{"error": "name is required"})
	}
	if config.URL == "" {
		return c.Status(400).JSON(fiber.Map{"error": "url is required"})
	}

	result, err := cicd.CreateNotificationConfig(&config)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(result)
}

func getNotificationHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	config, err := cicd.GetNotificationConfig(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(config)
}

func updateNotificationHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	var config cicd.NotificationConfig
	if err := c.BodyParser(&config); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	result, err := cicd.UpdateNotificationConfig(id, &config)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(result)
}

func deleteNotificationHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := cicd.DeleteNotificationConfig(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true})
}

func testNotificationHandler(c *fiber.Ctx) error {
	var req struct {
		URL     string            `json:"url"`
		Secret  string            `json:"secret"`
		Headers map[string]string `json:"headers"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.URL == "" {
		return c.Status(400).JSON(fiber.Map{"error": "url is required"})
	}

	// Create and save a temporary test notification config
	testConfig := &cicd.NotificationConfig{
		Name:    "Test Notification",
		Type:    cicd.NotificationTypeWebhook,
		Enabled: true,
		URL:     req.URL,
		Secret:  req.Secret,
		Headers: req.Headers,
		Events:  []cicd.NotificationEvent{cicd.NotificationEventBuildSucceeded},
	}

	// Create the config (this also loads it into memory)
	created, err := cicd.CreateNotificationConfig(testConfig)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to create test config: " + err.Error()})
	}

	// Create a test payload
	testBuild := &cicd.FreestyleBuild{
		ID:          "test-build",
		JobID:       "test-job",
		JobName:     "Test Job",
		BuildNumber: 1,
		Status:      cicd.RunStatusSucceeded,
		TriggerType: "test",
	}

	// Send test notification (this will use the config we just created)
	cicd.NotifyBuildEvent(cicd.NotificationEventBuildSucceeded, testBuild)

	// Clean up the test config
	cicd.DeleteNotificationConfig(created.ID)

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Test notification sent to " + req.URL,
	})
}

// S3 Storage handlers

func s3ConnectHandler(c *fiber.Ctx) error {
	var config database.S3Config
	if err := c.BodyParser(&config); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if config.Endpoint == "" {
		return c.Status(400).JSON(fiber.Map{"error": "endpoint is required"})
	}
	if config.AccessKeyID == "" || config.SecretAccessKey == "" {
		return c.Status(400).JSON(fiber.Map{"error": "access key and secret key are required"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := database.TestS3Connection(ctx, config)
	return c.JSON(result)
}

func s3ListBucketsHandler(c *fiber.Ctx) error {
	var config database.S3Config
	if err := c.BodyParser(&config); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	buckets, err := database.ListS3Buckets(ctx, config)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"buckets": buckets})
}

func s3CreateBucketHandler(c *fiber.Ctx) error {
	var req struct {
		database.S3Config
		Bucket string `json:"bucket"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Bucket == "" {
		return c.Status(400).JSON(fiber.Map{"error": "bucket name is required"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := database.CreateS3Bucket(ctx, req.S3Config, req.Bucket); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Bucket created"})
}

func s3DeleteBucketHandler(c *fiber.Ctx) error {
	var req struct {
		database.S3Config
		Bucket string `json:"bucket"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Bucket == "" {
		return c.Status(400).JSON(fiber.Map{"error": "bucket name is required"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := database.DeleteS3Bucket(ctx, req.S3Config, req.Bucket); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Bucket deleted"})
}

func s3ListObjectsHandler(c *fiber.Ctx) error {
	var req struct {
		database.S3Config
		Bucket  string `json:"bucket"`
		Prefix  string `json:"prefix"`
		MaxKeys int    `json:"max_keys"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Bucket == "" {
		return c.Status(400).JSON(fiber.Map{"error": "bucket is required"})
	}

	if req.MaxKeys <= 0 {
		req.MaxKeys = 1000
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	objects, err := database.ListS3Objects(ctx, req.S3Config, req.Bucket, req.Prefix, req.MaxKeys)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"objects": objects, "bucket": req.Bucket, "prefix": req.Prefix})
}

func s3ObjectInfoHandler(c *fiber.Ctx) error {
	var req struct {
		database.S3Config
		Bucket string `json:"bucket"`
		Key    string `json:"key"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Bucket == "" || req.Key == "" {
		return c.Status(400).JSON(fiber.Map{"error": "bucket and key are required"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	info, err := database.GetS3ObjectInfo(ctx, req.S3Config, req.Bucket, req.Key)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(info)
}

func s3UploadHandler(c *fiber.Ctx) error {
	// Get config from form fields
	config := database.S3Config{
		Endpoint:        c.FormValue("endpoint"),
		Region:          c.FormValue("region"),
		AccessKeyID:     c.FormValue("access_key_id"),
		SecretAccessKey: c.FormValue("secret_access_key"),
		UseSSL:          c.FormValue("use_ssl") == "true",
	}

	bucket := c.FormValue("bucket")
	prefix := c.FormValue("prefix")

	if bucket == "" {
		return c.Status(400).JSON(fiber.Map{"error": "bucket is required"})
	}

	// Get the file from the request
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "file is required"})
	}

	// Open the file
	src, err := file.Open()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to open file"})
	}
	defer src.Close()

	// Read file content
	data, err := io.ReadAll(src)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to read file"})
	}

	// Build the key (prefix + filename)
	key := file.Filename
	if prefix != "" {
		key = strings.TrimSuffix(prefix, "/") + "/" + file.Filename
	}

	// Detect content type from file header
	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := database.UploadS3Object(ctx, config, bucket, key, data, contentType); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "key": key, "size": len(data)})
}

func s3DownloadHandler(c *fiber.Ctx) error {
	var req struct {
		database.S3Config
		Bucket string `json:"bucket"`
		Key    string `json:"key"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Bucket == "" || req.Key == "" {
		return c.Status(400).JSON(fiber.Map{"error": "bucket and key are required"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	data, contentType, err := database.DownloadS3Object(ctx, req.S3Config, req.Bucket, req.Key)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Extract filename from key
	parts := strings.Split(req.Key, "/")
	filename := parts[len(parts)-1]

	c.Set("Content-Type", contentType)
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	return c.Send(data)
}

func s3DeleteHandler(c *fiber.Ctx) error {
	var req struct {
		database.S3Config
		Bucket string `json:"bucket"`
		Key    string `json:"key"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Bucket == "" || req.Key == "" {
		return c.Status(400).JSON(fiber.Map{"error": "bucket and key are required"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := database.DeleteS3Object(ctx, req.S3Config, req.Bucket, req.Key); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Object deleted"})
}

func s3PresignHandler(c *fiber.Ctx) error {
	var req struct {
		database.S3Config
		Bucket      string `json:"bucket"`
		Key         string `json:"key"`
		ExpiryHours int    `json:"expiry_hours"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Bucket == "" || req.Key == "" {
		return c.Status(400).JSON(fiber.Map{"error": "bucket and key are required"})
	}

	if req.ExpiryHours <= 0 {
		req.ExpiryHours = 24
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	url, err := database.GetPresignedURL(ctx, req.S3Config, req.Bucket, req.Key, time.Duration(req.ExpiryHours)*time.Hour)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"url": url, "expires_in_hours": req.ExpiryHours})
}

// Elasticsearch handlers

func esConnectHandler(c *fiber.Ctx) error {
	var config database.ESConfig
	if err := c.BodyParser(&config); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := database.TestESConnection(ctx, config)
	return c.JSON(result)
}

func esHealthHandler(c *fiber.Ctx) error {
	var config database.ESConfig
	if err := c.BodyParser(&config); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	health, err := database.GetESClusterHealth(ctx, config)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(health)
}

func esStatsHandler(c *fiber.Ctx) error {
	var config database.ESConfig
	if err := c.BodyParser(&config); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stats, err := database.GetESClusterStats(ctx, config)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(stats)
}

func esNodesHandler(c *fiber.Ctx) error {
	var config database.ESConfig
	if err := c.BodyParser(&config); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	nodes, err := database.GetESNodes(ctx, config)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Send(nodes)
}

func esIndicesHandler(c *fiber.Ctx) error {
	var config database.ESConfig
	if err := c.BodyParser(&config); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	indices, err := database.ListESIndices(ctx, config)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(indices)
}

func esCreateIndexHandler(c *fiber.Ctx) error {
	var req struct {
		database.ESConfig
		Index    string          `json:"index"`
		Settings json.RawMessage `json:"settings,omitempty"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Index == "" {
		return c.Status(400).JSON(fiber.Map{"error": "index name is required"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := database.CreateESIndex(ctx, req.ESConfig, req.Index, req.Settings); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Index created"})
}

func esDeleteIndexHandler(c *fiber.Ctx) error {
	var req struct {
		database.ESConfig
		Index string `json:"index"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Index == "" {
		return c.Status(400).JSON(fiber.Map{"error": "index name is required"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := database.DeleteESIndex(ctx, req.ESConfig, req.Index); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Index deleted"})
}

func esIndexMappingHandler(c *fiber.Ctx) error {
	var req struct {
		database.ESConfig
		Index string `json:"index"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Index == "" {
		return c.Status(400).JSON(fiber.Map{"error": "index name is required"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mapping, err := database.GetESIndexMapping(ctx, req.ESConfig, req.Index)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Send(mapping)
}

func esIndexSettingsHandler(c *fiber.Ctx) error {
	var req struct {
		database.ESConfig
		Index string `json:"index"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Index == "" {
		return c.Status(400).JSON(fiber.Map{"error": "index name is required"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	settings, err := database.GetESIndexSettings(ctx, req.ESConfig, req.Index)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Send(settings)
}

func esRefreshIndexHandler(c *fiber.Ctx) error {
	var req struct {
		database.ESConfig
		Index string `json:"index"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Index == "" {
		return c.Status(400).JSON(fiber.Map{"error": "index name is required"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := database.RefreshESIndex(ctx, req.ESConfig, req.Index); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Index refreshed"})
}

func esSearchHandler(c *fiber.Ctx) error {
	var req struct {
		database.ESConfig
		Index string `json:"index"`
		Query string `json:"query"`
		From  int    `json:"from"`
		Size  int    `json:"size"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Index == "" {
		return c.Status(400).JSON(fiber.Map{"error": "index name is required"})
	}

	if req.Size <= 0 {
		req.Size = 20
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := database.SearchESDocuments(ctx, req.ESConfig, req.Index, req.Query, req.From, req.Size)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(result)
}

func esGetDocumentHandler(c *fiber.Ctx) error {
	var req struct {
		database.ESConfig
		Index string `json:"index"`
		ID    string `json:"id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Index == "" || req.ID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "index and id are required"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	doc, err := database.GetESDocument(ctx, req.ESConfig, req.Index, req.ID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Send(doc)
}

func esDeleteDocumentHandler(c *fiber.Ctx) error {
	var req struct {
		database.ESConfig
		Index string `json:"index"`
		ID    string `json:"id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Index == "" || req.ID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "index and id are required"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := database.DeleteESDocument(ctx, req.ESConfig, req.Index, req.ID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Document deleted"})
}

func esQueryHandler(c *fiber.Ctx) error {
	var req struct {
		database.ESConfig
		Method string          `json:"method"`
		Path   string          `json:"path"`
		Body   json.RawMessage `json:"body,omitempty"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Method == "" {
		req.Method = "GET"
	}
	if req.Path == "" {
		req.Path = "/"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := database.ExecuteESQuery(ctx, req.ESConfig, req.Method, req.Path, req.Body)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(result)
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
