package aelog

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/logging"
	"google.golang.org/api/option"
	mrpb "google.golang.org/genproto/googleapis/api/monitoredres"
	logtypepb "google.golang.org/genproto/googleapis/logging/type"
)

var ctxKey = struct{ k string }{"hlog context key"}

type ctxValue struct {
	parent string
	hreq   *http.Request
	logger *logging.Logger
}

func WrapHandler(h http.Handler, logName string, opts ...option.ClientOption) (http.Handler, error) {
	ctx := context.Background()

	project := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if project == "" {
		return nil, errors.New("aelog: GOOGLE_CLOUD_PROJECT not set")
	}
	parent := "projects/" + project

	lc, err := logging.NewClient(ctx, parent)
	if err != nil {
		return nil, err
	}
	if logName == "" {
		logName = "app_log"
	}
	logger := lc.Logger(logName, logging.CommonResource(&mrpb.MonitoredResource{
		Type: "gae_app",
		Labels: map[string]string{
			"module_id":  os.Getenv("GAE_SERVICE"),
			"version_id": os.Getenv("GAE_VERSION"),
			"project_id": project,
		},
	}))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), ctxKey, &ctxValue{
			parent: parent,
			hreq:   r,
			logger: logger,
		})
		h.ServeHTTP(w, r.WithContext(ctx))
	}), nil
}

func logFromContext(ctx context.Context, sev logtypepb.LogSeverity, format string, args ...interface{}) {
	cv := ctx.Value(ctxKey)
	if cv == nil {
		// Handler wasn't wrapped.
		return
	}

	cvt := cv.(*ctxValue)

	e := logging.Entry{
		Timestamp: time.Now(),
		Severity:  logging.Severity(sev),
		Payload:   fmt.Sprintf(format, args...),
	}
	traceHeader := cvt.hreq.Header.Get("X-Cloud-Trace-Context")
	if traceHeader != "" {
		e.Trace = cvt.parent + "/traces/" + strings.Split(traceHeader, "/")[0]
	}
	cvt.logger.Log(e)
}

func Criticalf(ctx context.Context, format string, args ...interface{}) {
	logFromContext(ctx, logtypepb.LogSeverity_CRITICAL, format, args...)
}

func Debugf(ctx context.Context, format string, args ...interface{}) {
	logFromContext(ctx, logtypepb.LogSeverity_DEBUG, format, args...)
}

func Errorf(ctx context.Context, format string, args ...interface{}) {
	logFromContext(ctx, logtypepb.LogSeverity_ERROR, format, args...)
}

func Infof(ctx context.Context, format string, args ...interface{}) {
	logFromContext(ctx, logtypepb.LogSeverity_INFO, format, args...)
}

func Warningf(ctx context.Context, format string, args ...interface{}) {
	logFromContext(ctx, logtypepb.LogSeverity_WARNING, format, args...)
}
