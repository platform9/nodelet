package authn

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"

	bouncer "github.com/platform9/pf9-qbert/bouncer/pkg/api"
	"github.com/platform9/pf9-qbert/bouncer/pkg/cache"
	"github.com/platform9/pf9-qbert/bouncer/pkg/policy"
	"github.com/platform9/pf9-qbert/bouncer/pkg/utils"
)

var slowRequestWebhook = os.Getenv("BOUNCER_SLOW_REQUEST_WEBHOOK")

// authenticator holds the webhook state
type authenticator struct {
	keystone   bouncer.Keystone
	projectID  string
	cache      *cache.LRUExpireCache
	authTTL    time.Duration
	unauthTTL  time.Duration
	bcryptCost int
	mapper     *policy.RoleMapper
}

type credentialsCacheEntry struct {
	HashedPassword []byte
	Status         TokenReviewStatus
}

// New returns an initialized Authenticator http Handler
func New(keystone bouncer.Keystone, projectID string, authTTL, unauthTTL time.Duration, cacheSize, bcryptCost int, mapper *policy.RoleMapper) (*authenticator, error) {
	cache, err := cache.NewLRUExpireCache(cacheSize)
	if err != nil {
		return nil, err
	}
	return &authenticator{keystone, projectID, cache, authTTL, unauthTTL, bcryptCost, mapper}, nil
}

// Accepts a TokenReview request. Determines whether the token payload is a
// Keystone token ID, or credentials. Authenticates with Keystone using the token
// payload.  Returns a TokenReview with the Status field populated and the Spec
// field omitted.
// (ServeHTTP implements the http.Handler interface)
func (a *authenticator) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		log.Println("incorrect HTTP method:", r.Method)
		return
	}

	review := TokenReview{}
	if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		log.Println("decode tokenreview request: (error message not logged)")
		return
	}
	if err := review.ValidateRequest(); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		log.Println("validate tokenreview request: (error message not logged)")
		return
	}

	webhookToken := review.Spec.Token
	if IsKeystoneTokenID(webhookToken) {
		a.handleTokenID(webhookToken, &review)
	} else {
		username, password, err := Credentials(webhookToken)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			log.Println("decode credentials: (error message not logged)")
			return
		}
		start := time.Now()
		a.handleCredentials(username, password, &review)
		elapsed := time.Now().Sub(start)
		if elapsed > time.Second*30 && slowRequestWebhook != "" {
			fmtStr := "host=%s du=%s cluster=%s: authentication took too long"
			msg := fmt.Sprintf(fmtStr,
				os.Getenv("HOST_NAME"),
				os.Getenv("DU_FQDN"),
				os.Getenv("CLUSTER_ID"),
			)
			utils.PostToSlackBestEffort(slowRequestWebhook, msg)
		}
	}
	review.Spec = nil

	if err := review.ValidateResponse(); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		log.Println("validate tokenreview response: (error message not logged)")
		return
	}
	enc := json.NewEncoder(w)
	if err := enc.Encode(review); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		log.Println("encode tokenreview response: (error message not logged)")
		return
	}
}

// Fills user group information in status
// Gets groups from role to group mapping and Keystone
func (a *authenticator) fillMappedGroups(status *TokenReviewStatus, tokenWrapper *bouncer.KeystoneTokenWrapper) {
	groupsMappedFromRole := a.mapper.GetGroupsFromTokenRole(&tokenWrapper.Token)
	log.Println("User groups from roles to group mapping = ", groupsMappedFromRole)
	status.User.Groups = append(status.User.Groups, groupsMappedFromRole...)

	keystoneGroups, err := a.keystone.GroupsFromProjectToken(tokenWrapper)
	if err == nil {
		log.Println("User groups from keystone = ", keystoneGroups)
		status.User.Groups = append(status.User.Groups, keystoneGroups...)
	}
	log.Println("All user groups", status.User.Groups)
}

// handleCredentials populates the TokenReviewStatus in the webhook response.
// The Status cached under the username is used if the hashed password matches
// the hashed password in the cache; if the hashed passwords do not match, or
// if no cache entry exists, the webhook calls out to Keystone, creates the
// Status and caches it.
func (a *authenticator) handleCredentials(username, password string, review *TokenReview) {
	if e, ok := a.cache.Get(username); ok {
		// Assume an entry stored under a username key has a value of type `credentialsCacheEntry`
		ce, ok := e.(credentialsCacheEntry)
		if ok {
			if err := bcrypt.CompareHashAndPassword(ce.HashedPassword, []byte(password)); err == nil {
				review.Status = &ce.Status
				return
			} // Else hashed passwords do not match
		} else {
			log.Println("type convert credentials cache entry: incorrect type")
		}
	}
	status := TokenReviewStatus{}
	var ttl time.Duration
	if tokenWrapper, err := a.keystone.ProjectTokenFromCredentialsWithProjectId(
		username,
		password,
		a.projectID,
	); err != nil {
		log.Println("authn with credentials:", err)
		status.Authenticated = false
		ttl = a.unauthTTL
		if err, ok := err.(bouncer.KeystoneResponseError); !ok || err.StatusCode != http.StatusUnauthorized {
			// The Keystone client returned an error, but not one caused by a 401 response from Keystone
			review.Status = &status
			return
		}
	} else {
		status.Authenticated = true
		status.User = &TokenReviewStatusUser{
			Username: tokenWrapper.Token.User.Name,
			UID:      tokenWrapper.Token.User.ID,
		}
		a.fillMappedGroups(&status, &tokenWrapper)
		ttl = a.authTTL
	}
	review.Status = &status
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), a.bcryptCost)
	if err != nil {
		log.Println("cannot add to cache: hash password:", err)
		return
	}
	a.cache.Add(username, credentialsCacheEntry{hashedPassword, status}, ttl)
}

// handleTokenID populates the TokenReviewStatus in the webhook response. The
// Status cached under the tokenID is used; if no cache entry exists, the
// webhook calls out to Keystone, creates the Status and caches it.
func (a *authenticator) handleTokenID(tokenID string, review *TokenReview) {
	if e, ok := a.cache.Get(tokenID); ok {
		// Assume an entry stored under a tokenID key has a value of type `TokenReviewStatus`
		status, ok := e.(TokenReviewStatus)
		if ok {
			review.Status = &status
			return
		} else {
			log.Println("type convert tokenID cache entry: incorrect type")
		}
	}
	status := TokenReviewStatus{}
	var ttl time.Duration
	if tokenWrapper, err := a.keystone.ProjectTokenFromTokenID(tokenID, a.projectID); err != nil {
		log.Println("authn with tokenID:", err)
		status.Authenticated = false
		ttl = a.unauthTTL
		if err, ok := err.(bouncer.KeystoneResponseError); !ok || err.StatusCode != http.StatusUnauthorized {
			// The Keystone client returned an error, but not one caused by a 401 response from Keystone
			review.Status = &status
			return
		}
	} else {
		status.Authenticated = true
		status.User = &TokenReviewStatusUser{
			Username: tokenWrapper.Token.User.Name,
			UID:      tokenWrapper.Token.User.ID,
		}
		a.fillMappedGroups(&status, &tokenWrapper)
		ttl = a.authTTL
	}
	review.Status = &status
	a.cache.Add(tokenID, status, ttl)
}
