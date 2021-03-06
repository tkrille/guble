package server

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"

	"fmt"
	guble "github.com/smancke/guble/guble"
)

var aTestByteMessage = []byte("Hello World!")
var chanSize = 10

func TestAddAndRemoveRoutes(t *testing.T) {
	a := assert.New(t)

	// Given a Multiplexer
	router := NewPubSubRouter().Go()

	// when i add two routes in the same path
	channel := make(chan MsgAndRoute, chanSize)
	routeBlah1 := router.Subscribe(NewRoute("/blah", channel, "appid01", "user01"))
	routeBlah2 := router.Subscribe(NewRoute("/blah", channel, "appid02", "user01"))

	// and one route in another path
	routeFoo := router.Subscribe(NewRoute("/foo", channel, "appid01", "user01"))

	// then

	// the routes are stored
	a.Equal(2, len(router.routes[guble.Path("/blah")]))
	a.True(routeBlah1.equals(router.routes[guble.Path("/blah")][0]))
	a.True(routeBlah2.equals(router.routes[guble.Path("/blah")][1]))

	a.Equal(1, len(router.routes[guble.Path("/foo")]))
	a.True(routeFoo.equals(router.routes[guble.Path("/foo")][0]))

	// WHEN i remove routes
	router.Unsubscribe(routeBlah1)
	router.Unsubscribe(routeFoo)

	// then they are gone
	a.Equal(1, len(router.routes[guble.Path("/blah")]))
	a.True(routeBlah2.equals(router.routes[guble.Path("/blah")][0]))

	a.Nil(router.routes[guble.Path("/foo")])
}

func TestReplacingOfRoutes(t *testing.T) {
	a := assert.New(t)

	// Given a router with a route
	router := NewPubSubRouter().Go()
	router.Subscribe(NewRoute("/blah", nil, "appid01", "user01"))

	// when: i add another route with the same Application Id and Same Path
	router.Subscribe(NewRoute("/blah", nil, "appid01", "newUserId"))

	// then: the router only contains the new route
	a.Equal(1, len(router.routes))
	a.Equal(1, len(router.routes["/blah"]))
	a.Equal("newUserId", router.routes["/blah"][0].UserId)
}

func TestSimpleMessageSending(t *testing.T) {
	a := assert.New(t)

	// Given a Multiplexer with route
	router, r := aRouterRoute()

	// when i send a message to the route
	router.HandleMessage(&guble.Message{Path: r.Path, Body: aTestByteMessage})

	// then I can receive it a short time later
	assertChannelContainsMessage(a, r.C, aTestByteMessage)
}

func TestRoutingWithSubTopics(t *testing.T) {
	a := assert.New(t)

	// Given a Multiplexer with route
	router := NewPubSubRouter().Go()
	channel := make(chan MsgAndRoute, chanSize)
	r := router.Subscribe(NewRoute("/blah", channel, "appid01", "user01"))

	// when i send a message to a subroute
	router.HandleMessage(&guble.Message{Path: "/blah/blub", Body: aTestByteMessage})

	// then I can receive the message
	assertChannelContainsMessage(a, r.C, aTestByteMessage)

	// but, when i send a message to a resource, which is just a substring
	router.HandleMessage(&guble.Message{Path: "/blahblub", Body: aTestByteMessage})

	// then the message gets not delivered
	a.Equal(0, len(r.C))
}

func TestMatchesTopic(t *testing.T) {
	for _, test := range []struct {
		messagePath guble.Path
		routePath   guble.Path
		matches     bool
	}{
		{"/foo", "/foo", true},
		{"/foo/xyz", "/foo", true},
		{"/foo", "/bar", false},
		{"/fooxyz", "/foo", false},
		{"/foo", "/bar/xyz", false},
	} {
		if !test.matches == matchesTopic(test.messagePath, test.routePath) {
			t.Errorf("error: expected %v, but: matchesTopic(%q, %q) = %v", test.matches, test.messagePath, test.routePath, matchesTopic(test.messagePath, test.routePath))
		}
	}
}

func TestRouteIsRemovedIfChannelIsFull(t *testing.T) {
	a := assert.New(t)

	// Given a Multiplexer with route
	router, r := aRouterRoute()
	// where the channel is full of messages
	for i := 0; i < chanSize; i++ {
		router.HandleMessage(&guble.Message{Path: r.Path, Body: aTestByteMessage})
	}

	// when I send one more message
	done := make(chan bool, 1)
	go func() {
		router.HandleMessage(&guble.Message{Path: r.Path, Body: aTestByteMessage})
		done <- true
	}()

	// then: the it returns immediately
	select {
	case <-done:
	case <-time.After(time.Millisecond * 10):
		a.Fail("Not returning!")
	}

	time.Sleep(time.Millisecond * 1)

	// fetch messages from the channel
	for i := 0; i < chanSize; i++ {
		select {
		case _, open := <-r.C:
			a.True(open)
		case <-time.After(time.Millisecond * 10):
			a.Fail("error not enough messages in channel")
		}
	}

	// and the channel is closed
	select {
	case _, open := <-r.C:
		a.False(open)
	default:
		fmt.Printf("len(r.C): %v", len(r.C))
		a.Fail("channel was not closed")
	}
}

func aRouterRoute() (*PubSubRouter, *Route) {
	router := NewPubSubRouter().Go()
	return router, router.Subscribe(NewRoute("/blah", make(chan MsgAndRoute, chanSize), "appid01", "user01"))
}

func assertChannelContainsMessage(a *assert.Assertions, c chan MsgAndRoute, msg []byte) {
	//log.Println("DEBUG: start assertChannelContainsMessage-> select")
	select {
	case msgBack := <-c:
		a.Equal(string(msg), string(msgBack.Message.Body))
	case <-time.After(time.Millisecond):
		a.Fail("No message received")
	}
}
