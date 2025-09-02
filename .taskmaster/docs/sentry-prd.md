# Product Requirements Document: Sentry.io Integration

## 1. Introduction

This document outlines the requirements for integrating the Sentry.io error tracking and performance monitoring platform into the Pi-Controller application. This integration will cover both the Go backend (`pi-controller`, `pi-agent`) and the React frontend web UI.

## 2. Rationale

To ensure the stability, reliability, and performance of the Pi-Controller platform, a robust monitoring solution is required. Integrating Sentry.io will provide the following benefits:

*   **Proactive Error Detection:** Automatically capture and report unhandled errors and panics in real-time.
*   **Improved Debugging:** Gain rich context (stack traces, breadcrumbs, request data) for every issue, reducing time-to-resolution.
*   **Performance Monitoring:** Identify performance bottlenecks in both backend API endpoints and frontend user interactions.
*   **Release Health:** Track the stability of new releases and quickly identify regressions.

## 3. Scope

This project covers the full, production-ready integration of Sentry.io across the application stack.

*   **Backend:** Integration with the `sentry-go` SDK for the `pi-controller` and `pi-agent` services.
*   **Frontend:** Integration with the `@sentry/react` SDK for the web UI.
*   **Configuration:** Secure and flexible configuration of Sentry settings.

## 4. Requirements

### 4.1. Backend Integration (Go)

1.  **SDK Installation:** Add the `getsentry/sentry-go` and `sentry-gin` packages to the project dependencies.
2.  **Configuration:**
    *   Add a `sentry` section to the application `config.go` file.
    *   This section must include fields for `dsn`, `environment`, `release`, and `debug`.
    *   The Sentry DSN must be configurable via an environment variable (`SENTRY_DSN`) and should not be hardcoded.
    *   The application must run without errors if the DSN is not provided.
3.  **Initialization:**
    *   Initialize the Sentry SDK in the `main()` function of both `cmd/pi-controller/main.go` and `cmd/pi-agent/main.go`.
    *   The `release` should be dynamically set, ideally based on the application version (`app.Version`).
4.  **Error & Panic Capturing:**
    *   Integrate the `sentry-gin` middleware into the Gin router in `internal/api/server.go`.
    *   This middleware must recover from any panics, report them to Sentry, and return a 500 error to the user.
    *   It should also add request context (URL, method, headers, etc.) to Sentry events.
5.  **Performance Monitoring:**
    *   The `sentry-gin` middleware should be configured to enable performance tracing for all API requests.
    *   This will create a transaction for each incoming HTTP request.
6.  **Logging Integration:**
    *   Integrate Sentry with the existing `logrus` logger. Errors logged at `log.Error` or `log.Fatal` levels should be captured as Sentry events.

### 4.2. Frontend Integration (React)

1.  **SDK Installation:** Add the `@sentry/react` and `@sentry/tracing` packages to the `package.json` dependencies in the `web/` directory.
2.  **Configuration:**
    *   The Sentry DSN, environment, and release version must be configurable via environment variables (e.g., `REACT_APP_SENTRY_DSN`).
3.  **Initialization:**
    *   Initialize the Sentry SDK in the main entry point of the React application (e.g., `web/src/index.js`).
4.  **Error Capturing:**
    *   Wrap the main React component tree with a Sentry `<ErrorBoundary>` component to automatically capture any rendering errors.
5.  **Performance Monitoring:**
    *   Enable performance monitoring by adding the `BrowserTracing` integration during Sentry initialization. This will automatically trace page loads and navigation.

### 4.3. Documentation

1.  **Update `.env.example`:** Add `SENTRY_DSN=""` to the example environment file.
2.  **Update `getting-started.md`:** Add a section explaining how to configure Sentry for local development by setting the `SENTRY_DSN` environment variable.

## 5. Out of Scope

*   Setting up Sentry alerting rules, dashboards, or integrations with third-party services like Slack or Jira.
*   Implementing custom Sentry instrumentation beyond the standard middleware-based performance monitoring.
*   User-specific tracking or session replay features.

## 6. Acceptance Criteria

*   When a panic occurs in a backend API handler, it is captured in Sentry with a full stack trace.
*   When a frontend component fails to render, the error is captured in Sentry.
*   Performance data for backend API endpoints (e.g., `GET /api/v1/clusters`) is visible in the Sentry dashboard.
*   The application starts and runs correctly without any Sentry DSN configured.
*   The `README.md` or `getting-started.md` clearly explains how to enable Sentry.
