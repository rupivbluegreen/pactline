package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rupivbluegreen/pactline/internal/api"
	"github.com/rupivbluegreen/pactline/internal/api/handlers"
	"github.com/rupivbluegreen/pactline/internal/auth"
	"github.com/rupivbluegreen/pactline/internal/database"
	"github.com/rupivbluegreen/pactline/internal/email"
	"github.com/rupivbluegreen/pactline/internal/organizations"
)

func isInfraUnavailable(err error) bool {
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "i/o timeout")
}

func TestTenantIsolation_TwoUsersTwoOrgs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := database.Connect(ctx, database.DSNFromEnv())
	if err != nil {
		if isInfraUnavailable(err) {
			t.Skipf("db unavailable: %v", err)
		}
		t.Fatalf("connect: %v", err)
	}
	if err := database.MigrateUp(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	defer pool.Close()

	users := database.NewUserRepo(pool)
	orgs := database.NewOrganizationRepo(pool)
	mems := database.NewMembershipRepo(pool)
	sess := database.NewSessionRepo(pool)
	links := database.NewMagicLinkRepo(pool)
	authSvc := auth.NewService(pool, users, links, sess, email.NewSender(), "http://localhost:8000")
	orgSvc := organizations.NewService(pool)

	router := api.Router(api.Deps{
		Sessions: sess, Memberships: mems,
		Auth: &handlers.AuthHandlers{Svc: authSvc},
		Me:   &handlers.MeHandler{Users: users, Memberships: mems, Orgs: orgs},
		Orgs: &handlers.OrgsHandlers{Svc: orgSvc},
		Contracts: &handlers.ContractsHandler{
			Pool:              pool,
			Contracts:         database.NewContractRepo(pool),
			ContractTypes:     database.NewContractTypeRepo(pool),
			ContractDocuments: database.NewContractDocumentRepo(pool),
			ExtractedFields:   database.NewExtractedFieldRepo(pool),
		},
	})

	suffix1 := uuid.NewString()[:8]
	suffix2 := uuid.NewString()[:8]
	user1 := setupUserWithOrg(t, ctx, users, sess, orgSvc, "iso1-"+suffix1+"@example.com", "Org One "+suffix1)
	user2 := setupUserWithOrg(t, ctx, users, sess, orgSvc, "iso2-"+suffix2+"@example.com", "Org Two "+suffix2)

	me1 := callMe(t, router, user1.token)
	if !strings.Contains(me1, "Org One "+suffix1) || strings.Contains(me1, "Org Two "+suffix2) {
		t.Errorf("user1 /me leaked another org's data: %s", me1)
	}
	me2 := callMe(t, router, user2.token)
	if !strings.Contains(me2, "Org Two "+suffix2) || strings.Contains(me2, "Org One "+suffix1) {
		t.Errorf("user2 /me leaked another org's data: %s", me2)
	}

	c1 := callContracts(t, router, user1.token)
	if !strings.Contains(c1, `"contracts":[]`) {
		t.Errorf("user1 contracts unexpected: %s", c1)
	}
}

type seedUser struct {
	token database.SessionToken
}

func setupUserWithOrg(
	t *testing.T, ctx context.Context,
	users *database.UserRepo, sess *database.SessionRepo,
	orgSvc *organizations.Service, emailSeed, orgName string,
) seedUser {
	t.Helper()
	u, _, err := users.CreateOrGetByEmail(ctx, emailSeed)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	if _, err := orgSvc.CreateForUser(ctx, u.ID, orgName, ""); err != nil {
		t.Fatalf("org: %v", err)
	}
	tok, _, err := sess.Create(ctx, u.ID, time.Hour)
	if err != nil {
		t.Fatalf("session: %v", err)
	}
	return seedUser{token: tok}
}

func callMe(t *testing.T, router http.Handler, tok database.SessionToken) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+string(tok))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("/me code: %d body: %s", rec.Code, rec.Body.String())
	}
	var pretty map[string]any
	_ = json.NewDecoder(rec.Body).Decode(&pretty)
	b, _ := json.Marshal(pretty)
	return string(b)
}

func callContracts(t *testing.T, router http.Handler, tok database.SessionToken) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/contracts", nil)
	req.Header.Set("Authorization", "Bearer "+string(tok))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("/contracts code: %d body: %s", rec.Code, rec.Body.String())
	}
	return rec.Body.String()
}
