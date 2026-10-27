package handlers

import (
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/tigerwallet/admin/internal/models"
	"gorm.io/gorm/schema"
)

// --- billing subject resolution -------------------------------------------

// Billing endpoints live behind the admin auth middleware, which only sets
// "admin_id". They previously read a "user_id" context value that no
// middleware ever set, so every call returned 401.

func TestBillingSubjectResolvesFromQueryParam(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/api/v1/billing/invoices?user_id=user-42", nil)

	got, ok := billingSubject(c)
	if !ok || got != "user-42" {
		t.Fatalf("billingSubject = (%q, %v), want (user-42, true)", got, ok)
	}
}

func TestBillingSubjectFallsBackToAuthenticatedAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/api/v1/billing/invoices", nil)
	c.Set("admin_id", uint(7))

	got, ok := billingSubject(c)
	if !ok || got != "7" {
		t.Fatalf("billingSubject = (%q, %v), want (7, true)", got, ok)
	}
}

func TestBillingSubjectRejectsAnonymousRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/api/v1/billing/invoices", nil)

	if got, ok := billingSubject(c); ok {
		t.Fatalf("billingSubject = (%q, true), want not-ok for anonymous request", got)
	}
}

// --- column naming contract ----------------------------------------------

// gormColumnName returns the column a field maps to: an explicit
// `gorm:"column:..."` tag wins, otherwise GORM's default naming strategy is
// used. Handlers that reference these columns in raw SQL or update maps must
// agree with this value, otherwise the query fails at runtime.
func gormColumnName(t *testing.T, model any, field string) string {
	t.Helper()

	rt := reflect.TypeOf(model)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	sf, ok := rt.FieldByName(field)
	if !ok {
		t.Fatalf("%s has no field %s", rt.Name(), field)
	}

	tag := sf.Tag.Get("gorm")
	for _, part := range strings.Split(tag, ";") {
		if strings.HasPrefix(part, "column:") {
			return strings.TrimPrefix(part, "column:")
		}
	}

	return schema.NamingStrategy{}.ColumnName("", field)
}

// TestPersistedColumnNamesMatchHandlers pins the database column for every
// field whose Go name ends in a digit sequence. GORM does not insert an
// underscore before digits ("Volume24h" -> "volume24h"), while the handlers
// reference the natural snake_case spelling ("volume_24h"), so each of these
// needs an explicit column tag to stay in sync.
func TestPersistedColumnNamesMatchHandlers(t *testing.T) {
	cases := []struct {
		model any
		field string
		want  string
	}{
		{Feature{}, "RolloutPercent", "rollout_percent"},
		{LiquidityPool{}, "Volume24h", "volume_24h"},
		{LiquidityPool{}, "Fees24h", "fees_24h"},
		{models.TradingPair{}, "Volume24h", "volume_24h"},
		{models.TradingPair{}, "PriceChange24h", "price_change_24h"},
		{models.Token{}, "Volume24h", "volume_24h"},
		{models.Token{}, "PriceChange24h", "price_change_24h"},
	}

	for _, tc := range cases {
		got := gormColumnName(t, tc.model, tc.field)
		if got != tc.want {
			rt := reflect.TypeOf(tc.model)
			t.Errorf("%s.%s maps to column %q, want %q",
				rt.Name(), tc.field, got, tc.want)
		}
	}
}

// --- RBAC key type consistency -------------------------------------------

// TestRoleAssignmentAdminIDMatchesAdminPrimaryKey guards the join key type.
// admins.id is a bigint (uint in Go); an earlier version declared
// admin_role_assignments.admin_id as uuid, which made every role assignment
// fail against a bigint primary key.
func TestRoleAssignmentAdminIDMatchesAdminPrimaryKey(t *testing.T) {
	adminID := reflect.TypeOf(models.Admin{}).Field(0)
	if adminID.Name != "ID" {
		t.Fatalf("models.Admin field 0 is %q, expected the primary key ID", adminID.Name)
	}

	assignment := reflect.TypeOf(AdminRoleAssignment{})
	field, ok := assignment.FieldByName("AdminID")
	if !ok {
		t.Fatal("AdminRoleAssignment has no AdminID field")
	}

	if field.Type.Kind() != adminID.Type.Kind() {
		t.Fatalf("AdminRoleAssignment.AdminID is %s but models.Admin.ID is %s; "+
			"the role assignment join would fail", field.Type, adminID.Type)
	}

	// granted_by references the acting admin, so it must use the same key type.
	grantedBy, ok := assignment.FieldByName("GrantedBy")
	if !ok {
		t.Fatal("AdminRoleAssignment has no GrantedBy field")
	}
	if grantedBy.Type.Kind() != reflect.Ptr || grantedBy.Type.Elem().Kind() != adminID.Type.Kind() {
		t.Fatalf("AdminRoleAssignment.GrantedBy is %s, want pointer to %s",
			grantedBy.Type, adminID.Type)
	}
}
