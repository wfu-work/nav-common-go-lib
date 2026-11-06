package sse

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPAndStorageAdaptersCorrelateWithoutBodiesOrSQL(t *testing.T) {
	r := newTestRecorder(t, Options{})
	router := gin.New()
	router.Use(r.HTTPMiddleware())
	router.POST("/devices/:id", func(c *gin.Context) {
		e := FromContext(c.Request.Context())
		if e.TraceID == "" || e.RequestID == "" {
			t.Error("missing correlation")
		}
		r.GormLogger().Trace(c.Request.Context(), time.Now(), func() (string, int64) { t.Error("SQL interpolation called"); return "password=private-body", 0 }, errors.New("failed SQL password=private-body"))
		c.Status(503)
	})
	req := httptest.NewRequest("POST", "/devices/actual-id?token=private-query", strings.NewReader("private-body"))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	b := r.Snapshot(Filter{}, "", 200)
	if len(b.Events) != 2 || b.Events[0].TraceID != b.Events[1].TraceID || b.Events[1].Fields["route"] != "/devices/:id" {
		t.Fatalf("bad adapters %+v", b)
	}
	for _, e := range b.Events {
		if strings.Contains(e.Message, "private-") {
			t.Fatal("raw body leaked")
		}
	}
	r.GormLogger().Trace(Quiet(context.Background()), time.Now().Add(-time.Second), nil, errors.New("hidden"))
	if len(r.Snapshot(Filter{}, "", 200).Events) != 2 {
		t.Fatal("log query generated recursive events")
	}
}
