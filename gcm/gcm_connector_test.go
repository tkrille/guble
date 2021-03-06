package gcm

import (
	"github.com/smancke/guble/guble"
	"github.com/smancke/guble/server"
	"github.com/smancke/guble/store"

	"github.com/golang/mock/gomock"
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"

	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

var ctrl *gomock.Controller

func TestPostMessage(t *testing.T) {
	defer initCtrl(t)()

	a := assert.New(t)

	// given:  a rest api with a message sink
	routerMock := NewMockPubSubSource(ctrl)
	routerMock.EXPECT().Subscribe(gomock.Any()).Do(func(route *server.Route) {
		a.Equal("/notifications", string(route.Path))
		a.Equal("marvin", route.UserId)
		a.Equal("gcmId123", route.ApplicationId)
	})

	kvStore := store.NewMemoryKVStore()
	gcm := NewGCMConnector("/gcm/", "testApi")
	gcm.SetRouter(routerMock)
	gcm.SetKVStore(kvStore)

	url, _ := url.Parse("http://localhost/gcm/marvin/gcmId123/subscribe/notifications")
	// and a http context
	req := &http.Request{URL: url}
	w := httptest.NewRecorder()

	params := httprouter.Params{
		httprouter.Param{Key: "userid", Value: "marvin"},
		httprouter.Param{Key: "gcmid", Value: "gcmId123"},
		httprouter.Param{Key: "topic", Value: "/notifications"},
	}

	// when: I POST a message
	gcm.Subscribe(w, req, params)

	// the the result
	a.Equal("registered: /notifications\n", string(w.Body.Bytes()))
}

func TestSaveAndLoadSubscriptions(t *testing.T) {
	defer initCtrl(t)()
	defer enableDebugForMethod()()
	a := assert.New(t)

	// given: some test routes
	testRoutes := map[string]bool{
		"marvin:/foo:1234": true,
		"zappod:/bar:1212": true,
		"athur:/erde:42":   true,
	}

	routerMock := NewMockPubSubSource(ctrl)
	routerMock.EXPECT().Subscribe(gomock.Any()).Do(func(route *server.Route) {
		// delte the route from the map, if we got it in the test
		delete(testRoutes, fmt.Sprintf("%v:%v:%v", route.UserId, route.Path, route.ApplicationId))
	}).AnyTimes()

	kvStore := store.NewMemoryKVStore()
	gcm := NewGCMConnector("/gcm/", "testApi")
	gcm.SetRouter(routerMock)
	gcm.SetKVStore(kvStore)

	// when: we save the routes
	for k, _ := range testRoutes {
		splitedKey := strings.SplitN(k, ":", 3)
		userid := splitedKey[0]
		topic := splitedKey[1]
		gcmid := splitedKey[2]
		gcm.saveSubscription(userid, topic, gcmid)
	}

	// and reload the routes
	gcm.loadSubscriptions()

	time.Sleep(time.Millisecond * 100)

	// than: all expected subscriptions were called
	a.Equal(0, len(testRoutes))
}

func TestRemoveTailingSlash(t *testing.T) {
	assert.Equal(t, "/foo", removeTrailingSlash("/foo/"))
	assert.Equal(t, "/foo", removeTrailingSlash("/foo"))
}

func initCtrl(t *testing.T) func() {
	ctrl = gomock.NewController(t)
	return func() { ctrl.Finish() }
}

func enableDebugForMethod() func() {
	reset := guble.LogLevel
	guble.LogLevel = guble.LEVEL_DEBUG
	return func() { guble.LogLevel = reset }
}
