package authentication

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/pkg/errors"

	"github.com/openmultiplayer/web/server/src/db"
	"github.com/openmultiplayer/web/server/src/web"
)

// State stores state for performing authentication
type State struct {
	db *db.PrismaClient
	sc *securecookie.SecureCookie
}

// OAuthProvider describes a type that can provide an OAuth2 authentication
// method for users.
//
// Link simply returns a URL to start the OAuth2 process.
//
// Login is called by the callback and handles the code/token exchange and
// returns a User object to the caller to be encoded into a cookie.
type OAuthProvider interface {
	Link() string
	Login(ctx context.Context, state, code string) (*db.UserModel, error)
}

// New initialises a new authentication service
func New(
	db *db.PrismaClient,
	hashKey,
	blockKey []byte,
) *State {
	a := &State{
		db: db,
		sc: securecookie.New(hashKey, blockKey),
	}

	return a
}

// EncodeAuthCookie writes the secure user auth cookie to the response writer.
func (a *State) EncodeAuthCookie(w http.ResponseWriter, user db.UserModel) {
	encoded, err := a.sc.Encode(secureCookieName, Cookie{
		UserID:  user.ID,
		Created: time.Now(),
	})
	if err != nil {
		web.StatusUnauthorized(w, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     secureCookieName,
		Value:    encoded,
		Path:     "/",
		Domain:   ".open.mp",
		Secure:   true,
		HttpOnly: false,
	})
}

// GetAuthenticationInfo extracts auth info from a request context and, if not
// present, will write a 500 error to the response and return not-ok. In this
// failure case, the request should be immediately terminated.
func GetAuthenticationInfo(
	w http.ResponseWriter,
	r *http.Request,
) (*Info, bool) {
	if auth, ok := GetAuthenticationInfoFromContext(r.Context()); ok {
		return auth, true
	}
	web.StatusInternalServerError(w, web.WithSuggestion(
		errors.New("failed to extract auth context from request"),
		"Could not read session data from cookies.",
		"Try clearing your cookies and logging in to your account again."))
	return nil, false
}

// GetAuthenticationInfoFromContext pulls out auth data from a request context
func GetAuthenticationInfoFromContext(ctx context.Context) (*Info, bool) {
	if auth, ok := ctx.Value(contextKey).(Info); ok {
		return &auth, true
	}
	return nil, false
}
